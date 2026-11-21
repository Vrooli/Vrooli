package backlog

// KindMeta holds per-kind configuration. All code that needs kind-specific
// behaviour should read from KindConfig rather than hardcoding values.
type KindMeta struct {
	// Deliverable is reserved for a kind-owned local output. Canonical plans for
	// every backlog kind, including research, live in plan-manager via plan_ref.
	Deliverable string
	// Dir is the on-disk directory name for items of this kind.
	Dir string
}

// KindConfig is the central registry of per-kind metadata.
// To customise a new kind, add an entry here — callers pick up changes automatically.
var KindConfig = map[BacklogKind]KindMeta{
	KindIdea:     {Dir: "ideas"},
	KindResearch: {Dir: "research"},
	KindFix:      {Dir: "fix"},
	KindExecute:  {Dir: "execute"},
	KindChore:    {Dir: "chore"},
}

// DeliverableForKind returns the deliverable filename for the given kind.
// All current kinds use plan_ref instead of a local deliverable.
func DeliverableForKind(kind BacklogKind) string {
	if meta, ok := KindConfig[kind]; ok {
		return meta.Deliverable
	}
	return ""
}
