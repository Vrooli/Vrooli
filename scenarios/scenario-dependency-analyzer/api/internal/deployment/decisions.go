package deployment

import types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"

// IsTierBlocker treats an unknown or explicitly unsupported derived verdict as
// blocking. A deployment report must never turn missing evidence into a pass.
func IsTierBlocker(support types.TierSupportSummary) bool {
	return support.Supported == nil || !*support.Supported
}
