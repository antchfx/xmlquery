package xmlquery

import (
	"strings"
	"testing"
)

func TestParseWithLineNumbers(t *testing.T) {
	xml := `<?xml version="1.0"?>
<root>
  <item id="1">
    <title>First Item</title>
    <description>Description for first item</description>
  </item>
  <item id="2">
    <title>Second Item</title>
    <description>Description for second item</description>
  </item>
</root>`

	doc, err := ParseWithLineNumbers(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseWithLineNumbers() error = %v", err)
	}

	// Find the root element (should be on line 2)
	root := FindOne(doc, "//root")
	if root == nil {
		t.Fatal("root element not found")
	}
	if root.GetLineNumber() != 2 {
		t.Errorf("root element line number = %d, want 2", root.GetLineNumber())
	}

	// Find all item elements
	items := Find(doc, "//item")
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	// First item should be on line 3
	if items[0].GetLineNumber() != 3 {
		t.Errorf("first item line number = %d, want 3", items[0].GetLineNumber())
	}

	// Second item should be on line 7
	if items[1].GetLineNumber() != 7 {
		t.Errorf("second item line number = %d, want 7", items[1].GetLineNumber())
	}

	// Find title elements
	titles := Find(doc, "//title")
	if len(titles) != 2 {
		t.Fatalf("expected 2 titles, got %d", len(titles))
	}

	// First title should be on line 4
	if titles[0].GetLineNumber() != 4 {
		t.Errorf("first title line number = %d, want 4", titles[0].GetLineNumber())
	}

	// Second title should be on line 8
	if titles[1].GetLineNumber() != 8 {
		t.Errorf("second title line number = %d, want 8", titles[1].GetLineNumber())
	}

	// Find description elements
	descriptions := Find(doc, "//description")
	if len(descriptions) != 2 {
		t.Fatalf("expected 2 descriptions, got %d", len(descriptions))
	}

	// First description should be on line 5
	if descriptions[0].GetLineNumber() != 5 {
		t.Errorf("first description line number = %d, want 5", descriptions[0].GetLineNumber())
	}

	// Second description should be on line 9
	if descriptions[1].GetLineNumber() != 9 {
		t.Errorf("second description line number = %d, want 9", descriptions[1].GetLineNumber())
	}
}

func TestParseWithLineNumbersSingleLine(t *testing.T) {
	xml := `<root><item id="1"><title>Title</title></item></root>`

	doc, err := ParseWithLineNumbers(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseWithLineNumbers() error = %v", err)
	}

	// All elements should be on line 1
	root := FindOne(doc, "//root")
	if root.GetLineNumber() != 1 {
		t.Errorf("root line number = %d, want 1", root.GetLineNumber())
	}

	item := FindOne(doc, "//item")
	if item.GetLineNumber() != 1 {
		t.Errorf("item line number = %d, want 1", item.GetLineNumber())
	}

	title := FindOne(doc, "//title")
	if title.GetLineNumber() != 1 {
		t.Errorf("title line number = %d, want 1", title.GetLineNumber())
	}
}

func TestParseWithLineNumbersComments(t *testing.T) {
	xml := `<?xml version="1.0"?>
<!-- This is a comment -->
<root>
  <!-- Another comment -->
  <item>Content</item>
</root>`

	doc, err := ParseWithLineNumbers(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseWithLineNumbers() error = %v", err)
	}

	// Find comment nodes
	comments := Find(doc, "//comment()")
	if len(comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(comments))
	}

	// First comment should be on line 2
	if comments[0].GetLineNumber() != 2 {
		t.Errorf("first comment line number = %d, want 2", comments[0].GetLineNumber())
	}

	// Second comment should be on line 4
	if comments[1].GetLineNumber() != 4 {
		t.Errorf("second comment line number = %d, want 4", comments[1].GetLineNumber())
	}

	// Root element should be on line 3
	root := FindOne(doc, "//root")
	if root.GetLineNumber() != 3 {
		t.Errorf("root line number = %d, want 3", root.GetLineNumber())
	}

	// Item element should be on line 5
	item := FindOne(doc, "//item")
	if item.GetLineNumber() != 5 {
		t.Errorf("item line number = %d, want 5", item.GetLineNumber())
	}
}

func TestParseWithLineNumbersCDATA(t *testing.T) {
	xml := `<?xml version="1.0"?>
<root>
  <content><![CDATA[Some CDATA content]]></content>
  <item>Regular content</item>
</root>`

	doc, err := ParseWithLineNumbers(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseWithLineNumbers() error = %v", err)
	}

	// Content element should be on line 3
	content := FindOne(doc, "//content")
	if content.GetLineNumber() != 3 {
		t.Errorf("content line number = %d, want 3", content.GetLineNumber())
	}

	// Item element should be on line 4
	item := FindOne(doc, "//item")
	if item.GetLineNumber() != 4 {
		t.Errorf("item line number = %d, want 4", item.GetLineNumber())
	}
}

func TestGetLineNumberMethod(t *testing.T) {
	xml := `<root>
  <child>text</child>
</root>`

	doc, err := ParseWithLineNumbers(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseWithLineNumbers() error = %v", err)
	}

	root := FindOne(doc, "//root")
	if root == nil {
		t.Fatal("root element not found")
	}

	// Test the GetLineNumber method
	lineNum := root.GetLineNumber()
	if lineNum != 1 {
		t.Errorf("GetLineNumber() = %d, want 1", lineNum)
	}

	child := FindOne(doc, "//child")
	if child == nil {
		t.Fatal("child element not found")
	}

	childLineNum := child.GetLineNumber()
	if childLineNum != 2 {
		t.Errorf("child GetLineNumber() = %d, want 2", childLineNum)
	}
}

func TestLineNumbersWithNamespaces(t *testing.T) {
	xml := `<?xml version="1.0"?>
<root xmlns:ns="http://example.com/ns">
  <ns:item>
    <ns:title>Title</ns:title>
  </ns:item>
</root>`

	doc, err := ParseWithLineNumbers(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseWithLineNumbers() error = %v", err)
	}

	// Root should be on line 2
	root := FindOne(doc, "//root")
	if root.GetLineNumber() != 2 {
		t.Errorf("root line number = %d, want 2", root.GetLineNumber())
	}

	// Item should be on line 3 (using local-name to avoid namespace issues in XPath)
	item := FindOne(doc, "//*[local-name()='item']")
	if item.GetLineNumber() != 3 {
		t.Errorf("item line number = %d, want 3", item.GetLineNumber())
	}

	// Title should be on line 4
	title := FindOne(doc, "//*[local-name()='title']")
	if title.GetLineNumber() != 4 {
		t.Errorf("title line number = %d, want 4", title.GetLineNumber())
	}
}