package buildinfo

import "github.com/vrooli/vrooli/internal/rootmetrics"

type rebuildObservation interface {
	Async()
	Event(string, string)
	Flush()
}

var beginRebuildObservation = func(destination string) rebuildObservation {
	return BeginDirectRootInstall(destination)
}

// BeginDirectRootInstall observes only the canonical root target. Other CLI
// publications and rebaseline-owned publications are separate operations.
func BeginDirectRootInstall(destination string) *rootmetrics.Attempt {
	home, canonical, err := canonicalRootTarget(destination)
	if err != nil || !canonical {
		return nil
	}
	a := rootmetrics.Begin(home, "install")
	return a
}
