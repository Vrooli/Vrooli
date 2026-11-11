package main

import (
	"io"
	"net/http"
	"testing"

	"github.com/vrooli/cli-core/cliapptest"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func TestControllerCLIExplicitlyRequestsSavedOfflineInventory(t *testing.T) {
	called := false
	cliapptest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"healthy"}`))
			return
		}
		if r.URL.Path != "/vrooli.lighting_control.v1.control.ControlService/ListControllers" {
			t.Errorf("unexpected public path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			return
		}
		var request controlv1.ListControllersRequest
		if err := proto.Unmarshal(data, &request); err != nil {
			t.Error(err)
			return
		}
		if !request.IncludeUnavailable {
			t.Error("offline inspection was not explicit")
		}
		called = true
		response := &controlv1.ListControllersResponse{Backend: &controlv1.BackendStatus{State: controlv1.BackendState_BACKEND_STATE_UNAVAILABLE, Detail: "transport disconnected"}, Controllers: []*controlv1.Controller{{ControllerId: "saved", Enrolled: true, Zones: []*controlv1.ZoneCapability{{ZoneId: "header", LedCount: 24}}}}}
		data, err = proto.Marshal(response)
		if err != nil {
			t.Error(err)
			return
		}
		w.Header().Set("Content-Type", "application/proto")
		_, _ = w.Write(data)
	}))
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	output := cliapptest.CaptureStdout(t, func() error {
		return app.Run([]string{"control", "controllers", "--include-unavailable", "true", "--json"})
	})
	var response controlv1.ListControllersResponse
	if err := protojson.Unmarshal([]byte(output), &response); err != nil {
		t.Fatal(err)
	}
	if response.Backend.GetState() != controlv1.BackendState_BACKEND_STATE_UNAVAILABLE || response.Backend.GetDetail() != "transport disconnected" {
		t.Fatalf("backend error hidden in CLI response: %s", output)
	}
	if len(response.Controllers) != 1 || response.Controllers[0].ControllerId != "saved" || !response.Controllers[0].Enrolled || response.Controllers[0].Zones[0].ZoneId != "header" {
		t.Fatalf("saved wiring missing in CLI response: %s", output)
	}

	if !called {
		t.Fatal("CLI never requested public controller inventory")
	}
}
