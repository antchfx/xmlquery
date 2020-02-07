package xmlquery

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

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
		case TextNode:
			buf.WriteString(n.Data)
			return
		case CommentNode:
			return
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			output(buf, child)
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
	if n.Type == TextNode {
		xml.EscapeText(buf, []byte(n.sanitizedData(preserveSpaces)))
		return
	}
	if n.Type == CommentNode {
		buf.WriteString("<!--")
		buf.WriteString(n.Data)
		buf.WriteString("-->")
		return
	}
	if n.Type == DeclarationNode {
		buf.WriteString("<?" + n.Data)
	} else {
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
		buf.WriteByte(34) // "
		xml.EscapeText(buf, []byte(attr.Value))
		buf.WriteByte(34)
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

// LoadURL loads the XML document from the specified URL.
func LoadURL(url string) (*Node, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return parse(resp.Body)
}

type ParseOption func(cdg *parseConfig) error

type parseConfig struct {
	decoder      *xml.Decoder
	doc          *Node
	space2prefix map[string]string
	level        int
}

func parse(r io.Reader, opts ...ParseOption) (*Node, error) {
	var cfg = &parseConfig{
		decoder: xml.NewDecoder(r),
		doc:     &Node{Type: DocumentNode},
		space2prefix: map[string]string{
			// http://www.w3.org/XML/1998/namespace is bound by definition to the prefix xml.
			"http://www.w3.org/XML/1998/namespace": "xml",
		},
		level: 0,
	}
	cfg.decoder.CharsetReader = charset.NewReaderLabel

	for _, setOpt := range opts {
		var errOpt = setOpt(cfg)
		if errOpt != nil {
			return nil, errOpt
		}
	}

	var (
		decoder      = cfg.decoder
		doc          = cfg.doc
		space2prefix = cfg.space2prefix
		level        = cfg.level
	)
	prev := doc
	for {
		tok, err := decoder.Token()
		switch {
		case err == io.EOF:
			goto quit
		case err != nil:
			return nil, err
		}

		switch tok := tok.(type) {
		case xml.StartElement:
			if level == 0 {
				// mising XML declaration
				node := &Node{Type: DeclarationNode, Data: "xml", level: 1}
				addChild(prev, node)
				level = 1
				prev = node
			}
			// https://www.w3.org/TR/xml-names/#scoping-defaulting
			for _, att := range tok.Attr {
				if att.Name.Local == "xmlns" {
					space2prefix[att.Value] = ""
				} else if att.Name.Space == "xmlns" {
					space2prefix[att.Value] = att.Name.Local
				}
			}

			if tok.Name.Space != "" {
				if _, found := space2prefix[tok.Name.Space]; !found {
					return nil, errors.New("xmlquery: invalid XML document, namespace is missing")
				}
			}

			for i := 0; i < len(tok.Attr); i++ {
				att := &tok.Attr[i]
				if prefix, ok := space2prefix[att.Name.Space]; ok {
					att.Name.Space = prefix
				}
			}

			node := &Node{
				Type:         ElementNode,
				Data:         tok.Name.Local,
				Prefix:       space2prefix[tok.Name.Space],
				NamespaceURI: tok.Name.Space,
				Attr:         tok.Attr,
				level:        level,
			}
			//fmt.Println(fmt.Sprintf("start > %s : %d", node.Data, level))
			if level == prev.level {
				addSibling(prev, node)
			} else if level > prev.level {
				addChild(prev, node)
			} else if level < prev.level {
				for i := prev.level - level; i > 1; i-- {
					prev = prev.Parent
				}
				addSibling(prev.Parent, node)
			}
			prev = node
			level++
		case xml.EndElement:
			level--
		case xml.CharData:
			node := &Node{Type: TextNode, Data: string(tok), level: level}
			if level == prev.level {
				addSibling(prev, node)
			} else if level > prev.level {
				addChild(prev, node)
			} else if level < prev.level {
				for i := prev.level - level; i > 1; i-- {
					prev = prev.Parent
				}
				addSibling(prev.Parent, node)
			}
		case xml.Comment:
			node := &Node{Type: CommentNode, Data: string(tok), level: level}
			if level == prev.level {
				addSibling(prev, node)
			} else if level > prev.level {
				addChild(prev, node)
			} else if level < prev.level {
				for i := prev.level - level; i > 1; i-- {
					prev = prev.Parent
				}
				addSibling(prev.Parent, node)
			}
		case xml.ProcInst: // Processing Instruction
			if prev.Type != DeclarationNode {
				level++
			}
			node := &Node{Type: DeclarationNode, Data: tok.Target, level: level}
			pairs := strings.Split(string(tok.Inst), " ")
			for _, pair := range pairs {
				pair = strings.TrimSpace(pair)
				if i := strings.Index(pair, "="); i > 0 {
					addAttr(node, pair[:i], strings.Trim(pair[i+1:], `"`))
				}
			}
			if level == prev.level {
				addSibling(prev, node)
			} else if level > prev.level {
				addChild(prev, node)
			}
			prev = node
		case xml.Directive:
		}

	}
quit:
	return doc, nil
}

// Parse returns the parse tree for the XML from the given Reader.
func Parse(r io.Reader) (*Node, error) {
	return parse(r)
}

// ParseWith runs the parser with provided optional configuration.
func ParseWith(r io.Reader, opts ...ParseOption) (*Node, error) {
	return parse(r, opts...)
}

// WithDecoder applies patch function to the XML decoder.
func WithDecoder(patch func(dec *xml.Decoder)) ParseOption {
	return func(cfg *parseConfig) error {
		patch(cfg.decoder)
		return nil
	}
}

// WithStrictDecoder forces the strict XML parsing mode.
func WithStrictDecoder() ParseOption {
	return WithDecoder(func(dec *xml.Decoder) {
		dec.Strict = true
	})
}

// WithTolerantDecoder disables the strict XML parsing mode.
func WithTolerantDecoder() ParseOption {
	return WithDecoder(func(dec *xml.Decoder) {
		dec.Strict = false
	})
}
