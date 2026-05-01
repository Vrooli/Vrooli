package operatingmode

func itemLevelDefinition() Definition {
	return Definition{
		Mode:  ModeItemLevel,
		Label: "Item Level",
		Scope: ScopePolicy{Kind: ScopeBacklogItem},
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
