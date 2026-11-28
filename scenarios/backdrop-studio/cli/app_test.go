package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	releasev1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/release"
	releaseconnect "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/release/release_v1connect"
	"google.golang.org/protobuf/proto"
)

// TestNewAppConstructs is the smoke gate: NewApp() must succeed against
// the cli-core wiring declared in app.go. This catches the most common
// regression class — a misconfigured StandardScenarioOptions or a missing
// dependency from cli-core — before any tests touch real commands.
func TestNewAppConstructs(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	if app == nil || app.core == nil || app.core.CLI == nil {
		t.Fatal("NewApp() returned an incomplete app")
	}
}

// TestRunVersion exercises a non-API command path through cli-core.
// --version must succeed and must NOT trigger the NeedsAPI preflight
// (which would try to reach the configured API base and fail in CI).
func TestRunVersion(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	if err := app.Run([]string{"--version"}); err != nil {
		t.Fatalf("app.Run(--version) error: %v", err)
	}
}

// TestRunHelp exercises cli-core's help renderer through the scenario
// app's wiring. Help is rendered to stdout by cli-core; we only verify
// Run returns without error. The presence of each registered command in
// the actual help surface is covered by cli-core's own tests.
func TestRunHelp(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	if err := app.Run([]string{"--help"}); err != nil {
		t.Fatalf("app.Run(--help) error: %v", err)
	}
}

// TestMetadata pins the values app.go declares — appName must match the
// scenario id (post-substitution), appVersion must be non-empty.
// Catches accidental edits that decouple the binary identity from the
// scenario it belongs to.
func TestMetadata(t *testing.T) {
	if strings.TrimSpace(appName) == "" {
		t.Fatal("appName must not be empty")
	}
	if strings.TrimSpace(appVersion) == "" {
		t.Fatal("appVersion must not be empty")
	}
}

type releaseCLIProbe struct {
	releaseconnect.UnimplementedReleaseServiceHandler
	requests chan *releasev1.ReleaseRequest
	reject   bool
}

func (p *releaseCLIProbe) Release(_ context.Context, request *connect.Request[releasev1.ReleaseRequest]) (*connect.Response[releasev1.ReleasedBackdrop], error) {
	p.requests <- proto.Clone(request.Msg).(*releasev1.ReleaseRequest)
	if p.reject {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("candidate has no qualified bytes"))
	}
	return connect.NewResponse(&releasev1.ReleasedBackdrop{Id: "fixture-only-release", Width: 1440, Height: 720, ContentHash: strings.Repeat("a", 64), MimeType: "image/png", JobId: "fixture-only-job"}), nil
}

// LP-PRES-009: the real CLI command delegates qualification to the release
// owner. This in-process transport fixture never releases an actual candidate.
func TestReleaseCLIUsesOwnerQualificationAndPropagatesRejection(t *testing.T) {
	for _, rejected := range []bool{false, true} {
		name := "qualified-owner-response"
		if rejected {
			name = "owner-rejection"
		}
		t.Run(name, func(t *testing.T) {
			probe := &releaseCLIProbe{requests: make(chan *releasev1.ReleaseRequest, 1), reject: rejected}
			path, handler := releaseconnect.NewReleaseServiceHandler(probe)
			router := http.NewServeMux()
			router.Handle(path, handler)
			router.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"healthy"}`))
			})
			server := httptest.NewServer(router)
			defer server.Close()
			t.Setenv("BACKDROP_STUDIO_API_BASE", server.URL)
			t.Setenv("BACKDROP_STUDIO_CONFIG_DIR", t.TempDir())
			app, err := NewApp()
			if err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			err = app.core.CLI.RunWithWriters([]string{"--api-base", server.URL, "release", "create", "--candidate", "fixture-candidate", "--style", "survey-relief", "--strategy", "vector", "--surface", "web.hero", "--placement", "full_bleed", "--alt-text", "A contour study", "--json"}, &stdout, &stderr)
			if rejected {
				if err == nil || !strings.Contains(err.Error(), "candidate has no qualified bytes") {
					t.Fatalf("owner rejection was hidden: %v, output=%s", err, stdout.String())
				}
				if strings.Contains(stdout.String(), "fixture-only-release") {
					t.Fatal("failed release printed a success receipt")
				}
			} else if err != nil || !strings.Contains(stdout.String(), "fixture-only-release") {
				t.Fatalf("CLI did not project owner response: %v, output=%s", err, stdout.String())
			}
			select {
			case request := <-probe.requests:
				if request.GetCandidateId() != "fixture-candidate" || request.GetSurfaceId() != "web.hero" || request.GetAltText() != "A contour study" {
					t.Fatal("CLI lost configured release identity")
				}
				if request.GetLegibilityPasses() || request.GetContrastRatio() != 0 || len(request.GetImagePng()) != 0 || len(request.GetReservedRegions()) != 0 {
					t.Fatal("CLI fabricated caller-owned qualification")
				}
			default:
				t.Fatal("real release CLI command never reached the fixture owner")
			}
		})
	}
}
