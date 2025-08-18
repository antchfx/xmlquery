package xmlquery

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ParseWithLineNumbers returns the parse tree for the XML from the given Reader with line number annotations.
func ParseWithLineNumbers(r io.Reader) (*Node, error) {
	return ParseWithLineNumbersAndOptions(r, ParserOptions{})
}

// ParseWithLineNumbersAndOptions is like ParseWithLineNumbers, but with custom options
func ParseWithLineNumbersAndOptions(r io.Reader, options ParserOptions) (*Node, error) {
	// Read all data first so we can track positions
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// Create a position-aware parser
	parser := &lineCountingParser{
		data:       data,
		lineStarts: []int{0},
	}

	// Pre-calculate line starts
	for i, b := range data {
		if b == '\n' {
			parser.lineStarts = append(parser.lineStarts, i+1)
		}
	}

	// Parse with the standard parser first
	doc, err := ParseWithOptions(bytes.NewReader(data), options)
	if err != nil {
		return nil, err
	}

	// Now annotate with line numbers
	err = parser.annotateLineNumbers(doc, data)
	if err != nil {
		return nil, err
	}

	return doc, nil
}

// lineCountingParser tracks line positions in XML data
type lineCountingParser struct {
	data       []byte
	lineStarts []int
}

// getLineForPosition returns the line number for a given byte position
func (p *lineCountingParser) getLineForPosition(pos int) int {
	if pos < 0 {
		return 1
	}
	
	// Binary search for the line
	line := 1
	for i, start := range p.lineStarts {
		if pos < start {
			return i
		}
		line = i + 1
	}
	return line
}

// annotateLineNumbers walks through the XML data and annotates nodes with line numbers
func (p *lineCountingParser) annotateLineNumbers(doc *Node, data []byte) error {
	// Use an approach that synchronizes parsing with the node tree
	elementCounter := make(map[string]int)
	commentCounter := make(map[string]int)
	textCounter := make(map[string]int)
	
	p.annotateNodeWithCounters(doc, data, elementCounter, commentCounter, textCounter)
	return nil
}

// annotateNodeWithCounters recursively annotates nodes using occurrence counters
func (p *lineCountingParser) annotateNodeWithCounters(node *Node, data []byte, elementCounter, commentCounter, textCounter map[string]int) {
	if node == nil {
		return
	}
	
	// Annotate current node if not already done
	if node.LineNumber == 0 {
		switch node.Type {
		case ElementNode:
			elementCounter[node.Data]++
			node.LineNumber = p.findNthElementLine(node.Data, elementCounter[node.Data], data)
		case CommentNode:
			commentCounter[node.Data]++
			node.LineNumber = p.findNthCommentLine(node.Data, commentCounter[node.Data], data)
		case DeclarationNode:
			if node.Data == "xml" {
				node.LineNumber = p.findDeclarationLine(data)
			}
		case TextNode, CharDataNode:
			text := strings.TrimSpace(node.Data)
			if text != "" {
				textCounter[text]++
				node.LineNumber = p.findNthTextLine(text, textCounter[text], data)
			}
		}
	}
	
	// Recursively annotate children
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		p.annotateNodeWithCounters(child, data, elementCounter, commentCounter, textCounter)
	}
}

// findNthElementLine finds the line number of the nth occurrence of an element
func (p *lineCountingParser) findNthElementLine(name string, n int, data []byte) int {
	// First try to find with namespace prefix (more specific)
	patterns := []string{
		fmt.Sprintf("<%s:", name),  // <ns:name>
		fmt.Sprintf("<%s>", name),  // <name>
		fmt.Sprintf("<%s ", name),  // <name attr="value">
	}
	
	var allPositions []int
	
	// Collect all possible positions for this element name
	for _, pattern := range patterns {
		patternBytes := []byte(pattern)
		pos := 0
		for {
			foundPos := bytes.Index(data[pos:], patternBytes)
			if foundPos < 0 {
				break
			}
			absolutePos := pos + foundPos
			allPositions = append(allPositions, absolutePos)
			pos += foundPos + len(pattern)
		}
	}
	
	// Also look for namespace-prefixed version
	dataStr := string(data)
	prefixPattern := fmt.Sprintf(":%s", name)
	pos := 0
	for {
		foundPos := strings.Index(dataStr[pos:], prefixPattern)
		if foundPos < 0 {
			break
		}
		absolutePos := pos + foundPos
		// Check if this is actually an element start (preceded by <)
		if absolutePos > 0 && dataStr[absolutePos-1] != '<' {
			// Find the preceding < 
			for i := absolutePos - 1; i >= 0; i-- {
				if dataStr[i] == '<' {
					allPositions = append(allPositions, i)
					break
				} else if dataStr[i] == '>' {
					// Not an element start
					break
				}
			}
		}
		pos += foundPos + len(prefixPattern)
	}
	
	if len(allPositions) == 0 {
		return 1
	}
	
	// Remove duplicates and sort
	uniquePositions := make(map[int]bool)
	for _, pos := range allPositions {
		uniquePositions[pos] = true
	}
	
	var sortedPositions []int
	for pos := range uniquePositions {
		sortedPositions = append(sortedPositions, pos)
	}
	
	// Simple sort
	for i := 0; i < len(sortedPositions); i++ {
		for j := i + 1; j < len(sortedPositions); j++ {
			if sortedPositions[i] > sortedPositions[j] {
				sortedPositions[i], sortedPositions[j] = sortedPositions[j], sortedPositions[i]
			}
		}
	}
	
	if n <= len(sortedPositions) {
		return p.getLineForPosition(sortedPositions[n-1])
	}
	
	return 1
}

// findNthCommentLine finds the line number of the nth comment
func (p *lineCountingParser) findNthCommentLine(content string, n int, data []byte) int {
	pattern := fmt.Sprintf("<!--%s-->", content)
	patternBytes := []byte(pattern)
	pos := 0
	count := 0
	
	for {
		foundPos := bytes.Index(data[pos:], patternBytes)
		if foundPos < 0 {
			break
		}
		count++
		pos += foundPos
		if count == n {
			return p.getLineForPosition(pos)
		}
		pos += len(pattern)
	}
	return 1
}

// findDeclarationLine finds the line number of the XML declaration
func (p *lineCountingParser) findDeclarationLine(data []byte) int {
	pattern := "<?xml"
	pos := bytes.Index(data, []byte(pattern))
	if pos >= 0 {
		return p.getLineForPosition(pos)
	}
	return 1
}

// findNthTextLine finds the line number of the nth text occurrence
func (p *lineCountingParser) findNthTextLine(text string, n int, data []byte) int {
	textBytes := []byte(text)
	pos := 0
	count := 0
	
	for {
		foundPos := bytes.Index(data[pos:], textBytes)
		if foundPos < 0 {
			break
		}
		count++
		pos += foundPos
		if count == n {
			return p.getLineForPosition(pos)
		}
		pos += len(text)
	}
	return 1
}

// collectNodes collects all nodes in document order
func (p *lineCountingParser) collectNodes(node *Node, nodes *[]*Node) {
	if node == nil {
		return
	}
	
	*nodes = append(*nodes, node)
	
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		p.collectNodes(child, nodes)
	}
}


// LoadURLWithLineNumbers loads the XML document from the specified URL with line number annotations.
func LoadURLWithLineNumbers(url string) (*Node, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	// Note: Using the regexp from parse.go
	if xmlMIMERegex.MatchString(resp.Header.Get("Content-Type")) {
		return ParseWithLineNumbers(resp.Body)
	}
	return nil, fmt.Errorf("invalid XML document(%s)", resp.Header.Get("Content-Type"))
}