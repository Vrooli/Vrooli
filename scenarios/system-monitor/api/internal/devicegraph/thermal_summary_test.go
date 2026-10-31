package devicegraph

import "testing"

func TestSummarizeThermalUsesHottestSensorAndNearestTrip(t *testing.T) {
	graph := Graph{Devices: []Device{
		{Class: ClassThermalSensor, Readings: map[string]float64{
			readingTemperature:      48,
			readingSetpointMax:      70,
			readingSetpointCritical: 90,
		}},
		{Class: ClassThermalSensor, Readings: map[string]float64{
			readingTemperature:      61,
			readingSetpointMax:      75,
			readingSetpointCritical: 95,
		}},
	}}
	summary := graph.SummarizeThermal()
	if summary.Status != "measured" || summary.TemperatureC != 61 || summary.TripPointC != 70 {
		t.Fatalf("summary = %#v", summary)
	}
}

func TestSummarizeThermalRefusesMissingReadings(t *testing.T) {
	graph := Graph{Devices: []Device{{Class: ClassThermalSensor}}}
	summary := graph.SummarizeThermal()
	if summary.Status != "failed" || summary.TemperatureC != 0 {
		t.Fatalf("summary = %#v", summary)
	}
}
