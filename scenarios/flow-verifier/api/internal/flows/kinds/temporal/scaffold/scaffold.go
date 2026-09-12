// Package scaffold emits a minimal but fully-runnable temporal flow
// into <parent>/flow/ on disk. The flow has one state, one event,
// one self-transition, and one invariant — enough to verify, replay,
// and lint green with zero hand edits after the scaffold runs.
package scaffold

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"flow-verifier/internal/flows/kinds/temporal/layout"
)

// Options drives a single scaffold invocation.
type Options struct {
	// Root is the absolute or relative path to the scenario root.
	Root string
	// ParentDir is the directory (relative to Root) that should
	// contain the new flow/ subdirectory.
	ParentDir string
	// FlowID is the dotted "<domain>.<name>.<surface>" identifier.
	FlowID string
	// Language is the target runtime; "" means infer from ParentDir.
	Language layout.Language
}

// Write materializes the scaffold on disk. Returns the (relative)
// flow directory it created.
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
	lang, err := resolveLanguage(opts)
	if err != nil {
		return "", err
	}
	parts := strings.Split(opts.FlowID, ".")
	if len(parts) < 3 {
		return "", fmt.Errorf("scaffold: flowId %q must have at least three dotted segments (e.g. domain.name.surface)", opts.FlowID)
	}
	flowDirRel := filepath.ToSlash(filepath.Join(opts.ParentDir, layout.FlowDirName))
	flowDirAbs := filepath.Join(opts.Root, filepath.FromSlash(flowDirRel))
	if _, err := os.Stat(flowDirAbs); err == nil {
		return "", fmt.Errorf("scaffold: %s already exists; remove it or pick a different parent", flowDirRel)
	}

	data := buildTemplateData(opts.FlowID, parts, lang, opts.ParentDir)
	files, err := renderFiles(lang, data)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(flowDirAbs, 0o755); err != nil {
		return "", err
	}
	for name, body := range files {
		target := filepath.Join(flowDirAbs, name)
		if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
			return "", err
		}
	}
	return flowDirRel, nil
}

// resolveLanguage selects the runtime language for the scaffold.
// Explicit Options.Language wins; otherwise we infer from the parent
// dir prefix.
func resolveLanguage(opts Options) (layout.Language, error) {
	if opts.Language != "" {
		switch opts.Language {
		case layout.LanguageGo, layout.LanguageTypeScript:
			return opts.Language, nil
		default:
			return "", fmt.Errorf("scaffold: unsupported language %q", opts.Language)
		}
	}
	clean := filepath.ToSlash(filepath.Clean(opts.ParentDir))
	switch {
	case strings.HasPrefix(clean, "ui/"), clean == "ui":
		return layout.LanguageTypeScript, nil
	case strings.HasPrefix(clean, "api/"), clean == "api":
		return layout.LanguageGo, nil
	default:
		return "", fmt.Errorf("scaffold: cannot infer language from parent dir %q (expected ui/* or api/*); pass --lang ts|go explicitly", opts.ParentDir)
	}
}

type templateData struct {
	FlowID              string
	Domain              string
	Name                string // PascalCase
	LowerName           string // camelCase
	ModuleName          string
	GeneratedImportPath string // for Go scaffolds
}

func buildTemplateData(flowID string, parts []string, _ layout.Language, parentDir string) templateData {
	domain := parts[0]
	middle := strings.Join(parts[1:len(parts)-1], "-")
	name := pascalCase(middle)
	importPath := generatedImportPath(parentDir)
	return templateData{
		FlowID:              flowID,
		Domain:              domain,
		Name:                name,
		LowerName:           lowerFirst(name),
		ModuleName:          name + "Flow",
		GeneratedImportPath: importPath,
	}
}

// generatedImportPath mirrors layout.SubpackageImportPath: it strips
// a leading "api/" and emits a {{SCENARIO_ID}}-anchored import path
// suitable for the react-vite template materialization step.
func generatedImportPath(parentDir string) string {
	clean := filepath.ToSlash(filepath.Clean(parentDir))
	clean = strings.TrimPrefix(clean, "api/")
	return "{{SCENARIO_ID}}/" + clean + "/" + layout.FlowDirName + "/" + layout.GeneratedDirName
}

func renderFiles(lang layout.Language, data templateData) (map[string]string, error) {
	out := map[string]string{}
	if err := render(out, "flow.json", flowJSONTemplate(lang), data); err != nil {
		return nil, err
	}
	switch lang {
	case layout.LanguageTypeScript:
		if err := render(out, "transition.ts", transitionTSTemplate, data); err != nil {
			return nil, err
		}
		if err := render(out, "fixtures.ts", fixturesTSTemplate, data); err != nil {
			return nil, err
		}
		if err := render(out, "flow.test.ts", flowTestTSTemplate, data); err != nil {
			return nil, err
		}
	case layout.LanguageGo:
		if err := render(out, "transition.go", transitionGoTemplate, data); err != nil {
			return nil, err
		}
		if err := render(out, "flow_test.go", flowTestGoTemplate, data); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func render(out map[string]string, name string, body string, data templateData) error {
	tmpl, err := template.New(name).Parse(body)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}
	out[name] = buf.String()
	return nil
}

func pascalCase(value string) string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '-' || r == '_' || r == '.' || r == ' '
	})
	var b strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		lower := strings.ToLower(part)
		b.WriteString(strings.ToUpper(lower[:1]))
		if len(lower) > 1 {
			b.WriteString(lower[1:])
		}
	}
	return b.String()
}

func lowerFirst(value string) string {
	if value == "" {
		return value
	}
	return strings.ToLower(value[:1]) + value[1:]
}

func flowJSONTemplate(lang layout.Language) string {
	if lang == layout.LanguageGo {
		return flowJSONGoTemplate
	}
	return flowJSONTSTemplate
}

const flowJSONTSTemplate = `{
  "schemaVersion": 6,
  "kind": "temporal",
  "flowId": "{{.FlowID}}",
  "domain": "{{.Domain}}",
  "description": "Scaffolded {{.Name}} flow.",
  "model": {
    "module": "{{.ModuleName}}",
    "seed": "1",
    "maxSteps": 4,
    "traceCount": 4,
    "verify": { "invariants": ["TypeOK"] }
  },
  "states": [
    { "id": "idle", "quint": "Idle", "initial": true },
    { "id": "ready", "quint": "Ready" }
  ],
  "events": [
    { "id": "start", "quint": "Start" },
    { "id": "reset", "quint": "Reset" }
  ],
  "transitionDefaults": {
    "invalid": { "to": "self", "wantError": false }
  },
  "transitions": [
    { "from": "idle", "event": "start", "to": "ready" },
    { "from": "ready", "event": "reset", "to": "idle" }
  ],
  "invariants": [
    { "id": "type_ok", "quint": "TypeOK", "description": "Types remain valid." }
  ],
  "traces": [
    {
      "name": "smoke",
      "initial": "idle",
      "steps": [
        { "event": "start", "want": "ready", "wantError": false },
        { "event": "reset", "want": "idle", "wantError": false }
      ]
    }
  ],
  "runtime": {
    "typescript": {
      "statusType": "{{.Name}}Status",
      "eventType": "{{.Name}}EventType",
      "statusesConst": "{{.LowerName}}Statuses",
      "eventsConst": "{{.LowerName}}Events",
      "formalExpectationConst": "{{.LowerName}}FormalExpectation",
      "stateUnionType": "{{.Name}}State",
      "eventUnionType": "{{.Name}}Event",
      "stateVariants": { "idle": {}, "ready": {} },
      "eventVariants": { "start": {}, "reset": {} }
    }
  },
  "replay": {
    "transition": {
      "function": "transition{{.Name}}",
      "statusAccessor": "state.status"
    }
  }
}
`

const flowJSONGoTemplate = `{
  "schemaVersion": 6,
  "kind": "temporal",
  "flowId": "{{.FlowID}}",
  "domain": "{{.Domain}}",
  "description": "Scaffolded {{.Name}} flow.",
  "model": {
    "module": "{{.ModuleName}}",
    "seed": "1",
    "maxSteps": 4,
    "traceCount": 4,
    "verify": { "invariants": ["TypeOK"] }
  },
  "states": [
    { "id": "idle", "quint": "Idle", "initial": true },
    { "id": "ready", "quint": "Ready" }
  ],
  "events": [
    { "id": "start", "quint": "Start" },
    { "id": "reset", "quint": "Reset" }
  ],
  "transitionDefaults": {
    "invalid": { "to": "self", "wantError": false }
  },
  "transitions": [
    { "from": "idle", "event": "start", "to": "ready" },
    { "from": "ready", "event": "reset", "to": "idle" }
  ],
  "invariants": [
    { "id": "type_ok", "quint": "TypeOK", "description": "Types remain valid." }
  ],
  "traces": [
    {
      "name": "smoke",
      "initial": "idle",
      "steps": [
        { "event": "start", "want": "ready", "wantError": false },
        { "event": "reset", "want": "idle", "wantError": false }
      ]
    }
  ],
  "runtime": {
    "go": {
      "package": "flow",
      "statusType": "Status",
      "eventType": "Event",
      "constantPrefix": "{{.Name}}"
    }
  },
  "replay": {
    "transition": {
      "function": "Transition{{.Name}}",
      "stateType": "State",
      "statusField": "Status"
    }
  }
}
`

const transitionTSTemplate = `import type { {{.Name}}State, {{.Name}}Event } from "./generated/runtime";
import { next{{.Name}}Status } from "./generated/runtime";

export const transition{{.Name}} = (state: {{.Name}}State, event: {{.Name}}Event): {{.Name}}State => {
  const next = next{{.Name}}Status(state.status, event.type);
  return { status: next } as {{.Name}}State;
};
`

const fixturesTSTemplate = `import type { {{.Name}}FormalReplayFixtures } from "./generated/replay.helper";

export const {{.LowerName}}FormalFixtures = {
  stateFor: {
    idle: () => ({ status: "idle" as const }),
    ready: () => ({ status: "ready" as const }),
  },
  eventFor: {
    start: () => ({ type: "start" as const }),
    reset: () => ({ type: "reset" as const }),
  },
} satisfies {{.Name}}FormalReplayFixtures;
`

const flowTestTSTemplate = `import { runFormalReplay } from "./generated/replay.helper";
import { transition{{.Name}} } from "./transition";
import { {{.LowerName}}FormalFixtures } from "./fixtures";

runFormalReplay({
  transition: transition{{.Name}},
  fixtures: {{.LowerName}}FormalFixtures,
});
`

const transitionGoTemplate = `package flow

import (
	"{{.GeneratedImportPath}}"
)

// Transition{{.Name}} is the hand-authored wrapper around the
// generated state machine for the {{.FlowID}} flow.
func Transition{{.Name}}(status generated.Status, event generated.Event) (generated.Status, error) {
	return generated.Transition{{.Name}}Status(status, event)
}
`

const flowTestGoTemplate = `package flow

import (
	"testing"

	"{{.GeneratedImportPath}}"
)

func Test{{.Name}}FormalReplay(t *testing.T) {
	generated.RunReplay(t, func(status generated.Status, event generated.Event) (generated.Status, error) {
		return Transition{{.Name}}(status, event)
	})
}
`
