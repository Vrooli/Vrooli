package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	model, err := newNERModelFromEnv()
	if err != nil {
		// A model construction error is non-fatal: the regex pass still serves.
		fmt.Fprintln(os.Stderr, "pii-detector NER model disabled:", err)
		model = UnavailableModel{reason: err.Error()}
	}
	defer model.Close()

	addr := envOr("RESOURCE_LISTEN_ADDR", "127.0.0.1:11460")
	fmt.Fprintf(os.Stderr, "pii-detector listening on %s (model_id=%q)\n", addr, model.ModelID())
	if err := http.ListenAndServe(addr, newHandler(model)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
