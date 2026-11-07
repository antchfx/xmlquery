package xmlquery

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/antchfx/xpath"
)

// https://msdn.microsoft.com/en-us/library/ms762271(v=vs.85).aspx
const xmlDoc = `
    <?xml version="1.0"?>
<catalog>
   <!-- book list-->
   <book id="bk101">
      <author>Gambardella, Matthew</author>
      <title>XML Developer's Guide</title>
      <genre>Computer</genre>
      <price>44.95</price>
      <publish_date>2000-10-01</publish_date>
      <description>An in-depth look at creating applications
      with XML.</description>
   </book>
   <book id="bk102">
      <author>Ralls, Kim</author>
      <title>Midnight Rain</title>
      <genre>Fantasy</genre>
      <price>5.95</price>
      <publish_date>2000-12-16</publish_date>
      <description>A former architect battles corporate zombies,
      an evil sorceress, and her own childhood to become queen
      of the world.</description>
   </book>
   <book id="bk103">
      <author>Corets, Eva</author>
      <title>Maeve Ascendant</title>
      <genre>Fantasy</genre>
      <price>5.95</price>
      <publish_date>2000-11-17</publish_date>
      <description>After the collapse of a nanotechnology
      society in England, the young survivors lay the
      foundation for a new society.</description>
   </book>
</catalog>`

var doc = loadXML(xmlDoc)

func TestQueryAttributeNamespace(t *testing.T) {
	root, err := Parse(strings.NewReader(`<root xmlns:xlink="http://www.w3.org/1999/xlink"><node xlink:href="target" id="plain"/></root>`))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		expr, prefix, namespace, data, value string
	}{
		{"//@xlink:href", "xlink", "http://www.w3.org/1999/xlink", "href", "target"},
		{"//@id", "", "", "id", "plain"},
	} {
		node, err := Query(root, tc.expr)
		if err != nil || node == nil {
			t.Fatalf("Query(%q) = %v, %v", tc.expr, node, err)
		}
		if node.Prefix != tc.prefix || node.NamespaceURI != tc.namespace || node.Data != tc.data || node.InnerText() != tc.value {
			t.Errorf("Query(%q) = prefix %q, namespace %q, data %q, value %q", tc.expr, node.Prefix, node.NamespaceURI, node.Data, node.InnerText())
		}
	}
}

func TestXPath(t *testing.T) {
	if list := Find(doc, "//book"); len(list) != 3 {
		t.Fatal("count(//book) != 3")
	}
	if node := FindOne(doc, "//book[@id='bk101']"); node == nil {
		t.Fatal("//book[@id='bk101] is not found")
	}
	if node := FindOne(doc, "//book[price>=44.95]"); node == nil {
		t.Fatal("//book/price>=44.95 is not found")
	}
	if list := Find(doc, "//book[genre='Fantasy']"); len(list) != 2 {
		t.Fatal("//book[genre='Fantasy'] items count is not equal 2")
	}
	var c int
	FindEach(doc, "//book", func(i int, n *Node) {
		c++
	})
	l := len(Find(doc, "//book"))
	if c != l {
		t.Fatal("count(//book) != 3")
	}
	c = 0
	FindEachWithBreak(doc, "//book", func(i int, n *Node) bool {
		if c == l-1 {
			return false
		}
		c++
		return true
	})
	if c != l-1 {
		t.Fatal("FindEachWithBreak failed to stop.")
	}
	node := FindOne(doc, "//book[1]")
	if node.SelectAttr("id") != "bk101" {
		t.Fatal("//book[1]/@id != bk101")
	}
}

func TestXPathCdUp(t *testing.T) {
	doc := loadXML(`<a><b attr="1"/></a>`)
	node := FindOne(doc, "/a/b/@attr/..")
	t.Logf("node = %#v", node)
	if node == nil || node.Data != "b" {
		t.Fatal("//b/@id/.. != <b/>")
	}
}

func TestInvalidXPathExpression(t *testing.T) {
	doc := &Node{}
	_, err := QueryAll(doc, "//a[@a==1]")
	if err == nil {
		t.Fatal("expected a parsed error but nil")
	}
	_, err = Query(doc, "//a[@a==1]")
	if err == nil {
		t.Fatal("expected a parsed error but nil")
	}
}

func TestNavigator(t *testing.T) {
	nav := &NodeNavigator{curr: doc, root: doc, attr: -1}
	nav.MoveToChild() // New Line
	nav.MoveToNext()  // catalog
	if nav.curr.Data != "catalog" {
		t.Fatal("current node name != `catalog`")
	}
	nav.MoveToChild() // New Line
	nav.MoveToNext()  // comment node
	if nav.curr.Type != CommentNode {
		t.Fatal("node type not CommentNode")
	}
	nav.Value()
	nav.MoveToNext() //book
	nav.MoveToChild()
	nav.MoveToNext() // book/author
	if nav.LocalName() != "author" {
		t.Fatalf("node error")
	}
	nav.MoveToParent() // book
	nav.MoveToNext()   // next book
	if nav.curr.SelectAttr("id") != "bk102" {
		t.Fatal("node error")
	}
}

func TestAttributesNamespaces(t *testing.T) {
	doc := loadXML(`
		<root xmlns="ns://root" xmlns:nested="ns://nested" xmlns:other="ns://other">
			<tag id="1" attr="v"></tag>
			<tag id="2" nested:attr="v"></tag>
			<nested:tag id="3" nested:attr="v"></nested:tag>
			<nested:tag id="4" other:attr="v"></nested:tag>
			<nested:tag id="5" attr="v"></nested:tag>
		</root>
	`)
	results := Find(doc, "//*[@*[namespace-uri()='ns://nested' and local-name()='attr']]")
	parsed := make([]string, 0, 5)
	for _, tag := range results {
		parsed = append(parsed, tag.SelectAttr("id"))
	}
	got := fmt.Sprintf("%v", parsed)

	// unsure if 5 should be selected here
	if got != "[2 3]" {
		t.Fatalf("Expected tags [2 3], got %v", got)
	}
}

func loadXML(s string) *Node {
	node, err := Parse(strings.NewReader(s))
	if err != nil {
		panic(err)
	}
	return node
}

func TestMissingTextNodes(t *testing.T) {
	doc := loadXML(`
        <?xml version="1.0" encoding="utf-8"?>
        <corpus><p>Lorem <a>ipsum</a> dolor</p></corpus>
    `)
	results := Find(doc, "//text()")
	if len(results) != 3 {
		t.Fatalf("Expected text nodes 3, got %d", len(results))
	}
}

func TestXPathAttributeAxisExcludesNamespaceDeclarations(t *testing.T) {
	doc, err := Parse(strings.NewReader(`<root xmlns="urn:default" xmlns:p="urn:test" id="1" xml:lang="en" p:code="x" p:xmlns="ordinary"/>`))
	if err != nil {
		t.Fatal(err)
	}
	nodes := Find(doc, `//*[local-name()='root']/@*`)
	var names []string
	for _, n := range nodes {
		names = append(names, n.Prefix+":"+n.Data)
	}
	sort.Strings(names)
	want := []string{":id", "p:code", "p:xmlns", "xml:lang"}
	if len(names) != len(want) {
		t.Fatalf("attribute names=%v want%v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("attribute names=%v want%v", names, want)
		}
	}
	root := FindOne(doc, `//*[local-name()='root']`)
	if root.SelectAttr("xmlns:p") != "urn:test" || len(root.Attr) != 6 {
		t.Fatal("source namespace attributes were removed")
	}
	out := doc.OutputXML(false)
	if !strings.Contains(out, `xmlns="urn:default"`) || !strings.Contains(out, `xmlns:p="urn:test"`) {
		t.Fatalf("namespace declarations absent from serialized XML: %s", out)
	}
}
func TestNamespaceOnlyElementHasNoXPathAttributes(t *testing.T) {
	doc, err := Parse(strings.NewReader(`<root xmlns:p="urn:test"/>`))
	if err != nil {
		t.Fatal(err)
	}
	root := FindOne(doc, "//root")
	nav := CreateXPathNavigator(root)
	if nav.MoveToNextAttribute() {
		t.Error("namespace declaration exposed as an attribute")
	}
	if nav.LocalName() != "root" {
		t.Error("failed attribute move changed current node")
	}
}

func TestXPathDocumentAndCDATAStringValues(t *testing.T) {
	doc, err := Parse(strings.NewReader(`<root>before<item><![CDATA[42]]></item><!--excluded--><tail>after</tail></root>`))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		expr string
		want interface{}
	}{
		{`string(/)`, "before42after"}, {`string(//item/text())`, "42"}, {`number(//item/text())`, float64(42)}, {`boolean(//item/text()[.='42'])`, true}, {`string(//tail/text())`, "after"}, {`string(//comment())`, "excluded"},
	} {
		t.Run(tc.expr, func(t *testing.T) {
			got := xpath.MustCompile(tc.expr).Evaluate(CreateXPathNavigator(doc))
			if got != tc.want {
				t.Errorf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}
