package adapters

import (
	"fmt"
	"sort"
	"strings"
)

type Identity struct {
	ID      string
	Version string
}

type Match struct {
	Language  string
	Framework string
	Platform  string
}

type Adapter interface {
	Identity() Identity
	Matches(Match) bool
}

type Registry struct {
	byID map[string]map[string]Adapter
}

func NewRegistry() *Registry { return &Registry{byID: make(map[string]map[string]Adapter)} }

func (r *Registry) Register(adapter Adapter) error {
	if r == nil || adapter == nil {
		return fmt.Errorf("adapter registry: adapter is required")
	}
	identity := adapter.Identity()
	identity.ID = strings.TrimSpace(identity.ID)
	identity.Version = strings.TrimSpace(identity.Version)
	if identity.ID == "" || identity.Version == "" {
		return fmt.Errorf("adapter registry: identity id and version are required")
	}
	if r.byID == nil {
		r.byID = make(map[string]map[string]Adapter)
	}
	versions := r.byID[identity.ID]
	if versions == nil {
		versions = make(map[string]Adapter)
		r.byID[identity.ID] = versions
	}
	if _, exists := versions[identity.Version]; exists {
		return fmt.Errorf("adapter registry: duplicate adapter %s@%s", identity.ID, identity.Version)
	}
	versions[identity.Version] = adapter
	return nil
}

func (r *Registry) Resolve(identity Identity, match Match) (Adapter, error) {
	if r == nil {
		return nil, fmt.Errorf("adapter registry: registry is nil")
	}
	versions := r.byID[strings.TrimSpace(identity.ID)]
	if len(versions) == 0 {
		return nil, fmt.Errorf("adapter registry: unsupported adapter %s", identity.ID)
	}
	if strings.TrimSpace(identity.Version) != "" {
		adapter, ok := versions[strings.TrimSpace(identity.Version)]
		if !ok {
			return nil, fmt.Errorf("adapter registry: unsupported adapter %s@%s", identity.ID, identity.Version)
		}
		if !adapter.Matches(match) {
			return nil, fmt.Errorf("adapter registry: adapter %s@%s does not support requested match", identity.ID, identity.Version)
		}
		return adapter, nil
	}
	var matches []Adapter
	for _, adapter := range versions {
		if adapter.Matches(match) {
			matches = append(matches, adapter)
		}
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("adapter registry: no supported version for adapter %s", identity.ID)
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("adapter registry: ambiguous versions for adapter %s", identity.ID)
	}
	return matches[0], nil
}

func (r *Registry) Identities() []Identity {
	if r == nil {
		return nil
	}
	var out []Identity
	for id, versions := range r.byID {
		for version := range versions {
			out = append(out, Identity{ID: id, Version: version})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ID == out[j].ID {
			return out[i].Version < out[j].Version
		}
		return out[i].ID < out[j].ID
	})
	return out
}
