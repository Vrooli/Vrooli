package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"runtime"
	"sort"
)

// KeyInput contains every input that can change the meaning of unit evidence.
// A missing dimension is intentionally represented as an empty value and is
// still serialized, so callers cannot accidentally reuse an older key shape.
type KeyInput struct {
	SchemaVersion        string            `json:"schema_version"`
	SourceDigest         string            `json:"source_digest"`
	ConfigDigest         string            `json:"config_digest"`
	DependencyLockDigest string            `json:"dependency_lock_digest"`
	ToolchainIdentity    string            `json:"toolchain_identity"`
	AdapterID            string            `json:"adapter_id"`
	AdapterVersion       string            `json:"adapter_version"`
	PolicyDigest         string            `json:"policy_digest"`
	RunnerProfile        string            `json:"runner_profile"`
	OS                   string            `json:"os"`
	Architecture         string            `json:"architecture"`
	Environment          map[string]string `json:"environment,omitempty"`
	CoverageMode         string            `json:"coverage_mode"`
	ArtifactSchema       string            `json:"artifact_schema"`
}

type Key struct {
	Canonical []byte
	Digest    string
}

func NewKey(input KeyInput) (Key, error) {
	if input.SchemaVersion == "" {
		input.SchemaVersion = "unit-health.evidence.v1"
	}
	if input.OS == "" {
		input.OS = runtime.GOOS
	}
	if input.Architecture == "" {
		input.Architecture = runtime.GOARCH
	}
	input.Environment = canonicalEnvironment(input.Environment)
	canonical, err := json.Marshal(input)
	if err != nil {
		return Key{}, fmt.Errorf("evidence: canonicalize key: %w", err)
	}
	digest := sha256.Sum256(canonical)
	return Key{Canonical: canonical, Digest: hex.EncodeToString(digest[:])}, nil
}

func canonicalEnvironment(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make(map[string]string, len(in))
	for _, key := range keys {
		out[key] = in[key]
	}
	return out
}
