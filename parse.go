package xmlquery

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/antchfx/xpath"
	"golang.org/x/net/html/charset"
)

var xmlMIMERegex = regexp.MustCompile(`(?i)((application|image|message|model)/((\w|\.|-)+\+?)?|text/)(wb)?xml`)

// LoadURL loads the XML document from the specified URL.
func LoadURL(url string) (*Node, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// Make sure the Content-Type has a valid XML MIME type
	if xmlMIMERegex.MatchString(resp.Header.Get("Content-Type")) {
		return Parse(resp.Body)
	}
	return nil, fmt.Errorf("invalid XML document(%s)", resp.Header.Get("Content-Type"))
}

// Parse returns the parse tree for the XML from the given Reader.
func Parse(r io.Reader) (*Node, error) {
	return ParseWithOptions(r, ParserOptions{})
}

// ParseWithOptions is like parse, but with custom options
func ParseWithOptions(r io.Reader, options ParserOptions) (*Node, error) {
	var data []byte
	var lineStarts []int

	// If line numbers are requested, read all data for position tracking
	if options.WithLineNumbers {
		var err error
		data, err = io.ReadAll(r)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(data)

		// Pre-calculate line starts
		lineStarts = []int{0}
		for i, b := range data {
			if b == '\n' {
				lineStarts = append(lineStarts, i+1)
			}
		}
	}

	p := createParser(r)
	options.apply(p)
	var err error
	for err == nil {
		_, err = p.parse()
	}

	if err == io.EOF {
		// additional check for validity
		// according to: https://www.w3.org/TR/xml
		// the document MUST contain at least ONE element
		valid := false
		for doc := p.doc; doc != nil; doc = doc.NextSibling {
			for node := doc.FirstChild; node != nil; node = node.NextSibling {
				if node.Type == ElementNode {
					valid = true
					break
				}
			}
		}
		if !valid {
			return nil, fmt.Errorf("xmlquery: invalid XML document")
		}

		// If line numbers were requested, annotate the parsed document
		if options.WithLineNumbers {
			annotator := &lineNumberAnnotator{
				data:       data,
				lineStarts: lineStarts,
			}

			err = annotator.annotateLineNumbers(p.doc)
			if err != nil {
				return nil, err
			}
		}

		return p.doc, nil
	}

	return nil, err
}

type parser struct {
	decoder             *xml.Decoder
	doc                 *Node
	level               int
	prev                *Node
	streamElementXPath  *xpath.Expr   // Under streaming mode, this specifies the xpath to the target element node(s).
	streamElementFilter *xpath.Expr   // If specified, it provides further filtering on the target element.
	streamNode          *Node         // Need to remember the last target node So we can clean it up upon next Read() call.
	streamNodePrev      *Node         // Need to remember target node's prev so upon target node removal, we can restore correct prev.
	reader              *cachedReader // Need to maintain a reference to the reader, so we can determine whether a node contains CDATA.
	once                sync.Once
	space2prefix        map[string]*xmlnsPrefix
	currentLine         int // Track current line number during parsing
	lastProcessedPos    int // Track how much cached data we've already processed for line counting
}

type xmlnsPrefix struct {
	name  string
	level int
}

func createParser(r io.Reader) *parser {
	reader := newCachedReader(bufio.NewReader(r))
	p := &parser{
		decoder:          xml.NewDecoder(reader),
		doc:              &Node{Type: DocumentNode},
		level:            0,
		reader:           reader,
		currentLine:      0,
		lastProcessedPos: 0,
	}
	if p.decoder.CharsetReader == nil {
		p.decoder.CharsetReader = charset.NewReaderLabel
	}
	p.prev = p.doc
	return p
}

// updateLineNumber scans only new cached data for newlines to update current line position
func (p *parser) updateLineNumber() {
	cached := p.reader.CacheWithLimit(-1) // Get all cached data

	// Only process data we haven't seen before
	for i := p.lastProcessedPos; i < len(cached); i++ {
		if cached[i] == '\n' {
			p.currentLine++
		}
	}

	// Update our position to avoid reprocessing this data
	p.lastProcessedPos = len(cached)
}

func (p *parser) parse() (*Node, error) {
	p.once.Do(func() {
		p.space2prefix = map[string]*xmlnsPrefix{"http://www.w3.org/XML/1998/namespace": {name: "xml", level: 0}}
	})

	var streamElementNodeCounter int
	for {
		p.reader.StartCaching()
		tok, err := p.decoder.Token()
		p.reader.StopCaching()

		// Update line number based on processed content
		p.updateLineNumber()

		if err != nil {
			return nil, err
		}

		switch tok := tok.(type) {
		case xml.StartElement:
			if p.level == 0 {
				// mising XML declaration
				attributes := make([]Attr, 1)
				attributes[0].Name = xml.Name{Local: "version"}
				attributes[0].Value = "1.0"
				node := &Node{
					Type:       DeclarationNode,
					Data:       "xml",
					Attr:       attributes,
					level:      1,
					LineNumber: p.currentLine,
				}
				AddChild(p.prev, node)
				p.level = 1
				p.prev = node
			}

			for _, att := range tok.Attr {
				if att.Name.Local == "xmlns" {
					// https://github.com/antchfx/xmlquery/issues/67
					if prefix, ok := p.space2prefix[att.Value]; !ok || (ok && prefix.level >= p.level) {
						p.space2prefix[att.Value] = &xmlnsPrefix{name: "", level: p.level} // reset empty if exist the default namespace
					}
				} else if att.Name.Space == "xmlns" {
					// maybe there are have duplicate NamespaceURL?
					p.space2prefix[att.Value] = &xmlnsPrefix{name: att.Name.Local, level: p.level}
				}
			}

			if space := tok.Name.Space; space != "" {
				if _, found := p.space2prefix[space]; !found && p.decoder.Strict {
					return nil, fmt.Errorf("xmlquery: invalid XML document, namespace %s is missing", space)
				}
			}

			attributes := make([]Attr, len(tok.Attr))
			for i, att := range tok.Attr {
				name := att.Name
				if prefix, ok := p.space2prefix[name.Space]; ok {
					name.Space = prefix.name
				}
				attributes[i] = Attr{
					Name:         name,
					Value:        att.Value,
					NamespaceURI: att.Name.Space,
				}
			}

			node := &Node{
				Type:         ElementNode,
				Data:         tok.Name.Local,
				NamespaceURI: tok.Name.Space,
				Attr:         attributes,
				level:        p.level,
				LineNumber:   p.currentLine,
			}

			if p.level == p.prev.level {
				AddSibling(p.prev, node)
			} else if p.level > p.prev.level {
				AddChild(p.prev, node)
			} else if p.level < p.prev.level {
				for i := p.prev.level - p.level; i > 1; i-- {
					p.prev = p.prev.Parent
				}
				AddSibling(p.prev.Parent, node)
			}

			if node.NamespaceURI != "" {
				if v, ok := p.space2prefix[node.NamespaceURI]; ok {
					cached := string(p.reader.CacheWithLimit(len(v.name) + len(node.Data) + 2))
					if strings.HasPrefix(cached, fmt.Sprintf("%s:%s", v.name, node.Data)) || strings.HasPrefix(cached, fmt.Sprintf("<%s:%s", v.name, node.Data)) {
						node.Prefix = v.name
					}
				}
			}
			// If we're in the streaming mode, we need to remember the node if it is the target node
			// so that when we finish processing the node's EndElement, we know how/what to return to
			// caller. Also we need to remove the target node from the tree upon next Read() call so
			// memory doesn't grow unbounded.
			if p.streamElementXPath != nil {
				if p.streamNode == nil {
					if QuerySelector(p.doc, p.streamElementXPath) != nil {
						p.streamNode = node
						p.streamNodePrev = p.prev
						streamElementNodeCounter = 1
					}
				} else {
					streamElementNodeCounter++
				}
			}
			p.prev = node
			p.level++
		case xml.EndElement:
			p.level--
			// If we're in streaming mode, and we already have a potential streaming
			// target node identified (p.streamNode != nil) then we need to check if
			// this is the real one we want to return to caller.
			if p.streamNode != nil {
				streamElementNodeCounter--
				if streamElementNodeCounter == 0 {
					// Now we know this element node is the at least passing the initial
					// p.streamElementXPath check and is a potential target node candidate.
					// We need to have 1 more check with p.streamElementFilter (if given) to
					// ensure it is really the element node we want.
					// The reason we need a two-step check process is because the following
					// situation:
					//   <AAA><BBB>b1</BBB></AAA>
					// And say the p.streamElementXPath = "/AAA/BBB[. != 'b1']". Now during
					// xml.StartElement time, the <BBB> node is still empty, so it will pass
					// the p.streamElementXPath check. However, eventually we know this <BBB>
					// shouldn't be returned to the caller. Having a second more fine-grained
					// filter check ensures that. So in this case, the caller should really
					// setup the stream parser with:
					//   streamElementXPath = "/AAA/BBB["
					//   streamElementFilter = "/AAA/BBB[. != 'b1']"
					if p.streamElementFilter == nil || QuerySelector(p.doc, p.streamElementFilter) != nil {
						return p.streamNode, nil
					}
					// otherwise, this isn't our target node, clean things up.
					// note we also remove the underlying *Node from the node tree, to prevent
					// future stream node candidate selection error.
					RemoveFromTree(p.streamNode)
					p.prev = p.streamNodePrev
					p.streamNode = nil
					p.streamNodePrev = nil
				}
			}
		case xml.CharData:
			// First, normalize the cache...
			cached := bytes.ToUpper(p.reader.CacheWithLimit(9))
			nodeType := TextNode
			if bytes.HasPrefix(cached, []byte("<![CDATA[")) || bytes.HasPrefix(cached, []byte("![CDATA[")) {
				nodeType = CharDataNode
			}
			node := &Node{Type: nodeType, Data: string(tok), level: p.level, LineNumber: p.currentLine}
			if p.level == p.prev.level {
				AddSibling(p.prev, node)
			} else if p.level > p.prev.level {
				AddChild(p.prev, node)
			} else if p.level < p.prev.level {
				for i := p.prev.level - p.level; i > 1; i-- {
					p.prev = p.prev.Parent
				}
				AddSibling(p.prev.Parent, node)
			}
		case xml.Comment:
			node := &Node{Type: CommentNode, Data: string(tok), level: p.level, LineNumber: p.currentLine}
			if p.level == p.prev.level {
				AddSibling(p.prev, node)
			} else if p.level > p.prev.level {
				AddChild(p.prev, node)
			} else if p.level < p.prev.level {
				for i := p.prev.level - p.level; i > 1; i-- {
					p.prev = p.prev.Parent
				}
				AddSibling(p.prev.Parent, node)
			}
		case xml.ProcInst: // Processing Instruction
			if !(p.prev.Type == DeclarationNode || p.prev.Type == ProcessingInstruction) {
				p.level++
			}
			node := &Node{Type: DeclarationNode, Data: tok.Target, level: p.level, LineNumber: p.currentLine}
			pairs := strings.Split(string(tok.Inst), " ")
			for _, pair := range pairs {
				pair = strings.TrimSpace(pair)
				if i := strings.Index(pair, "="); i > 0 {
					AddAttr(node, pair[:i], strings.Trim(pair[i+1:], `"'`))
				}
			}
			if tok.Target != "xml" {
				node.Type = ProcessingInstruction
				node.ProcInst = &ProcInstData{Target: tok.Target, Inst: strings.TrimSpace(string(tok.Inst))}
			}
			if p.level == p.prev.level {
				AddSibling(p.prev, node)
			} else if p.level > p.prev.level {
				AddChild(p.prev, node)
			} else if p.level < p.prev.level {
				for i := p.prev.level - p.level; i > 1; i-- {
					p.prev = p.prev.Parent
				}
				AddSibling(p.prev.Parent, node)
			}
			p.prev = node
		case xml.Directive:
			node := &Node{Type: NotationNode, Data: string(tok), level: p.level, LineNumber: p.currentLine}
			if p.level == p.prev.level {
				AddSibling(p.prev, node)
			} else if p.level > p.prev.level {
				AddChild(p.prev, node)
			} else if p.level < p.prev.level {
				for i := p.prev.level - p.level; i > 1; i-- {
					p.prev = p.prev.Parent
				}
				AddSibling(p.prev.Parent, node)
			}
		}
	}
}

// StreamParser enables loading and parsing an XML document in a streaming
// fashion.
type StreamParser struct {
	p *parser
}

// CreateStreamParser creates a StreamParser. Argument streamElementXPath is
// required.
// Argument streamElementFilter is optional and should only be used in advanced
// scenarios.
//
// Scenario 1: simple case:
//
//	xml := `<AAA><BBB>b1</BBB><BBB>b2</BBB></AAA>`
//	sp, err := CreateStreamParser(strings.NewReader(xml), "/AAA/BBB")
//	if err != nil {
//	    panic(err)
//	}
//	for {
//	    n, err := sp.Read()
//	    if err != nil {
//	        break
//	    }
//	    fmt.Println(n.OutputXML(true))
//	}
//
// Output will be:
//
//	<BBB>b1</BBB>
//	<BBB>b2</BBB>
//
// Scenario 2: advanced case:
//
//	xml := `<AAA><BBB>b1</BBB><BBB>b2</BBB></AAA>`
//	sp, err := CreateStreamParser(strings.NewReader(xml), "/AAA/BBB", "/AAA/BBB[. != 'b1']")
//	if err != nil {
//	    panic(err)
//	}
//	for {
//	    n, err := sp.Read()
//	    if err != nil {
//	        break
//	    }
//	    fmt.Println(n.OutputXML(true))
//	}
//
// Output will be:
//
//	<BBB>b2</BBB>
//
// As the argument names indicate, streamElementXPath should be used for
// providing xpath query pointing to the target element node only, no extra
// filtering on the element itself or its children; while streamElementFilter,
// if needed, can provide additional filtering on the target element and its
// children.
//
// CreateStreamParser returns an error if either streamElementXPath or
// streamElementFilter, if provided, cannot be successfully parsed and compiled
// into a valid xpath query.
func CreateStreamParser(r io.Reader, streamElementXPath string, streamElementFilter ...string) (*StreamParser, error) {
	return CreateStreamParserWithOptions(r, ParserOptions{}, streamElementXPath, streamElementFilter...)
}

// CreateStreamParserWithOptions is like CreateStreamParser, but with custom options
func CreateStreamParserWithOptions(
	r io.Reader,
	options ParserOptions,
	streamElementXPath string,
	streamElementFilter ...string,
) (*StreamParser, error) {
	elemXPath, err := getQuery(streamElementXPath)
	if err != nil {
		return nil, fmt.Errorf("invalid streamElementXPath '%s', err: %s", streamElementXPath, err.Error())
	}
	elemFilter := (*xpath.Expr)(nil)
	if len(streamElementFilter) > 0 {
		elemFilter, err = getQuery(streamElementFilter[0])
		if err != nil {
			return nil, fmt.Errorf("invalid streamElementFilter '%s', err: %s", streamElementFilter[0], err.Error())
		}
	}
	parser := createParser(r)
	options.apply(parser)
	sp := &StreamParser{
		p: parser,
	}
	sp.p.streamElementXPath = elemXPath
	sp.p.streamElementFilter = elemFilter
	return sp, nil
}

// Read returns a target node that satisfies the XPath specified by caller at
// StreamParser creation time. If there is no more satisfying target nodes after
// reading the rest of the XML document, io.EOF will be returned. At any time,
// any XML parsing error encountered will be returned, and the stream parsing
// stopped. Calling Read() after an error is returned (including io.EOF) results
// undefined behavior. Also note, due to the streaming nature, calling Read()
// will automatically remove any previous target node(s) from the document tree.
func (sp *StreamParser) Read() (*Node, error) {
	// Because this is a streaming read, we need to release/remove last
	// target node from the node tree to free up memory.
	if sp.p.streamNode != nil {
		// We need to remove all siblings before the current stream node,
		// because the document may contain unwanted nodes between the target
		// ones (for example new line text node), which would otherwise
		// accumulate as first childs, and slow down the stream over time
		for sp.p.streamNode.PrevSibling != nil {
			RemoveFromTree(sp.p.streamNode.PrevSibling)
		}
		sp.p.prev = sp.p.streamNode.Parent
		RemoveFromTree(sp.p.streamNode)
		sp.p.streamNode = nil
		sp.p.streamNodePrev = nil
	}
	return sp.p.parse()
}

// lineNumberAnnotator handles post-processing line number annotation
type lineNumberAnnotator struct {
	data       []byte
	lineStarts []int
	tracker    *positionTracker
}

// getLineForPosition returns the line number for a given byte position
func (p *lineNumberAnnotator) getLineForPosition(pos int) int {
	if pos < 0 {
		return 1
	}

	line := 1
	for i, start := range p.lineStarts {
		if pos < start {
			return i // i is the line number (1-based because lineStarts[0] = 0 for line 1)
		}
		line = i + 1
	}
	return line
}

// annotateLineNumbers walks through the XML data and annotates nodes with line numbers
func (p *lineNumberAnnotator) annotateLineNumbers(doc *Node) error {
	// First reset all line numbers to ensure clean state
	p.resetLineNumbers(doc)
	// Use a simpler approach: walk through the document in order and match with positions
	p.annotateNodesByPosition(doc)
	return nil
}

// resetLineNumbers recursively resets all line numbers to 0
func (p *lineNumberAnnotator) resetLineNumbers(node *Node) {
	if node == nil {
		return
	}
	node.LineNumber = 0
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		p.resetLineNumbers(child)
	}
}

// annotateNodesByPosition recursively annotates nodes by finding their positions in source
func (p *lineNumberAnnotator) annotateNodesByPosition(node *Node) {
	if node == nil {
		return
	}

	// Annotate current node if not already done
	if node.LineNumber == 0 {
		switch node.Type {
		case ElementNode:
			node.LineNumber = p.findElementPosition(node.Data)
		case CommentNode:
			node.LineNumber = p.findCommentPosition(node.Data)
		case DeclarationNode:
			node.LineNumber = p.findDeclarationLine()
		case ProcessingInstruction:
			node.LineNumber = p.findProcessingInstructionPosition(node.Data)
		case TextNode, CharDataNode:
			text := strings.TrimSpace(node.Data)
			if text != "" {
				node.LineNumber = p.findTextPosition(text)
			}
		}
	}

	// Recursively annotate children
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		p.annotateNodesByPosition(child)
	}
}

// State to track positions as we traverse the document
type positionTracker struct {
	currentPos    int
	elementCounts map[string]int
	commentCounts map[string]int
	textCounts    map[string]int
}

// findElementPosition finds the line number for the next occurrence of an element
func (p *lineNumberAnnotator) findElementPosition(name string) int {
	if p.tracker == nil {
		p.tracker = &positionTracker{
			elementCounts: make(map[string]int),
			commentCounts: make(map[string]int),
			textCounts:    make(map[string]int),
		}
	}

	p.tracker.elementCounts[name]++
	return p.findNthElementOccurrence(name, p.tracker.elementCounts[name])
}

// findNthElementOccurrence finds the nth occurrence of an element
func (p *lineNumberAnnotator) findNthElementOccurrence(name string, n int) int {
	count := 0
	pos := 0
	dataStr := string(p.data)

	// Look for both prefixed and non-prefixed versions
	patterns := []string{
		fmt.Sprintf("<%s", name), // <name
		fmt.Sprintf(":%s", name), // prefix:name
	}

	for {
		earliestPos := len(p.data)
		foundPattern := ""

		// Find the earliest occurrence of any pattern
		for _, pattern := range patterns {
			foundPos := strings.Index(dataStr[pos:], pattern)
			if foundPos >= 0 {
				absolutePos := pos + foundPos
				if absolutePos < earliestPos {
					earliestPos = absolutePos
					foundPattern = pattern
				}
			}
		}

		if earliestPos == len(p.data) {
			break // No more occurrences found
		}

		// Validate the match
		nextCharPos := earliestPos + len(foundPattern)
		isValidMatch := false

		if foundPattern[0] == '<' {
			// Direct element match like <name
			if nextCharPos < len(p.data) {
				ch := p.data[nextCharPos]
				if ch == '>' || ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
					isValidMatch = true
				}
			}
		} else {
			// Namespace prefix match like :name
			// Make sure it's preceded by < and some prefix
			if earliestPos > 0 && nextCharPos < len(p.data) {
				ch := p.data[nextCharPos]
				if ch == '>' || ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
					// Look backwards to find the <
					foundOpenTag := false
					for i := earliestPos - 1; i >= 0; i-- {
						if p.data[i] == '<' {
							foundOpenTag = true
							break
						} else if p.data[i] == '>' {
							break // Found closing tag first, not valid
						}
					}
					if foundOpenTag {
						isValidMatch = true
					}
				}
			}
		}

		if isValidMatch {
			count++
			if count == n {
				// For namespace prefix matches, return the position of the <
				if foundPattern[0] == ':' {
					for i := earliestPos - 1; i >= 0; i-- {
						if p.data[i] == '<' {
							return p.getLineForPosition(i)
						}
					}
				}
				return p.getLineForPosition(earliestPos)
			}
		}

		pos = earliestPos + 1
	}

	return 1
}

// findCommentPosition finds the line number for the next occurrence of a comment
func (p *lineNumberAnnotator) findCommentPosition(content string) int {
	if p.tracker == nil {
		p.tracker = &positionTracker{
			elementCounts: make(map[string]int),
			commentCounts: make(map[string]int),
			textCounts:    make(map[string]int),
		}
	}

	p.tracker.commentCounts[content]++
	return p.findNthCommentOccurrence(content, p.tracker.commentCounts[content])
}

// findNthCommentOccurrence finds the nth occurrence of a comment
func (p *lineNumberAnnotator) findNthCommentOccurrence(content string, n int) int {
	pattern := fmt.Sprintf("<!--%s-->", content)
	count := 0
	pos := 0

	for {
		foundPos := strings.Index(string(p.data[pos:]), pattern)
		if foundPos < 0 {
			break
		}
		count++
		absolutePos := pos + foundPos
		if count == n {
			return p.getLineForPosition(absolutePos)
		}
		pos = absolutePos + len(pattern)
	}
	return 1
}

// findDeclarationLine finds the line number of the XML declaration
func (p *lineNumberAnnotator) findDeclarationLine() int {
	pattern := "<?xml"
	pos := bytes.Index(p.data, []byte(pattern))
	if pos >= 0 {
		return p.getLineForPosition(pos)
	}
	return 1
}

// findTextPosition finds the line number for the next occurrence of text
func (p *lineNumberAnnotator) findTextPosition(text string) int {
	if p.tracker == nil {
		p.tracker = &positionTracker{
			elementCounts: make(map[string]int),
			commentCounts: make(map[string]int),
			textCounts:    make(map[string]int),
		}
	}

	p.tracker.textCounts[text]++
	return p.findNthTextOccurrence(text, p.tracker.textCounts[text])
}

// findNthTextOccurrence finds the nth occurrence of text
func (p *lineNumberAnnotator) findNthTextOccurrence(text string, n int) int {
	count := 0
	pos := 0

	for {
		foundPos := strings.Index(string(p.data[pos:]), text)
		if foundPos < 0 {
			break
		}
		count++
		absolutePos := pos + foundPos
		if count == n {
			return p.getLineForPosition(absolutePos)
		}
		pos = absolutePos + len(text)
	}
	return 1
}

// findProcessingInstructionPosition finds the line number for a processing instruction
func (p *lineNumberAnnotator) findProcessingInstructionPosition(target string) int {
	pattern := fmt.Sprintf("<?%s", target)
	pos := strings.Index(string(p.data), pattern)
	if pos >= 0 {
		return p.getLineForPosition(pos)
	}
	return 1
}

// LoadURLWithLineNumbers loads the XML document from the specified URL with line number annotations.
func LoadURLWithLineNumbers(url string) (*Node, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if xmlMIMERegex.MatchString(resp.Header.Get("Content-Type")) {
		return ParseWithOptions(resp.Body, ParserOptions{WithLineNumbers: true})
	}
	return nil, fmt.Errorf("invalid XML document(%s)", resp.Header.Get("Content-Type"))
}
