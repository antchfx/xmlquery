package xmlquery

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
)

type foption func(*formatter)

// FormatOptionIndent indent the XML using the provided
// pattern.
func FormatOptionIndent(s string) foption {
	return func(t *formatter) {
		t.Ident = s
	}
}

// FormatOptionDeclaration enable/disable the xml declaration.
func FormatOptionDeclaration(b bool) foption {
	return func(t *formatter) {
		t.XMLDeclaration = b
	}
}

// FormatString formats a xml string.
func FormatString(data string, options ...foption) (string, error) {
	root, err := Parse(strings.NewReader(data))
	if err != nil {
		return "", err
	}

	return Format(root, options...), nil
}

// Format a tree with the provided options
func Format(n *Node, options ...foption) string {
	return formatter{
		Ident:          "",
		XMLDeclaration: true,
	}.merge(options...).String(n)
}

type formatter struct {
	Ident          string
	XMLDeclaration bool
}

func (t formatter) merge(options ...foption) formatter {
	for _, opt := range options {
		opt(&t)
	}
	return t
}

func (t formatter) Output(buf *bytes.Buffer, n *Node) {
	t.output(buf, n, 0, false)
}

func (t formatter) String(n *Node) string {
	var buf bytes.Buffer

	t.Output(&buf, n)

	return buf.String()
}

func (t formatter) increment(n *Node, level int) int {
	switch n.Type {
	case DocumentNode:
		return level
	default:
		return level + 1
	}
}

func (t formatter) output(buf *bytes.Buffer, n *Node, level int, preserve bool) {
	preserveSpaces := calculatePreserveSpaces(n, preserve)

	switch n.Type {
	case TextNode:
		xml.EscapeText(buf, []byte(n.sanitizedData(preserveSpaces)))
		return
	case CharDataNode:
		buf.WriteString("<![CDATA[")
		xml.EscapeText(buf, []byte(n.sanitizedData(preserveSpaces)))
		buf.WriteString("]]>")
		return
	case CommentNode:
		buf.WriteString("<!--")
		buf.WriteString(n.Data)
		buf.WriteString("-->")
		return
	case DeclarationNode:
		if t.XMLDeclaration {
			buf.WriteString("<?" + n.Data + "?>")
		}
		return
	default:
		if len(n.Data) > 0 {
			if len(t.Ident) > 0 && level > 0 {
				buf.WriteString("\n")
				buf.WriteString(strings.Repeat(t.Ident, level))
			}

			if n.Prefix == "" {
				buf.WriteString("<" + n.Data)
			} else {
				buf.WriteString("<" + n.Prefix + ":" + n.Data)
			}
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

	if len(n.Data) > 0 {
		buf.WriteString(">")
	}

	t.recurse(buf, n, t.increment(n, level), preserveSpaces)

	if len(n.Data) > 0 {
		if len(t.Ident) > 0 && !t.isText(n.LastChild) {
			buf.WriteString("\n")
			buf.WriteString(strings.Repeat(t.Ident, level))
		}

		if n.Prefix == "" {
			buf.WriteString(fmt.Sprintf("</%s>", n.Data))
		} else {
			buf.WriteString(fmt.Sprintf("</%s:%s>", n.Prefix, n.Data))
		}
	}
}

func (t formatter) recurse(buf *bytes.Buffer, n *Node, level int, preserve bool) {
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		t.output(buf, child, level, preserve)
	}
}

func (t formatter) isText(n *Node) bool {
	if n == nil {
		return false
	}

	switch n.Type {
	case CharDataNode, TextNode:
		return strings.Trim(n.Data, "\n\t\r") != ""
	default:
		return false
	}
}
