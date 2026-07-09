package backlog

// KindMeta holds per-kind configuration. All code that needs kind-specific
// behaviour should read from KindConfig rather than hardcoding values.
type KindMeta struct {
	// Deliverable is the local primary workshop output. Non-research item plans
	// live in plan-manager and are referenced by plan_ref, so only research has
	// a local deliverable.
	Deliverable string
	// Dir is the on-disk directory name for items of this kind.
	Dir string
}

// KindConfig is the central registry of per-kind metadata.
// To customise a new kind, add an entry here — callers pick up changes automatically.
var KindConfig = map[BacklogKind]KindMeta{
	KindIdea:     {Dir: "ideas"},
	KindResearch: {Deliverable: "conclusion.md", Dir: "research"},
	KindFix:      {Dir: "fix"},
	KindExecute:  {Dir: "execute"},
	KindChore:    {Dir: "chore"},
}

// DeliverableForKind returns the deliverable filename for the given kind.
// Non-research items use plan_ref instead of a local deliverable.
func DeliverableForKind(kind BacklogKind) string {
	if meta, ok := KindConfig[kind]; ok {
		return meta.Deliverable
	}
	return ""
}
