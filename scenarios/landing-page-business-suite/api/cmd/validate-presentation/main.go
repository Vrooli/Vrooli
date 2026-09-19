// Command validate-presentation decodes and validates a presentation document
// revision file, printing every validation issue. Dev tool; no server state.
package main

import (
	"fmt"
	"os"

	landing "landing-page-business-suite-api/handlers/config"
	"landing-page-business-suite-api/internal/presentation"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: validate-presentation <revision.json>")
		os.Exit(2)
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	document, err := presentation.DecodeDocument(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "decode: %v\n", err)
		os.Exit(1)
	}
	if err := presentation.Validate(document); err != nil {
		fmt.Fprintf(os.Stderr, "validate: %v\n", err)
		os.Exit(1)
	}
	if _, err := landing.PresentationDocumentProto(document); err != nil {
		fmt.Fprintf(os.Stderr, "proto: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("ok")
}
