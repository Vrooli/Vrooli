package main

import (
	"io"

	"github.com/vrooli/vrooli/internal/cli/topcli"
	"github.com/vrooli/vrooli/internal/cliout"
)

func runInfoCommand(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	req, err := topcli.ParseInfoRequest(args)
	if err != nil {
		if helpErr, ok := err.(interface{ HelpText() string }); ok {
			_, _ = io.WriteString(stdout, helpErr.HelpText())
			if text := helpErr.HelpText(); text == "" || text[len(text)-1] != '\n' {
				_, _ = io.WriteString(stdout, "\n")
			}
			return nil
		}
		return err
	}
	format, err := cliout.ParseFormat("", globals.JSON)
	if err != nil {
		return err
	}
	return topcli.RunInfo(root, format, req, stdout, stderr)
}

func collectInfoSourcesDetailed(root string) ([]string, []string, error) {
	return topcli.CollectInfoSourcesDetailed(root)
}

func resolveInfoPath(root, path string) string {
	return topcli.ResolveInfoPath(root, path)
}
