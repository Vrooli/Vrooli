package platform

import "reflect"

// NativeOwnedPersistentTypes exposes the actual native receipt/codec roots to
// owning domain governance without exporting internal storage implementations.
func NativeOwnedPersistentTypes() []reflect.Type {
	return []reflect.Type{
		reflect.TypeFor[OperationIntent](), reflect.TypeFor[OperationRef](),
		reflect.TypeFor[OperationSpec](), reflect.TypeFor[OwnedServiceSpec](),
		reflect.TypeFor[darwinOwnedRecord](),
		reflect.TypeFor[windowsOwnedPayload](), reflect.TypeFor[windowsOwnedRootRecord](), reflect.TypeFor[windowsOwnedTerminalRecord](), reflect.TypeFor[windowsOwnedCustodyRecord](), reflect.TypeFor[nativeSystemBrokerDescriptor](),
		reflect.TypeFor[NativeLauncherProof](),
	}
}
func NativeOwnedPersistentFacts() map[string]any {
	return map[string]any{
		"ownedServiceKind":                        ownedServiceKind,
		"native_identity_version":                 2,
		"native_launcher_proof_version":           NativeLauncherProofVersion,
		"native_launcher_profile_openrgb":         NativeLauncherProfileOpenRGB,
		"native_limits_advisory":                  "native-advisory",
		"native_limits_processes":                 "native-process-limits",
		"windows_payload_version":                 2,
		"windows_payload_max_json_bytes":          windowsOwnedPayloadMaxJSONBytes,
		"darwin_ownership_record_version":         2,
		"windows_terminal_record_version":         1,
		"windows_custody_record_version":          1,
		"windows_custody_ready_value":             "CustodyReady",
		"windows_custody_ready_digest":            "sha256-exact-pending-json-after-RegFlushKey",
		"windows_submission_states":               []string{string(windowsSubmissionReserved), string(windowsSubmissionPrepared), string(windowsSubmissionClaimed)},
		"windows_owned_task_prefix":               windowsOwnedTaskPrefix,
		"windows_owned_run_issued_value":          windowsOwnedRunIssuedValue,
		"windows_owned_revoked_value":             windowsOwnedRevokedValue,
		"windows_owned_marker":                    windowsOwnedMarker,
		"windows_owned_task_namespace":            windowsTaskNamespace,
		"native_system_broker_descriptor_version": nativeSystemBrokerSchemaVersion,
		"NativeSystemSupervisorVerb":              NativeSystemSupervisorVerb,
	}
}
