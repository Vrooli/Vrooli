package storage

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

type LeverScope string

const (
	ScopeProcessTree LeverScope = "process-tree"
	ScopeSession     LeverScope = "session"
	ScopeDaemon      LeverScope = "daemon"
	ScopeContainer   LeverScope = "container"
)

type BlastRadius string

const (
	BlastNarrow BlastRadius = "narrow"
	BlastBroad  BlastRadius = "broad"
)

type Lever struct {
	Key    string
	Owner  string
	Entry  string
	Target string
	Scope  LeverScope
	Radius BlastRadius
}
type LeverWarning struct {
	Key     string
	Message string
}
type LeverRegistry struct {
	Levers   []Lever
	Warnings []LeverWarning
}

var broadLevers = map[string]string{"XDG_CACHE_HOME": "GOCACHE or a component-specific cache lever", "XDG_DATA_HOME": "a component-specific data lever", "XDG_STATE_HOME": "a component-specific state lever", "HOME": "a component-specific home lever", "TMPDIR": "a component-specific temp lever", "TEMP": "a component-specific temp lever"}

// BuildLeverRegistry validates relocation declarations before deployment planning.
func BuildLeverRegistry(levers []Lever, environment func(string) string) (LeverRegistry, error) {
	if environment == nil {
		environment = os.Getenv
	}
	copyLevers := append([]Lever(nil), levers...)
	sort.Slice(copyLevers, func(i, j int) bool {
		if copyLevers[i].Key != copyLevers[j].Key {
			return copyLevers[i].Key < copyLevers[j].Key
		}
		return copyLevers[i].Entry < copyLevers[j].Entry
	})
	seen := map[string]Lever{}
	result := LeverRegistry{Levers: copyLevers}
	for _, lever := range copyLevers {
		where := fmt.Sprintf("%s/%s", lever.Owner, lever.Entry)
		if strings.HasPrefix(lever.Key, "VROOLI_") {
			return LeverRegistry{}, fmt.Errorf("%s: relocation lever %q uses reserved VROOLI_ prefix", where, lever.Key)
		}
		if lever.Scope == "" {
			return LeverRegistry{}, fmt.Errorf("%s: relocation lever %q must declare scope", where, lever.Key)
		}
		if alternative, broad := broadLevers[lever.Key]; broad && lever.Radius != BlastBroad {
			return LeverRegistry{}, fmt.Errorf("%s: broad lever %q must be declared broad; prefer %s", where, lever.Key, alternative)
		}
		// Container-scoped variables are local to each owner container. The same
		// variable name can therefore safely point at a different target for two
		// resources. Host process-tree and daemon variables share an environment
		// and must still agree globally.
		if prior, exists := seen[lever.Key]; exists && prior.Target != lever.Target && (prior.Scope != ScopeContainer || lever.Scope != ScopeContainer) {
			return LeverRegistry{}, fmt.Errorf("%s: lever %q targets %q but %s targets %q", where, lever.Key, lever.Target, prior.Owner+"/"+prior.Entry, prior.Target)
		}
		seen[lever.Key] = lever
		if ambient := strings.TrimSpace(environment(lever.Key)); ambient != "" && ambient != lever.Target {
			result.Warnings = append(result.Warnings, LeverWarning{Key: lever.Key, Message: fmt.Sprintf("ambient value %q differs from declared target %q", ambient, lever.Target)})
		}
	}
	return result, nil
}
