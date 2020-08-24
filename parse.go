package xmlquery

import (
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"strings"

	"golang.org/x/net/html/charset"
)

// LoadURL loads the XML document from the specified URL.
func LoadURL(url string) (*Node, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return Parse(resp.Body)
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

type parser struct {
	decoder      *xml.Decoder
	doc          *Node
	space2prefix map[string]string
	level        int
	prev         *Node
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
			p.prev = node
			p.level++
		case xml.EndElement:
			p.level--
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
