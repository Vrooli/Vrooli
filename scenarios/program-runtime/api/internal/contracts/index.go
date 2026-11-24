// Package contracts indexes scenario-owned program contracts for reuse.
package contracts

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/vrooli/api-core/researchhome"
	repocontract "github.com/vrooli/repo-contract-go"
)

type Contract struct {
	LearningTask       *LearningTask   `json:"learning_task,omitempty"`
	Learning           LearningSummary `json:"learning,omitempty"`
	Memory             *Memory         `json:"memory,omitempty"`
	Verbs              []string
	Fixtures           []Fixture
	Bindings           []BindingRef
	Assumptions        []string
	Invariants         []string
	Signals            map[string]string
	Scenario           string
	Name               string
	ID                 string
	Version            string
	Purpose            string
	InputNames         []string
	Inputs             map[string]InputSpec
	BindingIDs         []string
	WallMS             int64
	OutputSchemaPath   string
	OutputSchema       *jsonschema.Schema `json:"-"`
	Declaration        []byte
	OutputSchemaSource []byte
	OutputBytes        int64
	Async              bool
	AsyncReason        string
	Rung               string
	OwnerSkill         string
	SourcePath         string
	Digest             string
	Source             string
	SourceMissing      bool
	ValidationError    string
	// Research marks a research-home program (research/<topic>/<name> or
	// research/<name>). It runs by id but is not lib-callable.
	Research bool
}

// RuntimeDeclaration returns the immutable on-disk declaration augmented with
// its already-validated output schema. The schema body travels with a library
// snapshot so the kernel can validate nested results without reading files or
// trusting a path supplied by a program.
func (c Contract) RuntimeDeclaration() ([]byte, error) {
	declaration := append([]byte(nil), c.Declaration...)
	if c.Digest != "" && c.Digest != ContentDigest(c.Declaration, []byte(c.Source), c.OutputSchemaSource) {
		return nil, errors.New("contract content digest does not match its declaration, source, and output schema")
	}
	if len(c.OutputSchemaSource) == 0 {
		return declaration, nil
	}
	if !json.Valid(c.OutputSchemaSource) {
		return nil, errors.New("compiled output schema source is not valid JSON")
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(declaration, &document); err != nil {
		return nil, fmt.Errorf("decode contract declaration for runtime: %w", err)
	}
	if document == nil {
		return nil, errors.New("contract declaration for runtime must be an object")
	}
	document["_runtime_output_schema"] = append(json.RawMessage(nil), c.OutputSchemaSource...)
	encoded, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("encode runtime contract declaration: %w", err)
	}
	return encoded, nil
}

// LearningSummary is the analyzer-derived projection of the in-program
// learning surface declared by a contract. It is intentionally small: callers
// need to know which verbs and note kinds are present, not duplicate source.
type LearningSummary struct {
	Verbs                    []string             `json:"verbs,omitempty"`
	UsesMemory               bool                 `json:"uses_memory,omitempty"`
	NoteKinds                []string             `json:"note_kinds,omitempty"`
	FreeTextInputsWithoutKey bool                 `json:"free_text_inputs_without_key,omitempty"`
	Capabilities             []LearningCapability `json:"capabilities,omitempty"`
}

type LearningCapability struct {
	Scenario string   `json:"scenario"`
	Effects  []string `json:"effects"`
}

type Memory struct {
	Scope      string   `json:"scope"`
	ReadsIn    string   `json:"reads_in"`
	WritesIn   string   `json:"writes_in"`
	EntryKinds []string `json:"entry_kinds"`
	ScopeInput string   `json:"scope_input,omitempty"`
	WritesWhen string   `json:"writes_when,omitempty"`
	ReadScopes []string `json:"read_scopes,omitempty"`
}

type Fixture struct {
	ID       string   `json:"id"`
	Requires []string `json:"requires,omitempty"`
}

type BindingRef struct {
	ID       string `json:"id"`
	Effect   string `json:"effect"`
	Optional bool   `json:"optional,omitempty"`
	Via      string `json:"via,omitempty"`
	Note     string `json:"note,omitempty"`
}

// LearningTask registers one task boundary. Paths are dot-separated object
// keys, never expressions. The domain owner supplies outcome semantics.
type LearningTask struct {
	Scope         string   `json:"scope"`
	Operation     string   `json:"operation"`
	ContextFields []string `json:"context_fields"`
	Outcome       struct {
		StatusPath    string            `json:"status_path"`
		Mapping       map[string]string `json:"mapping"`
		EvidencePaths []string          `json:"evidence_paths"`
	} `json:"outcome"`
}

type InputSpec struct {
	Type     string
	Required bool
	Default  json.RawMessage
	Enum     []any
	FreeText bool
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
	loadMu       sync.Mutex
	mu           sync.RWMutex
	contracts    []Contract
	mtimes       map[string]int64
	researchHome string
}

func NewIndex() *Index { return &Index{mtimes: map[string]int64{}} }

// SetResearchHome adds the research home's programs to the index. Without
// it the index holds repository contracts only, so validation and tests stay
// independent of the operator's research files.
func (i *Index) SetResearchHome(dir string) {
	i.mu.Lock()
	i.researchHome = dir
	i.mu.Unlock()
}

// programDir is one folder of program contracts and the scope that names
// them: a scenario, or research / research/<topic>.
type programDir struct{ scope, dir, home string } // home is set for research folders

func (i *Index) researchDirs() ([]programDir, error) {
	i.mu.RLock()
	home := i.researchHome
	i.mu.RUnlock()
	if home == "" {
		return nil, nil
	}
	scopes, err := researchhome.Scopes(home)
	dirs := make([]programDir, 0, len(scopes))
	for _, scope := range scopes {
		dirs = append(dirs, programDir{scope: strings.TrimSuffix(researchhome.Prefix+scope.Topic, "/"), dir: filepath.Join(scope.Dir, "programs"), home: home})
	}
	return dirs, err
}

func (i *Index) Load(repoRoot string) error {
	return i.loadSnapshot(func() ([]Contract, map[string]int64, error) {
		root := strings.TrimSpace(repoRoot)
		if root == "" {
			return nil, nil, fmt.Errorf("repository root is required")
		}
		repo, err := repocontract.LoadDefault(root)
		if err != nil {
			return nil, nil, fmt.Errorf("load repository contract: %w", err)
		}
		targets, err := repo.EnumerateTargets(root)
		if err != nil {
			return nil, nil, fmt.Errorf("enumerate repository targets: %w", err)
		}
		compiler := jsonschema.NewCompiler()
		schemaPath := filepath.Join(root, "scenarios", "program-runtime", "schemas", "program-contract.schema.json")
		if err := compiler.AddResource(schemaPath, bytes.NewReader(mustRead(schemaPath))); err != nil {
			return nil, nil, fmt.Errorf("load program contract schema: %w", err)
		}
		compiled, err := compiler.Compile(schemaPath)
		if err != nil {
			return nil, nil, fmt.Errorf("compile program contract schema: %w", err)
		}

		loaded := make([]Contract, 0)
		mtimes := make(map[string]int64)
		if info, err := os.Stat(schemaPath); err == nil {
			mtimes[schemaPath] = info.ModTime().UnixNano()
		}
		var dirs []programDir
		for _, target := range targets {
			if target.Kind == repocontract.TargetKindScenario {
				dirs = append(dirs, programDir{scope: target.ID, dir: filepath.Join(root, "scenarios", target.ID, ".vrooli", "program-runtime")})
			}
		}
		research, researchErr := i.researchDirs()
		if researchErr != nil {
			loaded = append(loaded, Contract{Scenario: "research", ID: "research", Research: true, ValidationError: "research home: " + researchErr.Error()})
		}
		for _, d := range append(dirs, research...) {
			loaded = loadProgramDir(d, compiled, loaded, mtimes)
		}
		sort.SliceStable(loaded, func(a, b int) bool {
			if loaded[a].Scenario != loaded[b].Scenario {
				return loaded[a].Scenario < loaded[b].Scenario
			}
			return loaded[a].Name < loaded[b].Name
		})
		return loaded, mtimes, nil
	})
}

// loadProgramDir reads every contract in one program folder.
func loadProgramDir(d programDir, compiled *jsonschema.Schema, loaded []Contract, mtimes map[string]int64) []Contract {
	entries, err := os.ReadDir(d.dir)
	if os.IsNotExist(err) {
		return loaded
	}
	if err != nil {
		return append(loaded, Contract{Scenario: d.scope, ID: d.scope, SourcePath: d.dir, ValidationError: err.Error(), Research: researchhome.IsID(d.scope + "/")})
	}
	for _, entry := range entries {
		if !isContractFile(entry) {
			continue
		}
		path := filepath.Join(d.dir, entry.Name())
		if info, statErr := entry.Info(); statErr == nil {
			mtimes[path] = info.ModTime().UnixNano()
		}
		sourcePath := strings.TrimSuffix(path, filepath.Ext(path)) + ".py"
		if sourceInfo, sourceErr := os.Stat(sourcePath); sourceErr == nil {
			mtimes[sourcePath] = sourceInfo.ModTime().UnixNano()
		}
		contract := readContract(d.scope, path, compiled)
		if d.home != "" {
			contract = confineToResearchHome(contract, d.home)
		}
		if contract.OutputSchemaPath != "" {
			if info, err := os.Stat(contract.OutputSchemaPath); err == nil {
				mtimes[contract.OutputSchemaPath] = info.ModTime().UnixNano()
			}
		}
		loaded = append(loaded, contract)
	}
	return loaded
}

// confineToResearchHome refuses a research contract whose declaration,
// source or output schema resolves outside the research home (a symlink out
// of it), so nothing outside the home is loaded or run.
func confineToResearchHome(c Contract, home string) Contract {
	for _, p := range []string{c.SourcePath, strings.TrimSuffix(c.SourcePath, ".json") + ".py", c.OutputSchemaPath} {
		if p == "" {
			continue
		}
		if _, err := researchhome.Resolve(home, p); errors.Is(err, researchhome.ErrOutside) {
			return Contract{Scenario: c.Scenario, ID: c.ID, Name: c.Name, SourcePath: c.SourcePath, Research: true, ValidationError: err.Error()}
		}
	}
	return c
}

func isContractFile(entry os.DirEntry) bool {
	name := entry.Name()
	return !entry.IsDir() && name != "note-kinds.json" && strings.HasSuffix(name, ".json") && !strings.HasSuffix(name, ".output.schema.json")
}

func (i *Index) loadSnapshot(read func() ([]Contract, map[string]int64, error)) error {
	// Serialize reading and publication together. If concurrent loads only lock
	// the final assignment, an older disk snapshot can finish after and replace a
	// newer one.
	i.loadMu.Lock()
	defer i.loadMu.Unlock()

	loaded, mtimes, err := read()
	if err != nil {
		return err
	}
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
	schemaPaths := []string{filepath.Join(root, "scenarios", "program-runtime", "schemas", "program-contract.schema.json")}
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
	var dirs []string
	for _, scenario := range scenarios {
		if scenario.IsDir() {
			dirs = append(dirs, filepath.Join(root, "scenarios", scenario.Name(), ".vrooli", "program-runtime"))
		}
	}
	research, _ := i.researchDirs()
	for _, d := range research {
		dirs = append(dirs, d.dir)
	}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return false, err
		}
		for _, entry := range entries {
			if entry.IsDir() || entry.Name() == "note-kinds.json" || strings.HasSuffix(entry.Name(), ".output.schema.json") || (!strings.HasSuffix(entry.Name(), ".json") && !strings.HasSuffix(entry.Name(), ".py")) {
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

// List returns the repository contracts: the lib-callable set that coverage,
// preflight and nested calls use. Research programs are in Research.
func (i *Index) List() []Contract { return i.filter(false) }

// Research returns the research-home programs.
func (i *Index) Research() []Contract { return i.filter(true) }

func (i *Index) filter(research bool) []Contract {
	i.mu.RLock()
	defer i.mu.RUnlock()
	out := make([]Contract, 0, len(i.contracts))
	for _, c := range i.contracts {
		if c.Research == research {
			out = append(out, c)
		}
	}
	return out
}

// Lookup resolves a program name as callers write it: <scenario>.<program>
// or research/[<topic>/]<program>.
func (i *Index) Lookup(name string) (Contract, bool) {
	name = strings.TrimSpace(name)
	if researchhome.IsID(name) {
		i.mu.RLock()
		defer i.mu.RUnlock()
		for _, c := range i.contracts {
			if c.Research && c.ID == name {
				return c, true
			}
		}
		return Contract{}, false
	}
	scenario, program, ok := strings.Cut(name, ".")
	if !ok || scenario == "" || program == "" {
		return Contract{}, false
	}
	return i.Get(scenario, program)
}

func (i *Index) Get(scenario, name string) (Contract, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	for _, c := range i.contracts {
		if !c.Research && c.Scenario == scenario && c.Name == name {
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
		if contract.ValidationError != "" || contract.Research {
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
	LearningTask *LearningTask `json:"learning_task"`
	Learning     struct {
		NoteKinds    map[string]json.RawMessage `json:"note_kinds"`
		Capabilities []LearningCapability       `json:"capabilities"`
	} `json:"learning"`
	Memory      *Memory   `json:"memory"`
	Verbs       []string  `json:"verbs"`
	Fixtures    []Fixture `json:"fixtures"`
	Assumptions []string  `json:"assumptions"`
	Invariants  []string  `json:"invariants"`
	Outputs     struct {
		Signals map[string]string `json:"signals"`
	} `json:"outputs"`
	OutputSchema string                     `json:"output_schema"`
	Name         string                     `json:"name"`
	Version      string                     `json:"version"`
	Purpose      string                     `json:"purpose"`
	Inputs       map[string]json.RawMessage `json:"inputs"`
	Budget       struct {
		WallMS      int64  `json:"wall_ms"`
		OutputBytes int64  `json:"output_bytes"`
		Async       bool   `json:"async"`
		AsyncReason string `json:"async_reason"`
	} `json:"budget"`
	Bindings   []BindingRef `json:"bindings"`
	Rung       string       `json:"rung"`
	OwnerSkill string       `json:"owner_skill"`
}

func readContract(scenario, path string, schema *jsonschema.Schema) Contract {
	stem := strings.TrimSuffix(filepath.Base(path), ".json")
	separator := "."
	research := researchhome.IsID(scenario + "/")
	if research {
		separator = "/"
	}
	c := Contract{Scenario: scenario, SourcePath: path, ID: scenario + separator + stem, Name: stem, Research: research}
	data, err := os.ReadFile(path)
	if err != nil {
		c.ValidationError = err.Error()
		return c
	}
	c.Declaration = append([]byte{}, data...)
	sourcePath := strings.TrimSuffix(path, filepath.Ext(path)) + ".py"
	source, sourceErr := os.ReadFile(sourcePath)
	if sourceErr != nil {
		if os.IsNotExist(sourceErr) {
			c.SourceMissing = true
		} else {
			c.ValidationError = fmt.Sprintf("source file %s: %v", sourcePath, sourceErr)
		}
	} else {
		c.Source = string(source)
	}
	c.Digest = ContentDigest(data, source, nil)
	var raw rawContract
	if err := json.Unmarshal(data, &raw); err != nil {
		c.ValidationError = err.Error()
		return c
	}
	c.OutputBytes = raw.Budget.OutputBytes
	c.LearningTask = raw.LearningTask
	c.Memory = raw.Memory
	c.Verbs = append([]string(nil), raw.Verbs...)
	c.Fixtures = append([]Fixture(nil), raw.Fixtures...)
	c.Assumptions = append([]string(nil), raw.Assumptions...)
	c.Invariants = append([]string(nil), raw.Invariants...)
	c.Signals = raw.Outputs.Signals
	c.Bindings = append([]BindingRef(nil), raw.Bindings...)
	c.Async = raw.Budget.Async
	c.AsyncReason = raw.Budget.AsyncReason
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
		compiler.LoadURL = rejectExternalSchema
		if err = compiler.AddResource(c.OutputSchemaPath, bytes.NewReader(schemaBytes)); err == nil {
			c.OutputSchema, err = compiler.Compile(c.OutputSchemaPath)
		}
		if err != nil {
			c.ValidationError = err.Error()
			return c
		}
		c.OutputSchemaSource = append([]byte{}, schemaBytes...)
		c.Digest = ContentDigest(data, source, schemaBytes)
	}
	c.Name, c.Version, c.Purpose, c.Rung, c.OwnerSkill, c.WallMS = raw.Name, raw.Version, raw.Purpose, raw.Rung, raw.OwnerSkill, raw.Budget.WallMS
	if research && raw.Name != c.ID {
		c.ValidationError = fmt.Sprintf("contract name %q must be %q", raw.Name, c.ID)
	}
	if prefix := scenario + separator; strings.HasPrefix(c.Name, prefix) {
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
			FreeText bool            `json:"free_text"`
		}
		if err := json.Unmarshal(encoded, &spec); err != nil {
			c.ValidationError = fmt.Sprintf("input %s: %v", name, err)
			continue
		}
		c.Inputs[name] = InputSpec{Type: spec.Type, Required: spec.Required, Default: spec.Default, Enum: spec.Enum, FreeText: spec.FreeText}
	}
	sort.Strings(c.InputNames)
	c.Learning = learningSummary(c.Source, raw.Learning.NoteKinds, c.Inputs)
	c.Learning.Capabilities = raw.Learning.Capabilities
	if c.LearningTask != nil {
		for _, field := range c.LearningTask.ContextFields {
			if _, ok := c.Inputs[field]; !ok {
				c.ValidationError = fmt.Sprintf("learning context field %q is not a declared input", field)
			}
		}
		if scenario == "vrooli-memory" && (c.Name == "prepare-attempt" || c.Name == "finish-attempt" || c.Name == "run-task") {
			c.ValidationError = "learning infrastructure must not register a learning_task boundary"
		}
	}
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

var (
	learnCallPattern        = regexp.MustCompile(`\blearn\.([a-z][a-z0-9_]*)\s*\(`)
	explicitLearnKeyPattern = regexp.MustCompile(`\blearn\.task\s*\([^)]*\bkey\s*=`)
)

func learningSummary(source string, declaredNoteKinds map[string]json.RawMessage, inputs map[string]InputSpec) LearningSummary {
	seen := map[string]bool{}
	for _, match := range learnCallPattern.FindAllStringSubmatch(source, -1) {
		if len(match) == 2 {
			seen[match[1]] = true
		}
	}
	verbs := make([]string, 0, len(seen))
	for verb := range seen {
		verbs = append(verbs, verb)
	}
	sort.Strings(verbs)
	noteKinds := make([]string, 0, len(declaredNoteKinds))
	for kind := range declaredNoteKinds {
		noteKinds = append(noteKinds, kind)
	}
	sort.Strings(noteKinds)
	freeText := false
	for _, spec := range inputs {
		freeText = freeText || spec.FreeText
	}
	return LearningSummary{Verbs: verbs, UsesMemory: len(verbs) > 0, NoteKinds: noteKinds, FreeTextInputsWithoutKey: freeText && !explicitLearnKeyPattern.MatchString(source)}
}

func mustRead(path string) []byte { data, _ := os.ReadFile(path); return data }

// ContentDigest is shared by discovery and archived artifact verification.
func ContentDigest(declaration, source, schema []byte) string {
	h := sha256.New()
	h.Write(declaration)
	h.Write([]byte{0})
	h.Write(source)
	if len(schema) > 0 {
		h.Write([]byte{0})
		h.Write(schema)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (c *Contract) RestoreOutputSchema() error {
	if len(c.OutputSchemaSource) == 0 {
		return nil
	}
	compiler := jsonschema.NewCompiler()
	compiler.LoadURL = rejectExternalSchema
	const uri = "https://program-runtime.invalid/archived-output.schema.json"
	if err := compiler.AddResource(uri, bytes.NewReader(c.OutputSchemaSource)); err != nil {
		return err
	}
	schema, err := compiler.Compile(uri)
	if err == nil {
		c.OutputSchema = schema
	}
	return err
}

func rejectExternalSchema(uri string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("output schema must contain its references in one portable artifact: %s", uri)
}
