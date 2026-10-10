package xmlquery_test

import (
	"fmt"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/antchfx/xpath"
	"golang.org/x/net/html/charset"
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

func ExampleParseWithOptions_customEntities() {
	s := `<name>&auml; &amp; &#228;</name>`
	options := xmlquery.ParserOptions{
		Decoder: &xmlquery.DecoderOptions{
			Strict: true,
			Entity: map[string]string{"auml": "ä"},
		},
	}
	doc, err := xmlquery.ParseWithOptions(strings.NewReader(s), options)
	if err != nil {
		panic(err)
	}
	fmt.Println(xmlquery.FindOne(doc, "//name").InnerText())

	for _, input := range []string{`<name>ä</name>`, `<name>&#228;</name>`, `<name>&#xE4;</name>`} {
		doc, err := xmlquery.Parse(strings.NewReader(input))
		if err != nil {
			panic(err)
		}
		fmt.Println(xmlquery.FindOne(doc, "//name").InnerText())
	}

	_, err = xmlquery.Parse(strings.NewReader(`<name>&auml;</name>`))
	fmt.Println(err != nil)
	_, err = xmlquery.ParseWithOptions(strings.NewReader(`<name>&unknown;</name>`), options)
	fmt.Println(err != nil)
	_, err = xmlquery.ParseWithOptions(strings.NewReader(`<name></other>`), options)
	fmt.Println(err != nil)

	options.Decoder.CharsetReader = charset.NewReaderLabel
	latin1 := "<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?><name>\xe4 &auml;</name>"
	doc, err = xmlquery.ParseWithOptions(strings.NewReader(latin1), options)
	if err != nil {
		panic(err)
	}
	fmt.Println(xmlquery.FindOne(doc, "//name").InnerText())

	// Output:
	// ä & ä
	// ä
	// ä
	// ä
	// true
	// true
	// true
	// ä ä
}
