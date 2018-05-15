xmlquery
====
[![Build Status](https://travis-ci.org/antchfx/xmlquery.svg?branch=master)](https://travis-ci.org/antchfx/xmlquery)
[![Coverage Status](https://coveralls.io/repos/github/antchfx/xmlquery/badge.svg?branch=master)](https://coveralls.io/github/antchfx/xmlquery?branch=master)
[![GoDoc](https://godoc.org/github.com/antchfx/xmlquery?status.svg)](https://godoc.org/github.com/antchfx/xmlquery)
[![Go Report Card](https://goreportcard.com/badge/github.com/antchfx/xmlquery)](https://goreportcard.com/report/github.com/antchfx/xmlquery)

Overview
===

**xmlquery** is an XML Parser and XPath package, supports extract data or evaluate from XML documents using XPath expression.

[htmlquery](https://github.com/antchfx/htmlquery) is similar to this package, but supports HTML document using XPath expression.

Installation
====

$ go get github.com/antchfx/xmlquery

Dependencies
====

- [xpath](https://github.com/antchfx/xpath)

Getting Started
===

#### Parse a XML from URL

```go
doc, _ := xmlquery.LoadURL("http://www.example.com/sitemap.xml")
fmt.Println(doc.OutputXML(false))
```

#### Parse a XML from string

```go
s := `<?xml version="1.0" encoding="utf-8"?><rss version="2.0"></rss>`
doc, _ := xmlquery.Parse(strings.NewReader(s))
fmt.Println(doc.OutputXML(false))
```

#### Select all elements

```go
doc := loadTestXML()
for _, n := range xmlquery.Find(doc, "//book") {
	fmt.Printf(n.OutputXML(true))
}
```

#### Select first of all matched elements

```go
doc := loadTestXML()
n := xmlquery.FindOne(doc, "//book")
fmt.Printf(n.OutputXML(true))
```

#### Select children element of current element

```go
doc := loadTestXML()
n := xmlquery.FindOne(doc, "//book")
c1 := xmlquery.FindOne(n, "/author")
c2 := n.SelectElement("author")
fmt.Println(c1 == c2)
```

#### Select all elements with "id" attribute conditions

```go
doc := loadTestXML()
for _, n := range xmlquery.Find(doc, "//book[@id='bk104']") {
	fmt.Printf(n.OutputXML(true))
}
```

#### Gets element text or attribute value

```go
doc := loadTestXML()
n := xmlquery.FindOne(doc, "//book")
fmt.Println(n.InnerText())
fmt.Println(n.SelectAttr("id"))
```

#### Evaluate total prices

```go
doc := loadTestXML()
expr, err := xpath.Compile("sum(//book/price)")
price := expr.Evaluate(xmlquery.CreateXPathNavigator(doc)).(float64)
fmt.Printf("total price: %f\n", price)
```

#### Evaluate element count

```go
doc := loadTestXML()
expr, err := xpath.Compile("count(//book)")
price := expr.Evaluate(xmlquery.CreateXPathNavigator(doc)).(float64)
fmt.Printf("total count is %f\n", price)
```

#### Create XML document

```go
doc := &xmlquery.Node{
	Type: xmlquery.DeclarationNode,
	Data: "xml",
	Attr: []xml.Attr{
		xml.Attr{Name: xml.Name{Local: "version"}, Value: "1.0"},
	},
}
root := &xmlquery.Node{
	Data: "rss",
	Type: xmlquery.ElementNode,
}
doc.FirstChild = root
channel := &xmlquery.Node{
	Data: "channel",
	Type: xmlquery.ElementNode,
}
root.FirstChild = channel
title := &xmlquery.Node{
	Data: "title",
	Type: xmlquery.ElementNode,
}
title_text := &xmlquery.Node{
	Data: "W3Schools Home Page",
	Type: xmlquery.TextNode,
}
title.FirstChild = title_text
channel.FirstChild = title
fmt.Println(doc.OutputXML(true))
// <?xml version="1.0"?><rss><channel><title>W3Schools Home Page</title></channel></rss>
```

Questions
===
If you have any questions, create an issue and welcome to contribute.