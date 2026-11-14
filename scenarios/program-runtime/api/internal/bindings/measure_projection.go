// Package bindings consumes explicit, owner-generated typed measure declarations.
package bindings

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	bindingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings"
	"program-runtime/internal/jsoncheck"
)

type runtimeMeasureProjection struct {
	Version       int                     `json:"version"`
	Scenario      string                  `json:"scenario"`
	Bindings      []runtimeMeasureBinding `json:"bindings"`
	SourceDigests map[string]string       `json:"source_digests"`
}

type runtimeMeasureBinding struct {
	ID           string   `json:"id"`
	Service      string   `json:"service"`
	Method       string   `json:"method"`
	Effect       string   `json:"effect"`
	RunEligible  bool     `json:"run_eligible"`
	Permissions  []string `json:"permissions"`
	PrimaryField string   `json:"primary_field"`
}

var (
	runtimeMeasureCommand = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)
	runtimeMeasureMethod  = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
)

func runtimeMeasureOwner(path string) string {
	generated := filepath.Dir(filepath.Clean(path))
	config := filepath.Dir(generated)
	if filepath.Base(generated) != "generated" || filepath.Base(config) != ".vrooli" {
		return ""
	}
	return filepath.Base(filepath.Dir(config))
}

// addRuntimeMeasureProjection stages the entire owner surface before adding
// any binding. A malformed or ambiguous projection cannot partially widen it.
func (r *Registry) addRuntimeMeasureProjection(path string, methods map[string][]methodInfo) error {
	owner := runtimeMeasureOwner(path)
	if owner == "" {
		return errors.New("invalid owner projection path")
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open projection: %w", err)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 1024*1024+1))
	if err != nil || len(data) > 1024*1024 {
		return errors.New("projection unreadable or too large")
	}
	if err := jsoncheck.UniqueKeys(data); err != nil {
		return fmt.Errorf("invalid projection: %w", err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var document runtimeMeasureProjection
	if err := decoder.Decode(&document); err != nil {
		return fmt.Errorf("invalid projection: %w", err)
	}
	if _, err := decoder.Token(); err != io.EOF {
		return errors.New("projection contains trailing data")
	}
	if document.Version != 1 || document.Scenario != owner || len(document.Bindings) > 256 {
		return errors.New("projection owner, version or binding count invalid")
	}
	if len(document.SourceDigests) != 1 {
		return errors.New("projection source fingerprint is unavailable or invalid")
	}
	const ownerSource = "api/internal/measures/measures.go"
	expected, ok := document.SourceDigests[ownerSource]
	if !ok || !strings.HasPrefix(expected, "sha256:") || len(expected) != 71 {
		return errors.New("projection source fingerprint is invalid")
	}
	if _, err := hex.DecodeString(expected[7:]); err != nil {
		return errors.New("projection source fingerprint is invalid")
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Clean(path))))
	sourcePath := root
	for _, part := range strings.Split(ownerSource, "/") {
		sourcePath = filepath.Join(sourcePath, part)
		info, err := os.Lstat(sourcePath)
		if err != nil {
			return fmt.Errorf("owner source fingerprint unavailable: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("owner source fingerprint unavailable: symlink")
		}
	}
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("owner source fingerprint unavailable: %w", err)
	}
	digest := sha256.Sum256(source)
	if "sha256:"+hex.EncodeToString(digest[:]) != expected {
		return errors.New("owner source fingerprint stale")
	}
	type stagedBinding struct {
		binding *bindingsv1.Binding
		info    methodInfo
	}
	staged := make([]stagedBinding, 0, len(document.Bindings))
	ids, methodKeys := map[string]bool{}, map[string]bool{}
	for _, item := range document.Bindings {
		prefix := owner + "/measures/"
		command := strings.TrimPrefix(item.ID, prefix)
		if !strings.HasPrefix(item.ID, prefix) || !runtimeMeasureCommand.MatchString(command) || len(command) > 128 || !strings.Contains(item.Service, ".") || !runtimeMeasureMethod.MatchString(item.Method) || item.Effect != "read" || !item.RunEligible || len(item.Permissions) != 1 || item.Permissions[0] != "network:internal" {
			return fmt.Errorf("invalid governed measure binding %q", item.ID)
		}
		key := item.Service + "." + item.Method
		if ids[item.ID] || methodKeys[key] {
			return fmt.Errorf("duplicate measure binding %q", item.ID)
		}
		ids[item.ID], methodKeys[key] = true, true
		if existing := r.byID[item.ID]; existing != nil {
			return fmt.Errorf("measure binding %q conflicts with CLI binding", item.ID)
		}
		info, err := resolveMethod(owner, key, methods, nil)
		if err != nil || !scenarioOwnsDescriptorSource(owner, info.source) {
			return fmt.Errorf("measure %q does not resolve uniquely to its owner descriptor", item.ID)
		}
		for _, existing := range r.bindings {
			existingInfo, found := r.methods[existing.GetId()]
			if found && existing.GetScenario() == owner && existingInfo.service == info.service && existing.GetMethod() == item.Method {
				return fmt.Errorf("measure binding %q conflicts with existing RPC", item.ID)
			}
		}
		if item.PrimaryField != "" {
			found := false
			for i := 0; i < info.output.Fields().Len(); i++ {
				field := info.output.Fields().Get(i)
				if normalizeField(string(field.Name())) == normalizeField(item.PrimaryField) || normalizeField(field.JSONName()) == normalizeField(item.PrimaryField) {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("measure %q primary field is not in its response descriptor", item.ID)
			}
		}
		projection := responseProjectionFor(info.output, item.PrimaryField)
		if len(projection.candidateFields) > 0 {
			return fmt.Errorf("measure %q response projection is ambiguous", item.ID)
		}
		binding := &bindingsv1.Binding{Id: item.ID, Scenario: owner, Group: "measures", Command: command, Service: item.Service, Method: item.Method, RequestType: string(info.input.FullName()), ResponseType: string(info.output.FullName()), Effect: "read", RunEligible: true, Permissions: append([]string(nil), item.Permissions...), Description: "Owner-governed typed measure.", Signature: fmt.Sprintf("%s.%s(%s) -> %s", item.Service, item.Method, info.input.FullName(), info.output.FullName()), RowsField: projection.rowsField, MetaFields: append([]string(nil), projection.metaFields...), RowFieldCandidates: append([]string(nil), projection.candidateFields...)}
		staged = append(staged, stagedBinding{binding, info})
	}
	for _, item := range staged {
		binding := item.binding
		r.bindings = append(r.bindings, binding)
		r.byID[binding.Id] = binding
		r.methods[binding.Id] = item.info
		r.required[binding.Id] = map[string]bool{}
		r.schemas[binding.Id] = cliapp.ArgSchema{}
		for _, key := range []string{normalizeField(binding.Command), normalizeField("measures/" + binding.Command), normalizeField(binding.Service + "." + binding.Method), normalizeField(binding.Id)} {
			r.operation[key] = append(r.operation[key], binding)
		}
	}
	// The CLI may omit the same RPC while the owner explicitly governs it as a
	// program capability. It is not an unbound RPC in that case.
	filtered := r.unbound[:0]
	for _, entry := range r.unbound {
		resolved := false
		if entry.GetScenario() == owner && entry.GetReason() == bindingsv1.UnboundReason_UNBOUND_REASON_OMITTED_RPC {
			for _, item := range staged {
				service := item.binding.Service
				short := service[strings.LastIndex(service, ".")+1:]
				if entry.GetMethod() == item.binding.Method && (entry.GetService() == short || entry.GetService() == service) {
					resolved = true
					break
				}
			}
		}
		if !resolved {
			filtered = append(filtered, entry)
		}
	}
	r.unbound = filtered
	return nil
}
