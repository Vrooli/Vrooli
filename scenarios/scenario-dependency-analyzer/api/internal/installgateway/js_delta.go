package installgateway

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// LockDelta lists the locked packages (name@version) the staged change drops
// from and adds to the lockfile. It lets a preview show what a removal or an
// install really does to the closure, not just the direct dependency.
func (s *JSStage) LockDelta() (removed, added []string, err error) {
	before, err := s.lockedPackages(s.base)
	if err != nil {
		return nil, nil, err
	}
	after, err := s.lockedPackages(s.merged())
	if err != nil {
		return nil, nil, err
	}
	for key := range before {
		if !after[key] {
			removed = append(removed, key)
		}
	}
	for key := range after {
		if !before[key] {
			added = append(added, key)
		}
	}
	sort.Strings(removed)
	sort.Strings(added)
	return removed, added, nil
}

// LockedVersions reports, for each named package, the versions the lockfile
// will hold after publishing (sorted). A name the lock does not hold is absent.
func (s *JSStage) LockedVersions(names []string) (map[string][]string, error) {
	after, err := s.lockedPackages(s.merged())
	if err != nil {
		return nil, err
	}
	out := map[string][]string{}
	for _, name := range names {
		for key := range after {
			if version, ok := strings.CutPrefix(key, name+"@"); ok && version != "" {
				out[name] = append(out[name], version)
			}
		}
		sort.Strings(out[name])
	}
	return out, nil
}

// merged is the inputs as they will be after publishing.
func (s *JSStage) merged() map[string][]byte {
	out := make(map[string][]byte, len(s.base))
	for name, data := range s.base {
		out[name] = data
	}
	for name, data := range s.next {
		out[name] = data
	}
	return out
}

func (s *JSStage) lockedPackages(inputs map[string][]byte) (map[string]bool, error) {
	set := map[string]bool{}
	switch s.manager.name {
	case "pnpm":
		packages, err := readPnpmLockPackages(inputs["pnpm-lock.yaml"])
		if err != nil {
			return nil, err
		}
		for key := range packages {
			set[key] = true
		}
	case "npm":
		if len(inputs["package-lock.json"]) == 0 {
			return set, nil
		}
		var document struct {
			Packages map[string]npmLockEntry `json:"packages"`
		}
		if err := json.Unmarshal(inputs["package-lock.json"], &document); err != nil {
			return nil, fmt.Errorf("decode package-lock.json: %w", err)
		}
		for key, entry := range document.Packages {
			if key == "" || entry.Link {
				continue
			}
			name := entry.Name
			if name == "" {
				if idx := strings.LastIndex(key, "node_modules/"); idx >= 0 {
					name = key[idx+len("node_modules/"):]
				}
			}
			set[name+"@"+entry.Version] = true
		}
	}
	return set, nil
}
