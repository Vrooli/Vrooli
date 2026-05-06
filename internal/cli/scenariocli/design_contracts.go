package scenariocli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/cliout"
)

type DesignCopyRule struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type DesignAdapterManifest struct {
	ID       string           `json:"id,omitempty"`
	Copy     []DesignCopyRule `json:"copy,omitempty"`
	Requires string           `json:"requires,omitempty"`
}

type DesignKitAdapter struct {
	Path     string   `json:"path"`
	Supports []string `json:"supports,omitempty"`
}

type DesignKitManifest struct {
	ID          string                      `json:"id"`
	Name        string                      `json:"name"`
	Version     string                      `json:"version"`
	Default     bool                        `json:"default,omitempty"`
	Description string                      `json:"description,omitempty"`
	Tags        []string                    `json:"tags,omitempty"`
	Adapters    map[string]DesignKitAdapter `json:"adapters,omitempty"`
}

type DesignKitInfo struct {
	ID       string
	Path     string
	Manifest DesignKitManifest
	Missing  bool
}

type DesignValidationIssue struct {
	Kit     string `json:"kit,omitempty"`
	Adapter string `json:"adapter,omitempty"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

type DesignValidationReport struct {
	Count  int                     `json:"count"`
	Issues []DesignValidationIssue `json:"issues,omitempty"`
}

type (
	DesignListRequest     struct{}
	DesignShowRequest     struct{ ID string }
	DesignValidateRequest struct {
		ID  string
		All bool
	}
)

func RenderDesignListResponse(w io.Writer, format cliout.Format, kits []DesignKitInfo) error {
	if format == cliout.FormatJSON {
		return cliout.WriteSuccessJSON(w, "designKits", kits)
	}
	rows := make([][]string, 0, len(kits))
	for _, kit := range kits {
		defaultMarker := ""
		if kit.Manifest.Default {
			defaultMarker = "yes"
		}
		adapters := sortedDesignAdapterIDs(kit.Manifest.Adapters)
		rows = append(rows, []string{kit.ID, kit.Manifest.Name, kit.Manifest.Version, defaultMarker, strings.Join(adapters, ", ")})
	}
	_ = cliout.RenderTable(w, []string{"ID", "Name", "Version", "Default", "Adapters"}, rows)
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Tip: vrooli scenario design show <kit-id>")
	return nil
}

func RenderDesignShowResponse(w io.Writer, format cliout.Format, info DesignKitInfo) error {
	if format == cliout.FormatJSON {
		return cliout.WriteSuccessJSON(w, "designKit", info)
	}
	manifest := info.Manifest
	_, _ = fmt.Fprintf(w, "%s (%s)\n", manifest.Name, manifest.ID)
	if manifest.Description != "" {
		_, _ = fmt.Fprintln(w, manifest.Description)
	}
	if manifest.Version != "" {
		_, _ = fmt.Fprintf(w, "Version: %s\n", manifest.Version)
	}
	if manifest.Default {
		_, _ = fmt.Fprintln(w, "Default: yes")
	}
	if len(manifest.Tags) > 0 {
		_, _ = fmt.Fprintf(w, "Tags: %s\n", strings.Join(manifest.Tags, ", "))
	}
	_, _ = fmt.Fprintf(w, "Canonical design: %s/DESIGN.md\n", info.Path)
	if len(manifest.Adapters) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "Adapters:")
		for _, id := range sortedDesignAdapterIDs(manifest.Adapters) {
			adapter := manifest.Adapters[id]
			_, _ = fmt.Fprintf(w, "  - %s\n", id)
			_, _ = fmt.Fprintf(w, "      path: %s\n", adapter.Path)
			if len(adapter.Supports) > 0 {
				_, _ = fmt.Fprintf(w, "      supports: %s\n", strings.Join(adapter.Supports, ", "))
			}
		}
	}
	return nil
}

func RenderDesignValidateResponse(w io.Writer, format cliout.Format, report DesignValidationReport) error {
	if format == cliout.FormatJSON {
		return cliout.WriteSuccessJSON(w, "designValidation", report)
	}
	if len(report.Issues) == 0 {
		_, _ = fmt.Fprintf(w, "Validated %d design kits\n", report.Count)
		return nil
	}
	_, _ = fmt.Fprintf(w, "Design validation found %d issue(s) across %d kit(s):\n", len(report.Issues), report.Count)
	for _, issue := range report.Issues {
		scope := issue.Kit
		if issue.Adapter != "" {
			scope += "/" + issue.Adapter
		}
		if issue.Path != "" {
			_, _ = fmt.Fprintf(w, "  - %s (%s): %s\n", scope, issue.Path, issue.Message)
		} else {
			_, _ = fmt.Fprintf(w, "  - %s: %s\n", scope, issue.Message)
		}
	}
	return nil
}

func sortedDesignAdapterIDs(adapters map[string]DesignKitAdapter) []string {
	ids := make([]string, 0, len(adapters))
	for id := range adapters {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
