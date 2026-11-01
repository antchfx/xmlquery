package xmlquery

import (
	"encoding/xml"
	"io"
	"strings"
	"testing"
)

func TestOutputRestoresNamespaceAfterChild(t *testing.T) {
	input := `<root xmlns:p="urn:test"><child xmlns:q="urn:test"><q:inside/></child><p:after p:id="v"/></root>`
	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	d := xml.NewDecoder(strings.NewReader(doc.OutputXML(false)))
	found := false
	for {
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if x, ok := tok.(xml.StartElement); ok && x.Name.Local == "after" {
			found = true
			if x.Name.Space != "urn:test" {
				t.Errorf("namespace lost: %#v output%s", x.Name, doc.OutputXML(false))
			}
			if len(x.Attr) != 1 || x.Attr[0].Name != (xml.Name{Space: "urn:test", Local: "id"}) || x.Attr[0].Value != "v" {
				t.Errorf("attribute namespace/value lost: %#v", x.Attr)
			}
		}
	}
	if !found {
		t.Fatal("after missing")
	}
}

func TestNamespaceScopeAfterDefaultNamespaceChild(t *testing.T) {
	doc, err := Parse(strings.NewReader(`<root xmlns:p="urn:test"><child xmlns="urn:test"><inside/></child><p:after/></root>`))
	if err != nil {
		t.Fatal(err)
	}
	after := FindOne(doc, "//*[local-name()='after']")
	if after == nil {
		t.Fatal("after missing")
	}
	if after.Prefix != "p" || after.NamespaceURI != "urn:test" {
		t.Errorf("after=%#v", after)
	}
}

func TestStreamNamespaceScopeBetweenRecords(t *testing.T) {
	sp, err := CreateStreamParser(strings.NewReader(`<root xmlns:p="urn:test"><child xmlns:q="urn:test"><q:inside/></child><p:after/></root>`), "/root/*")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = sp.Read(); err != nil {
		t.Fatal(err)
	}
	n, err := sp.Read()
	if err != nil {
		t.Fatal(err)
	}
	if n.Data != "after" || n.Prefix != "p" || n.NamespaceURI != "urn:test" {
		t.Errorf("unexpected streamed node: %#v", n)
	}
	if _, err = sp.Read(); err != io.EOF {
		t.Errorf("want EOF, got %v", err)
	}
}

func TestNestedNamespaceAliasScope(t *testing.T) {
	doc, err := Parse(strings.NewReader(`<root xmlns:p="urn:test"><child xmlns:q="urn:test"><grandchild xmlns:r="urn:test"><r:inside/></grandchild><q:middle q:id="inner"/></child><p:after p:id="outer"/></root>`))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, prefix, value string }{{"middle", "q", "inner"}, {"after", "p", "outer"}} {
		n := FindOne(doc, "//*[local-name()='"+tc.name+"']")
		if n == nil {
			t.Fatal(tc.name)
		}
		if n.Prefix != tc.prefix || n.NamespaceURI != "urn:test" || n.SelectAttr(tc.prefix+":id") != tc.value {
			t.Errorf("namespace scope not restored for%s: %#v", tc.name, n)
		}
	}
	decoder := xml.NewDecoder(strings.NewReader(doc.OutputXML(false)))
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if n, ok := tok.(xml.StartElement); ok && (n.Name.Local == "middle" || n.Name.Local == "after") {
			if n.Name.Space != "urn:test" || len(n.Attr) != 1 || n.Attr[0].Name.Space != "urn:test" {
				t.Errorf("invalid roundtrip expanded names: %#v", n)
			}
		}
	}
}
