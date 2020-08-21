package xmlquery

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
)

// Produces a marshaling friendly name for a given *Node.
func nodeName(n *Node) string {
	if n == nil {
		return "(nil)"
	}
	switch n.Type {
	case DocumentNode:
		return "(ROOT)"
	case DeclarationNode:
		return fmt.Sprintf("(DECL '%s')", n.Data)
	case ElementNode:
		name := fmt.Sprintf("(ELEM %s)", n.Data)
		if n.Prefix != "" {
			name = fmt.Sprintf("(ELEM %s:%s)", n.Prefix, n.Data)
		}
		return name
	case TextNode:
		return fmt.Sprintf("(TEXT '%s')", n.Data)
	case CharDataNode:
		return fmt.Sprintf("(CDATA '%s')", n.Data)
	case CommentNode:
		return fmt.Sprintf("(COMMENT '%s')", n.Data)
	case AttributeNode:
		return fmt.Sprintf("(ATTR '%s')", n.Data)
	default:
		return fmt.Sprintf("(UNKNOWN '%s')", n.Data)
	}
}

// We have to implement a custom json marshaler for Node given its self-referencing
// pointers which would cause json marshaler infinite recursion and stack overflow.
// Keep the marshaler in _test.go so it doesn't end up in production library code.
func (n Node) MarshalJSON() ([]byte, error) {
	ptrToStr := func(link *Node) string {
		return "ptr:" + nodeName(link)
	}
	children := []*Node{}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		children = append(children, child)
	}
	return json.Marshal(&struct {
		Parent, FirstChild, LastChild, PrevSibling, NextSibling string
		Type                                                    NodeType
		Data                                                    string
		Prefix                                                  string
		NamespaceURI                                            string
		Attr                                                    []xml.Attr
		Level                                                   int
		Children                                                []*Node `json:"#children"`
	}{
		Parent:       ptrToStr(n.Parent),
		FirstChild:   ptrToStr(n.FirstChild),
		LastChild:    ptrToStr(n.LastChild),
		PrevSibling:  ptrToStr(n.PrevSibling),
		NextSibling:  ptrToStr(n.NextSibling),
		Type:         n.Type,
		Data:         n.Data,
		Prefix:       n.Prefix,
		NamespaceURI: n.NamespaceURI,
		Attr:         n.Attr,
		Level:        n.level,
		Children:     children,
	})
}

// Make marshaling of NodeType (an int type) human readable.
var nodeTypeStrings = map[NodeType]string{
	DocumentNode:    "DocumentNode",
	DeclarationNode: "DeclarationNode",
	ElementNode:     "ElementNode",
	TextNode:        "TextNode",
	CharDataNode:    "CharDataNode",
	CommentNode:     "CommentNode",
	AttributeNode:   "AttributeNode",
}

func (nt NodeType) String() string {
	if s, found := nodeTypeStrings[nt]; found {
		return s
	}
	return fmt.Sprintf("(unknown:%d)", nt)
}

func (nt NodeType) MarshalJSON() ([]byte, error) {
	return []byte(`"` + nt.String() + `"`), nil
}

// A best-effort json marshaling helper for test
func prettyJSONMarshal(v interface{}) string {
	valueBuf := new(bytes.Buffer)
	enc := json.NewEncoder(valueBuf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "\t")
	err := enc.Encode(v)
	if err != nil {
		return "{}"
	}
	return valueBuf.String()
}
