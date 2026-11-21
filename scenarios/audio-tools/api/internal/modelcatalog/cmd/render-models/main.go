// Command render-models regenerates docs/reference/models.md from the embedded
// model catalog. Run it from the api/ directory:
//
//	go run ./internal/modelcatalog/cmd/render-models
//
// The Go test TestDocsMatchRenderer fails when the checked-in document drifts,
// so a seed edit must be followed by this command.
package main

import (
	"flag"
	"fmt"
	"os"

	"audio-tools/internal/modelcatalog"
)

func main() {
	out := flag.String("out", "../docs/reference/models.md", "output path for the generated Markdown")
	flag.Parse()

	rendered := modelcatalog.Default().RenderMarkdown()
	if err := os.WriteFile(*out, []byte(rendered), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "render-models: write %s: %v\n", *out, err)
		os.Exit(1)
	}
	fmt.Printf("render-models: wrote %s (%d bytes)\n", *out, len(rendered))
}
