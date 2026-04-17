// Package metadata provides CLI commands for OG metadata fetching.
//
// DOC: docs/reference/cli-commands.md#metadata
package metadata

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"prompt-manager/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// OGMetadata represents Open Graph metadata
type OGMetadata struct {
	URL         string `json:"url"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Image       string `json:"image,omitempty"`
	SiteName    string `json:"siteName,omitempty"`
	Type        string `json:"type,omitempty"`
	Favicon     string `json:"favicon,omitempty"`
}

// Commands returns the metadata command groups using noun-verb pattern.
func Commands(ctx appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Metadata",
		Commands: []cliapp.Command{
			{
				Name:        "metadata",
				Aliases:     []string{"meta", "og"},
				NeedsAPI:    true,
				Description: "Fetch Open Graph metadata (fetch)",
				Run: func(args []string) error {
					return route(ctx, args)
				},
			},
		},
	}
}

// route dispatches to the appropriate subcommand.
func route(ctx appctx.Context, args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "fetch", "get":
		return cmdFetch(ctx, subArgs)
	default:
		return fmt.Errorf("unknown subcommand: %s\n\n%s", subcommand, usageText())
	}
}

func printUsage() error {
	fmt.Println(usageText())
	return nil
}

func usageText() string {
	return `Usage: prompt-manager metadata <subcommand> [args]

Subcommands:
  fetch, get <url>    Fetch Open Graph metadata from a URL`
}

func cmdFetch(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("fetch", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: metadata fetch <url>")
	}
	targetURL := fs.Arg(0)

	// Validate URL
	if _, err := url.Parse(targetURL); err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	query := url.Values{}
	query.Set("url", targetURL)

	var meta OGMetadata
	if err := ctx.GetWithQuery("/og-metadata", query, &meta); err != nil {
		return fmt.Errorf("failed to fetch metadata: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(meta)
	}

	fmt.Printf("URL: %s\n", meta.URL)
	if meta.Title != "" {
		fmt.Printf("Title: %s\n", meta.Title)
	}
	if meta.Description != "" {
		fmt.Printf("Description: %s\n", meta.Description)
	}
	if meta.SiteName != "" {
		fmt.Printf("Site: %s\n", meta.SiteName)
	}
	if meta.Type != "" {
		fmt.Printf("Type: %s\n", meta.Type)
	}
	if meta.Image != "" {
		fmt.Printf("Image: %s\n", meta.Image)
	}
	if meta.Favicon != "" {
		fmt.Printf("Favicon: %s\n", meta.Favicon)
	}
	return nil
}
