package capacityhandlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	capacityapp "github.com/vrooli/vrooli/internal/app/capacity"
	engine "github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/cli/capacitycli"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
)

func TestResidencyCLIHandlerCallsGenerationBoundOwnerAPI(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "capacity.db")
	zeroFloor := int64(0)
	store, err := engine.NewSQLiteStore(ctx, engine.Config{DBPath: dbPath, MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	gpuIndex := 0
	claim, err := store.CreateClaim(ctx, engine.CapacityClaim{
		OwnerKind: engine.OwnerKindResource, OwnerID: "whisper", ResourceKind: engine.ResourceKindVRAM,
		GPUIndex: &gpuIndex, AmountBytes: 1024, PreferredBytes: 1024, FloorBytes: 0,
		Status: engine.StatusGranted, ActivityState: engine.ActivityIdle,
		DegradeProfile: &engine.DegradeProfile{Model: engine.ModelIdentity{Name: "ggml-medium.bin", Digest: "6c14d5adee5f86394037b4e4e8b59f1673b6cee10e3cf0b11bbdbee79c156208"}},
	}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	startedAt := time.Now().UTC().Add(-time.Second)
	request := capacityapp.ResidencyRequest{
		OwnerID: "whisper", ClaimID: claim.ClaimID, ClaimGeneration: claim.Generation,
		Transition: engine.ResidencyTransition{
			ID: "whisper-cli-transition", Model: engine.ModelIdentity{Name: "ggml-medium.bin", Digest: "6c14d5adee5f86394037b4e4e8b59f1673b6cee10e3cf0b11bbdbee79c156208"},
			Desired: engine.ResidencyPlacement{Backend: engine.ResidencyBackendCPU, Known: true},
			Reason:  "pressure", StartedAt: startedAt,
			PreFootprint: engine.ResidencyFootprint{Bytes: 1024, Source: engine.FootprintSourceMeasured, ObservedAt: startedAt.Add(-time.Millisecond), Verified: true},
		},
	}
	requestPath := filepath.Join(t.TempDir(), "request.json")
	raw, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(requestPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	deps := rootcli.HandlerDeps[string]{
		Stdout:       func(string) io.Writer { return &stdout },
		OutputFormat: func(string) (cliout.Format, error) { return cliout.FormatJSON, nil },
		ServiceFor: func(string) capacityapp.Service {
			return capacityapp.Service{OpenStore: func(ctx context.Context) (capacityapp.Store, error) {
				return engine.NewSQLiteStore(ctx, engine.Config{DBPath: dbPath, MinimumDiskFloorBytes: &zeroFloor})
			}}
		},
	}
	handlers := commandtree.BuildHandlerMap(buildCommandTable(deps))
	if err := handlers[string(capacitycli.CommandResidency)]("test", []string{"begin", "--request-file", requestPath}); err != nil {
		t.Fatalf("capacity residency begin: %v", err)
	}
	var record engine.ResidencyTransitionRecord
	if err := json.Unmarshal(stdout.Bytes(), &record); err != nil {
		t.Fatalf("decode residency output: %v", err)
	}
	if record.ClaimID != claim.ClaimID || record.ClaimGeneration != claim.Generation || record.Transition.Model.Name != "ggml-medium.bin" {
		t.Fatalf("record lost exact claim/model identity: %+v", record)
	}
	unknownAt := time.Now().UTC()
	request.Revision = record.Revision
	request.CurrentClaimGeneration = claim.Generation
	request.Transition.Observed = engine.ResidencyPlacement{Known: false, Reason: "probe unavailable"}
	request.Transition.Footprint = engine.ResidencyFootprint{Bytes: 1024, Source: engine.FootprintSourceMeasured, ObservedAt: unknownAt, Verified: true}
	raw, err = json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(requestPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := handlers[string(capacitycli.CommandResidency)]("test", []string{"recover", "--request-file", requestPath}); err != nil {
		t.Fatalf("capacity residency recover: %v", err)
	}
	var recovered engine.ResidencyTransitionRecord
	if err := json.Unmarshal(stdout.Bytes(), &recovered); err != nil {
		t.Fatalf("decode recovery output: %v", err)
	}
	if recovered.Revision != record.Revision+1 || recovered.Transition.Observed.Known {
		t.Fatalf("recovery response=%+v", recovered)
	}
	stdout.Reset()
	if err := handlers[string(capacitycli.CommandResidency)]("test", []string{"history", "--transition-id", request.Transition.ID}); err != nil {
		t.Fatalf("capacity residency history: %v", err)
	}
	var events []engine.ResidencyTransitionEvent
	if err := json.Unmarshal(stdout.Bytes(), &events); err != nil {
		t.Fatalf("decode recovery history: %v", err)
	}
	if len(events) != 2 || events[0].EventKind != "started" || events[1].EventKind != "recovery_unknown" {
		t.Fatalf("recovery history=%+v", events)
	}
}
