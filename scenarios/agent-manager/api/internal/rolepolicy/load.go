package rolepolicy

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

// Revision is an immutable validated catalog revision. Digest is calculated
// from exact source bytes so an execution snapshot can cite a reviewable
// declared-state artifact.
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
		return nil, domain.NewConfigInvalidError("rolePolicyCatalog", "failed to parse catalog: "+err.Error(), err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("catalog contains multiple JSON values")
		}
		return nil, domain.NewConfigInvalidError("rolePolicyCatalog", "failed to parse catalog: "+err.Error(), err)
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
		return nil, domain.NewConfigInvalidError("rolePolicyCatalog", fmt.Sprintf("failed to read catalog at %s", path), err)
	}
	return Parse(data)
}

// ResolvePath returns the one repository-owned role-policy location. Tests
// can override it without reading user-owned coding-agent configuration.
func ResolvePath() string {
	path, _ := os.LookupEnv("AGENT_MANAGER_ROLE_POLICY_CATALOG_PATH")
	if path = strings.TrimSpace(path); path != "" {
		return path
	}
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		root = "."
	}
	if resolved, err := repocontract.ResolveScenarioPath(root, "agent-manager"); err == nil {
		return filepath.Join(resolved, "config", "role-policy-catalog.json")
	}
	return filepath.Join(root, "scenarios", "agent-manager", "config", "role-policy-catalog.json")
}
