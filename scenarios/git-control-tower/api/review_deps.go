package main

import "context"

// ReviewTidinessClient is the subset of TidinessManagerClient used by review handlers.
type ReviewTidinessClient interface {
	GetTidinessScore(ctx context.Context, scenario string) (*TidinessScoreResponse, error)
	GetStaleness(ctx context.Context, scenario string) (*TidinessStalenessInfo, error)
	TriggerLightScan(ctx context.Context, req TidinessLightScanRequest) (*TidinessLightScanResult, error)
}

// ReviewTestGenieClient is the subset of TestGenieClient used by review handlers.
type ReviewTestGenieClient interface {
	ListExecutions(ctx context.Context, scenario string, limit int) (*TestExecutionListResponse, error)
	ExecuteSuite(ctx context.Context, req TestExecutionRequest) (*TestExecutionResult, error)
}

// ReviewAuditorClient is the subset of AuditorClient used by review handlers.
type ReviewAuditorClient interface {
	GetViolations(ctx context.Context, scenarioName string) (*AuditorViolationsResponse, error)
	StartCheck(ctx context.Context, scenarioName, checkType string) (*AuditorCheckJobResponse, error)
}

// ReviewVisualStorage is the subset of VisualCaptureStorage used by review handlers.
type ReviewVisualStorage interface {
	ListSnapshotSets(repoID int64, scenarioSlug string) ([]SnapshotSetMeta, error)
}

// ReviewCapabilities is the subset of CapabilityRegistry used by review handlers.
type ReviewCapabilities interface {
	IsAvailable(ctx context.Context, capabilityID string) bool
}
