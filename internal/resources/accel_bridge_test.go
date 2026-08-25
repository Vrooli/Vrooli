package resources

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/accel"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
)

// Feature: every accelerator-capable driver verifies where its resource landed
//
//	As an operator
//	I want every driver to read the backend its resource actually runs on
//	So that a CPU fallback is a reported state instead of something discovered
//	fifteen hours later.

// acceleratedManifest builds a manifest declaring cuda with a cpu floor.
func acceleratedManifest(name, driver string) ResourceManifest {
	return ResourceManifest{
		Name:   name,
		Driver: driver,
		Acceleration: &manifestpkg.AccelerationSpec{
			Backends: []string{manifestpkg.BackendCUDA, manifestpkg.BackendCPU},
			Require:  manifestpkg.RequirePreferred,
			Backend: map[string]manifestpkg.BackendConfig{
				manifestpkg.BackendCUDA: {MinCompute: "8.9"},
				manifestpkg.BackendCPU:  {},
			},
		},
	}
}

// Scenario: a manifest's accelerator declaration becomes a placement spec.
func TestAccelSpecForReadsTheOneDeclaration(t *testing.T) {
	cases := []struct {
		scenario       string
		manifest       ResourceManifest
		wantAccelerate bool
		wantBackends   []accel.Backend
	}{
		{
			scenario:       "Given an authored acceleration block, Then the spec carries its backends in order",
			manifest:       acceleratedManifest("kyutai-stt", "compose-service"),
			wantAccelerate: true,
			wantBackends:   []accel.Backend{accel.BackendCUDA, accel.BackendCPU},
		},
		{
			scenario: "Given a block naming only the cpu backend, Then there is nothing to verify",
			manifest: ResourceManifest{
				Name:   "sherpa-onnx",
				Driver: "managed-service",
				Acceleration: &manifestpkg.AccelerationSpec{
					Backends: []string{manifestpkg.BackendCPU},
					Backend:  map[string]manifestpkg.BackendConfig{manifestpkg.BackendCPU: {}},
				},
			},
			wantAccelerate: false,
		},
		{
			scenario:       "Given no accelerator declaration at all, Then there is nothing to verify",
			manifest:       ResourceManifest{Name: "postgres", Driver: "managed-service"},
			wantAccelerate: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.scenario, func(t *testing.T) {
			// When the bridge projects the manifest
			spec, ok := accelSpecFor(tc.manifest)

			// Then it agrees about whether there is an accelerator to verify
			if ok != tc.wantAccelerate {
				t.Fatalf("accelSpecFor() accelerated = %v, want %v (spec = %+v)", ok, tc.wantAccelerate, spec)
			}
			if !tc.wantAccelerate {
				return
			}
			// And the declared backends survive in order
			if len(spec.Backends) != len(tc.wantBackends) {
				t.Fatalf("Backends = %v, want %v", spec.Backends, tc.wantBackends)
			}
			for i, backend := range tc.wantBackends {
				if spec.Backends[i] != backend {
					t.Fatalf("Backends = %v, want %v", spec.Backends, tc.wantBackends)
				}
			}
			if spec.Resource != tc.manifest.Name {
				t.Fatalf("Resource = %q, want %q", spec.Resource, tc.manifest.Name)
			}
		})
	}
}

// Scenario: each driver resolves the placement target its runtime shape implies.
func TestPlacementTargetForMatchesTheDriverRuntimeShape(t *testing.T) {
	cases := []struct {
		scenario string
		driver   string
		wantKind string
	}{
		{scenario: "Given a managed-service resource, Then the target is a host process", driver: "managed-service", wantKind: "accel.HostProcess"},
		{scenario: "Given a native-cli resource, Then the target is a host process", driver: "native-cli", wantKind: "accel.HostProcess"},
	}

	controller := NewController(t.TempDir(), t.TempDir())
	for _, tc := range cases {
		t.Run(tc.scenario, func(t *testing.T) {
			// Given an accelerated manifest on that driver
			m := acceleratedManifest("fixture", tc.driver)

			// When the placement target is resolved
			target, ok, err := placementTargetFor(context.Background(), controller, m)
			// Then a target of the expected kind comes back
			if err != nil {
				t.Fatalf("placementTargetFor() = %v, want nil", err)
			}
			if !ok {
				t.Fatal("placementTargetFor() resolved nothing; every accelerator-capable driver must have a target kind")
			}
			if got := targetKindName(target); got != tc.wantKind {
				t.Fatalf("target kind = %q, want %q", got, tc.wantKind)
			}
			// And the host-process target carries the artifact prefix, so a
			// child process holding the device is attributed to this resource
			process, isProcess := target.(accel.HostProcess)
			if isProcess && !strings.Contains(process.ExecutablePrefix, "fixture") {
				t.Fatalf("ExecutablePrefix = %q, want it to name the resource's artifact tree", process.ExecutablePrefix)
			}
		})
	}

	// And a driver with no accelerator runtime shape resolves nothing
	if _, ok, err := placementTargetFor(context.Background(), controller, acceleratedManifest("fixture", "cloud-api")); err != nil || ok {
		t.Fatalf("placementTargetFor(cloud-api) = ok %v err %v, want ok=false", ok, err)
	}
}

func targetKindName(target accel.PlacementTarget) string {
	switch target.(type) {
	case accel.HostProcess:
		return "accel.HostProcess"
	case accel.Container:
		return "accel.Container"
	case accel.ComposeService:
		return "accel.ComposeService"
	}
	return "unknown"
}

// Scenario: a resource with no accelerator declaration is never probed.
func TestVerifyStartedPlacementSkipsAResourceWithNoAccelerator(t *testing.T) {
	// Given a resource declaring no accelerator
	controller := NewController(t.TempDir(), t.TempDir())
	m := ResourceManifest{Name: "postgres", Driver: "managed-service"}
	var warnings strings.Builder

	// When its placement is verified after start
	err := verifyStartedPlacement(context.Background(), controller, m, &warnings)
	// Then nothing is probed and nothing is reported
	if err != nil {
		t.Fatalf("verifyStartedPlacement() = %v, want nil", err)
	}
	if warnings.Len() != 0 {
		t.Fatalf("warnings = %q, want none", warnings.String())
	}
}

// Scenario: VROOLI_GPU changes which backend is used, never whether it is reported.
//
// A status surface that hides a deliberate downgrade is indistinguishable from
// one that hides an accidental downgrade, so the override never suppresses the
// verdict — it only explains it.
func TestGPUOverrideChangesSelectionWithoutHidingDrift(t *testing.T) {
	spec := accel.Spec{
		Resource: "ollama",
		Backends: []accel.Backend{accel.BackendCUDA, accel.BackendCPU},
		Require:  accel.RequirePreferred,
	}
	onCUDA := accel.ReadinessResult{Selected: accel.BackendCUDA, Declared: accel.BackendCUDA}

	cases := []struct {
		scenario     string
		override     string
		readiness    accel.ReadinessResult
		wantSelected accel.Backend
		wantDrift    bool
		wantReason   string
	}{
		{
			scenario:     "Given VROOLI_GPU=off on a CUDA-capable host, Then the cpu is selected and the drift is still reported",
			override:     "off",
			readiness:    onCUDA,
			wantSelected: accel.BackendCPU,
			wantDrift:    true,
			wantReason:   "off selected the cpu backend",
		},
		{
			scenario:     "Given VROOLI_GPU=on where the probe found nothing, Then the declared backend is forced and it is not drift",
			override:     "on",
			readiness:    accel.ReadinessResult{Selected: accel.BackendCPU, Declared: accel.BackendCUDA, Drift: true},
			wantSelected: accel.BackendCUDA,
			wantDrift:    false,
			wantReason:   "on forced the declared backend",
		},
		{
			scenario:     "Given VROOLI_GPU unset, Then the host decides and nothing is added",
			override:     "",
			readiness:    onCUDA,
			wantSelected: accel.BackendCUDA,
			wantDrift:    false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.scenario, func(t *testing.T) {
			// Given the override the operator set
			t.Setenv(gpuOverrideEnvVar, tc.override)

			// When it is folded into the readiness verdict
			got, reason := applyGPUOverride(tc.readiness, spec)

			// Then the selected backend and the drift verdict follow the override
			if got.Selected != tc.wantSelected {
				t.Fatalf("Selected = %q, want %q", got.Selected, tc.wantSelected)
			}
			if got.Drift != tc.wantDrift {
				t.Fatalf("Drift = %v, want %v", got.Drift, tc.wantDrift)
			}
			// And the operator can see the override was theirs
			if tc.wantReason == "" {
				if reason != "" {
					t.Fatalf("reason = %q, want none", reason)
				}
				return
			}
			if !strings.Contains(reason, tc.wantReason) {
				t.Fatalf("reason = %q, want it to contain %q", reason, tc.wantReason)
			}
		})
	}
}

// Scenario: a drifted resource is warned about as drift, not as unknown.
func TestVerifyStartedPlacementReportsDriftDistinctlyFromUnknown(t *testing.T) {
	// Given an operator who forced the cpu on a resource that declared cuda
	t.Setenv(gpuOverrideEnvVar, "off")
	controller := NewController(t.TempDir(), t.TempDir())
	var warnings strings.Builder

	// When its placement is verified after start
	err := verifyStartedPlacement(context.Background(), controller, acceleratedManifest("ollama", "managed-service"), &warnings)
	// Then the start succeeds, because a drifted resource is still serving
	if err != nil {
		t.Fatalf("verifyStartedPlacement() = %v, want nil", err)
	}
	// And the warning says what it declared and what it is actually on
	warning := warnings.String()
	if !strings.Contains(warning, "declared cuda but is running on cpu") {
		t.Fatalf("warning = %q, want it to name the declared and observed backends", warning)
	}
	// And it is not mislabelled as unknown, which would hide a known state
	if strings.Contains(warning, "is unknown") {
		t.Fatalf("warning = %q, want drift reported as drift rather than unknown", warning)
	}
}

// Scenario: every accelerator-capable driver calls the placement verifier.
//
// This is a source-level assertion because the alternative — a live start per
// driver — needs docker, compose and an accelerator, none of which a unit test
// may assume.
func TestEveryAcceleratorCapableDriverVerifiesPlacement(t *testing.T) {
	// Given the start path of every driver that can run an accelerated resource
	wantCallers := map[string]string{
		"composeServiceDriver.Run":        "compose-service",
		"startDockerService":              "docker-service",
		"nativeCLIDriver.Run":             "native-cli",
		"managedServiceDriver.runPrivate": "managed-service",
	}

	// When each driver file's call graph is read. The implementation is split
	// into whole-symbol part files, so do not restrict this contract to the
	// package entrypoints.
	callers := map[string]int{}
	files, err := filepath.Glob("drivers*.go")
	if err != nil {
		t.Fatalf("glob driver files: %v", err)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		source, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		fileSet := token.NewFileSet()
		parsed, err := parser.ParseFile(fileSet, file, source, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, declaration := range parsed.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			name := function.Name.Name
			if receiver := receiverTypeName(function.Recv); receiver != "" {
				name = receiver + "." + name
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				if identifier, ok := call.Fun.(*ast.Ident); ok && identifier.Name == "verifyStartedPlacement" {
					callers[name]++
				}
				return true
			})
		}
	}

	// Then every one of them reaches the verifier
	for caller, driver := range wantCallers {
		if callers[caller] == 0 {
			t.Errorf("%s (the %s start path) does not call verifyStartedPlacement; a resource on that driver could fall back to the CPU unnoticed", caller, driver)
		}
	}
	// And no other function calls it, so there is exactly one verification seam
	for caller := range callers {
		if _, expected := wantCallers[caller]; !expected {
			t.Errorf("%s calls verifyStartedPlacement but is not a declared driver start path; keep the seam to one call site per driver", caller)
		}
	}
}

func receiverTypeName(fields *ast.FieldList) string {
	if fields == nil || len(fields.List) == 0 {
		return ""
	}
	switch expression := fields.List[0].Type.(type) {
	case *ast.Ident:
		return expression.Name
	case *ast.StarExpr:
		if identifier, ok := expression.X.(*ast.Ident); ok {
			return identifier.Name
		}
	}
	return ""
}
