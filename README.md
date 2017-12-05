xmlquery
====
[![Build Status](https://travis-ci.org/antchfx/xmlquery.svg?branch=master)](https://travis-ci.org/antchfx/xmlquery)
[![Coverage Status](https://coveralls.io/repos/github/antchfx/xmlquery/badge.svg?branch=master)](https://coveralls.io/github/antchfx/xmlquery?branch=master)
[![GoDoc](https://godoc.org/github.com/antchfx/xmlquery?status.svg)](https://godoc.org/github.com/antchfx/xmlquery)
[![Go Report Card](https://goreportcard.com/badge/github.com/antchfx/xmlquery)](https://goreportcard.com/report/github.com/antchfx/xmlquery)

xmlquery is an XML parser thats builds a read/modified DOM and supports XPath feature to extract data 
from XML documents using XPath expression.

Its depend on [xpath](https://github.com/antchfx/xpath) package.

Installation
====

    $ go get github.com/antchfx/xmlquery

Examples
===

```go
package main

import (
	"github.com/antchfx/xmlquery"
)

func main() {
	// Load XML document from file.
	f, err := os.Open(`./examples/test.xml`)
	if err != nil {
		panic(err)
	}
	// Parse XML document.
	doc, err := xmlquery.Parse(f)
	if err != nil{
		panic(err)
	}

	// Selects all the `book` elements.
	for _, n := range xmlquery.Find(doc, "//book") {
		fmt.Printf("%s : %s \n", n.SelectAttr("id"), xmlquery.FindOne(n, "title").InnerText())
	}

	// Selects all the `book` elements that have a "id" attribute
	// with a value of "bk104"
	n := xmlquery.FindOne(doc, "//book[@id='bk104']")
	fmt.Printf("%s \n", n.OutputXML(true))
}
```

### Evaluate an XPath expression

Using `Evaluate()` to evaluates XPath expressions.

```go
expr, err := xpath.Compile("sum(//book/price)")
if err != nil {
	panic(err)
}
fmt.Printf("total price: %f\n", expr.Evaluate(xmlquery.CreateXPathNavigator(doc)).(float64))	
```
