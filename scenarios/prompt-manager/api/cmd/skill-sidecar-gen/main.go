// Command skill-sidecar-gen regenerates compatibility skill.json files from
// the SKILL.md frontmatter source of truth.
package main

import (
	"flag"
	"fmt"
	"os"

	"prompt-manager/internal/skills"
)

func main() {
	root := flag.String("root", "", "skill root to walk")
	flag.Parse()
	if *root == "" {
		fmt.Fprintln(os.Stderr, "-root is required")
		os.Exit(2)
	}
	if err := skills.GenerateCorpus(*root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
