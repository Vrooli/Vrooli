package templatecontracts

// DestroyRequest selects a generated scenario to tear down.
type DestroyRequest struct {
	Name string
	// DryRun previews the full footprint without deleting anything.
	DryRun bool
	// ProtoOnly leaves the scenario directory alone and reaps only the shared
	// packages/proto footprint. This is how an already-deleted scenario's
	// stranded codegen gets cleaned up.
	ProtoOnly bool
	// Force is required to destroy a scenario whose directory still exists.
	Force bool
}

// DestroyResult is the outcome of a `template-manager lifecycle destroy` run.
type DestroyResult struct {
	Scenario  string `json:"scenario"`
	DryRun    bool   `json:"dryRun,omitempty"`
	ProtoOnly bool   `json:"protoOnly,omitempty"`
	// PathsRemoved lists everything deleted (or that would be, under DryRun).
	PathsRemoved []string `json:"pathsRemoved,omitempty"`
	// PathsAbsent lists footprint members that did not exist. Reported rather
	// than hidden so a partial teardown is visible instead of looking complete.
	PathsAbsent []string `json:"pathsAbsent,omitempty"`
	// NeedsProtoGenerate is true when codegen must re-run so the destroyed
	// surface leaves the descriptor sets.
	NeedsProtoGenerate bool   `json:"needsProtoGenerate,omitempty"`
	Message            string `json:"message,omitempty"`
}
