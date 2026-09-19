package app

import (
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/resources/doc-ocr/cli/internal/discovery"
	"github.com/vrooli/vrooli/resources/doc-ocr/cli/internal/domain"
	"github.com/vrooli/vrooli/resources/doc-ocr/cli/internal/env"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

type BuildInfo struct{ Name, Version, Description, Fingerprint, Timestamp, SourceRoot string }

func BuildCommandApp(info BuildInfo) (*cliapp.App, error) {
	service := domain.NewService(env.Load(), discovery.DiscoverRuntime(info.SourceRoot))
	stale := cliutil.NewStaleChecker(info.Name, info.Fingerprint, info.Timestamp, info.SourceRoot, "VROOLI_CLI_SOURCE_ROOT")
	stale.SourceContextPath = ".."
	stale.ManifestSourcePath = "resource.json"
	stale.FreshnessInputs = []string{"cli/**", "cli/internal/**", "docs/**", "README.md", "resource.json"}
	return cliapp.NewApp(cliapp.AppOptions{
		Name: info.Name, Version: info.Version, Description: info.Description,
		Commands: []cliapp.CommandGroup{
			{Title: "Resource", Commands: []cliapp.Command{
				{Name: "health", Description: "Verify model checksum and OCR readiness", Run: func([]string) error { return service.Health() }},
				{Name: "status", Description: "Show readiness and active operating mode", Run: func([]string) error { return service.PrintStatus() }},
				{Name: "capabilities", Description: "List OCR capabilities", Run: func([]string) error { return service.Capabilities() }},
				{Name: "languages", Description: "List installed OCR languages", Run: func([]string) error { return service.Languages() }},
				{Name: "version", Description: "Show engine and model versions", Run: func([]string) error { return service.Version(info.Name, info.Version) }},
			}},
			{Title: "OCR", Commands: []cliapp.Command{
				{Name: "ocr", Usage: "ocr <image-or-pdf-page> [--languages eng]", Description: "Recognize a local page", Run: func(args []string) error {
					input, language, err := parseOCRArgs(args)
					if err != nil {
						return err
					}
					return service.OCR(input, language)
				}},
			}},
		}, StaleChecker: stale,
	}), nil
}

func parseOCRArgs(args []string) (string, string, error) {
	language := "eng"
	input := ""
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--languages":
			if i+1 >= len(args) {
				return "", "", fmt.Errorf("--languages requires a value")
			}
			i++
			language = args[i]
		case strings.HasPrefix(args[i], "--languages="):
			language = strings.TrimPrefix(args[i], "--languages=")
		case strings.HasPrefix(args[i], "-"):
			return "", "", fmt.Errorf("unknown OCR option %q", args[i])
		case input == "":
			input = args[i]
		default:
			return "", "", fmt.Errorf("ocr accepts one input path")
		}
	}
	if input == "" {
		return "", "", fmt.Errorf("usage: ocr <image-or-pdf-page> [--languages eng]")
	}
	return input, language, nil
}
