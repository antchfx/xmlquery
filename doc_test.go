package xmlquery_test

import (
	"fmt"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/antchfx/xpath"
)

func Example() {
	// XPATH syntax and functions see https://github.com/antchfx/xpath
	s := `<?xml version="1.0"?>
	<bookstore>
	   <book id="bk101">
		  <author>Gambardella, Matthew</author>
		  <title>XML Developer's Guide</title>
		  <genre>Computer</genre>
		  <price>44.95</price>
		  <publish_date>2000-10-01</publish_date>
	   </book>
	   <book id="bk102">
		  <author>Ralls, Kim</author>
		  <title>Midnight Rain</title>
		  <genre>Fantasy</genre>
		  <price>5.95</price>
		  <publish_date>2000-12-16</publish_date>
	   </book>
	   <book id="bk103">
		  <author>Corets, Eva</author>
		  <title>Maeve Ascendant</title>
		  <genre>Fantasy</genre>
		  <price>5.95</price>
		  <publish_date>2000-11-17</publish_date>
	   </book>
	</bookstore>`

	doc, err := xmlquery.Parse(strings.NewReader(s))
	if err != nil {
		panic(err)
	}
	// Quick query all books.
	books := xmlquery.Find(doc, `/bookstore/book`)
	for _, n := range books {
		fmt.Println(n)
	}

	// Find all books with price rather than 10.
	books = xmlquery.Find(doc, `//book[price < 10]`)
	fmt.Println(len(books))

	// Find books with @id=bk102 or @id=bk101
	books = xmlquery.Find(doc, `//book[@id = "bk102" or @id = "bk101"]`)
	fmt.Println(len(books))

	// Find books by author: Corets, Eva
	book := xmlquery.FindOne(doc, `//book[author = "Corets, Eva"]`)
	fmt.Println(book.SelectElement("title").InnerText()) // > Output: Maeve Ascendant

	// Calculate the total prices of all books
	nav := xmlquery.CreateXPathNavigator(doc)
	prices := xpath.MustCompile(`sum(//book/price)`).Evaluate(nav).(float64)
	fmt.Println(prices) // > Output: 56.85
}
