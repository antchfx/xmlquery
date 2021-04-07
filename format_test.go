package xmlquery

import (
	"strings"
	"testing"
)

func TestFormat(t *testing.T) {
	expected := `<d>
	<e>hello world</e>
</d>`

	s := `<d><e>hello world</e></d>`

	n, err := Parse(strings.NewReader(s))
	if err != nil {
		t.Fatal(err)
	}

	formatted := Format(n, FormatOptionIndent("\t"), FormatOptionDeclaration(false))
	testOutputXMLString(t, "should pretty the output", formatted, expected)
}
