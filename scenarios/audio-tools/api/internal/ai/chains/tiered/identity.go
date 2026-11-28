package tiered

// StampIdentity applies the shared tier/provider/model decoration every
// capability chain performs on a provider response. Result types stay
// capability-owned: callers pass the addresses of the three common fields so
// the helper can fill the provider id and model only when the provider did not
// report them.
//
// overwriteProvider preserves the one intentional difference between tiers:
// BYOK dispatches to exactly one adapter, which is authoritative for its own
// id, while an LPBS response may already carry a more specific provider id.
func StampIdentity[T ~string](tier *T, tierValue T, providerID, modelID *string, providerValue, modelValue string, overwriteProvider bool) {
	if tier != nil {
		*tier = tierValue
	}
	if providerID != nil && (overwriteProvider || *providerID == "") {
		*providerID = providerValue
	}
	if modelID != nil && *modelID == "" {
		*modelID = modelValue
	}
}
