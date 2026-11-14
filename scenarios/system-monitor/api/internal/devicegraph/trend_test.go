package devicegraph

import (
	"strings"
	"testing"
	"time"
)

// Thermal is a rung-two snapshot until a second sample exists; only then does
// it become a rung-five signal, banded against the sensor's own setpoints.
func TestTemperatureTrendPromotesThermalToAnticipation(t *testing.T) {
	fixture := buildReferenceHost(t)
	first := collectFixture(t, referenceEnv(t, fixture))

	sensor := deviceByID(t, first, "sensor:nvme@0000:02:00.0")
	pending := assertRung(t, sensor.Rungs, RungAnticipation, StateUnmeasurable)
	if !strings.Contains(pending.Reason, "not_yet_sampled") {
		t.Fatalf("first sample anticipation reason = %q, want not_yet_sampled", pending.Reason)
	}

	tracker := NewTrendTracker()
	tracker.Observe(&first)

	fixture.write("devices/pci0000:00/0000:02:00.0/hwmon/hwmon1/temp1_input", "43850")
	later := referenceEnv(t, fixture)
	later.Now = func() time.Time { return fixtureNow(t).Add(time.Minute) }
	second := collectFixture(t, later)
	tracker.Observe(&second)

	warmed := deviceByID(t, second, "sensor:nvme@0000:02:00.0")
	assertRung(t, warmed.Rungs, RungAnticipation, StateMeasured)
	if got := warmed.Readings[readingTemperature+trendSuffix]; got != 6 {
		t.Errorf("temperature trend = %v C/min, want 6", got)
	}
	if warmed.Attributes[AttributeBand] != string(BandNominal) {
		t.Errorf("band = %q, want nominal at 43.85C against a 82.85C max", warmed.Attributes[AttributeBand])
	}
}

func TestTemperatureBandsAgainstTheSensorsOwnSetpoints(t *testing.T) {
	fixture := buildReferenceHost(t)
	// k10temp declares a max of 95C and no critical setpoint.
	fixture.write("devices/pci0000:00/0000:00:18.3/hwmon/hwmon2/temp1_input", "96500")
	first := collectFixture(t, referenceEnv(t, fixture))
	tracker := NewTrendTracker()
	tracker.Observe(&first)

	later := referenceEnv(t, fixture)
	later.Now = func() time.Time { return fixtureNow(t).Add(time.Minute) }
	second := collectFixture(t, later)
	tracker.Observe(&second)

	hot := deviceByID(t, second, "sensor:k10temp@0000:00:18.3")
	if hot.Attributes[AttributeBand] != string(BandElevated) {
		t.Errorf("band = %q, want elevated at 96.5C against a 95C max", hot.Attributes[AttributeBand])
	}

	// The amdgpu sensor declares only a critical setpoint, and sits under it.
	nominal := deviceByID(t, second, "sensor:amdgpu@0000:79:00.0")
	if nominal.Attributes[AttributeBand] != string(BandNominal) {
		t.Errorf("band = %q, want nominal at 52C against a 100C critical", nominal.Attributes[AttributeBand])
	}
}

// A sensor with no declared bar is reported as unbandable with the reason. The
// alternative — inventing a threshold — would be wrong on different silicon.
func TestSensorWithoutSetpointsIsUnbandable(t *testing.T) {
	fixture := buildReferenceHost(t)
	first := collectFixture(t, referenceEnv(t, fixture))
	tracker := NewTrendTracker()
	tracker.Observe(&first)

	later := referenceEnv(t, fixture)
	later.Now = func() time.Time { return fixtureNow(t).Add(time.Minute) }
	second := collectFixture(t, later)
	tracker.Observe(&second)

	sensor := deviceByID(t, second, "sensor:spd5118@0-0050")
	if sensor.Attributes[AttributeBand] != string(BandUnbandable) {
		t.Fatalf("band = %q, want unbandable", sensor.Attributes[AttributeBand])
	}
	if sensor.Attributes[AttributeBandReason] == "" {
		t.Error("an unbandable sensor must say why")
	}
}

func TestInterfaceErrorTrendIsARungFiveSignal(t *testing.T) {
	fixture := buildReferenceHost(t)
	first := collectFixture(t, referenceEnv(t, fixture))
	tracker := NewTrendTracker()
	tracker.Observe(&first)

	fixture.write("devices/pci0000:00/0000:0b:00.0/net/enp11s0/statistics/rx_errors", "12")
	later := referenceEnv(t, fixture)
	later.Now = func() time.Time { return fixtureNow(t).Add(2 * time.Minute) }
	second := collectFixture(t, later)
	tracker.Observe(&second)

	device := deviceByID(t, second, "net:enp11s0")
	assertRung(t, device.Rungs, RungAnticipation, StateMeasured)
	if got := device.Readings[readingInterfaceErrors+trendSuffix]; got != 5 {
		t.Errorf("error rate = %v per minute, want 5 ((17-7)/2)", got)
	}
}

// A device whose telemetry could not be read never acquires a trend; it keeps
// the reason that blocked the measurement.
func TestUnmeasurableTelemetryNeverAcquiresATrend(t *testing.T) {
	fixture := buildReferenceHost(t)
	tracker := NewTrendTracker()
	for offset := 0; offset < 3; offset++ {
		env := referenceEnv(t, fixture)
		env.Now = func() time.Time { return fixtureNow(t).Add(time.Duration(offset) * time.Minute) }
		graph := collectFixture(t, env)
		tracker.Observe(&graph)

		silent := deviceByID(t, graph, "sensor:asus@asus-nb-wmi")
		state := assertRung(t, silent.Rungs, RungAnticipation, StateUnmeasurable)
		if state.Reason == "" {
			t.Fatal("a sensor with no temperature must explain why it has no trend")
		}
		if _, banded := silent.Attributes[AttributeBand]; banded {
			t.Fatal("a sensor with no temperature must not be banded")
		}
	}
}

// The tracker keeps only the series present in the latest graph, so a device
// that disappears does not leak a sample forever.
func TestTrackerForgetsDevicesThatDisappear(t *testing.T) {
	fixture := buildReferenceHost(t)
	first := collectFixture(t, referenceEnv(t, fixture))
	tracker := NewTrendTracker()
	tracker.Observe(&first)
	if len(tracker.samples) == 0 {
		t.Fatal("the tracker retained nothing from a graph full of measured sensors")
	}

	empty := Graph{CollectedAt: fixtureNow(t).Add(time.Minute)}
	tracker.Observe(&empty)
	if len(tracker.samples) != 0 {
		t.Fatalf("tracker retained %d samples for devices no longer present", len(tracker.samples))
	}
}
