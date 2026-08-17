package fakes

import (
	"context"
	"fmt"
	"sync"

	"device-control/strategy"
)

// FloorOnly proves that ID and Describe are sufficient for a strategy that
// has no screen and accepts no input.
type FloorOnly struct{ Declaration strategy.Declaration }

func NewFloorOnly(id string) *FloorOnly {
	return &FloorOnly{Declaration: strategy.Declaration{
		StrategyID: id, Description: "floor-only fixture", Status: strategy.StatusAvailable,
		Capabilities: map[string]strategy.Capability{
			strategy.CapScreenshot: {Name: strategy.CapScreenshot, Status: strategy.StatusUnavailable, Reason: "fixture has no screen"},
			strategy.CapInput:      {Name: strategy.CapInput, Status: strategy.StatusUnavailable, Reason: "fixture has no input"},
		},
		Promotable: true, EvidenceClass: "fixture",
	}}
}

func (f *FloorOnly) ID() string { return f.Declaration.StrategyID }
func (f *FloorOnly) Describe(context.Context) (strategy.Declaration, error) {
	return f.Declaration, nil
}

type PropertyOnly struct {
	Declaration strategy.Declaration
	Values      map[string]any
	mu          sync.Mutex
	Changes     []strategy.PropertySet
}

func NewPropertyOnly(id string, descriptor strategy.PropertyDescriptor, value any) *PropertyOnly {
	return &PropertyOnly{
		Declaration: strategy.Declaration{
			StrategyID: id, Description: "property-only fixture", Status: strategy.StatusAvailable,
			Capabilities: map[string]strategy.Capability{
				strategy.CapProperty:   {Name: strategy.CapProperty, Status: strategy.StatusAvailable, StateClass: strategy.StateBearing},
				strategy.CapScreenshot: {Name: strategy.CapScreenshot, Status: strategy.StatusUnavailable, Reason: "property fixture has no screen"},
				strategy.CapInput:      {Name: strategy.CapInput, Status: strategy.StatusUnavailable, Reason: "property fixture has no input"},
			}, Properties: []strategy.PropertyDescriptor{descriptor}, Promotable: true, EvidenceClass: "fixture",
		}, Values: map[string]any{descriptor.Name: value},
	}
}

func (f *PropertyOnly) ID() string { return f.Declaration.StrategyID }
func (f *PropertyOnly) Describe(context.Context) (strategy.Declaration, error) {
	return f.Declaration, nil
}
func (f *PropertyOnly) GetProperty(_ context.Context, name string) (any, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	value, ok := f.Values[name]
	if !ok {
		return nil, fmt.Errorf("unknown property %q", name)
	}
	return value, nil
}
func (f *PropertyOnly) SetProperty(_ context.Context, set strategy.PropertySet) error {
	for _, descriptor := range f.Declaration.Properties {
		if descriptor.Name != set.Name {
			continue
		}
		if err := strategy.ValidatePropertyValue(descriptor, set.Value); err != nil {
			return err
		}
		f.mu.Lock()
		f.Values[set.Name] = set.Value
		f.Changes = append(f.Changes, set)
		f.mu.Unlock()
		return nil
	}
	return fmt.Errorf("unknown property %q", set.Name)
}

type SensorOnly struct {
	Declaration strategy.Declaration
	Readings    []strategy.SensorReading
}

func NewSensorOnly(id string, readings ...strategy.SensorReading) *SensorOnly {
	return &SensorOnly{Declaration: strategy.Declaration{
		StrategyID: id, Description: "sensor-only fixture", Status: strategy.StatusAvailable,
		Capabilities: map[string]strategy.Capability{
			strategy.CapSensor:     {Name: strategy.CapSensor, Status: strategy.StatusAvailable, StateClass: strategy.StateBearing},
			strategy.CapInput:      {Name: strategy.CapInput, Status: strategy.StatusUnavailable, Reason: "sensor is read-only"},
			strategy.CapProperty:   {Name: strategy.CapProperty, Status: strategy.StatusUnavailable, Reason: "sensor has no writable properties"},
			strategy.CapMedia:      {Name: strategy.CapMedia, Status: strategy.StatusUnavailable, Reason: "sensor is not a media endpoint"},
			strategy.CapScreenshot: {Name: strategy.CapScreenshot, Status: strategy.StatusUnavailable, Reason: "sensor has no screen"},
		}, Promotable: true, EvidenceClass: "fixture",
	}, Readings: readings}
}
func (f *SensorOnly) ID() string { return f.Declaration.StrategyID }
func (f *SensorOnly) Describe(context.Context) (strategy.Declaration, error) {
	return f.Declaration, nil
}
func (f *SensorOnly) ReadSensors(context.Context) ([]strategy.SensorReading, error) {
	return append([]strategy.SensorReading(nil), f.Readings...), nil
}

type MediaOnly struct {
	Declaration strategy.Declaration
	Commands    []strategy.MediaCommand
}

func NewMediaOnly(id string) *MediaOnly {
	return &MediaOnly{Declaration: strategy.Declaration{
		StrategyID: id, Description: "media fixture", Status: strategy.StatusAvailable,
		Capabilities: map[string]strategy.Capability{
			strategy.CapMedia:      {Name: strategy.CapMedia, Status: strategy.StatusAvailable, StateClass: strategy.EventBearing},
			strategy.CapInput:      {Name: strategy.CapInput, Status: strategy.StatusUnavailable, Reason: "media fixture has no generic input"},
			strategy.CapScreenshot: {Name: strategy.CapScreenshot, Status: strategy.StatusUnavailable, Reason: "media fixture has no screen"},
		}, Promotable: true, EvidenceClass: "fixture",
	}}
}
func (f *MediaOnly) ID() string { return f.Declaration.StrategyID }
func (f *MediaOnly) Describe(context.Context) (strategy.Declaration, error) {
	return f.Declaration, nil
}
func (f *MediaOnly) ControlMedia(_ context.Context, command strategy.MediaCommand) error {
	switch command.Action {
	case "play", "pause", "stop", "next", "previous", "volume":
		f.Commands = append(f.Commands, command)
		return nil
	default:
		return fmt.Errorf("unsupported media command %q", command.Action)
	}
}
