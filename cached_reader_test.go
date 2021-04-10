package xmlquery

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestCaching(t *testing.T) {
	buf := strings.NewReader(`ABCDEF`)
	bufReader := bufio.NewReader(buf)
	CachedReader := NewCachedReader(bufReader)

	b, err := CachedReader.ReadByte()
	if err != nil {
		t.Fatal(err.Error())
	}

	if b != 'A' {
		t.Fatalf("Expected read byte to be A, got %c instead.", b)
	}

	CachedReader.StartCaching()
	tmpBuf := make([]byte, 10)
	n, err := CachedReader.Read(tmpBuf)
	if err != nil {
		t.Fatal(err.Error())
	}

	if n != 5 {
		t.Fatalf("Expected 5 bytes to be read. Got %d instead.", n)
	}
	if !bytes.Equal(tmpBuf[:n], []byte("BCDEF")) {
		t.Fatalf("Incorrect read buffer value")
	}

	cached := CachedReader.Cache()
	if !bytes.Equal(cached, []byte("BCDEF")) {
		t.Fatalf("Incorrect cached buffer value")
	}
}
