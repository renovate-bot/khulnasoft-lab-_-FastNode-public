// Package testast implements functions to test the generated ASTs.
package testast

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/khulnasoft-lab/fastnode/fastnode-go/lang/python/pythonparser/epytext/ast"
)

// Assert compares the expected pretty-printed form of the AST with the
// one produced by printing node. It displays a visual diff if they
// are not identical.
func Assert(t *testing.T, expected string, node ast.Node) {
	var buf bytes.Buffer
	ast.Print(node, &buf, "\t")

	actual := strings.TrimSpace(buf.String())
	expected = strings.TrimSpace(expected)

	if actual != expected {
		expectedLines := strings.Split(expected, "\n")
		actualLines := strings.Split(actual, "\n")

		n := len(expectedLines)
		if len(actualLines) > n {
			n = len(actualLines)
		}

		errorLine := -1
		sidebyside := fmt.Sprintf("      | %-40s | %-40s |\n", "EXPECTED", "ACTUAL")
		var errorExpected, errorActual string
		for i := 0; i < n; i++ {
			var expectedLine, actualLine string
			if i < len(expectedLines) {
				expectedLine = strings.Replace(expectedLines[i], "\t", "    ", -1)
			}
			if i < len(actualLines) {
				actualLine = strings.Replace(actualLines[i], "\t", "    ", -1)
			}
			symbol := "   "
			if actualLine != expectedLine {
				symbol = "***"
				if errorLine == -1 {
					errorLine = i
					errorExpected = strings.TrimSpace(expectedLine)
					errorActual = strings.TrimSpace(actualLine)
				}
			}
			sidebyside += fmt.Sprintf("%-6s| %-40s | %-40s |\n", symbol, expectedLine, actualLine)
		}
		t.Errorf("expected %s but got %s (line %d):\n%s", errorExpected, errorActual, errorLine, sidebyside)
	}

	t.Log("\n" + actual)
}
