package topcli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	contextinfo "github.com/vrooli/vrooli/internal/app/contextinfo"
	"github.com/vrooli/vrooli/internal/cli/clipolicy"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
)

var DefaultInfoFiles = []string{"docs/context.md"}

type InfoRequest struct {
	ListOnly bool
}

const InfoUsageText = "Usage: vrooli info [--list]\n\nDisplay consolidated Vrooli project context in a single stream.\n\n    --list     Print the resolved file paths without emitting file contents."

type infoManifest struct {
	Files []string `json:"files"`
}

type InfoFileOutput struct {
	Path     string `json:"path"`
	Contents string `json:"contents,omitempty"`
}

type InfoOutput struct {
	Root  string           `json:"root"`
	Files []InfoFileOutput `json:"files"`
}

func ParseInfoRequest(args []string) (InfoRequest, error) {
	req := InfoRequest{}
	for _, arg := range args {
		switch arg {
		case "--list":
			req.ListOnly = true
		case "--help", "-h":
			return InfoRequest{}, clipolicy.CommandHelpOnly(InfoUsageText)
		default:
			return InfoRequest{}, clipolicy.UnknownOptionError("info", arg)
		}
	}
	return req, nil
}

func RunInfo(root string, format cliout.Format, req InfoRequest, stdout, stderr io.Writer) error {
	service := contextinfo.Service{
		CollectSources: CollectInfoSourcesDetailed,
		ResolvePath:    ResolveInfoPath,
		ReadFile:       os.ReadFile,
	}
	if req.ListOnly {
		paths, warnings, err := service.List(root)
		if err != nil {
			return err
		}
		if len(paths) == 0 {
			return errors.New("no context sources defined for vrooli info")
		}
		for _, warning := range warnings {
			_, _ = fmt.Fprintf(stderr, "[WARNING] %s\n", warning)
		}
		if format == cliout.FormatJSON {
			return cliout.WriteSuccessFields(stdout, map[string]any{"root": root, "files": paths})
		}
		for _, path := range paths {
			_, _ = fmt.Fprintln(stdout, path)
		}
		return nil
	}

	payload, warnings, err := service.Load(root)
	if err != nil {
		return err
	}
	if len(payload.Files) == 0 {
		return errors.New("no context sources defined for vrooli info")
	}
	for _, warning := range warnings {
		_, _ = fmt.Fprintf(stderr, "[WARNING] %s\n", warning)
	}
	if format == cliout.FormatJSON {
		out := InfoOutput{Root: payload.Root, Files: make([]InfoFileOutput, 0, len(payload.Files))}
		for _, file := range payload.Files {
			out.Files = append(out.Files, InfoFileOutput{Path: file.Path, Contents: file.Contents})
		}
		return cliout.WriteSuccessFields(stdout, map[string]any{"root": out.Root, "files": out.Files})
	}
	_, _ = fmt.Fprintln(stdout, "[HEADER]  Vrooli Context Briefing")
	_, _ = fmt.Fprintf(stdout, "[INFO]    Project root: %s\n", root)
	for _, file := range payload.Files {
		_, _ = fmt.Fprintf(stdout, "\n===== %s =====\n", file.Path)
		_, _ = io.WriteString(stdout, file.Contents)
		if file.Contents == "" || file.Contents[len(file.Contents)-1] != '\n' {
			_, _ = fmt.Fprintln(stdout)
		}
	}
	return nil
}

func CollectInfoSources(root string) ([]string, error) {
	files, _, err := CollectInfoSourcesDetailed(root)
	return files, err
}

func CollectInfoSourcesDetailed(root string) ([]string, []string, error) {
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

	manifestPath := repocontractmeta.InfoManifestPath(root)
	if data, err := os.ReadFile(manifestPath); err == nil {
		var manifest infoManifest
		if err := json.Unmarshal(data, &manifest); err == nil && len(manifest.Files) > 0 {
			return manifest.Files, nil, nil
		} else if err != nil {
			return append([]string(nil), DefaultInfoFiles...),
				[]string{fmt.Sprintf("Invalid info manifest %s: %v. Falling back to defaults.", manifestPath, err)},
				nil
		}
	}

	return append([]string(nil), DefaultInfoFiles...), nil, nil
}

func ResolveInfoPath(root, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(root, filepath.FromSlash(path))
}
