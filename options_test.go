package xmlquery

import (
	"bytes"
	"encoding/xml"
	"testing"
)

func TestApplyOptions(t *testing.T) {
	parser := &parser{
		decoder: xml.NewDecoder(bytes.NewReader(make([]byte, 0))),
	}
	options := ParserOptions{
		Decoder: &DecoderOptions{
			Strict: false,
			AutoClose: []string{"foo"},
			Entity: map[string]string{
				"bar": "baz",
			},
		},
	}

	options.apply(parser)
	if parser.decoder.Strict != options.Decoder.Strict {
		t.Fatalf("Expected Strict attribute of %v, got %v instead", options.Decoder.Strict, parser.decoder.Strict)
	}
	if parser.decoder.AutoClose[0] != options.Decoder.AutoClose[0] {
		t.Fatalf("Expected AutoClose attribute with %v, got %v instead", options.Decoder.AutoClose, parser.decoder.AutoClose)
	}
	if parser.decoder.Entity["bar"] != options.Decoder.Entity["bar"] {
		t.Fatalf("Expected Entity mode of %v, got %v instead", options.Decoder.Entity, parser.decoder.Entity)
	}
}

func TestApplyEmptyOptions(t *testing.T) {
	parser := &parser{
		decoder: xml.NewDecoder(bytes.NewReader(make([]byte, 0))),
	}
	options := ParserOptions{
		Decoder: nil,
	}

	// Only testing for the absence of errors since we are not
	// expecting this call to do anything
	options.apply(parser)
}
