package xmlquery

import (
	"github.com/antchfx/xpath"
	"strings"
	"testing"
)

func TestMoveToRootClearsAttributeCursor(t *testing.T) {
	doc, err := Parse(strings.NewReader(`<root id="root"><child id="child"/></root>`))
	if err != nil {
		t.Fatal(err)
	}
	for _, top := range []*Node{doc, FindOne(doc, "//root")} {
		nav := CreateXPathNavigator(top)
		if top == doc {
			nav.MoveToChild()
			for nav.NodeType() != xpath.ElementNode {
				if !nav.MoveToNext() {
					t.Fatal("missing element")
				}
			}
		}
		if !nav.MoveToNextAttribute() {
			t.Fatal("missing attribute")
		}
		nav.MoveToRoot()
		if got := nav.NodeType(); got != CreateXPathNavigator(top).NodeType() {
			t.Errorf("root type=%v", got)
		}
		if !nav.MoveToChild() {
			t.Fatal("root retained attribute cursor, cannot descend")
		}
		if nav.Current().Parent != top {
			t.Fatal("wrong root child")
		}
	}
}
