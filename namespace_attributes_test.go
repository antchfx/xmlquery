package xmlquery

import (
	"sort"
	"strings"
	"testing"
)

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
