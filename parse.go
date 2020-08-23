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
	return parse(resp.Body)
}

func parse(r io.Reader) (*Node, error) {
	var (
		decoder      = xml.NewDecoder(r)
		doc          = &Node{Type: DocumentNode}
		space2prefix = make(map[string]string)
		level        = 0
	)
	// http://www.w3.org/XML/1998/namespace is bound by definition to the prefix xml.
	space2prefix["http://www.w3.org/XML/1998/namespace"] = "xml"
	decoder.CharsetReader = charset.NewReaderLabel
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
			node := &Node{Type: CharDataNode, Data: string(tok), level: level}
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
