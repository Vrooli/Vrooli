package routing

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	"ai-gateway/internal/providers"
)

func timeoutDuration(timeoutMs int32) time.Duration {
	if timeoutMs <= 0 {
		return 0
	}
	return time.Duration(timeoutMs) * time.Millisecond
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func newID(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(b[:])
}

// samplingSupportText and samplingSupportEnum store the declaration as text
// rather than an ordinal. Route evidence is meant to be read by an operator
// running ad-hoc queries, and its neighbouring columns (breaker_state,
// failure_class, rejection_reason) are already stable lowercase strings for
// exactly that reason.
func samplingSupportText(support sharedv1.SamplingSupport) string {
	switch support {
	case sharedv1.SamplingSupport_SAMPLING_SUPPORT_HONORED:
		return string(providers.SamplingHonored)
	case sharedv1.SamplingSupport_SAMPLING_SUPPORT_IGNORED:
		return string(providers.SamplingIgnored)
	case sharedv1.SamplingSupport_SAMPLING_SUPPORT_REJECTED:
		return string(providers.SamplingRejected)
	case sharedv1.SamplingSupport_SAMPLING_SUPPORT_UNKNOWN:
		return string(providers.SamplingUnknown)
	default:
		return ""
	}
}

func samplingSupportEnum(text string) sharedv1.SamplingSupport {
	switch providers.SamplingSupport(strings.TrimSpace(text)) {
	case providers.SamplingHonored:
		return sharedv1.SamplingSupport_SAMPLING_SUPPORT_HONORED
	case providers.SamplingIgnored:
		return sharedv1.SamplingSupport_SAMPLING_SUPPORT_IGNORED
	case providers.SamplingRejected:
		return sharedv1.SamplingSupport_SAMPLING_SUPPORT_REJECTED
	case providers.SamplingUnknown:
		return sharedv1.SamplingSupport_SAMPLING_SUPPORT_UNKNOWN
	default:
		return sharedv1.SamplingSupport_SAMPLING_SUPPORT_UNSPECIFIED
	}
}
