package xmlquery

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/antchfx/xpath"
	"golang.org/x/net/html/charset"
)

// A NodeType is the type of a Node.
type NodeType uint

const (
	// DocumentNode is a document object that, as the root of the document tree,
	// provides access to the entire XML document.
	DocumentNode NodeType = iota
	// DeclarationNode is the document type declaration, indicated by the following
	// tag (for example, <!DOCTYPE...> ).
	DeclarationNode
	// ElementNode is an element (for example, <item> ).
	ElementNode
	// TextNode is the text content of a node.
	TextNode
	// CharDataNode node <![CDATA[content]]>
	CharDataNode
	// CommentNode a comment (for example, <!-- my comment --> ).
	CommentNode
	// AttributeNode is an attribute of element.
	AttributeNode
)

// A Node consists of a NodeType and some Data (tag name for
// element nodes, content for text) and are part of a tree of Nodes.
type Node struct {
	Parent, FirstChild, LastChild, PrevSibling, NextSibling *Node

	Type         NodeType
	Data         string
	Prefix       string
	NamespaceURI string
	Attr         []xml.Attr

	level int // node level in the tree
}

// InnerText returns the text between the start and end tags of the object.
func (n *Node) InnerText() string {
	var output func(*bytes.Buffer, *Node)
	output = func(buf *bytes.Buffer, n *Node) {
		switch n.Type {
		case TextNode, CharDataNode:
			buf.WriteString(n.Data)
		case CommentNode:
		default:
			for child := n.FirstChild; child != nil; child = child.NextSibling {
				output(buf, child)
			}
		}
	}

	var buf bytes.Buffer
	output(&buf, n)
	return buf.String()
}

func (n *Node) sanitizedData(preserveSpaces bool) string {
	if preserveSpaces {
		return strings.Trim(n.Data, "\n\t")
	}
	return strings.TrimSpace(n.Data)
}

func calculatePreserveSpaces(n *Node, pastValue bool) bool {
	if attr := n.SelectAttr("xml:space"); attr == "preserve" {
		return true
	} else if attr == "default" {
		return false
	}
	return pastValue
}

func outputXML(buf *bytes.Buffer, n *Node, preserveSpaces bool) {
	preserveSpaces = calculatePreserveSpaces(n, preserveSpaces)
	switch n.Type {
	case TextNode, CharDataNode:
		xml.EscapeText(buf, []byte(n.sanitizedData(preserveSpaces)))
		return
	case CommentNode:
		buf.WriteString("<!--")
		buf.WriteString(n.Data)
		buf.WriteString("-->")
		return
	case DeclarationNode:
		buf.WriteString("<?" + n.Data)
	default:
		if n.Prefix == "" {
			buf.WriteString("<" + n.Data)
		} else {
			buf.WriteString("<" + n.Prefix + ":" + n.Data)
		}
	}

	for _, attr := range n.Attr {
		if attr.Name.Space != "" {
			buf.WriteString(fmt.Sprintf(` %s:%s=`, attr.Name.Space, attr.Name.Local))
		} else {
			buf.WriteString(fmt.Sprintf(` %s=`, attr.Name.Local))
		}
		buf.WriteByte('"')
		xml.EscapeText(buf, []byte(attr.Value))
		buf.WriteByte('"')
	}
	if n.Type == DeclarationNode {
		buf.WriteString("?>")
	} else {
		buf.WriteString(">")
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		outputXML(buf, child, preserveSpaces)
	}
	if n.Type != DeclarationNode {
		if n.Prefix == "" {
			buf.WriteString(fmt.Sprintf("</%s>", n.Data))
		} else {
			buf.WriteString(fmt.Sprintf("</%s:%s>", n.Prefix, n.Data))
		}
	}
}

// OutputXML returns the text that including tags name.
func (n *Node) OutputXML(self bool) string {
	var buf bytes.Buffer
	if self {
		outputXML(&buf, n, false)
	} else {
		for n := n.FirstChild; n != nil; n = n.NextSibling {
			outputXML(&buf, n, false)
		}
	}

	return buf.String()
}

func addAttr(n *Node, key, val string) {
	var attr xml.Attr
	if i := strings.Index(key, ":"); i > 0 {
		attr = xml.Attr{
			Name:  xml.Name{Space: key[:i], Local: key[i+1:]},
			Value: val,
		}
	} else {
		attr = xml.Attr{
			Name:  xml.Name{Local: key},
			Value: val,
		}
	}

	n.Attr = append(n.Attr, attr)
}

func addChild(parent, n *Node) {
	n.Parent = parent
	if parent.FirstChild == nil {
		parent.FirstChild = n
	} else {
		parent.LastChild.NextSibling = n
		n.PrevSibling = parent.LastChild
	}

	parent.LastChild = n
}

func addSibling(sibling, n *Node) {
	for t := sibling.NextSibling; t != nil; t = t.NextSibling {
		sibling = t
	}
	n.Parent = sibling.Parent
	sibling.NextSibling = n
	n.PrevSibling = sibling
	if sibling.Parent != nil {
		sibling.Parent.LastChild = n
	}
}

// removes a node and its subtree from the tree it is in. If the node is the root of the tree, then it's no-op.
func remove(n *Node) {
	if n.Parent == nil {
		return
	}

	if n.Parent.FirstChild == n {
		if n.Parent.LastChild == n {
			n.Parent.FirstChild = nil
			n.Parent.LastChild = nil
		} else {
			n.Parent.FirstChild = n.NextSibling
			n.NextSibling.PrevSibling = nil
		}
	} else {
		if n.Parent.LastChild == n {
			n.Parent.LastChild = n.PrevSibling
			n.PrevSibling.NextSibling = nil
		} else {
			n.PrevSibling.NextSibling = n.NextSibling
			n.NextSibling.PrevSibling = n.PrevSibling
		}
	}
	n.Parent = nil
	n.PrevSibling = nil
	n.NextSibling = nil
}

// LoadURL loads the XML document from the specified URL.
func LoadURL(url string) (*Node, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return Parse(resp.Body)
}

type parser struct {
	decoder              *xml.Decoder
	doc                  *Node
	space2prefix         map[string]string
	level                int
	prev                 *Node
	streamXPath          *xpath.Expr
	streamNode           *Node
	streamNodePrev       *Node
	streamElementCounter int
}

func createParser(r io.Reader) *parser {
	p := &parser{
		decoder:      xml.NewDecoder(r),
		doc:          &Node{Type: DocumentNode},
		space2prefix: make(map[string]string),
		level:        0,
	}
	// http://www.w3.org/XML/1998/namespace is bound by definition to the prefix xml.
	p.space2prefix["http://www.w3.org/XML/1998/namespace"] = "xml"
	p.decoder.CharsetReader = charset.NewReaderLabel
	p.prev = p.doc
	return p
}

func (p *parser) parse() (*Node, error) {
	for {
		tok, err := p.decoder.Token()
		if err != nil {
			return nil, err
		}

		switch tok := tok.(type) {
		case xml.StartElement:
			if p.level == 0 {
				// mising XML declaration
				node := &Node{Type: DeclarationNode, Data: "xml", level: 1}
				addChild(p.prev, node)
				p.level = 1
				p.prev = node
			}
			// https://www.w3.org/TR/xml-names/#scoping-defaulting
			for _, att := range tok.Attr {
				if att.Name.Local == "xmlns" {
					p.space2prefix[att.Value] = ""
				} else if att.Name.Space == "xmlns" {
					p.space2prefix[att.Value] = att.Name.Local
				}
			}

			if tok.Name.Space != "" {
				if _, found := p.space2prefix[tok.Name.Space]; !found {
					return nil, errors.New("xmlquery: invalid XML document, namespace is missing")
				}
			}

			for i := 0; i < len(tok.Attr); i++ {
				att := &tok.Attr[i]
				if prefix, ok := p.space2prefix[att.Name.Space]; ok {
					att.Name.Space = prefix
				}
			}

			node := &Node{
				Type:         ElementNode,
				Data:         tok.Name.Local,
				Prefix:       p.space2prefix[tok.Name.Space],
				NamespaceURI: tok.Name.Space,
				Attr:         tok.Attr,
				level:        p.level,
			}
			//fmt.Println(fmt.Sprintf("start > %s : %d", node.Data, node.level))
			if p.level == p.prev.level {
				addSibling(p.prev, node)
			} else if p.level > p.prev.level {
				addChild(p.prev, node)
			} else if p.level < p.prev.level {
				for i := p.prev.level - p.level; i > 1; i-- {
					p.prev = p.prev.Parent
				}
				addSibling(p.prev.Parent, node)
			}
			// If we're in the streaming mode, we need to remember the node if it is the target node
			// so that when we finish processing the node's EndElement, we know how/what to return to
			// caller.
			if p.streamXPath != nil {
				if p.streamNode == nil {
					if p.isStreamTarget() {
						p.streamNode = node
						p.streamNodePrev = p.prev
						p.streamElementCounter = 1
					}
				} else {
					p.streamElementCounter++
				}
			}
			p.prev = node
			p.level++
		case xml.EndElement:
			p.level--
			if p.streamNode != nil {
				p.streamElementCounter--
				if p.streamElementCounter == 0 {
					return p.streamNode, nil
				}
			}
		case xml.CharData:
			node := &Node{Type: CharDataNode, Data: string(tok), level: p.level}
			if p.level == p.prev.level {
				addSibling(p.prev, node)
			} else if p.level > p.prev.level {
				addChild(p.prev, node)
			} else if p.level < p.prev.level {
				for i := p.prev.level - p.level; i > 1; i-- {
					p.prev = p.prev.Parent
				}
				addSibling(p.prev.Parent, node)
			}
		case xml.Comment:
			node := &Node{Type: CommentNode, Data: string(tok), level: p.level}
			if p.level == p.prev.level {
				addSibling(p.prev, node)
			} else if p.level > p.prev.level {
				addChild(p.prev, node)
			} else if p.level < p.prev.level {
				for i := p.prev.level - p.level; i > 1; i-- {
					p.prev = p.prev.Parent
				}
				addSibling(p.prev.Parent, node)
			}
		case xml.ProcInst: // Processing Instruction
			if p.prev.Type != DeclarationNode {
				p.level++
			}
			node := &Node{Type: DeclarationNode, Data: tok.Target, level: p.level}
			pairs := strings.Split(string(tok.Inst), " ")
			for _, pair := range pairs {
				pair = strings.TrimSpace(pair)
				if i := strings.Index(pair, "="); i > 0 {
					addAttr(node, pair[:i], strings.Trim(pair[i+1:], `"`))
				}
			}
			if p.level == p.prev.level {
				addSibling(p.prev, node)
			} else if p.level > p.prev.level {
				addChild(p.prev, node)
			}
			p.prev = node
		case xml.Directive:
		}
	}
}

func (p *parser) isStreamTarget() bool {
	return QuerySelector(p.doc, p.streamXPath) != nil
}

// Parse returns the parse tree for the XML from the given Reader.
func Parse(r io.Reader) (*Node, error) {
	p := createParser(r)
	for {
		_, err := p.parse()
		if err == io.EOF {
			return p.doc, nil
		}
		if err != nil {
			return nil, err
		}
	}
}

type StreamParser struct {
	p *parser
}

func CreateStreamParser(r io.Reader, streamXPath string) *StreamParser {
	if streamXPath == "" {
		panic("streamXPath cannot be empty")
	}
	expr, err := getQuery(streamXPath)
	if err != nil {
		panic(fmt.Sprintf("invalid streamXPath '%s', err: %s", streamXPath, err.Error()))
	}
	sp := &StreamParser{
		p: createParser(r),
	}
	sp.p.streamXPath = expr
	return sp
}

func (sp *StreamParser) Read() (*Node, error) {
	// Because this is a streaming read, we need to release/remove last
	// target node from the node tree to free up memory.
	if sp.p.streamNode != nil {
		remove(sp.p.streamNode)
		sp.p.prev = sp.p.streamNodePrev
		sp.p.streamNode = nil
		sp.p.streamNodePrev = nil
		sp.p.streamElementCounter = 0
	}
	return sp.p.parse()
}
