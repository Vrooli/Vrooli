package modelpolicy

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"agent-manager/internal/domain"

	repocontract "github.com/vrooli/repo-contract-go"
)

// Revision is an immutable, validated catalog snapshot. Its digest is the
// SHA-256 of the exact source bytes, so operator-visible revisions correspond
// to a reviewable Git artifact rather than a re-marshaled approximation.
type Revision struct {
	digest  string
	catalog *Catalog
}

func (r *Revision) Digest() string {
	if r == nil {
		return ""
	}
	return r.digest
}

func (r *Revision) Catalog() *Catalog {
	if r == nil {
		return nil
	}
	return r.catalog.Clone()
}

func Parse(data []byte) (*Revision, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var catalog Catalog
	if err := decoder.Decode(&catalog); err != nil {
		return nil, domain.NewConfigInvalidError("modelPolicyCatalog", "failed to parse catalog: "+err.Error(), err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return nil, domain.NewConfigInvalidError("modelPolicyCatalog", "failed to parse catalog: "+err.Error(), err)
	}
	if err := catalog.Validate(); err != nil {
		return nil, err
	}

	sum := sha256.Sum256(data)
	return &Revision{
		digest:  "sha256:" + hex.EncodeToString(sum[:]),
		catalog: catalog.Clone(),
	}, nil
}

func Load(path string) (*Revision, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, domain.NewConfigInvalidError("modelPolicyCatalog", fmt.Sprintf("failed to read catalog at %s", path), err)
	}
	return Parse(data)
}

func ResolvePath() string {
	if path := strings.TrimSpace(os.Getenv("AGENT_MANAGER_MODEL_POLICY_CATALOG_PATH")); path != "" {
		return path
	}
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		root = "."
	}
	if resolved, err := repocontract.ResolveScenarioPath(root, "agent-manager"); err == nil {
		return filepath.Join(resolved, "config", "model-policy-catalog.json")
	}
	return filepath.Join(root, "scenarios", "agent-manager", "config", "model-policy-catalog.json")
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if err == io.EOF {
		return nil
	}
	if err == nil {
		return fmt.Errorf("catalog contains multiple JSON values")
	}
	return err
}
