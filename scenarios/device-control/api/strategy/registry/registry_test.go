package registry

import (
	"context"
	"testing"
	"time"

	"device-control/strategy"

	"github.com/stretchr/testify/require"
)

type concurrentProbe struct {
	id     string
	ready  chan struct{}
	blocks bool
}

type panicProbe struct{}

func (panicProbe) ID() string                                             { return "panic" }
func (panicProbe) Describe(context.Context) (strategy.Declaration, error) { panic("probe failed") }

func TestListIsolatesPanickingStrategy(t *testing.T) {
	ready := make(chan struct{})
	results := New(panicProbe{}, &concurrentProbe{id: "healthy", ready: ready}).List(context.Background())
	require.Len(t, results, 2)
	require.Equal(t, "healthy", results[0].StrategyID)
	require.Equal(t, strategy.StatusAvailable, results[0].Status)
	require.Equal(t, "panic", results[1].StrategyID)
	require.Equal(t, strategy.StatusUnavailable, results[1].Status)
}

func TestListDoesNotStartProbesAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	results := New(panicProbe{}).List(ctx)
	require.Len(t, results, 1)
	require.Equal(t, strategy.StatusUnavailable, results[0].Status)
	require.Contains(t, results[0].NextActions, "context canceled")
}

func (p *concurrentProbe) ID() string { return p.id }
func (p *concurrentProbe) Describe(ctx context.Context) (strategy.Declaration, error) {
	if p.blocks {
		select {
		case <-p.ready:
		case <-ctx.Done():
			return strategy.Declaration{}, ctx.Err()
		}
	} else {
		close(p.ready)
	}
	return strategy.Declaration{StrategyID: p.id, Status: strategy.StatusAvailable}, nil
}

func TestListProbesStrategiesIndependentlyAndKeepsOrder(t *testing.T) {
	ready := make(chan struct{})
	registry := New(&concurrentProbe{id: "a-slow", ready: ready, blocks: true}, &concurrentProbe{id: "z-fast", ready: ready})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	results := registry.List(ctx)
	require.Len(t, results, 2)
	require.Equal(t, "a-slow", results[0].StrategyID)
	require.Equal(t, "z-fast", results[1].StrategyID)
	require.Equal(t, strategy.StatusAvailable, results[0].Status)
	require.Equal(t, strategy.StatusAvailable, results[1].Status)
}

type fixtureHub struct {
	serial string
}

func (h *fixtureHub) ID() string { return "fixture-hub" }

func (h *fixtureHub) Describe(context.Context) (strategy.Declaration, error) {
	capabilities := map[string]strategy.Capability{}
	for _, name := range []string{strategy.CapInput, strategy.CapScreenshot, strategy.CapProperty, strategy.CapSensor, strategy.CapMedia} {
		capabilities[name] = strategy.Capability{Name: name, Status: strategy.StatusUnavailable, Reason: "fixture hub target does not expose this modality"}
	}
	switch h.serial {
	case "light-1":
		capabilities[strategy.CapProperty] = strategy.Capability{Name: strategy.CapProperty, Status: strategy.StatusAvailable, StateClass: strategy.StateBearing}
	case "tv-1":
		capabilities[strategy.CapInput] = strategy.Capability{Name: strategy.CapInput, Status: strategy.StatusAvailable, StateClass: strategy.EventBearing}
		capabilities[strategy.CapMedia] = strategy.Capability{Name: strategy.CapMedia, Status: strategy.StatusAvailable, StateClass: strategy.EventBearing}
	case "sensor-1":
		capabilities[strategy.CapSensor] = strategy.Capability{Name: strategy.CapSensor, Status: strategy.StatusAvailable, StateClass: strategy.StateBearing}
	}
	return strategy.Declaration{StrategyID: h.ID(), DeviceID: h.serial, Transport: "fixture", Description: "fixture hub target", Status: strategy.StatusAvailable, Capabilities: capabilities}, nil
}

func (h *fixtureHub) Enumerate(context.Context) ([]strategy.Device, error) {
	return []strategy.Device{
		{ID: "fixture-light", Serial: "light-1", Name: "Kitchen light", StrategyID: h.ID(), Transport: "hub", Health: strategy.StatusAvailable},
		{ID: "fixture-tv", Serial: "tv-1", Name: "Living room TV", StrategyID: h.ID(), Transport: "hub", Health: strategy.StatusAvailable},
		{ID: "fixture-sensor", Serial: "sensor-1", Name: "Temperature sensor", StrategyID: h.ID(), Transport: "hub", Health: strategy.StatusAvailable},
	}, nil
}

func (h *fixtureHub) ForDevice(serial string) strategy.Strategy { return &fixtureHub{serial: serial} }

func TestListDevicesReportsPerTargetCapabilityProfiles(t *testing.T) {
	devices := New(&fixtureHub{}).ListDevices(context.Background())
	require.Len(t, devices, 3)
	profiles := map[string][]string{}
	for _, item := range devices {
		available := make([]string, 0)
		for name, capability := range item.Declaration.Capabilities {
			if capability.Status == strategy.StatusAvailable {
				available = append(available, name)
			}
		}
		profiles[item.Device.Serial] = available
	}
	require.ElementsMatch(t, []string{strategy.CapProperty}, profiles["light-1"])
	require.ElementsMatch(t, []string{strategy.CapInput, strategy.CapMedia}, profiles["tv-1"])
	require.ElementsMatch(t, []string{strategy.CapSensor}, profiles["sensor-1"])
}
