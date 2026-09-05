// Package contracts indexes scenario-owned program contracts for reuse.
package contracts

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
	repocontract "github.com/vrooli/repo-contract-go"
)

type Contract struct {
	Scenario         string
	Name             string
	ID               string
	Version          string
	Purpose          string
	InputNames       []string
	Inputs           map[string]InputSpec
	BindingIDs       []string
	WallMS           int64
	OutputSchemaPath string
	OutputSchema     *jsonschema.Schema
	OutputBytes      int64
	Rung             string
	OwnerSkill       string
	SourcePath       string
	Digest           string
	Source           string
	ValidationError  string
}

type InputSpec struct {
	Type     string
	Required bool
	Default  json.RawMessage
	Enum     []any
}

// ResolveInputs applies declared defaults and rejects unknown, missing, or
// shallowly mistyped values before a contract reaches the Python kernel. A
// program's nested schema remains the authority for domain-specific fields.
func (c Contract) ResolveInputs(provided map[string]any) (map[string]any, error) {
	resolved := make(map[string]any, len(c.Inputs))
	for name := range provided {
		if _, ok := c.Inputs[name]; !ok {
			return nil, fmt.Errorf("unknown input %q", name)
		}
	}
	for name, spec := range c.Inputs {
		value, ok := provided[name]
		if !ok && len(spec.Default) > 0 {
			if err := json.Unmarshal(spec.Default, &value); err != nil {
				return nil, fmt.Errorf("decode default for input %q: %w", name, err)
			}
			ok = true
		}
		if !ok {
			if spec.Required {
				return nil, fmt.Errorf("missing required input %q", name)
			}
			continue
		}
		if !matchesInputType(value, spec.Type) {
			return nil, fmt.Errorf("input %q must have type %s", name, spec.Type)
		}
		if len(spec.Enum) > 0 {
			matched := false
			for _, candidate := range spec.Enum {
				matched = matched || reflect.DeepEqual(value, candidate)
			}
			if !matched {
				return nil, fmt.Errorf("input %q is outside its declared enum", name)
			}
		}
		resolved[name] = value
	}
	return resolved, nil
}

func matchesInputType(value any, kind string) bool {
	switch kind {
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "number":
		_, ok := value.(float64)
		return ok
	case "integer":
		n, ok := value.(float64)
		return ok && !math.IsNaN(n) && !math.IsInf(n, 0) && math.Trunc(n) == n
	case "array":
		_, ok := value.([]any)
		return ok
	case "object":
		_, ok := value.(map[string]any)
		return ok
	default:
		return false
	}
}

type Index struct {
	mu        sync.RWMutex
	contracts []Contract
	mtimes    map[string]int64
}

func NewIndex() *Index { return &Index{mtimes: map[string]int64{}} }

func (i *Index) Load(repoRoot string) error {
	root := strings.TrimSpace(repoRoot)
	if root == "" {
		return fmt.Errorf("repository root is required")
	}
	repo, err := repocontract.LoadDefault(root)
	if err != nil {
		return fmt.Errorf("load repository contract: %w", err)
	}
	targets, err := repo.EnumerateTargets(root)
	if err != nil {
		return fmt.Errorf("enumerate repository targets: %w", err)
	}
	compiler := jsonschema.NewCompiler()
	schemaPath := filepath.Join(root, "scenarios", "program-runtime", "schemas", "program-contract.schema.json")
	if err := compiler.AddResource(schemaPath, bytes.NewReader(mustRead(schemaPath))); err != nil {
		return fmt.Errorf("load program contract schema: %w", err)
	}
	compiled, err := compiler.Compile(schemaPath)
	if err != nil {
		return fmt.Errorf("compile program contract schema: %w", err)
	}

	loaded := make([]Contract, 0)
	mtimes := make(map[string]int64)
	for _, target := range targets {
		if target.Kind != repocontract.TargetKindScenario {
			continue
		}
		dir := filepath.Join(root, "scenarios", target.ID, ".vrooli", "program-runtime")
		entries, readErr := os.ReadDir(dir)
		if readErr != nil {
			if os.IsNotExist(readErr) {
				continue
			}
			loaded = append(loaded, Contract{Scenario: target.ID, ID: target.ID, SourcePath: dir, ValidationError: readErr.Error()})
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			info, statErr := entry.Info()
			if statErr == nil {
				mtimes[path] = info.ModTime().UnixNano()
			}
			sourcePath := strings.TrimSuffix(path, filepath.Ext(path)) + ".py"
			if sourceInfo, sourceErr := os.Stat(sourcePath); sourceErr == nil {
				mtimes[sourcePath] = sourceInfo.ModTime().UnixNano()
			}
			contract := readContract(target.ID, path, compiled)
			if contract.OutputSchemaPath != "" {
				if info, err := os.Stat(contract.OutputSchemaPath); err == nil {
					mtimes[contract.OutputSchemaPath] = info.ModTime().UnixNano()
				}
			}
			loaded = append(loaded, contract)
		}
	}
	sort.SliceStable(loaded, func(a, b int) bool {
		if loaded[a].Scenario != loaded[b].Scenario {
			return loaded[a].Scenario < loaded[b].Scenario
		}
		return loaded[a].Name < loaded[b].Name
	})
	i.mu.Lock()
	i.contracts, i.mtimes = loaded, mtimes
	i.mu.Unlock()
	return nil
}

func (i *Index) Refresh(repoRoot string) (bool, error) {
	// Loading is cheap compared with scenario startup. Compare the discovered
	// file set and mtimes before rebuilding the in-memory projection.
	root := strings.TrimSpace(repoRoot)
	current := map[string]int64{}
	i.mu.RLock()
	schemaPaths := []string{}
	for _, contract := range i.contracts {
		if contract.OutputSchemaPath != "" {
			schemaPaths = append(schemaPaths, contract.OutputSchemaPath)
		}
	}
	i.mu.RUnlock()
	for _, path := range schemaPaths {
		if info, err := os.Stat(path); err == nil {
			current[path] = info.ModTime().UnixNano()
		}
	}

	// Only scan the declared program directories; walking node_modules and
	// build artifacts would turn a fresh kernel into a repository-wide crawl.
	scenarios, err := os.ReadDir(filepath.Join(root, "scenarios"))
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	for _, scenario := range scenarios {
		if !scenario.IsDir() {
			continue
		}
		dir := filepath.Join(root, "scenarios", scenario.Name(), ".vrooli", "program-runtime")
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return false, err
		}
		for _, entry := range entries {
			if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".json") && !strings.HasSuffix(entry.Name(), ".py")) {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				return false, err
			}
			current[filepath.Join(dir, entry.Name())] = info.ModTime().UnixNano()
		}
	}

	i.mu.RLock()
	unchanged := len(current) == len(i.mtimes)
	if unchanged {
		for path, mod := range current {
			if i.mtimes[path] != mod {
				unchanged = false
				break
			}
		}
	}
	i.mu.RUnlock()
	if unchanged {
		return false, nil
	}
	return true, i.Load(root)
}

func (i *Index) List() []Contract {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return append([]Contract(nil), i.contracts...)
}

func (i *Index) Get(scenario, name string) (Contract, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	for _, c := range i.contracts {
		if c.Scenario == scenario && c.Name == name {
			return c, true
		}
	}
	return Contract{}, false
}

func (i *Index) CoverageFor(bindingIDs []string) (Contract, bool) {
	need := make(map[string]struct{}, len(bindingIDs))
	for _, id := range bindingIDs {
		need[id] = struct{}{}
	}
	i.mu.RLock()
	defer i.mu.RUnlock()
	var best Contract
	covered := false
	for _, contract := range i.contracts {
		if contract.ValidationError != "" {
			continue
		}
		available := make(map[string]struct{}, len(contract.BindingIDs))
		for _, id := range contract.BindingIDs {
			available[id] = struct{}{}
		}
		ok := true
		for id := range need {
			if _, exists := available[id]; !exists {
				ok = false
				break
			}
		}
		if ok && (!covered || len(contract.BindingIDs) < len(best.BindingIDs) || (len(contract.BindingIDs) == len(best.BindingIDs) && contract.ID < best.ID)) {
			best, covered = contract, true
		}
	}
	return best, covered
}

func (i *Index) CoveredBy(bindingIDs []string) string {
	contract, ok := i.CoverageFor(bindingIDs)
	if !ok {
		return ""
	}
	return contract.ID
}

type rawContract struct {
	OutputSchema string                     `json:"output_schema"`
	Name         string                     `json:"name"`
	Version      string                     `json:"version"`
	Purpose      string                     `json:"purpose"`
	Inputs       map[string]json.RawMessage `json:"inputs"`
	Budget       struct {
		WallMS      int64 `json:"wall_ms"`
		OutputBytes int64 `json:"output_bytes"`
	} `json:"budget"`
	Bindings []struct {
		ID string `json:"id"`
	} `json:"bindings"`
	Rung       string `json:"rung"`
	OwnerSkill string `json:"owner_skill"`
}

func readContract(scenario, path string, schema *jsonschema.Schema) Contract {
	c := Contract{Scenario: scenario, SourcePath: path, ID: scenario + "." + strings.TrimSuffix(filepath.Base(path), ".json")}
	data, err := os.ReadFile(path)
	if err != nil {
		c.ValidationError = err.Error()
		return c
	}
	sourcePath := strings.TrimSuffix(path, filepath.Ext(path)) + ".py"
	source, sourceErr := os.ReadFile(sourcePath)
	if sourceErr != nil {
		c.ValidationError = fmt.Sprintf("source file %s: %v", sourcePath, sourceErr)
	} else {
		c.Source = string(source)
	}
	digest := sha256.Sum256(append(append(data, 0), source...))
	c.Digest = hex.EncodeToString(digest[:])
	var raw rawContract
	if err := json.Unmarshal(data, &raw); err != nil {
		c.ValidationError = err.Error()
		return c
	}
	c.OutputBytes = raw.Budget.OutputBytes
	if c.OutputBytes == 0 {
		c.OutputBytes = 4096
	}
	if raw.OutputSchema != "" {
		clean := filepath.Clean(raw.OutputSchema)
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			c.ValidationError = "output_schema must be local to the program directory"
			return c
		}
		c.OutputSchemaPath = filepath.Join(filepath.Dir(path), clean)
		schemaBytes, err := os.ReadFile(c.OutputSchemaPath)
		if err != nil {
			c.ValidationError = err.Error()
			return c
		}
		compiler := jsonschema.NewCompiler()
		if err = compiler.AddResource(c.OutputSchemaPath, bytes.NewReader(schemaBytes)); err == nil {
			c.OutputSchema, err = compiler.Compile(c.OutputSchemaPath)
		}
		if err != nil {
			c.ValidationError = err.Error()
			return c
		}
		digest = sha256.Sum256(append(append(append(append([]byte{}, data...), 0), source...), schemaBytes...))
		c.Digest = hex.EncodeToString(digest[:])
	}
	c.Name, c.Version, c.Purpose, c.Rung, c.OwnerSkill, c.WallMS = raw.Name, raw.Version, raw.Purpose, raw.Rung, raw.OwnerSkill, raw.Budget.WallMS
	if prefix := scenario + "."; strings.HasPrefix(c.Name, prefix) {
		c.Name = strings.TrimPrefix(c.Name, prefix)
	}
	c.Inputs = make(map[string]InputSpec, len(raw.Inputs))
	for name, encoded := range raw.Inputs {
		c.InputNames = append(c.InputNames, name)
		var spec struct {
			Type     string          `json:"type"`
			Required bool            `json:"required"`
			Default  json.RawMessage `json:"default"`
			Enum     []any           `json:"enum"`
		}
		if err := json.Unmarshal(encoded, &spec); err != nil {
			c.ValidationError = fmt.Sprintf("input %s: %v", name, err)
			continue
		}
		c.Inputs[name] = InputSpec{Type: spec.Type, Required: spec.Required, Default: spec.Default, Enum: spec.Enum}
	}
	sort.Strings(c.InputNames)
	for _, binding := range raw.Bindings {
		c.BindingIDs = append(c.BindingIDs, binding.ID)
	}
	sort.Strings(c.BindingIDs)
	var document any
	if err := json.Unmarshal(data, &document); err == nil {
		if err := schema.Validate(document); err != nil {
			c.ValidationError = err.Error()
		}
	}
	if c.Name == "" {
		c.Name = strings.TrimSuffix(filepath.Base(path), ".json")
	}
	return c
}

func mustRead(path string) []byte { data, _ := os.ReadFile(path); return data }
