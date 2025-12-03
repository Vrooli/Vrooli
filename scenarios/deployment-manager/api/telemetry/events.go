// Package telemetry provides deployment telemetry ingestion and classification.
package telemetry

// Telemetry event constants define the types of events recorded during deployment.
// These events are emitted by the runtime and ingested by deployment-manager
// for monitoring and debugging deployment health.
const (
	// EventAppStart is emitted when the bundled application starts.
	EventAppStart = "app_start"

	// EventReady is emitted when the application becomes ready to serve.
	EventReady = "ready"

	// EventShutdown is emitted when the application shuts down.
	EventShutdown = "shutdown"

	// EventRestart is emitted when a service is restarted.
	EventRestart = "restart"

	// EventDependencyUnreachable is emitted when a required dependency
	// cannot be reached (e.g., external service timeout).
	EventDependencyUnreachable = "dependency_unreachable"

	// EventSwapMissingAsset is emitted when a swap resource is missing
	// required assets that should have been bundled.
	EventSwapMissingAsset = "swap_missing_asset"

	// EventAssetMissing is emitted when a required bundled asset
	// is missing or corrupted.
	EventAssetMissing = "asset_missing"

	// EventMigrationFailed is emitted when a database migration fails.
	EventMigrationFailed = "migration_failed"

	// EventAPIUnreachable is emitted when the runtime control API
	// cannot be reached.
	EventAPIUnreachable = "api_unreachable"

	// EventSecretsMissing is emitted when required secrets are not
	// available at startup.
	EventSecretsMissing = "secrets_missing"

	// EventHealthFailed is emitted when a service health check fails.
	EventHealthFailed = "health_failed"
)

// failureEvents is the set of telemetry events that indicate a failure condition.
// These events are highlighted in the telemetry UI and tracked for failure analysis.
var failureEvents = map[string]bool{
	EventDependencyUnreachable: true,
	EventSwapMissingAsset:      true,
	EventAssetMissing:          true,
	EventMigrationFailed:       true,
	EventAPIUnreachable:        true,
	EventSecretsMissing:        true,
	EventHealthFailed:          true,
}

// IsFailureEvent decides whether a telemetry event type indicates a failure.
// Failure events are tracked separately for debugging and alerting purposes.
//
// Decision criteria:
// - Events that prevent the application from becoming ready
// - Events that indicate broken dependencies or missing resources
// - Events that require operator intervention to resolve
func IsFailureEvent(event string) bool {
	return failureEvents[event]
}

// GetAllFailureEventTypes returns the list of all failure event types.
// Useful for UI filtering and documentation.
func GetAllFailureEventTypes() []string {
	types := make([]string, 0, len(failureEvents))
	for event := range failureEvents {
		types = append(types, event)
	}
	return types
}

// lifecycleEvents are events that track normal application lifecycle.
var lifecycleEvents = map[string]bool{
	EventAppStart: true,
	EventReady:    true,
	EventShutdown: true,
	EventRestart:  true,
}

// IsLifecycleEvent decides whether a telemetry event type is a lifecycle event.
// Lifecycle events track normal application state transitions.
func IsLifecycleEvent(event string) bool {
	return lifecycleEvents[event]
}

// ContentType constants for parsing request bodies.
const (
	ContentTypeJSON  = "application/json"
	ContentTypeJSONL = "application/x-jsonl"
)

// IsJSONLContentType decides whether a content type header indicates JSONL format.
// JSONL (JSON Lines) uses one JSON object per line without array wrapper.
func IsJSONLContentType(contentType string) bool {
	return contentType == ContentTypeJSONL ||
		(contentType != "" && contentType != ContentTypeJSON &&
			// Also check for explicit jsonl in content type
			len(contentType) >= 5 && contentType[len(contentType)-5:] == "jsonl")
}
