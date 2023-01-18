package xmlquery

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoadURLSuccess(t *testing.T) {
	contentTypes := []string{
		"application/vnd.paos.xml",
		"application/vnd.otps.ct-kip+xml",
		"application/vnd.openxmlformats-package.core-properties+xml",
		"application/CDFX+XML",
		"application/ATXML",
		"application/3gpdash-qoe-report+xml",
		"application/vnd.nokia.pcd+wbxml",
		"image/svg+xml",
		"message/imdn+xml",
		"model/vnd.collada+xml",
		"text/xml-external-parsed-entity",
		"text/xml",
		"aPPLIcaTioN/xMl; charset=UTF-8",
		"application/xhtml+xml",
		"application/xml",
		"text/xmL; charset=UTF-8",
		"application/aTOM+xmL; charset=UTF-8",
		"application/RsS+xmL; charset=UTF-8",
		"application/maTHml+xmL; charset=UTF-8",
		"application/xslt+xmL; charset=UTF-8",
	}

	for _, contentType := range contentTypes {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s := `<?xml version="1.0"?>
				<parent>
					<child></child>
				</parent>
			`
			w.Header().Set("Content-Type", contentType)
			w.Write([]byte(s))
		}))
		defer server.Close()
		_, err := LoadURL(server.URL)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestLoadURLFailure(t *testing.T) {
	contentTypes := []string{
		"application/pdf",
		"application/json",
		"application/tlsrpt+gzip",
		"application/vnd.3gpp.pic-bw-small",
		"application/vnd.collabio.xodocuments.document-template",
		"application/vnd.ctc-posml",
		"application/vnd.gov.sk.e-form+zip",
		"audio/mp4",
		"audio/vnd.sealedmedia.softseal.mpeg",
		"image/png",
		"image/vnd.adobe.photoshop",
		"message/example",
		"message/vnd.wfa.wsc",
		"model/vnd.usdz+zip",
		"model/vnd.valve.source.compiled-map",
		"multipart/signed",
		"text/css",
		"text/html",
		"video/quicktime",
		"video/JPEG",
	}

	for _, contentType := range contentTypes {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", contentType)
		}))
		defer server.Close()
		_, err := LoadURL(server.URL)
		if err != nil && err.Error() == fmt.Sprintf("invalid XML document(%s)", contentType) {
			return
		}

		t.Fatalf("Want invalid XML document(%s), got %v", contentType, err)
	}
}

func TestDefaultNamespace_1(t *testing.T) {
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
	<svg
	   xmlns:svg="http://www.w3.org/2000/svg"
	   xmlns="http://www.w3.org/2000/svg"
	>
	   <text xml:space="preserve">
		  <tspan>Multiline</tspan>
		  <tspan>Multiline text</tspan>
	   </text>
	</svg>`

	doc, err := Parse(strings.NewReader(s))
	if err != nil {
		t.Fatal(err)
	}
	if n := FindOne(doc, "//svg"); n == nil {
		t.Fatal("should find a `svg` but got nil")
	}
	list := Find(doc, "//tspan")
	if found, expected := len(list), 2; found != expected {
		t.Fatalf("should found %d tspan but found %d", expected, found)
	}
}

func TestDefaultNamespace_2(t *testing.T) {
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
	<svg
	   xmlns="http://www.w3.org/2000/svg"
	   xmlns:svg="http://www.w3.org/2000/svg"
	>
	   <text xml:space="preserve">
		  <tspan>Multiline</tspan>
		  <tspan>Multiline text</tspan>
	   </text>
	</svg>`

	doc, err := Parse(strings.NewReader(s))
	if err != nil {
		t.Fatal(err)
	}
	if n := FindOne(doc, "//svg"); n == nil {
		t.Fatal("should find a `svg` but got nil")
	}
	list := Find(doc, "//tspan")
	if found, expected := len(list), 2; found != expected {
		t.Fatalf("should found %d tspan but found %d", expected, found)
	}
}

func TestNamespaceURL(t *testing.T) {
	s := `
<?xml version="1.0"?>
<rss version="2.0" xmlns="http://www.example.com/" xmlns:dc="https://purl.org/dc/elements/1.1/">
<!-- author -->
<dc:creator><![CDATA[Richard ]]><![CDATA[Lawler]]></dc:creator>
<dc:identifier>21|22021348</dc:identifier>
</rss>
	`
	doc, err := Parse(strings.NewReader(s))
	if err != nil {
		t.Fatal(err)
	}
	top := FindOne(doc, "//rss")
	if top == nil {
		t.Fatal("rss feed invalid")
	}
	node := FindOne(top, "dc:creator")
	if node.Prefix != "dc" {
		t.Fatalf("expected node prefix name is dc but is=%s", node.Prefix)
	}
	if node.NamespaceURI != "https://purl.org/dc/elements/1.1/" {
		t.Fatalf("dc:creator != %s", node.NamespaceURI)
	}
	if strings.Index(top.InnerText(), "author") > 0 {
		t.Fatalf("InnerText() include comment node text")
	}
	if !strings.Contains(top.OutputXML(true), "author") {
		t.Fatal("OutputXML shoud include comment node,but not")
	}
}

func TestMultipleProcInst(t *testing.T) {
	s := `<?xml version="1.0" encoding="UTF-8"?>
<?xml-stylesheet type="text/xsl" media="screen" href="/~d/styles/rss2full.xsl"?>
<?xml-stylesheet type="text/css" media="screen" href="http://feeds.reuters.com/~d/styles/itemcontent.css"?>
<rss xmlns:feedburner="http://rssnamespace.org/feedburner/ext/1.0" version="2.0">
</rss>
	`
	doc, err := Parse(strings.NewReader(s))
	if err != nil {
		t.Fatal(err)
	}

	node := doc.FirstChild // <?xml ?>
	if node.Data != "xml" {
		t.Fatal("node.Data != xml")
	}
	node = node.NextSibling // New Line
	node = node.NextSibling // <?xml-stylesheet?>
	if node.Data != "xml-stylesheet" {
		t.Fatal("node.Data != xml-stylesheet")
	}
}

func TestParse(t *testing.T) {
	s := `<?xml version="1.0" encoding="UTF-8"?>
<bookstore>
<book>
  <title lang="en">Harry Potter</title>
  <price>29.99</price>
</book>
<book>
  <title lang="en">Learning XML</title>
  <price>39.95</price>
</book>
</bookstore>`
	root, err := Parse(strings.NewReader(s))
	if err != nil {
		t.Error(err)
	}
	if root.Type != DocumentNode {
		t.Fatal("top node of tree is not DocumentNode")
	}

	declarNode := root.FirstChild
	if declarNode.Type != DeclarationNode {
		t.Fatal("first child node of tree is not DeclarationNode")
	}

	if declarNode.Attr[0].Name.Local != "version" && declarNode.Attr[0].Value != "1.0" {
		t.Fatal("version attribute not expected")
	}

	bookstore := root.LastChild
	if bookstore.Data != "bookstore" {
		t.Fatal("bookstore elem not found")
	}
	if bookstore.FirstChild.Data != "\n" {
		t.Fatal("first child node of bookstore is not empty node(\n)")
	}
	books := childNodes(bookstore, "book")
	if len(books) != 2 {
		t.Fatalf("expected book element count is 2, but got %d", len(books))
	}
	// first book element
	testNode(t, findNode(books[0], "title"), "title")
	testAttr(t, findNode(books[0], "title"), "lang", "en")
	testValue(t, findNode(books[0], "price").InnerText(), "29.99")
	testValue(t, findNode(books[0], "title").InnerText(), "Harry Potter")

	// second book element
	testNode(t, findNode(books[1], "title"), "title")
	testAttr(t, findNode(books[1], "title"), "lang", "en")
	testValue(t, findNode(books[1], "price").InnerText(), "39.95")

	testValue(t, books[0].OutputXML(true), `<book><title lang="en">Harry Potter</title><price>29.99</price></book>`)
}

func TestMissDeclaration(t *testing.T) {
	s := `<AAA>
		<BBB></BBB>
		<CCC></CCC>
	</AAA>`
	doc, err := Parse(strings.NewReader(s))
	if err != nil {
		t.Fatal(err)
	}
	node := FindOne(doc, "//AAA")
	if node == nil {
		t.Fatal("//AAA is nil")
	}
}

func TestMissingNamespace(t *testing.T) {
	s := `<root>
	<myns:child id="1">value 1</myns:child>
	<myns:child id="2">value 2</myns:child>
  </root>`
	_, err := Parse(strings.NewReader(s))
	if err == nil {
		t.Fatal("err is nil, want got invalid XML document")
	}
}

func TestTooNested(t *testing.T) {
	s := `<?xml version="1.0" encoding="UTF-8"?>
	<!-- comment here-->
    <AAA>
        <BBB>
            <DDD>
                <CCC>
                    <DDD/>
                    <EEE/>
                </CCC>
            </DDD>
        </BBB>
        <CCC>
            <DDD>
                <EEE>
                    <DDD>
                        <FFF/>
                    </DDD>
                </EEE>
            </DDD>
        </CCC>
     </AAA>`
	root, err := Parse(strings.NewReader(s))
	if err != nil {
		t.Error(err)
	}
	aaa := findNode(root, "AAA")
	if aaa == nil {
		t.Fatal("AAA node not exists")
	}
	ccc := aaa.LastChild.PrevSibling
	if ccc.Data != "CCC" {
		t.Fatalf("expected node is CCC,but got %s", ccc.Data)
	}
	bbb := ccc.PrevSibling.PrevSibling
	if bbb.Data != "BBB" {
		t.Fatalf("expected node is bbb,but got %s", bbb.Data)
	}
	ddd := findNode(bbb, "DDD")
	testNode(t, ddd, "DDD")
	testNode(t, ddd.LastChild.PrevSibling, "CCC")
}

func TestAttributeWithNamespace(t *testing.T) {
	s := `<?xml version="1.0" encoding="UTF-8"?><root xmlns:n1="http://www.w3.org">
   <good a="1" b="2" />
   <good a="1" n1:a="2" /></root>`
	doc, _ := Parse(strings.NewReader(s))
	n := FindOne(doc, "//good[@n1:a='2']")
	if n == nil {
		t.Fatal("n is nil")
	}
}

func TestIllegalAttributeChars(t *testing.T) {
	s := `<MyTag attr="If a&lt;b &amp; b&lt;c then a&lt;c, it&#39;s obvious"></MyTag>`
	doc, _ := Parse(strings.NewReader(s))
	e := "If a<b & b<c then a<c, it's obvious"
	if n := FindOne(doc, "//MyTag/@attr"); n.InnerText() != e {
		t.Fatalf("MyTag expected: %s but got: %s", e, n.InnerText())
	}
	if g := doc.LastChild.OutputXML(true); g != s {
		t.Fatalf("not expected body: %s", g)
	}
}

func TestCharData(t *testing.T) {
	s := `
<?xml version="1.0"?>
<rss version="2.0" xmlns="http://www.example.com/" xmlns:dc="https://purl.org/dc/elements/1.1/">
<dc:creator><![CDATA[Richard Lawler]]></dc:creator>
</rss>
	`
	doc, err := Parse(strings.NewReader(s))
	if err != nil {
		t.Fatal(err)
	}
	top := FindOne(doc, "//rss")
	if top == nil {
		t.Fatal("rss feed invalid")
	}
	node := FindOne(top, "dc:creator")
	if node.Prefix != "dc" {
		t.Fatalf("expected node prefix name is dc but is=%s", node.Prefix)
	}
	cdata := node.FirstChild
	if cdata == nil || cdata.Type != CharDataNode {
		t.Fatalf("expected cdata child, received %d", cdata.Type)
	}

	testValue(t, cdata.InnerText(), "Richard Lawler")
}

func TestStreamParser_InvalidXPath(t *testing.T) {
	sp, err := CreateStreamParser(strings.NewReader(""), "[invalid")
	if err == nil || err.Error() != "invalid streamElementXPath '[invalid', err: expression must evaluate to a node-set" {
		t.Fatalf("got non-expected error: %v", err)
	}
	if sp != nil {
		t.Fatal("expected nil for sp, but got none-nil value")
	}

	sp, err = CreateStreamParser(strings.NewReader(""), ".", "[invalid")
	if err == nil || err.Error() != "invalid streamElementFilter '[invalid', err: expression must evaluate to a node-set" {
		t.Fatalf("got non-expected error: %v", err)
	}
	if sp != nil {
		t.Fatal("expected nil for sp, but got none-nil value")
	}
}

func testOutputXML(t *testing.T, msg string, expectedXML string, n *Node) {
	if n.OutputXML(true) != expectedXML {
		t.Fatalf("%s, expected XML: '%s', actual: '%s'", msg, expectedXML, n.OutputXML(true))
	}
}

func TestStreamParser_Success1(t *testing.T) {
	s := `
	<ROOT>
		<AAA>
			<CCC>c1</CCC>
			<BBB>b1</BBB>
			<DDD>d1</DDD>
			<BBB>b2<ZZZ z="1">z1</ZZZ></BBB>
			<BBB>b3</BBB>
		</AAA>
		<ZZZ>
			<BBB>b4</BBB>
			<BBB>b5</BBB>
			<CCC>c3</CCC>
		</ZZZ>
	</ROOT>`

	sp, err := CreateStreamParser(strings.NewReader(s), "/ROOT/*/BBB", "/ROOT/*/BBB[. != 'b3']")
	if err != nil {
		t.Fatal(err.Error())
	}

	// First `<BBB>` read
	n, err := sp.Read()
	if err != nil {
		t.Fatal(err.Error())
	}
	testOutputXML(t, "first call result", `<BBB>b1</BBB>`, n)
	testOutputXML(t, "doc after first call",
		`<?xml version="1.0"?><ROOT><AAA><CCC>c1</CCC><BBB>b1</BBB></AAA></ROOT>`, findRoot(n))

	// Second `<BBB>` read
	n, err = sp.Read()
	if err != nil {
		t.Fatal(err.Error())
	}
	testOutputXML(t, "second call result", `<BBB>b2<ZZZ z="1">z1</ZZZ></BBB>`, n)
	testOutputXML(t, "doc after second call",
		`<?xml version="1.0"?><ROOT><AAA><DDD>d1</DDD><BBB>b2<ZZZ z="1">z1</ZZZ></BBB></AAA></ROOT>`, findRoot(n))

	// Third `<BBB>` read (Note we will skip 'b3' since the streamElementFilter excludes it)
	n, err = sp.Read()
	if err != nil {
		t.Fatal(err.Error())
	}
	testOutputXML(t, "third call result", `<BBB>b4</BBB>`, n)
	// Note the inclusion of `<BBB>b3</BBB>` in the document? This is because `<BBB>b3</BBB>` has
	// been filtered out and is not our target node, thus it is considered just like any other
	// non target nodes such as `<CCC>`` or `<DDD>`
	testOutputXML(t, "doc after third call",
		`<?xml version="1.0"?><ROOT><AAA></AAA><ZZZ><BBB>b4</BBB></ZZZ></ROOT>`,
		findRoot(n))

	// Fourth `<BBB>` read
	n, err = sp.Read()
	if err != nil {
		t.Fatal(err.Error())
	}
	testOutputXML(t, "fourth call result", `<BBB>b5</BBB>`, n)
	testOutputXML(t, "doc after fourth call",
		`<?xml version="1.0"?><ROOT><AAA></AAA><ZZZ><BBB>b5</BBB></ZZZ></ROOT>`,
		findRoot(n))

	_, err = sp.Read()
	if err != io.EOF {
		t.Fatalf("io.EOF expected, but got %v", err)
	}
}

func TestStreamParser_Success2(t *testing.T) {
	s := `
	<AAA>
		<CCC>c1</CCC>
		<BBB>b1</BBB>
		<DDD>d1</DDD>
		<BBB>b2</BBB>
		<CCC>c2</CCC>
	</AAA>`

	sp, err := CreateStreamParser(strings.NewReader(s), "/AAA/CCC | /AAA/DDD")
	if err != nil {
		t.Fatal(err.Error())
	}

	// First Read() should return c1
	n, err := sp.Read()
	if err != nil {
		t.Fatal(err.Error())
	}
	testOutputXML(t, "first call result", `<CCC>c1</CCC>`, n)
	testOutputXML(t, "doc after first call", `<?xml version="1.0"?><AAA><CCC>c1</CCC></AAA>`, findRoot(n))

	// Second Read() should return d1
	n, err = sp.Read()
	if err != nil {
		t.Fatal(err.Error())
	}
	testOutputXML(t, "second call result", `<DDD>d1</DDD>`, n)
	testOutputXML(t, "doc after second call",
		`<?xml version="1.0"?><AAA><BBB>b1</BBB><DDD>d1</DDD></AAA>`, findRoot(n))

	// Third call should return c2
	n, err = sp.Read()
	if err != nil {
		t.Fatal(err.Error())
	}
	testOutputXML(t, "third call result", `<CCC>c2</CCC>`, n)
	testOutputXML(t, "doc after third call",
		`<?xml version="1.0"?><AAA><BBB>b2</BBB><CCC>c2</CCC></AAA>`, findRoot(n))

	_, err = sp.Read()
	if err != io.EOF {
		t.Fatalf("io.EOF expected, but got %v", err)
	}
}

func TestCDATA(t *testing.T) {
	s := `
	<AAA>
		<CCC><![CDATA[c1]]></CCC>
	</AAA>`

	sp, err := CreateStreamParser(strings.NewReader(s), "/AAA/CCC")
	if err != nil {
		t.Fatal(err.Error())
	}

	n, err := sp.Read()
	if err != nil {
		t.Fatal(err.Error())
	}
	testOutputXML(t, "first call result", `<CCC><![CDATA[c1]]></CCC>`, n)
}

func TestXMLPreservation(t *testing.T) {
	s := `
	<?xml version="1.0" encoding="UTF-8"?>
	<AAA>
		<CCC><![CDATA[c1]]></CCC>
	</AAA>`

	doc, err := Parse(strings.NewReader(s))
	if err != nil {
		t.Fatal(err)
	}

	testOutputXML(t, "first call result",
		`<?xml version="1.0" encoding="UTF-8"?><AAA><CCC><![CDATA[c1]]></CCC></AAA>`, doc)
}
