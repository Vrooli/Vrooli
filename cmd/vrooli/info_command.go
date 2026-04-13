package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/cliout"
)

var infoDefaultFiles = []string{"docs/context.md"}

type infoManifest struct {
	Files []string `json:"files"`
}

type infoFileOutput struct {
	Path     string `json:"path"`
	Contents string `json:"contents,omitempty"`
}

type infoOutput struct {
	Root  string           `json:"root"`
	Files []infoFileOutput `json:"files"`
}

func runInfoCommand(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	showPathsOnly := false
	for _, arg := range args {
		switch arg {
		case "--list":
			showPathsOnly = true
		case "--help", "-h":
			_, _ = fmt.Fprintln(stdout, "Usage: vrooli info [--list]")
			_, _ = fmt.Fprintln(stdout)
			_, _ = fmt.Fprintln(stdout, "Display consolidated Vrooli project context in a single stream.")
			_, _ = fmt.Fprintln(stdout)
			_, _ = fmt.Fprintln(stdout, "    --list     Print the resolved file paths without emitting file contents.")
			return nil
		default:
			return unknownOptionError("info", arg)
		}
	}

	format, err := formatFromJSON(globals.json)
	if err != nil {
		return err
	}

	infoFiles, warnings, err := collectInfoSourcesDetailed(root)
	if err != nil {
		return err
	}
	if len(infoFiles) == 0 {
		return errors.New("no context sources defined for vrooli info")
	}
	for _, warning := range warnings {
		_, _ = fmt.Fprintf(stderr, "[WARNING] %s\n", warning)
	}

	if showPathsOnly {
		paths := make([]string, 0, len(infoFiles))
		for _, file := range infoFiles {
			paths = append(paths, resolveInfoPath(root, file))
		}
		if format == cliout.FormatJSON {
			return cliout.WriteJSON(stdout, map[string]any{
				"root":  root,
				"files": paths,
			})
		}
		for _, path := range paths {
			_, _ = fmt.Fprintln(stdout, path)
		}
		return nil
	}

	if format == cliout.FormatJSON {
		payload := infoOutput{
			Root:  root,
			Files: make([]infoFileOutput, 0, len(infoFiles)),
		}
		for _, source := range infoFiles {
			resolved := resolveInfoPath(root, source)
			contents, readErr := os.ReadFile(resolved)
			if readErr != nil {
				if os.IsNotExist(readErr) {
					_, _ = fmt.Fprintf(stderr, "[WARNING] Skipping missing context file: %s\n", source)
					continue
				}
				return fmt.Errorf("read info source %s: %w", resolved, readErr)
			}
			payload.Files = append(payload.Files, infoFileOutput{
				Path:     resolved,
				Contents: string(contents),
			})
		}
		return cliout.WriteJSON(stdout, payload)
	}

	_, _ = fmt.Fprintln(stdout, "[HEADER]  Vrooli Context Briefing")
	_, _ = fmt.Fprintf(stdout, "[INFO]    Project root: %s\n", root)
	for _, source := range infoFiles {
		resolved := resolveInfoPath(root, source)
		contents, readErr := os.ReadFile(resolved)
		if readErr != nil {
			if os.IsNotExist(readErr) {
				_, _ = fmt.Fprintf(stderr, "[WARNING] Skipping missing context file: %s\n", source)
				continue
			}
			return fmt.Errorf("read info source %s: %w", resolved, readErr)
		}
		_, _ = fmt.Fprintf(stdout, "\n===== %s =====\n", resolved)
		_, _ = stdout.Write(contents)
		if len(contents) == 0 || contents[len(contents)-1] != '\n' {
			_, _ = fmt.Fprintln(stdout)
		}
	}

	return nil
}

func collectInfoSources(root string) ([]string, error) {
	files, _, err := collectInfoSourcesDetailed(root)
	return files, err
}

func collectInfoSourcesDetailed(root string) ([]string, []string, error) {
	if envValue := strings.TrimSpace(os.Getenv("VROOLI_INFO_FILES")); envValue != "" {
		parts := strings.Split(envValue, ":")
		files := make([]string, 0, len(parts))
		for _, entry := range parts {
			entry = strings.TrimSpace(entry)
			if entry != "" {
				files = append(files, entry)
			}
		}
		return files, nil, nil
	}

	manifestPath := filepath.Join(root, ".vrooli", "info-manifest.json")
	if data, err := os.ReadFile(manifestPath); err == nil {
		var manifest infoManifest
		if err := json.Unmarshal(data, &manifest); err == nil && len(manifest.Files) > 0 {
			return manifest.Files, nil, nil
		} else if err != nil {
			return append([]string(nil), infoDefaultFiles...),
				[]string{fmt.Sprintf("Invalid info manifest %s: %v. Falling back to defaults.", manifestPath, err)},
				nil
		}
	}

	return append([]string(nil), infoDefaultFiles...), nil, nil
}

func resolveInfoPath(root, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(root, filepath.FromSlash(path))
}
