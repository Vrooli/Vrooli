package backlog

// KindMeta holds per-kind configuration. All code that needs kind-specific
// behaviour should read from KindConfig rather than hardcoding values.
type KindMeta struct {
	// Deliverable is the filename of the primary workshop output
	// ("plan.md" for most kinds, "conclusion.md" for research).
	Deliverable string
	// Dir is the on-disk directory name for items of this kind.
	Dir string
}

// KindConfig is the central registry of per-kind metadata.
// To customise a new kind, add an entry here — callers pick up changes automatically.
var KindConfig = map[BacklogKind]KindMeta{
	KindIdea:     {Deliverable: "plan.md", Dir: "ideas"},
	KindResearch: {Deliverable: "conclusion.md", Dir: "research"},
	KindFix:      {Deliverable: "plan.md", Dir: "fix"},
	KindExecute:  {Deliverable: "plan.md", Dir: "execute"},
	KindChore:    {Deliverable: "plan.md", Dir: "chore"},
}

// DeliverableForKind returns the deliverable filename for the given kind.
// Falls back to "plan.md" for unknown kinds.
func DeliverableForKind(kind BacklogKind) string {
	if meta, ok := KindConfig[kind]; ok {
		return meta.Deliverable
	}
	return "plan.md"
}
