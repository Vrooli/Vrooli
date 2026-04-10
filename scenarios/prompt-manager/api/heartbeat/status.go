package heartbeat

// Terminal run statuses returned by agent-manager.
//
// agent-manager serializes via protojson with UseProtoNames=true, so
// responses contain proto enum names (e.g. "RUN_STATUS_COMPLETE").
// Lowercase variants are included as a safety net for domain-style values.
//
// Source of truth: packages/proto/schemas/agent-manager/v1/domain/types.proto
// Domain values: scenarios/agent-manager/api/internal/domain/types.go
var terminalStatuses = map[string]bool{
	"RUN_STATUS_COMPLETE":  true,
	"RUN_STATUS_FAILED":    true,
	"RUN_STATUS_CANCELLED": true,
	"complete":             true,
	"failed":               true,
	"cancelled":            true,
}

// IsTerminalStatus reports whether s represents a finished run.
func IsTerminalStatus(s string) bool { return terminalStatuses[s] }

// IsFailedStatus reports whether s represents a failed run.
func IsFailedStatus(s string) bool {
	return s == "RUN_STATUS_FAILED" || s == "failed"
}

// IsCancelledStatus reports whether s represents a cancelled run.
func IsCancelledStatus(s string) bool {
	return s == "RUN_STATUS_CANCELLED" || s == "cancelled"
}
