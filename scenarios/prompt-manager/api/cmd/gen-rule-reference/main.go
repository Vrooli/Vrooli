// Command gen-rule-reference rewrites the generated rule-catalog tables in
// docs/agent-system from the live validation catalog.
//
// Usage: go run ./cmd/gen-rule-reference <docs-dir>
package main

import (
	"fmt"
	"os"

	"prompt-manager/internal/memberflow"
)

func main() {
	docsDir := "../../docs"
	if len(os.Args) > 1 {
		docsDir = os.Args[1]
	}
	changed, err := memberflow.GenerateRuleReference(docsDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if len(changed) == 0 {
		fmt.Println("rule reference already up to date")
		return
	}
	for _, path := range changed {
		fmt.Println("regenerated", path)
	}
}
