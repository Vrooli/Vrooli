package devicegraph

import (
	"fmt"
	"sync"
	"time"
)

// Reading keys the graph publishes for banded, trended signals.
const (
	readingTemperature       = "temperature_celsius"
	readingSetpointMax       = "setpoint_max_celsius"
	readingSetpointCritical  = "setpoint_critical_celsius"
	readingInterfaceErrors   = "error_events_total"
	readingCorrectableErrs   = "correctable_errors_total"
	readingUncorrectableErrs = "uncorrectable_errors_total"
	trendSuffix              = "_per_minute"
	trendMechanism           = "in-process rate tracker across collection cycles"
)

// Band is the position of a reading against the hardware's own declared bar.
type Band string

const (
	BandNominal    Band = "nominal"
	BandElevated   Band = "elevated"
	BandCritical   Band = "critical"
	BandUnbandable Band = "unbandable"
)

// AttributeBand and AttributeBandReason are the attribute keys a banded device
// carries.
const (
	AttributeBand       = "band"
	AttributeBandReason = "band_reason"
)

// trendSpec declares which readings of a device class become a rung-five
// signal once a second sample exists.
type trendSpec struct {
	readings []string
	subject  string
	banded   bool
}

var trendSpecs = map[Class]trendSpec{
	ClassThermalSensor: {
		readings: []string{readingTemperature},
		subject:  "temperature",
		banded:   true,
	},
	ClassNetworkInterface: {
		readings: []string{readingInterfaceErrors},
		subject:  "interface error",
	},
	ClassMemoryController: {
		readings: []string{readingCorrectableErrs, readingUncorrectableErrs},
		subject:  "memory error",
	},
	ClassMemoryModule: {
		readings: []string{readingCorrectableErrs, readingUncorrectableErrs},
		subject:  "memory error",
	},
}

// pendingTrend is the anticipation grade a device carries before a second
// sample exists. A trend that has not been sampled twice is unmeasurable with
// that reason — it is never reported as a flat, reassuring zero.
func (b *builder) pendingTrend(telemetry RungState, subject string) RungState {
	switch telemetry.State {
	case StateMeasured:
		return b.grader.unmeasurable(RungAnticipation,
			fmt.Sprintf("not_yet_sampled: the %s trend requires a previous sample", subject), trendMechanism)
	case StateNotApplicable:
		return b.grader.notApplicable(RungAnticipation, telemetry.Reason)
	case StateUnavailable:
		return b.grader.unavailable(RungAnticipation,
			fmt.Sprintf("no %s trend: %s", subject, telemetry.Reason), trendMechanism)
	default:
		return b.grader.unmeasurable(RungAnticipation,
			fmt.Sprintf("no %s trend: %s", subject, telemetry.Reason), trendMechanism)
	}
}

type trendSample struct {
	value float64
	at    time.Time
}

// TrendTracker turns successive graphs into rates, which is what promotes
// thermal and error telemetry from a rung-two snapshot to a rung-five
// forward-looking signal. It is safe for concurrent use and keeps only the
// series present in the most recent graph, so a device that disappears does
// not leak a sample forever.
type TrendTracker struct {
	mu      sync.Mutex
	samples map[string]trendSample
}

// NewTrendTracker builds an empty tracker.
func NewTrendTracker() *TrendTracker {
	return &TrendTracker{samples: map[string]trendSample{}}
}

// Observe folds one graph into the tracker and upgrades the anticipation rung
// of every device whose tracked reading now has a predecessor.
func (t *TrendTracker) Observe(graph *Graph) {
	if t == nil || graph == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	retained := make(map[string]trendSample, len(t.samples))
	at := graph.CollectedAt
	stamp := grader{at: at}

	for index := range graph.Devices {
		device := &graph.Devices[index]
		spec, tracked := trendSpecs[device.Class]
		if !tracked {
			continue
		}
		if device.Rungs[RungTelemetry].State != StateMeasured {
			continue
		}

		rated := false
		for _, reading := range spec.readings {
			value, present := device.Readings[reading]
			if !present {
				continue
			}
			key := device.ID + "|" + reading
			previous, had := t.samples[key]
			retained[key] = trendSample{value: value, at: at}
			if !had || !at.After(previous.at) {
				continue
			}
			minutes := at.Sub(previous.at).Minutes()
			if minutes <= 0 {
				continue
			}
			setReading(device, reading+trendSuffix, (value-previous.value)/minutes)
			rated = true
		}
		if !rated {
			continue
		}
		device.Rungs[RungAnticipation] = stamp.measured(RungAnticipation, trendMechanism)
		if spec.banded {
			applyBand(device)
		}
	}

	t.samples = retained
}

// applyBand positions the current temperature against the setpoints the
// hardware itself declares. When the sensor declares no bar, the band is
// reported as unbandable with the reason rather than being invented.
func applyBand(device *Device) {
	current, hasCurrent := device.Readings[readingTemperature]
	if !hasCurrent {
		return
	}
	critical, hasCritical := device.Readings[readingSetpointCritical]
	maximum, hasMaximum := device.Readings[readingSetpointMax]
	switch {
	case hasCritical && current >= critical:
		setAttribute(device, AttributeBand, string(BandCritical))
	case hasMaximum && current >= maximum:
		setAttribute(device, AttributeBand, string(BandElevated))
	case hasCritical || hasMaximum:
		setAttribute(device, AttributeBand, string(BandNominal))
	default:
		setAttribute(device, AttributeBand, string(BandUnbandable))
		setAttribute(device, AttributeBandReason,
			"this sensor declares no max or critical setpoint, so its temperature cannot be banded without inventing a threshold")
	}
}
