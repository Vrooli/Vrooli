package operatingmode

func itemLevelDefinition() Definition {
	return Definition{
		Mode:        ModeItemLevel,
		Label:       "Item Level",
		Description: "Default mode. Each backlog item flows through the existing item execution pipeline; agents only read initiative state.",
		Scope:       ScopePolicy{Kind: ScopeBacklogItem},
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
