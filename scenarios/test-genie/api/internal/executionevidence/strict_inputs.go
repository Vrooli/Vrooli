package executionevidence

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/freshness-go/treedigest"
)

const StrictInputArtifactSchemaVersion = 1

// StrictInputArtifactMaxBytes is a defensive encoding ceiling, not evidence
// that every supported product closure fits. Oversize closures refuse proof.
const StrictInputArtifactMaxBytes int64 = 64 * 1024 * 1024

// StrictExecutionBinding is observation data. It cannot acquire custody,
// authorize a run, assert passed phases, or satisfy acceptance criteria.
type StrictExecutionBinding struct {
	RunID               string `json:"run_id"`
	Target              string `json:"target"`
	OwnerBindingDigest  string `json:"owner_binding_digest"`
	ExecutionEpoch      uint64 `json:"execution_epoch"`
	ConfigurationDigest string `json:"configuration_digest"`
	DescriptorDigest    string `json:"descriptor_digest"`
	PhaseSetDigest      string `json:"phase_set_digest"`
}

type strictInputPayload struct {
	SchemaVersion int                      `json:"schema_version"`
	Binding       StrictExecutionBinding   `json:"binding"`
	Manifest      treedigest.InputManifest `json:"manifest"`
}

// StrictInputArtifact owns a validated private snapshot. Accessors return
// copies; no writer, registration, or generated transport is composed here.
type StrictInputArtifact struct{ payload strictInputPayload }

func validStrictDigest(s string) bool {
	if len(s) != 71 || !strings.HasPrefix(s, "sha256:") {
		return false
	}
	raw, err := hex.DecodeString(s[7:])
	return err == nil && len(raw) == sha256.Size && strings.ToLower(s) == s
}
func (b StrictExecutionBinding) validate() error {
	if strings.TrimSpace(b.RunID) == "" || b.RunID != strings.TrimSpace(b.RunID) || strings.TrimSpace(b.Target) == "" || b.Target != strings.TrimSpace(b.Target) || b.ExecutionEpoch == 0 {
		return fmt.Errorf("strict execution identity and nonzero epoch required")
	}
	for _, d := range []string{b.OwnerBindingDigest, b.ConfigurationDigest, b.DescriptorDigest, b.PhaseSetDigest} {
		if !validStrictDigest(d) {
			return fmt.Errorf("invalid strict binding digest")
		}
	}
	return nil
}
func cloneStrictManifest(m treedigest.InputManifest) treedigest.InputManifest {
	roots := make([]treedigest.ManifestRoot, len(m.Roots))
	copy(roots, m.Roots)
	for i := range roots {
		roots[i].Files = append([]treedigest.ManifestFile(nil), m.Roots[i].Files...)
	}
	m.Roots = roots
	clone := func(in map[string]string) map[string]string {
		if in == nil {
			return nil
		}
		out := make(map[string]string, len(in))
		for k, v := range in {
			out[k] = v
		}
		return out
	}
	m.Configuration = clone(m.Configuration)
	m.Toolchain = clone(m.Toolchain)
	return m
}
func NewStrictInputArtifact(m treedigest.InputManifest, b StrictExecutionBinding) (*StrictInputArtifact, error) {
	if err := b.validate(); err != nil {
		return nil, err
	}
	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("strict manifest: %w", err)
	}
	return &StrictInputArtifact{payload: strictInputPayload{StrictInputArtifactSchemaVersion, b, cloneStrictManifest(m)}}, nil
}
func (a *StrictInputArtifact) Manifest() treedigest.InputManifest {
	if a == nil {
		return treedigest.InputManifest{}
	}
	return cloneStrictManifest(a.payload.Manifest)
}
func (a *StrictInputArtifact) Binding() StrictExecutionBinding {
	if a == nil {
		return StrictExecutionBinding{}
	}
	return a.payload.Binding
}
func (a *StrictInputArtifact) Encode(maxBytes int64) ([]byte, string, error) {
	if a == nil || maxBytes < 1 || maxBytes > StrictInputArtifactMaxBytes {
		return nil, "", fmt.Errorf("strict artifact and supported byte budget required")
	}
	if _, err := NewStrictInputArtifact(a.payload.Manifest, a.payload.Binding); err != nil {
		return nil, "", err
	}
	raw, err := json.Marshal(a.payload)
	if err != nil {
		return nil, "", err
	}
	if int64(len(raw)) > maxBytes {
		return nil, "", fmt.Errorf("complete strict artifact exceeds byte budget")
	}
	sum := sha256.Sum256(raw)
	return raw, "sha256:" + hex.EncodeToString(sum[:]), nil
}

// DecodeStrictInputArtifact binds exact immutable bytes to an installed expected
// execution identity. Expected metadata must come from the owner, never the
// offered artifact. Reader errors/cancellation/trailing or ambiguous JSON refuse.
func DecodeStrictInputArtifact(ctx context.Context, r io.Reader, maxBytes int64, expectedDigest string, expected StrictExecutionBinding) (*StrictInputArtifact, error) {
	if ctx == nil || r == nil || maxBytes < 1 || maxBytes > StrictInputArtifactMaxBytes || !validStrictDigest(expectedDigest) {
		return nil, fmt.Errorf("strict reader, context, digest and supported budget required")
	}
	if err := expected.validate(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	raw, err := io.ReadAll(io.LimitReader(&strictContextReader{ctx, r}, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if int64(len(raw)) > maxBytes {
		return nil, fmt.Errorf("strict artifact exceeds byte budget")
	}
	sum := sha256.Sum256(raw)
	if "sha256:"+hex.EncodeToString(sum[:]) != expectedDigest {
		return nil, fmt.Errorf("strict artifact byte digest mismatch")
	}
	if err := rejectDuplicateStrictJSON(raw); err != nil {
		return nil, err
	}
	var p strictInputPayload
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&p); err != nil {
		return nil, err
	}
	if d.Decode(new(any)) != io.EOF {
		return nil, fmt.Errorf("trailing strict artifact JSON")
	}
	if p.SchemaVersion != StrictInputArtifactSchemaVersion || p.Binding != expected {
		return nil, fmt.Errorf("strict artifact schema or execution binding mismatch")
	}
	return NewStrictInputArtifact(p.Manifest, p.Binding)
}

type strictContextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r *strictContextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}

// Exact field spelling is necessary because encoding/json otherwise treats
// struct field names case-insensitively. Scalar maps retain arbitrary unique keys.
func rejectDuplicateStrictJSON(raw []byte) error {
	fields := map[string]map[string]string{
		"artifact":    {"schema_version": "scalar", "binding": "binding", "manifest": "manifest"},
		"binding":     {"run_id": "scalar", "target": "scalar", "owner_binding_digest": "scalar", "execution_epoch": "scalar", "configuration_digest": "scalar", "descriptor_digest": "scalar", "phase_set_digest": "scalar"},
		"manifest":    {"schema_version": "scalar", "identity": "scalar", "roots": "roots", "configuration": "map", "toolchain": "map", "attribution": "attribution"},
		"root":        {"name": "scalar", "identity": "scalar", "files": "files"},
		"file":        {"path": "scalar", "digest": "scalar", "size": "scalar"},
		"attribution": {"commit": "scalar", "branch": "scalar", "dirty": "scalar"},
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	var value func(string) error
	value = func(kind string) error {
		token, err := d.Token()
		if err != nil {
			return err
		}
		delim, object := token.(json.Delim)
		if !object {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				token, err := d.Token()
				if err != nil {
					return err
				}
				key, ok := token.(string)
				if !ok || seen[key] {
					return fmt.Errorf("duplicate strict JSON key")
				}
				seen[key] = true
				child := "scalar"
				if kind != "map" {
					var allowed bool
					child, allowed = fields[kind][key]
					if !allowed {
						return fmt.Errorf("unsupported or incorrectly spelled strict JSON field %q", key)
					}
				}
				if err := value(child); err != nil {
					return err
				}
			}
		case '[':
			child := "scalar"
			if kind == "roots" {
				child = "root"
			}
			if kind == "files" {
				child = "file"
			}
			for d.More() {
				if err := value(child); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("invalid strict JSON delimiter")
		}
		_, err = d.Token()
		return err
	}
	if err := value("artifact"); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return fmt.Errorf("trailing strict JSON")
	}
	return nil
}
