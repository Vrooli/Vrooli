package retention

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/vrooli/api-core/storage"
)

// The manifest block is parsed exactly once, here. durable_data has no shared Go
// model and two consumers each hand-rolled a local mirror of the same JSON
// shape; the mirrors then drifted. This package owns the parse so consumers use
// the typed result instead of re-deriving it.

// TargetKind names what a budget bounds.
type TargetKind string

const (
	// TargetSQLiteTable bounds rows in one table of one SQLite database.
	TargetSQLiteTable TargetKind = "sqlite_table"
	// TargetDirectory bounds the top-level entries of one directory.
	TargetDirectory TargetKind = "directory"
	// TargetFile bounds one regenerable file. It is removed atomically when
	// its declared age or byte ceiling is exceeded.
	TargetFile TargetKind = "file"
)

// PrunerMode names who selects what dies.
type PrunerMode string

const (
	// PrunerBuiltin uses the framework pruner for the target kind. A component
	// declaring this writes no Go code.
	PrunerBuiltin PrunerMode = "builtin"
	// PrunerCustom requires the component to register a Pruner under the budget
	// name.
	PrunerCustom PrunerMode = "custom"
)

// Target describes what a budget bounds and where it lives. Paths are relative
// to Class's root and are resolved through api-core/storage, so a shadow
// variant prunes its own data and never live's.
type Target struct {
	// Kind selects which pruner shape applies.
	Kind TargetKind
	// Class is the storage class root the paths below are relative to.
	Class storage.Class
	// Database is the SQLite database path, for TargetSQLiteTable.
	Database string
	// Table is the bounded table, for TargetSQLiteTable.
	Table string
	// TimeColumn carries each row's age; it orders deletes oldest-first and is
	// what MaxAge is evaluated against. For TargetSQLiteTable.
	TimeColumn string
	// Path is the bounded directory, for TargetDirectory.
	Path string
}

// RelPath returns the class-relative path this target occupies.
func (t Target) RelPath() string {
	switch t.Kind {
	case TargetSQLiteTable:
		return t.Database
	case TargetDirectory:
		return t.Path
	case TargetFile:
		return t.Path
	default:
		return ""
	}
}

// Resolve returns the absolute on-disk path of the target, joined safely under
// the resolved class root for opts.
func (t Target) Resolve(resolver *storage.Resolver, opts storage.Options) (string, error) {
	if resolver == nil {
		return "", fmt.Errorf("%w: %s target needs a storage resolver", ErrInvalidTarget, t.Kind)
	}
	rel := t.RelPath()
	if rel == "" {
		return "", fmt.Errorf("%w: %s target declares no path", ErrInvalidTarget, t.Kind)
	}
	return resolver.Path(opts, t.Class, rel)
}

// Spec is one fully parsed budget declaration: the enforceable bounds, what they
// bound, and who prunes.
type Spec struct {
	// Budget carries the declared bounds.
	Budget Budget
	// Target is what the budget bounds.
	Target Target
	// Mode selects the builtin pruner or a registered custom one.
	Mode PrunerMode
	// Rationale is the author's stated reason for this ceiling, carried through
	// to findings so whoever reads one knows what drives the volume.
	Rationale string
}

// rawManifest is the wire shape of the retention block. It mirrors
// common.schema.json#/definitions/retention exactly.
type rawManifest struct {
	Retention *struct {
		Budgets map[string]rawBudget `json:"budgets"`
	} `json:"retention"`
	Storage *struct {
		Entries map[string]rawStorageEntry `json:"entries"`
	} `json:"storage"`
}

type rawStorageEntry struct {
	Rung    string          `json:"rung"`
	Path    json.RawMessage `json:"path"`
	Kind    string          `json:"kind"`
	Class   string          `json:"class"`
	Subpath string          `json:"subpath"`
	Budget  *struct {
		MaxAge    string `json:"max_age"`
		MaxBytes  string `json:"max_bytes"`
		Rationale string `json:"rationale"`
	} `json:"budget"`
	Rationale string `json:"rationale"`
}

type rawBudget struct {
	Target struct {
		Kind       string `json:"kind"`
		Class      string `json:"class"`
		Database   string `json:"database"`
		Table      string `json:"table"`
		TimeColumn string `json:"time_column"`
		Path       string `json:"path"`
	} `json:"target"`
	MaxAge    string `json:"max_age"`
	MaxBytes  string `json:"max_bytes"`
	Pruner    string `json:"pruner"`
	Rationale string `json:"rationale"`
}

// ParseManifest reads a component manifest and returns its declared budgets,
// ordered by name so cycles and logs are deterministic.
//
// A manifest with no retention block yields no specs and no error: declaring a
// budget is not yet mandatory, and a component that declares nothing must keep
// working unchanged.
func ParseManifest(data []byte) ([]Spec, error) {
	var raw rawManifest
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse retention manifest: %w", err)
	}
	legacyCount := 0
	if raw.Retention != nil {
		legacyCount = len(raw.Retention.Budgets)
	}
	storageCount := 0
	if raw.Storage != nil {
		storageCount = len(raw.Storage.Entries)
	}
	if legacyCount == 0 && storageCount == 0 {
		return nil, nil
	}

	names := make([]string, 0, legacyCount+storageCount)
	if raw.Retention != nil {
		for name := range raw.Retention.Budgets {
			names = append(names, name)
		}
	}
	if raw.Storage != nil {
		for name := range raw.Storage.Entries {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	specs := make([]Spec, 0, len(names))
	paths := make(map[string]string, len(names))
	for _, name := range names {
		var spec Spec
		var err error
		if raw.Storage != nil {
			if entry, ok := raw.Storage.Entries[name]; ok {
				if entry.Budget == nil || entry.Kind != "dir" {
					continue
				}
				spec, err = parseStorageEntry(name, entry)
			} else {
				spec, err = parseBudget(name, raw.Retention.Budgets[name])
			}
		} else {
			spec, err = parseBudget(name, raw.Retention.Budgets[name])
		}
		if err != nil {
			return nil, err
		}
		if spec.Target.Kind == "" {
			// Explicit by-OS and host-owned paths remain governed by the
			// storage-manager entry enforcer; the class-root retention engine
			// cannot safely reinterpret them as a canonical class path.
			continue
		}
		key := targetConflictKey(spec.Target)
		if previous, exists := paths[key]; exists {
			return nil, fmt.Errorf("storage declaration %q conflicts with %q at path %q", name, previous, spec.Target.RelPath())
		}
		paths[key] = name
		specs = append(specs, spec)
	}
	return specs, nil
}

func targetConflictKey(target Target) string {
	if target.Kind == TargetSQLiteTable {
		return string(target.Kind) + ":" + string(target.Class) + ":" + target.RelPath() + ":" + target.Table
	}
	return string(target.Kind) + ":" + string(target.Class) + ":" + target.RelPath()
}

func parseStorageEntry(name string, raw rawStorageEntry) (Spec, error) {
	if raw.Budget == nil {
		return Spec{}, fmt.Errorf("storage entry %q has no retention budget", name)
	}
	if raw.Kind != "dir" {
		return Spec{}, fmt.Errorf("storage entry %q: only dir entries can use builtin retention", name)
	}
	var path string
	if len(raw.Path) > 0 && string(raw.Path) != "null" {
		if err := json.Unmarshal(raw.Path, &path); err != nil {
			// A by-OS path is an intentional host-managed exception. It is
			// still visible to storage-manager's resolver/enforcer, but it is
			// not a canonical class-root target for this package.
			return Spec{}, nil
		}
	} else {
		path = raw.Subpath
	}
	if path == "" {
		path = "."
	}
	if strings.HasPrefix(path, "/") || strings.Contains(path, "..") {
		return Spec{}, nil
	}
	class := raw.Class
	if class == "" {
		class = "data"
	}
	spec, err := parseBudget(name, rawBudget{
		Target: struct {
			Kind       string `json:"kind"`
			Class      string `json:"class"`
			Database   string `json:"database"`
			Table      string `json:"table"`
			TimeColumn string `json:"time_column"`
			Path       string `json:"path"`
		}{Kind: "directory", Class: class, Path: path},
		MaxAge: raw.Budget.MaxAge, MaxBytes: raw.Budget.MaxBytes, Pruner: "builtin", Rationale: raw.Budget.Rationale,
	})
	if err != nil {
		return Spec{}, err
	}
	if spec.Rationale == "" {
		spec.Rationale = strings.TrimSpace(raw.Rationale)
	}
	return spec, nil
}

func parseBudget(name string, raw rawBudget) (Spec, error) {
	spec := Spec{
		Budget:    Budget{Name: name},
		Rationale: strings.TrimSpace(raw.Rationale),
	}

	if raw.MaxAge != "" {
		age, err := ParseAge(raw.MaxAge)
		if err != nil {
			return Spec{}, fmt.Errorf("budget %q max_age: %w", name, err)
		}
		spec.Budget.MaxAge = age
	}
	if raw.MaxBytes != "" {
		size, err := ParseBytes(raw.MaxBytes)
		if err != nil {
			return Spec{}, fmt.Errorf("budget %q max_bytes: %w", name, err)
		}
		spec.Budget.MaxBytes = size
	}
	if !spec.Budget.HasAgeBound() && !spec.Budget.HasByteBound() {
		return Spec{}, fmt.Errorf("budget %q: %w", name, ErrNoBound)
	}

	mode, err := parsePrunerMode(raw.Pruner)
	if err != nil {
		return Spec{}, fmt.Errorf("budget %q: %w", name, err)
	}
	spec.Mode = mode

	target, err := parseTarget(raw)
	if err != nil {
		return Spec{}, fmt.Errorf("budget %q: %w", name, err)
	}
	spec.Target = target

	return spec, nil
}

// parsePrunerMode defaults to builtin so the common case declares nothing extra.
func parsePrunerMode(s string) (PrunerMode, error) {
	switch strings.TrimSpace(s) {
	case "", string(PrunerBuiltin):
		return PrunerBuiltin, nil
	case string(PrunerCustom):
		return PrunerCustom, nil
	default:
		return "", fmt.Errorf("pruner %q is not builtin or custom", s)
	}
}

// parseStorageClass defaults to the data class, where primary mutable
// application state lives and where unbounded growth actually happens.
func parseStorageClass(s string) (storage.Class, error) {
	switch storage.Class(strings.TrimSpace(s)) {
	case "":
		return storage.ClassData, nil
	case storage.ClassConfig:
		return storage.ClassConfig, nil
	case storage.ClassData:
		return storage.ClassData, nil
	case storage.ClassCache:
		return storage.ClassCache, nil
	case storage.ClassLogs:
		return storage.ClassLogs, nil
	case storage.ClassState:
		return storage.ClassState, nil
	default:
		return "", fmt.Errorf("%w: unknown storage class %q", ErrInvalidTarget, s)
	}
}

func parseTarget(raw rawBudget) (Target, error) {
	class, err := parseStorageClass(raw.Target.Class)
	if err != nil {
		return Target{}, err
	}
	target := Target{Class: class}

	switch TargetKind(strings.TrimSpace(raw.Target.Kind)) {
	case TargetSQLiteTable:
		target.Kind = TargetSQLiteTable
		target.Database = strings.TrimSpace(raw.Target.Database)
		target.Table = strings.TrimSpace(raw.Target.Table)
		target.TimeColumn = strings.TrimSpace(raw.Target.TimeColumn)
		if target.Database == "" || target.Table == "" || target.TimeColumn == "" {
			return Target{}, fmt.Errorf("%w: sqlite_table needs database, table, and time_column", ErrInvalidTarget)
		}
	case TargetDirectory:
		target.Kind = TargetDirectory
		target.Path = strings.TrimSpace(raw.Target.Path)
		if target.Path == "" {
			return Target{}, fmt.Errorf("%w: directory needs path", ErrInvalidTarget)
		}
	case TargetFile:
		target.Kind = TargetFile
		target.Path = strings.TrimSpace(raw.Target.Path)
		if target.Path == "" {
			return Target{}, fmt.Errorf("%w: file needs path", ErrInvalidTarget)
		}
	default:
		return Target{}, fmt.Errorf("%w: unknown kind %q", ErrInvalidTarget, raw.Target.Kind)
	}
	return target, nil
}
