package xmlquery

import (
	"encoding/xml"
	"html"
	"reflect"
	"strings"
	"testing"
)

func findRoot(n *Node) *Node {
	if n == nil {
		return nil
	}
	for ; n.Parent != nil; n = n.Parent {
	}
	return n
}

func findNode(root *Node, name string) *Node {
	node := root.FirstChild
	for {
		if node == nil || node.Data == name {
			break
		}
		node = node.NextSibling
	}
	return node
}

func childNodes(root *Node, name string) []*Node {
	var list []*Node
	node := root.FirstChild
	for {
		if node == nil {
			break
		}
		if node.Data == name {
			list = append(list, node)
		}
		node = node.NextSibling
	}
	return list
}

func testNode(t *testing.T, n *Node, expected string) {
	if n.Data != expected {
		t.Fatalf("expected node name is %s,but got %s", expected, n.Data)
	}
}

func testAttr(t *testing.T, n *Node, name, expected string) {
	for _, attr := range n.Attr {
		if attr.Name.Local == name && attr.Value == expected {
			return
		}
	}
	t.Fatalf("not found attribute %s in the node %s", name, n.Data)
}

func testValue(t *testing.T, val, expected interface{}) {
	if val == expected {
		return
	}
	if reflect.DeepEqual(val, expected) {
		return
	}
	t.Fatalf("expected value is %+v, but got %+v", expected, val)
}

func testTrue(t *testing.T, v bool) {
	if v {
		return
	}
	t.Fatal("expected value is true, but got false")
}

// Given a *Node, verify that all the pointers (parent, first child, next sibling, etc.) of
// - the node itself,
// - all its child nodes, and
// - pointers along the silbling chain
// are valid.
func verifyNodePointers(t *testing.T, n *Node) {
	if n == nil {
		return
	}
	if n.FirstChild != nil {
		testValue(t, n, n.FirstChild.Parent)
	}
	if n.LastChild != nil {
		testValue(t, n, n.LastChild.Parent)
	}

	verifyNodePointers(t, n.FirstChild)
	// There is no need to call verifyNodePointers(t, n.LastChild)
	// because verifyNodePointers(t, n.FirstChild) will traverse all its
	// siblings to the end, and if the last one isn't n.LastChild then it will fail.

	parent := n.Parent // parent could be nil if n is the root of a tree.

	// Verify the PrevSibling chain
	cur, prev := n, n.PrevSibling
	for ; prev != nil; cur, prev = prev, prev.PrevSibling {
		testValue(t, prev.Parent, parent)
		testValue(t, prev.NextSibling, cur)
	}
	testTrue(t, cur.PrevSibling == nil)
	testTrue(t, parent == nil || parent.FirstChild == cur)

	// Verify the NextSibling chain
	cur, next := n, n.NextSibling
	for ; next != nil; cur, next = next, next.NextSibling {
		testValue(t, next.Parent, parent)
		testValue(t, next.PrevSibling, cur)
	}
	testTrue(t, cur.NextSibling == nil)
	testTrue(t, parent == nil || parent.LastChild == cur)
}

func TestAddAttr(t *testing.T) {
	for _, test := range []struct {
		name     string
		n        *Node
		key      string
		val      string
		expected string
	}{
		{
			name:     "node has no existing attr",
			n:        &Node{Type: AttributeNode},
			key:      "ns:k1",
			val:      "v1",
			expected: `< ns:k1="v1"></>`,
		},
		{
			name:     "node has existing attrs",
			n:        &Node{Type: AttributeNode, Attr: []Attr{{Name: xml.Name{Local: "k1"}, Value: "v1"}}},
			key:      "k2",
			val:      "v2",
			expected: `< k1="v1" k2="v2"></>`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			AddAttr(test.n, test.key, test.val)
			testValue(t, test.n.OutputXML(true), test.expected)
		})
	}
}

func TestSetAttr(t *testing.T) {
	for _, test := range []struct {
		name     string
		n        *Node
		key      string
		val      string
		expected string
	}{
		{
			name:     "node has no existing attr",
			n:        &Node{Type: AttributeNode},
			key:      "ns:k1",
			val:      "v1",
			expected: `< ns:k1="v1"></>`,
		},
		{
			name:     "node has an existing attr, overwriting",
			n:        &Node{Type: AttributeNode, Attr: []Attr{{Name: xml.Name{Space: "ns", Local: "k1"}, Value: "v1"}}},
			key:      "ns:k1",
			val:      "v2",
			expected: `< ns:k1="v2"></>`,
		},
		{
			name:     "node has no existing attr, no ns",
			n:        &Node{Type: AttributeNode},
			key:      "k1",
			val:      "v1",
			expected: `< k1="v1"></>`,
		},
		{
			name:     "node has an existing attr, no ns, overwriting",
			n:        &Node{Type: AttributeNode, Attr: []Attr{{Name: xml.Name{Local: "k1"}, Value: "v1"}}},
			key:      "k1",
			val:      "v2",
			expected: `< k1="v2"></>`,
		},
	} {

		t.Run(test.name, func(t *testing.T) {
			test.n.SetAttr(test.key, test.val)
			testValue(t, test.n.OutputXML(true), test.expected)
		})
	}
}

func TestRemoveAttr(t *testing.T) {
	for _, test := range []struct {
		name     string
		n        *Node
		key      string
		expected string
	}{
		{
			name:     "node has no existing attr",
			n:        &Node{Type: AttributeNode},
			key:      "ns:k1",
			expected: `<></>`,
		},
		{
			name:     "node has an existing attr, overwriting",
			n:        &Node{Type: AttributeNode, Attr: []Attr{{Name: xml.Name{Space: "ns", Local: "k1"}, Value: "v1"}}},
			key:      "ns:k1",
			expected: `<></>`,
		},
		{
			name:     "node has no existing attr, no ns",
			n:        &Node{Type: AttributeNode},
			key:      "k1",
			expected: `<></>`,
		},
		{
			name:     "node has an existing attr, no ns, overwriting",
			n:        &Node{Type: AttributeNode, Attr: []Attr{{Name: xml.Name{Local: "k1"}, Value: "v1"}}},
			key:      "k1",
			expected: `<></>`,
		},
	} {

		t.Run(test.name, func(t *testing.T) {
			test.n.RemoveAttr(test.key)
			testValue(t, test.n.OutputXML(true), test.expected)
		})
	}
}

func TestRemoveFromTree(t *testing.T) {
	xml := `<?procinst?>
		<!--comment-->
		<aaa><bbb/>
			<ddd><eee><fff/></eee></ddd>
		<ggg/></aaa>`
	parseXML := func() *Node {
		doc, err := Parse(strings.NewReader(xml))
		testTrue(t, err == nil)
		return doc
	}

	t.Run("remove an elem node that is the only child of its parent", func(t *testing.T) {
		doc := parseXML()
		n := FindOne(doc, "//aaa/ddd/eee")
		testTrue(t, n != nil)
		RemoveFromTree(n)
		verifyNodePointers(t, doc)
		testValue(t, doc.OutputXML(false),
			`<?procinst?><!--comment--><aaa><bbb></bbb><ddd></ddd><ggg></ggg></aaa>`)
	})

	t.Run("remove an elem node that is the first but not the last child of its parent", func(t *testing.T) {
		doc := parseXML()
		n := FindOne(doc, "//aaa/bbb")
		testTrue(t, n != nil)
		RemoveFromTree(n)
		verifyNodePointers(t, doc)
		testValue(t, doc.OutputXML(false),
			`<?procinst?><!--comment--><aaa><ddd><eee><fff></fff></eee></ddd><ggg></ggg></aaa>`)
	})

	t.Run("remove an elem node that is neither the first nor  the last child of its parent", func(t *testing.T) {
		doc := parseXML()
		n := FindOne(doc, "//aaa/ddd")
		testTrue(t, n != nil)
		RemoveFromTree(n)
		verifyNodePointers(t, doc)
		testValue(t, doc.OutputXML(false),
			`<?procinst?><!--comment--><aaa><bbb></bbb><ggg></ggg></aaa>`)
	})

	t.Run("remove an elem node that is the last but not the first child of its parent", func(t *testing.T) {
		doc := parseXML()
		n := FindOne(doc, "//aaa/ggg")
		testTrue(t, n != nil)
		RemoveFromTree(n)
		verifyNodePointers(t, doc)
		testValue(t, doc.OutputXML(false),
			`<?procinst?><!--comment--><aaa><bbb></bbb><ddd><eee><fff></fff></eee></ddd></aaa>`)
	})

	t.Run("remove decl node works", func(t *testing.T) {
		doc := parseXML()
		procInst := doc.FirstChild
		testValue(t, procInst.Type, DeclarationNode)
		RemoveFromTree(procInst)
		verifyNodePointers(t, doc)
		testValue(t, doc.OutputXML(false),
			`<!--comment--><aaa><bbb></bbb><ddd><eee><fff></fff></eee></ddd><ggg></ggg></aaa>`)
	})

	t.Run("remove comment node works", func(t *testing.T) {
		doc := parseXML()
		commentNode := doc.FirstChild.NextSibling.NextSibling // First .NextSibling is an empty text node.
		testValue(t, commentNode.Type, CommentNode)
		RemoveFromTree(commentNode)
		verifyNodePointers(t, doc)
		testValue(t, doc.OutputXML(false),
			`<?procinst?><aaa><bbb></bbb><ddd><eee><fff></fff></eee></ddd><ggg></ggg></aaa>`)
	})

	t.Run("remove call on root does nothing", func(t *testing.T) {
		doc := parseXML()
		RemoveFromTree(doc)
		verifyNodePointers(t, doc)
		testValue(t, doc.OutputXML(false),
			`<?procinst?><!--comment--><aaa><bbb></bbb><ddd><eee><fff></fff></eee></ddd><ggg></ggg></aaa>`)
	})
}

func TestSelectElement(t *testing.T) {
	s := `<?xml version="1.0" encoding="UTF-8"?>
    <AAA>
        <BBB id="1"/>
        <CCC id="2">
            <DDD/>
        </CCC>
		<CCC id="3">
            <DDD/>
        </CCC>
     </AAA>`
	root, err := Parse(strings.NewReader(s))
	if err != nil {
		t.Error(err)
	}
	version := root.FirstChild.SelectAttr("version")
	if version != "1.0" {
		t.Fatal("version!=1.0")
	}
	aaa := findNode(root, "AAA")
	var n *Node
	n = aaa.SelectElement("BBB")
	if n == nil {
		t.Fatalf("n is nil")
	}
	n = aaa.SelectElement("CCC")
	if n == nil {
		t.Fatalf("n is nil")
	}

	var ns []*Node
	ns = aaa.SelectElements("CCC")
	if len(ns) != 2 {
		t.Fatalf("len(ns)!=2")
	}
}

func TestEscapeOutputValue(t *testing.T) {
	data := `<AAA>&lt;*&gt;</AAA>`

	root, err := Parse(strings.NewReader(data))
	if err != nil {
		t.Error(err)
	}

	escapedInnerText := root.OutputXML(true)
	if !strings.Contains(escapedInnerText, "&lt;*&gt;") {
		t.Fatal("Inner Text has not been escaped")
	}

}

func TestUnnecessaryEscapeOutputValue(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
	<class_list xml:space="preserve">
		<student>
			<name> Robert </name>
			<grade>A+</grade>

		</student>
	</class_list>`

	root, err := Parse(strings.NewReader(data))
	if err != nil {
		t.Error(err)
	}

	escapedInnerText := root.OutputXML(true)
	if strings.Contains(escapedInnerText, "&#x9") {
		t.Fatal("\\n has been escaped unnecessarily")
	}

	if strings.Contains(escapedInnerText, "&#xA") {
		t.Fatal("\\t has been escaped unnecessarily")
	}

}

func TestHtmlUnescapeStringOriginString(t *testing.T) {
	// has escape html character and \t
	data := `<?xml version="1.0" encoding="utf-8"?>
	<example xml:space="preserve"><word>&amp;#48;		</word></example>`

	root, err := Parse(strings.NewReader(data))
	if err != nil {
		t.Error(err)
	}

	escapedInnerText := root.OutputXML(false)
	unescapeString := html.UnescapeString(escapedInnerText)
	if strings.Contains(unescapeString, "&amp;") {
		t.Fatal("&amp; need unescape")
	}
	if !strings.Contains(escapedInnerText, "&amp;#48;\t\t") {
		t.Fatal("Inner Text should keep plain text")
	}

}

func TestOutputXMLWithNamespacePrefix(t *testing.T) {
	s := `<?xml version="1.0" encoding="UTF-8"?><S:Envelope xmlns:S="http://schemas.xmlsoap.org/soap/envelope/"><S:Body></S:Body></S:Envelope>`
	doc, _ := Parse(strings.NewReader(s))
	if s != doc.OutputXML(false) {
		t.Fatal("xml document missing some characters")
	}
}

func TestOutputXMLWithCommentNode(t *testing.T) {
	s := `<?xml version="1.0" encoding="utf-8"?>
	<!-- Students grades are updated bi-monthly -->
	<class_list>
		<student>
			<name>Robert</name>
			<grade>A+</grade>
		</student>
	<!--
		<student>
			<name>Lenard</name>
			<grade>A-</grade>
		</student>
	-->
	</class_list>`
	doc, _ := Parse(strings.NewReader(s))
	t.Log(doc.OutputXML(true))
	if e, g := "<!-- Students grades are updated bi-monthly -->", doc.OutputXML(true); !strings.Contains(g, e) {
		t.Fatal("missing some comment-node.")
	}
	n := FindOne(doc, "//class_list")
	t.Log(n.OutputXML(false))
	if e, g := "<name>Lenard</name>", n.OutputXML(false); !strings.Contains(g, e) {
		t.Fatal("missing some comment-node")
	}
}

func TestOutputXMLWithSpaceParent(t *testing.T) {
	s := `<?xml version="1.0" encoding="utf-8"?>
	<class_list>
		<student xml:space="preserve">
			<name> Robert </name>
			<grade>A+</grade>
		</student>
	</class_list>`
	doc, _ := Parse(strings.NewReader(s))
	t.Log(doc.OutputXML(true))

	expected := "<name> Robert </name>"
	if g := doc.OutputXML(true); !strings.Contains(g, expected) {
		t.Errorf(`expected "%s", obtained "%s"`, expected, g)
	}

	n := FindOne(doc, "/class_list/student")
	output := html.UnescapeString(n.OutputXML(false))
	expected = "\n\t\t\t<name> Robert </name>\n\t\t\t<grade>A+</grade>\n\t\t"
	if !(output == expected) {
		t.Errorf(`expected "%s", obtained "%s"`, expected, output)
	}
	t.Log(n.OutputXML(false))
}

func TestOutputXMLWithSpaceDirect(t *testing.T) {
	s := `<?xml version="1.0" encoding="utf-8"?>
	<class_list>
		<student>
			<name xml:space="preserve"> Robert </name>
			<grade>A+</grade>
		</student>
	</class_list>`
	doc, _ := Parse(strings.NewReader(s))
	t.Log(doc.OutputXML(true))

	n := FindOne(doc, "/class_list/student/name")
	expected := `<name xml:space="preserve"> Robert </name>`
	if g := doc.OutputXML(false); !strings.Contains(g, expected) {
		t.Errorf(`expected "%s", obtained "%s"`, expected, g)
	}

	output := html.UnescapeString(doc.OutputXML(true))
	if strings.Contains(output, "\n") {
		t.Errorf("the outputted xml contains newlines")
	}
	t.Log(n.OutputXML(false))
}

func TestOutputXMLWithSpaceOverwrittenToPreserve(t *testing.T) {
	s := `<?xml version="1.0" encoding="utf-8"?>
	<class_list>
		<student xml:space="default">
			<name xml:space="preserve"> Robert </name>
			<grade>A+</grade>
		</student>
	</class_list>`
	doc, _ := Parse(strings.NewReader(s))
	t.Log(doc.OutputXML(true))

	n := FindOne(doc, "/class_list/student")
	expected := `<name xml:space="preserve"> Robert </name>`
	if g := n.OutputXML(false); !strings.Contains(g, expected) {
		t.Errorf(`expected "%s", obtained "%s"`, expected, g)
	}

	output := html.UnescapeString(doc.OutputXML(true))
	if strings.Contains(output, "\n") {
		t.Errorf("the outputted xml contains newlines")
	}
	t.Log(n.OutputXML(false))
}

func TestOutputXMLWithSpaceOverwrittenToDefault(t *testing.T) {
	s := `<?xml version="1.0" encoding="utf-8"?>
	<class_list>
		<student xml:space="preserve">
			<name xml:space="default"> Robert </name>
			<grade>A+</grade>
		</student>
	</class_list>`
	doc, _ := Parse(strings.NewReader(s))
	t.Log(doc.OutputXML(true))

	expected := `<name xml:space="default">Robert</name>`
	if g := doc.OutputXML(false); !strings.Contains(g, expected) {
		t.Errorf(`expected "%s", obtained "%s"`, expected, g)
	}

	n := FindOne(doc, "/class_list/student")
	output := html.UnescapeString(n.OutputXML(false))
	expected = "\n\t\t\t<name xml:space=\"default\">Robert</name>\n\t\t\t<grade>A+</grade>\n\t\t"
	if !(output == expected) {
		t.Errorf(`expected "%s", obtained "%s"`, expected, output)
	}
	t.Log(n.OutputXML(false))
}

func TestOutputXMLWithXMLInCDATA(t *testing.T) {
	s := `<?xml version="1.0" encoding="utf-8"?><node><![CDATA[<greeting>Hello, world!</greeting>]]></node>`
	doc, _ := Parse(strings.NewReader(s))
	t.Log(doc.OutputXML(false))
	if doc.OutputXML(false) != s {
		t.Errorf("the outputted xml escaped CDATA section")
	}
}

func TestOutputXMLWithDefaultOptions(t *testing.T) {
	s := `<?xml version="1.0" encoding="utf-8"?><node><empty></empty></node>`
	expected := `<?xml version="1.0" encoding="utf-8"?><node><empty></empty></node>`

	doc, _ := Parse(strings.NewReader(s))
	result := doc.OutputXMLWithOptions()
	t.Log(result)
	if result != expected {
		t.Errorf("output was not expected. expected %v but got %v", expected, result)
	}
}

func TestOutputXMLWithOptions(t *testing.T) {
	s := `<?xml version="1.0" encoding="utf-8"?><node><empty></empty></node>`
	expected := `<?xml version="1.0" encoding="utf-8"?><node><empty/></node>`

	doc, _ := Parse(strings.NewReader(s))
	result := doc.OutputXMLWithOptions(WithEmptyTagSupport())
	t.Log(result)
	if result != expected {
		t.Errorf("output was not expected. expected %v but got %v", expected, result)
	}
}
