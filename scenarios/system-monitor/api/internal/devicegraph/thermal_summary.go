package devicegraph

import "math"

// ThermalSummary is the compact thermal evidence consumed by high-frequency
// metric views. The device graph remains the owner of sensor enumeration and
// provenance; this projection avoids a second hwmon/thermal-zone walk.
type ThermalSummary struct {
	TemperatureC float64
	TripPointC   float64
	Status       string
	Reason       string
	Provenance   string
}

// SummarizeThermal returns the hottest readable thermal sensor and the nearest
// declared trip point at or above it. It deliberately returns no numeric value
// when the graph has no readable thermal evidence.
func (g Graph) SummarizeThermal() ThermalSummary {
	summary := ThermalSummary{
		Status:     "unsupported",
		Reason:     "the cached device graph exposes no readable thermal sensor",
		Provenance: "cached device graph thermal sensors",
	}
	hottest := -math.MaxFloat64
	seenSensor := false
	for _, device := range g.DevicesOfClass(ClassThermalSensor) {
		seenSensor = true
		temperature, hasTemperature := device.Readings[readingTemperature]
		if !hasTemperature {
			continue
		}
		if temperature > hottest {
			hottest = temperature
		}
	}
	if !seenSensor {
		return summary
	}
	if hottest == -math.MaxFloat64 {
		summary.Status = "failed"
		summary.Reason = "thermal sensors were enumerated but no temperature reading was available"
		return summary
	}
	summary.Status = "measured"
	summary.Reason = ""
	summary.TemperatureC = hottest
	trip := math.MaxFloat64
	for _, device := range g.DevicesOfClass(ClassThermalSensor) {
		for _, key := range []string{readingSetpointMax, readingSetpointCritical} {
			if candidate, ok := device.Readings[key]; ok && candidate >= hottest && candidate < trip {
				trip = candidate
			}
		}
	}
	if trip != math.MaxFloat64 {
		summary.TripPointC = trip
	}
	return summary
}
