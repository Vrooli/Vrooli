package actions

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
	repocontract "github.com/vrooli/repo-contract-go"
)

// cliManifestSchemaName mirrors the canonical filename under .vrooli/schemas/
// and matches Phase 2's cli-manifest/v1 $id.
const cliManifestSchemaName = "cli-manifest.schema.json"

// cliManifest is the in-memory shape of a scenario's cli/manifest.json after
// schema validation. Mirrors .vrooli/schemas/cli-manifest.schema.json. Only
// the fields the resolver consumes are decoded; unknown fields are ignored.
type cliManifest struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Groups      []cliManifestGroup `json:"groups"`
}

type cliManifestGroup struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Flat        bool                 `json:"flat"`
	Commands    []cliManifestCommand `json:"commands"`
}

type cliManifestCommand struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Governance  cliManifestGovernance `json:"governance"`
}

type cliManifestGovernance struct {
	Effect                    string   `json:"effect"`
	RunEligible               bool     `json:"run_eligible"`
	HasRequiresConfirmation   bool     `json:"-"`
	RequiresConfirmationValue bool     `json:"-"`
	Permissions               []string `json:"permissions"`
}

// UnmarshalJSON captures whether requires_confirmation was set explicitly
// vs. defaulted, so the resolver can apply the "destructive defaults to true"
// rule without overriding an explicit false.
func (g *cliManifestGovernance) UnmarshalJSON(data []byte) error {
	var raw struct {
		Effect               string   `json:"effect"`
		RunEligible          bool     `json:"run_eligible"`
		RequiresConfirmation *bool    `json:"requires_confirmation"`
		Permissions          []string `json:"permissions"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	g.Effect = raw.Effect
	g.RunEligible = raw.RunEligible
	g.Permissions = raw.Permissions
	if raw.RequiresConfirmation != nil {
		g.HasRequiresConfirmation = true
		g.RequiresConfirmationValue = *raw.RequiresConfirmation
	}
	return nil
}

// cliManifestLoadResult carries one of three outcomes:
//   - Manifest != nil → schema-valid manifest loaded (or empty if missing).
//   - Err != nil      → manifest file exists but is unreadable or schema-invalid.
//   - Missing == true → no manifest file; caller treats as "unvalidated".
type cliManifestLoadResult struct {
	Manifest *cliManifest
	Missing  bool
	Err      error
}

// cliManifestCache resolves and caches per-scenario manifests, sharing one
// compiled schema across calls. Safe for concurrent use.
type cliManifestCache struct {
	repoRoot  string
	mu        sync.Mutex
	schema    *jsonschema.Schema
	schemaErr error
	manifests map[string]cliManifestLoadResult
}

func newCLIManifestCache(repoRoot string) *cliManifestCache {
	return &cliManifestCache{
		repoRoot:  repoRoot,
		manifests: map[string]cliManifestLoadResult{},
	}
}

func (c *cliManifestCache) load(scenario string) cliManifestLoadResult {
	c.mu.Lock()
	defer c.mu.Unlock()
	if cached, ok := c.manifests[scenario]; ok {
		return cached
	}
	result := c.loadLocked(scenario)
	c.manifests[scenario] = result
	return result
}

func (c *cliManifestCache) loadLocked(scenario string) cliManifestLoadResult {
	manifestPath, err := repocontract.ScenarioCLIManifestPath(c.repoRoot, scenario)
	if err != nil {
		return cliManifestLoadResult{Err: fmt.Errorf("resolve manifest path: %w", err)}
	}
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cliManifestLoadResult{Missing: true}
		}
		return cliManifestLoadResult{Err: fmt.Errorf("read %s: %w", manifestPath, err)}
	}
	schema, schemaErr := c.compileSchema()
	if schemaErr != nil {
		// Without a schema we cannot validate; treat as missing so the
		// resolver falls back to the unvalidated (CertaintyOwnerOnly) path
		// rather than rejecting actions during a transient infra problem.
		return cliManifestLoadResult{Missing: true}
	}
	var doc any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return cliManifestLoadResult{Err: fmt.Errorf("parse %s: %w", manifestPath, err)}
	}
	if err := schema.Validate(doc); err != nil {
		return cliManifestLoadResult{Err: fmt.Errorf("validate %s against %s: %w", manifestPath, cliManifestSchemaName, err)}
	}
	manifest := &cliManifest{}
	if err := json.Unmarshal(raw, manifest); err != nil {
		return cliManifestLoadResult{Err: fmt.Errorf("decode %s: %w", manifestPath, err)}
	}
	return cliManifestLoadResult{Manifest: manifest}
}

func (c *cliManifestCache) compileSchema() (*jsonschema.Schema, error) {
	if c.schema != nil || c.schemaErr != nil {
		return c.schema, c.schemaErr
	}
	schemaPath, err := repocontract.SchemaPath(c.repoRoot, cliManifestSchemaName)
	if err != nil {
		c.schemaErr = err
		return nil, err
	}
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		c.schemaErr = err
		return nil, err
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(cliManifestSchemaName, bytes.NewReader(schemaBytes)); err != nil {
		c.schemaErr = err
		return nil, err
	}
	schema, err := compiler.Compile(cliManifestSchemaName)
	if err != nil {
		c.schemaErr = err
		return nil, err
	}
	c.schema = schema
	return c.schema, nil
}

// resolveManifestCommand walks argv[1:] against the manifest. argv[0] is the
// scenario CLI target and is consumed by the caller. Normal groups use
// <group> <command>; flat groups use <command> directly. Returns
// (cmd, group, true) on hit; (_, _, false) if no declared command matches.
func resolveManifestCommand(manifest *cliManifest, argv []string) (cliManifestCommand, cliManifestGroup, bool) {
	if manifest == nil || len(argv) < 2 {
		return cliManifestCommand{}, cliManifestGroup{}, false
	}
	for _, group := range manifest.Groups {
		if group.Flat {
			for _, cmd := range group.Commands {
				if cmd.Name == argv[1] {
					return cmd, group, true
				}
			}
			continue
		}
		if len(argv) < 3 || group.Name != argv[1] {
			continue
		}
		commandName := argv[2]
		for _, cmd := range group.Commands {
			if cmd.Name == commandName {
				return cmd, group, true
			}
		}
	}
	return cliManifestCommand{}, cliManifestGroup{}, false
}

// manifestEffectToCommandEffect maps the manifest's effect vocabulary
// (read|write|destructive) to prompt-manager's CommandEffect. The manifest
// schema deliberately omits "admin"; admin-level commands stay outside the
// auto-derivable surface for v1.
func manifestEffectToCommandEffect(effect string) (CommandEffect, bool) {
	switch effect {
	case "read":
		return EffectRead, true
	case "write":
		return EffectWrite, true
	case "destructive":
		return EffectDestructive, true
	default:
		return "", false
	}
}
