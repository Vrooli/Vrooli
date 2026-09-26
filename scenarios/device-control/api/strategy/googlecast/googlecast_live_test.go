package googlecast

import (
	"context"
	"os"
	"testing"
	"time"

	"device-control/strategy"
)

func TestLiveGoogleCastReceiver(t *testing.T) {
	if os.Getenv("GOOGLECAST_LIVE_CHECK") != "1" {
		t.Skip("set GOOGLECAST_LIVE_CHECK=1 for an operator-LAN Cast witness")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	transport := New()
	devices, err := transport.DiscoverCast(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var target Device
	for _, device := range devices {
		if device.IdentityKey != "" && device.Endpoint != "" && device.Model == "SmartTV 4K" {
			target = device
			break
		}
	}
	if target.Endpoint == "" {
		t.Fatalf("operator LAN did not return the expected SmartTV Cast receiver: %#v", devices)
	}

	bound := New(WithDevices(target), WithObservationInterval(2*time.Second))
	state, err := bound.ReadState(ctx)
	if err != nil {
		t.Fatal(err)
	}
	application, _ := state.Properties["application"].Value.(string)
	if application == "" {
		// The receiver may be idle when the live check begins. YouTube is a
		// stable Cast application id and exercises the launch path below.
		application = "YouTube"
	}
	volume, ok := state.Properties["volume"].Value.(float64)
	if !ok || volume < 0 || volume > 1 {
		t.Fatalf("receiver did not report a valid volume: %#v", state)
	}
	t.Logf("Cast receiver identity=%s endpoint=%s application=%s volume=%.3f player_state=%v", target.IdentityKey, target.Endpoint, application, volume, state.Properties["player_state"].Value)

	// Reapply the observed values so the live command exercises the receiver
	// setters without changing the household's current volume or application.
	if err := bound.SetProperty(ctx, strategy.PropertySet{Name: "volume", Value: volume}); err != nil {
		t.Fatal(err)
	}
	if err := bound.SetProperty(ctx, strategy.PropertySet{Name: "application", Value: application}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		followUp, readErr := bound.ReadState(ctx)
		if readErr == nil {
			if got, _ := followUp.Properties["application"].Value.(string); got == application {
				return
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("application %q was not reported after launch", application)
}
