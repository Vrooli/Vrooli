// Package coreset reads the operator-granted core-set authority. The lists are
// manifest/operator data, not code-owned instance knowledge.
package coreset

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Authority struct {
	Seed        []string `json:"seed"`
	TrustedBase []string `json:"trusted_base"`
}

// Validate checks the structural invariants of operator-granted core
// authority. A trusted-base grant cannot name a scenario outside the seed
// authority: doing so would grant protection to an object that is not part of
// the operator's declared core.
func (a Authority) Validate() error {
	seed := normalizeSorted(a.Seed)
	trusted := normalizeSorted(a.TrustedBase)
	if len(seed) == 0 || len(trusted) == 0 {
		return os.ErrInvalid
	}
	seedSet := make(map[string]struct{}, len(seed))
	for _, name := range seed {
		seedSet[name] = struct{}{}
	}
	for _, name := range trusted {
		if _, ok := seedSet[name]; !ok {
			return fmt.Errorf("trusted-base member %q is not a core seed", name)
		}
	}
	return nil
}

type operatorState struct {
	Core Authority `json:"core"`
}

// Load reads operator-state.json from repoRoot. It returns an error instead of
// fabricating a fallback authority when the operator has not granted one.
func Load(repoRoot string) (Authority, error) {
	path := filepath.Join(strings.TrimSpace(repoRoot), ".vrooli", "operator-state.json")
	raw, err := os.ReadFile(path) // #nosec G304 -- repoRoot is a control-plane workspace path.
	if err != nil {
		return Authority{}, err
	}
	var state operatorState
	if err := json.Unmarshal(raw, &state); err != nil {
		return Authority{}, err
	}
	state.Core.Seed = normalizeSorted(state.Core.Seed)
	state.Core.TrustedBase = normalizeSorted(state.Core.TrustedBase)
	if err := state.Core.Validate(); err != nil {
		return Authority{}, err
	}
	return state.Core, nil
}

func currentAuthority() Authority {
	if root := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); root != "" {
		if authority, err := Load(root); err == nil {
			return authority
		}
	}
	dir, err := os.Getwd()
	if err != nil {
		return Authority{}
	}
	for {
		if authority, err := Load(dir); err == nil {
			return authority
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return Authority{}
		}
		dir = parent
	}
}

func CoreSeedScenarios() []string { return append([]string(nil), currentAuthority().Seed...) }

func TrustedBaseScenarios() []string { return append([]string(nil), currentAuthority().TrustedBase...) }

func DefaultFallbackCoreSet() []string { return CoreSeedScenarios() }

func IsCoreSeed(name string) bool { return contains(currentAuthority().Seed, name) }

func IsTrustedBase(name string) bool { return contains(currentAuthority().TrustedBase, name) }

func normalizeSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" {
			seen[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func contains(values []string, name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, value := range values {
		if value == name {
			return true
		}
	}
	return false
}
