// Pure proto<->domain mappers for the usage package. No I/O, no handler
// state — safe to call from any goroutine.
package usage

import (
	"time"

	"audio-tools/internal/clock"
	"audio-tools/internal/protomap"
	"audio-tools/internal/store"

	usagev1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/usage"
)

func resolveSince(clk clock.Clock, sinceSeconds int64) time.Time {
	if sinceSeconds <= 0 {
		sinceSeconds = 86400
	}
	if clk == nil {
		clk = clock.System{}
	}
	return clk.Now().UTC().Add(-time.Duration(sinceSeconds) * time.Second)
}

func rowToProto(r store.UsageRow) *usagev1.UsageRow {
	return &usagev1.UsageRow{
		OperationId:          r.OperationID,
		EmittedAt:            protomap.TimeToProto(r.EmittedAt),
		Capability:           r.Capability,
		Operation:            r.Operation,
		ProviderTier:         protomap.ProviderTierToProto(r.ProviderTier),
		ProviderId:           r.ProviderID,
		ModelId:              r.ModelID,
		LatencyMs:            r.LatencyMs,
		CreditsCharged:       r.CreditsCharged,
		PromptTokens:         r.PromptTokens,
		OutputTokens:         r.OutputTokens,
		AudioDurationSeconds: r.AudioDurationSeconds,
		Error:                r.Error,
		FallbackReason:       r.FallbackReason,
		UserIdentity:         r.UserIdentity,
	}
}
