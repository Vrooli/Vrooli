// Package scaffold emits a minimal but schema-valid navigation contract
// into <parent>/flow/navigation.json on disk. The contract has two
// routes, one persistent container, and one affordance — enough to
// validate and round-trip with zero hand edits after the scaffold runs.
package scaffold

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// flowDirName matches the convention enforced by discovery: any
// *.json under a directory literally named "flow" is a contract.
const flowDirName = "flow"

// Options drives a single scaffold invocation.
type Options struct {
	// Root is the absolute or relative path to the scenario root.
	Root string
	// ParentDir is the directory (relative to Root) that should
	// contain the new flow/ subdirectory.
	ParentDir string
	// FlowID is the dotted "<scenario>.<feature>.<surface>" identifier.
	FlowID string
}

// Write materializes the scaffold on disk. Returns the (relative) flow
// directory it created.
func Write(opts Options) (string, error) {
	if opts.Root == "" {
		return "", fmt.Errorf("scaffold: Root is required")
	}
	if opts.ParentDir == "" {
		return "", fmt.Errorf("scaffold: ParentDir is required")
	}
	if opts.FlowID == "" {
		return "", fmt.Errorf("scaffold: FlowID is required")
	}
	parts := strings.Split(opts.FlowID, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("scaffold: flowId %q must have at least two dotted segments", opts.FlowID)
	}

	flowDirRel := filepath.ToSlash(filepath.Join(opts.ParentDir, flowDirName))
	flowDirAbs := filepath.Join(opts.Root, filepath.FromSlash(flowDirRel))
	target := filepath.Join(flowDirAbs, "navigation.json")
	if _, err := os.Stat(target); err == nil {
		return "", fmt.Errorf("scaffold: %s/navigation.json already exists; remove it or pick a different parent", flowDirRel)
	}

	body, err := renderNavigationJSON(opts.FlowID, parts[0])
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(flowDirAbs, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
		return "", err
	}
	return flowDirRel, nil
}

type templateData struct {
	FlowID string
	Domain string
}

func renderNavigationJSON(flowID, domain string) (string, error) {
	tmpl, err := template.New("navigation.json").Parse(navigationJSONTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData{FlowID: flowID, Domain: domain}); err != nil {
		return "", err
	}
	return buf.String(), nil
}

const navigationJSONTemplate = `{
  "$schema": "https://vrooli.dev/schemas/navigation.v1.json",
  "schemaVersion": 1,
  "kind": "navigation",
  "flowId": "{{.FlowID}}",
  "domain": "{{.Domain}}",
  "description": "Scaffolded navigation graph.",
  "contexts": {
    "viewport": {
      "kind": "enum",
      "values": ["mobile", "desktop"],
      "default": "desktop"
    }
  },
  "routes": [
    {
      "id": "home",
      "path": "/",
      "page": "ui/src/pages/Home.tsx",
      "parents": []
    },
    {
      "id": "about",
      "path": "/about",
      "page": "ui/src/pages/About.tsx",
      "parents": ["home"]
    }
  ],
  "containers": [
    {
      "id": "top_nav_bar",
      "kind": "persistent",
      "host_routes": ["*"],
      "disclosure": "always_visible"
    }
  ],
  "affordances": [
    {
      "id": "nav_about",
      "to": "about",
      "presentations": [
        {
          "in": "top_nav_bar",
          "label": "About",
          "test_id": "top-nav-about",
          "reachable_via": ["mouse", "keyboard"]
        }
      ]
    }
  ]
}
`
