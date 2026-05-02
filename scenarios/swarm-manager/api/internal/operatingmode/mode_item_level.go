package operatingmode

func itemLevelDefinition() Definition {
	return Definition{
		Mode:        ModeItemLevel,
		Label:       "Item Level",
		Description: "Default mode. Each backlog item flows through the existing item execution pipeline; agents only read initiative state.",
		BestFor: []string{
			"Items are right-sized for one agent run each",
			"Items are loosely coupled and reviewable in isolation",
			"Items are stable — scope won't shift mid-execution",
			"Parallelism across many items is valuable",
		},
		NotFor: []string{
			"Items are coupled by a shared substrate they all change",
			"The natural unit of validation is the system as a whole",
			"Item shape will shift mid-execution as new ground truth emerges",
			"Intermediate states between items leave the system inconsistent",
		},
		Tradeoffs: []string{
			"Highest parallelism — many items in flight at once",
			"Bounded blast radius per item",
			"Per-item provenance and review surface",
			"Slowest when items aren't already well-shaped",
		},
		// item-level is the registered safe default; no recommended fallback.
		WhenInDoubtPickInstead: "",
		Scope:                  ScopePolicy{Kind: ScopeBacklogItem},
		RunStrategy: RunStrategyPolicy{
			Kind: RunStrategyExistingItemFlow,
		},
		Profile: ProfilePolicy{DefaultProfileKey: ProfileDefault},
		BacklogSync: BacklogSyncPolicy{
			Capabilities: []BacklogSyncCapability{BacklogSyncReadOnly},
			EventSource:  "item-level",
		},
		Metrics: MetricsPolicy{EventSource: "item-level"},
		UI:      UIPolicy{WorkspaceTabID: "info"},
	}
}
