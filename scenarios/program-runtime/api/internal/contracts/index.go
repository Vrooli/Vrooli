// Package contracts indexes scenario-owned program contracts for reuse.
package contracts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
	repocontract "github.com/vrooli/repo-contract-go"
)

type Contract struct {
	Scenario        string
	Name            string
	ID              string
	Version         string
	Purpose         string
	InputNames      []string
	BindingIDs      []string
	Rung            string
	OwnerSkill      string
	SourcePath      string
	Source          string
	ValidationError string
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
			loaded = append(loaded, readContract(target.ID, path, compiled))
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
	_ = filepath.WalkDir(filepath.Join(root, "scenarios"), func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".json") && !strings.HasSuffix(entry.Name(), ".py")) || !strings.Contains(filepath.ToSlash(path), "/.vrooli/program-runtime/") {
			return nil
		}
		if info, statErr := entry.Info(); statErr == nil {
			current[path] = info.ModTime().UnixNano()
		}
		return nil
	})
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
	Name     string                     `json:"name"`
	Version  string                     `json:"version"`
	Purpose  string                     `json:"purpose"`
	Inputs   map[string]json.RawMessage `json:"inputs"`
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
	var raw rawContract
	if err := json.Unmarshal(data, &raw); err != nil {
		c.ValidationError = err.Error()
		return c
	}
	c.Name, c.Version, c.Purpose, c.Rung, c.OwnerSkill = raw.Name, raw.Version, raw.Purpose, raw.Rung, raw.OwnerSkill
	if prefix := scenario + "."; strings.HasPrefix(c.Name, prefix) {
		c.Name = strings.TrimPrefix(c.Name, prefix)
	}
	for name := range raw.Inputs {
		c.InputNames = append(c.InputNames, name)
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
