package xmlquery

import (
	"github.com/antchfx/xpath"
	"strings"
	"testing"
)

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
