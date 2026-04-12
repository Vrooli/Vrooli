package main

import (
	"io"

	"github.com/vrooli/vrooli/internal/cliout"
)

func parseOutputFormat(globals globalOptions) (cliout.Format, error) {
	return cliout.ParseFormat("", globals.json)
}

func writeSuccessData(w io.Writer, key string, value any) error {
	return cliout.WriteJSON(w, map[string]any{
		"success": true,
		key:       value,
	})
}
