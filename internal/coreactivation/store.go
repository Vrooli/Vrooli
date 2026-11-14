package coreactivation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/internal/config"
)

const maxRecordBytes = 64 << 10

const initializedContent = "vrooli-core-activation-store-v1\n"

type Store struct {
	home  string
	path  string
	write func(string, []byte, os.FileMode) error
}

func NewStore(home string) (*Store, error) {
	if !filepath.IsAbs(home) || filepath.Clean(home) != home {
		return nil, fmt.Errorf("core activation requires an absolute canonical home")
	}
	state, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return nil, err
	}
	return NewStoreAt(home, filepath.Join(state, "core-activation"))
}

// NewStoreAt opens the directory frozen from the canonical runtime-home
// contract at admission/enrollment. Native worker and recovery definitions must
// retain this directory explicitly: resolving a source contract during recovery
// would make a missing source tree disable the owner needed to restore it.
// Callers resolve home through the platform identity owner, not untrusted argv.
func NewStoreAt(home, directory string) (*Store, error) {
	if !filepath.IsAbs(home) || filepath.Clean(home) != home || !filepath.IsAbs(directory) || filepath.Clean(directory) != directory || filepath.Base(directory) != "core-activation" {
		return nil, fmt.Errorf("core activation requires canonical home and frozen store directory")
	}
	rel, err := filepath.Rel(home, directory)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("core activation store is outside its user home")
	}
	return &Store{home: home, path: filepath.Join(directory, "current.json"), write: config.WriteOwnedFileDurable}, nil
}

// Directory is metadata for a fixed native definition, not authority to select
// a different store at execution time.
func (s *Store) Directory() string { return filepath.Dir(s.path) }

func (s *Store) lock(ctx context.Context) (func(), error) {
	if ctx == nil {
		return nil, fmt.Errorf("core activation context is required")
	}
	return artifactlock.AcquireWithPathContext(ctx, "core-activation:"+s.home, func(string) (string, error) { return filepath.Join(filepath.Dir(s.path), "operation.lock"), nil })
}

// Current is an atomic metadata observation. It does not create an operation,
// repair a corrupt record or infer native process absence from missing state.
func (s *Store) Current() (Record, bool, error) {
	initialized, err := s.initialized()
	if err != nil {
		return Record{}, false, err
	}
	info, err := os.Lstat(s.path)
	if errors.Is(err, os.ErrNotExist) {
		if initialized {
			return Record{}, false, fmt.Errorf("core activation state is missing from an initialized store; owner recovery required")
		}
		return Record{}, false, nil
	}
	if err != nil {
		return Record{}, false, err
	}
	if !initialized {
		return Record{}, false, fmt.Errorf("core activation initialization witness is missing")
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxRecordBytes {
		return Record{}, false, fmt.Errorf("invalid bounded core activation record")
	}
	f, err := os.Open(s.path)
	if err != nil {
		return Record{}, false, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return Record{}, false, fmt.Errorf("core activation record changed while opening")
	}
	data, err := io.ReadAll(io.LimitReader(f, maxRecordBytes+1))
	if err != nil || len(data) > maxRecordBytes {
		return Record{}, false, fmt.Errorf("core activation record exceeded read bounds")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err := validateJSONShape(d, reflect.TypeFor[Record](), 0); err != nil {
		return Record{}, false, err
	}
	if _, err := d.Token(); err != io.EOF {
		return Record{}, false, fmt.Errorf("trailing core activation record data")
	}
	var r Record
	if err := json.Unmarshal(data, &r); err != nil {
		return Record{}, false, fmt.Errorf("invalid core activation record values")
	}
	if err := r.validate(s.home); err != nil {
		return Record{}, false, err
	}
	return r, true, nil
}

func (s *Store) put(r *Record) error {
	next := *r
	next.Revision++
	if err := next.validate(s.home); err != nil {
		return err
	}
	data, err := json.Marshal(next)
	if err != nil {
		return err
	}
	if len(data) > maxRecordBytes {
		return fmt.Errorf("core activation record exceeds write bounds")
	}
	initialized, err := s.initialized()
	if err != nil {
		return err
	}
	if !initialized {
		// This witness precedes first admission. A crash in between is an
		// explicit incomplete store, never permission for fresh dispatch.
		if err := config.WriteOwnedFileDurable(s.initializedPath(), []byte(initializedContent), 0o600); err != nil {
			return fmt.Errorf("core activation initialization is unresolved: %w", err)
		}
	}
	if err := s.write(s.path, data, 0o600); err != nil {
		return fmt.Errorf("core activation commit may be visible; reconcile the same operation: %w", err)
	}
	*r = next
	return nil
}

func (s *Store) initializedPath() string { return filepath.Join(filepath.Dir(s.path), "initialized") }

func (s *Store) initialized() (bool, error) {
	path := s.initializedPath()
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil || !info.Mode().IsRegular() || info.Size() != int64(len(initializedContent)) {
		return false, fmt.Errorf("invalid core activation initialization witness")
	}
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return false, fmt.Errorf("core activation initialization witness changed")
	}
	data, err := io.ReadAll(io.LimitReader(f, int64(len(initializedContent)+1)))
	if err != nil || string(data) != initializedContent {
		return false, fmt.Errorf("invalid core activation initialization witness")
	}
	return true, nil
}

// Exact schema names apply to structs; map keys retain their own semantics.
// This domain's record has no open maps or custom JSON types. Missing required,
// duplicate, unknown and case-alias fields are rejected recursively.
func validateJSONShape(d *json.Decoder, t reflect.Type, depth int) error {
	if depth > 32 {
		return fmt.Errorf("core activation record nesting exceeded")
	}
	optional := t.Kind() == reflect.Pointer
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	token, err := d.Token()
	if err != nil {
		return fmt.Errorf("invalid core activation JSON")
	}
	if token == nil {
		if optional || t.Kind() == reflect.Slice {
			return nil
		}
		return fmt.Errorf("null required core activation field")
	}
	if t.Kind() == reflect.Struct {
		if token != json.Delim('{') {
			return fmt.Errorf("invalid core activation object")
		}
		fields := map[string]reflect.StructField{}
		required := map[string]bool{}
		for _, f := range reflect.VisibleFields(t) {
			if !f.IsExported() {
				continue
			}
			tag := strings.Split(f.Tag.Get("json"), ",")
			name := tag[0]
			if name == "-" {
				continue
			}
			if name == "" {
				name = f.Name
			}
			fields[name] = f
			required[name] = !strings.Contains(f.Tag.Get("json"), ",omitempty")
		}
		seen := map[string]bool{}
		for d.More() {
			token, err := d.Token()
			if err != nil {
				return fmt.Errorf("invalid core activation field")
			}
			name, ok := token.(string)
			field, exists := fields[name]
			if !ok || !exists || seen[name] {
				return fmt.Errorf("unknown or duplicate core activation field")
			}
			seen[name] = true
			if err := validateJSONShape(d, field.Type, depth+1); err != nil {
				return err
			}
		}
		for name, needed := range required {
			if needed && !seen[name] {
				return fmt.Errorf("missing required core activation field")
			}
		}
		end, err := d.Token()
		if err != nil || end != json.Delim('}') {
			return fmt.Errorf("invalid core activation object end")
		}
		return nil
	}
	if t.Kind() == reflect.Slice {
		if token != json.Delim('[') {
			return fmt.Errorf("invalid core activation array")
		}
		count := 0
		for d.More() {
			count++
			if count > 32 {
				return fmt.Errorf("core activation array bound exceeded")
			}
			if err := validateJSONShape(d, t.Elem(), depth+1); err != nil {
				return err
			}
		}
		end, err := d.Token()
		if err != nil || end != json.Delim(']') {
			return fmt.Errorf("invalid core activation array end")
		}
		return nil
	}
	if _, compound := token.(json.Delim); compound {
		return fmt.Errorf("compound value in scalar core activation field")
	}
	return nil
}
