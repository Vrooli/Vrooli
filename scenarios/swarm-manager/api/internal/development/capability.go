package development

import (
	"fmt"
	"sort"
	"strings"
)

const (
	LaneNative   = "native-goal"
	LaneFallback = "workflow-fallback"
)

// CapabilityRequirements are the guarantees a development engagement needs
// before it can share one grant across repairs and recovery.
type CapabilityRequirements struct {
	Binding      bool
	Continuation bool
	Metering     bool
	Cancellation bool
	Containment  bool
}

// LaneCapability is owner-issued capability evidence. A model name or native
// goal marker is not capability evidence and cannot satisfy this contract.
type LaneCapability struct {
	Lane         string
	Revision     string
	Binding      bool
	Continuation bool
	Metering     bool
	Cancellation bool
	Containment  bool
}

type LaneSelection struct {
	Lane     string
	Revision string
	Reason   string
}

// QualifiedFallbackCapability is the capability revision of the governed
// workflow path. Native remains absent until its owner earns parity.
func QualifiedFallbackCapability() LaneCapability {
	return LaneCapability{Lane: LaneFallback, Revision: "workflow-fallback/v1", Binding: true, Continuation: true, Metering: true, Cancellation: true, Containment: true}
}

func SelectLane(preference string, requirements CapabilityRequirements, capabilities []LaneCapability) (LaneSelection, error) {
	preference = strings.TrimSpace(strings.ToLower(preference))
	switch preference {
	case "", "auto":
		preference = "auto"
	case LaneNative, "native-required":
		preference = LaneNative
	case LaneFallback, "fallback-required":
		preference = LaneFallback
	default:
		return LaneSelection{}, fmt.Errorf("unsupported execution preference %q: %w", preference, ErrInvalid)
	}
	ordered := append([]LaneCapability(nil), capabilities...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if preference == "auto" && ordered[i].Lane != ordered[j].Lane {
			return ordered[i].Lane == LaneNative
		}
		return ordered[i].Lane < ordered[j].Lane
	})
	for _, capability := range ordered {
		if preference != "auto" && capability.Lane != preference {
			continue
		}
		if strings.TrimSpace(capability.Revision) == "" || !supports(capability, requirements) {
			continue
		}
		reason := "selected the owner-qualified " + capability.Lane + " lane at " + capability.Revision
		if preference == "auto" && capability.Lane == LaneFallback {
			reason = "native parity is unavailable; selected the owner-qualified fallback lane at " + capability.Revision
		}
		return LaneSelection{Lane: capability.Lane, Revision: capability.Revision, Reason: reason}, nil
	}
	return LaneSelection{}, fmt.Errorf("no %s execution lane satisfies binding, continuation, metering, cancellation and containment guarantees: %w", preference, ErrDenied)
}

func supports(capability LaneCapability, requirements CapabilityRequirements) bool {
	return (!requirements.Binding || capability.Binding) && (!requirements.Continuation || capability.Continuation) && (!requirements.Metering || capability.Metering) && (!requirements.Cancellation || capability.Cancellation) && (!requirements.Containment || capability.Containment)
}
