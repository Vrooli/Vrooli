package main

import (
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
	"google.golang.org/protobuf/proto"
	"io"
	"net/http"
	"testing"

	"github.com/vrooli/cli-core/cliapptest"
)

func TestConfigureZoneCLIUsesTypedPublicRequest(t *testing.T) {
	called := false
	cliapptest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"healthy"}`))
			return
		}
		if r.URL.Path != "/vrooli.lighting_control.v1.control.ControlService/ConfigureZone" {
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		var body controlv1.ConfigureZoneRequest
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if err := proto.Unmarshal(data, &body); err != nil {
			t.Error(err)
		}
		if body.ControllerId != "terminal" || body.ZoneId != "header" || body.LedCount != 24 {
			t.Errorf("wrong typed request: %+v", &body)
		}
		called = true
		w.Header().Set("Content-Type", "application/proto")
		data, err = proto.Marshal(&controlv1.ConfigureZoneResponse{Zone: &controlv1.ZoneCapability{ZoneId: "header", ConfiguredLedCount: 24}})
		if err != nil {
			t.Error(err)
		}
		w.Write(data)
	}))
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Run([]string{"control", "configure-zone", "--controller-id", "terminal", "--zone-id", "header", "--led-count", "24"}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("configuration did not reach public API")
	}
}
