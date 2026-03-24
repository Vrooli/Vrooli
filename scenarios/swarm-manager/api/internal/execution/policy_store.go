package execution

import (
	"path/filepath"
	"strings"

	"swarm-manager/internal/pathutil"
	"swarm-manager/internal/storage"
)

// DOC: docs/reference/configuration.md
// DOC: docs/reference/operational-targets.md
// DOC: docs/internal/TEMPORAL-FLOWS.md

// PolicyStore persists execution policy.
type PolicyStore struct {
	path string
}

func defaultPolicy() Policy {
	return Policy{
		DefaultMode:         ModeManual,
		DefaultDelaySeconds: 300,
		MaxFixupAttempts:    2,
		AutoFixup:           false,
	}
}

func newPolicyStore(path string) *PolicyStore {
	if strings.TrimSpace(path) == "" {
		path = filepath.Join(pathutil.ResolveScenarioRoot("swarm-manager"), ".vrooli", "execution-policy.json")
	}
	return &PolicyStore{path: path}
}

func (s *PolicyStore) Load() (Policy, error) {
	var policy Policy
	exists, err := storage.ReadJSON(s.path, &policy)
	if err != nil {
		return Policy{}, err
	}
	if !exists {
		return defaultPolicy(), nil
	}
	return normalizePolicy(policy), nil
}

func (s *PolicyStore) Save(policy Policy) error {
	return storage.WriteJSONAtomic(s.path, normalizePolicy(policy))
}

func normalizePolicy(policy Policy) Policy {
	mode := normalizeMode(policy.DefaultMode)
	if mode == "" {
		mode = ModeManual
	}
	delay := policy.DefaultDelaySeconds
	if delay < 0 {
		delay = 0
	}
	maxFixup := policy.MaxFixupAttempts
	if maxFixup < 0 {
		maxFixup = 0
	}
	if maxFixup > 5 {
		maxFixup = 5
	}
	return Policy{
		DefaultMode:         mode,
		DefaultDelaySeconds: delay,
		MaxFixupAttempts:    maxFixup,
		AutoFixup:           policy.AutoFixup,
	}
}
