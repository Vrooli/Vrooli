package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/workspace-sandbox/v1/workspace"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func TestApprovalCommandsForwardReviewedPatch(t *testing.T) {
	for _, command := range []string{"approve", "promote"} {
		t.Run(command, func(t *testing.T) {
			digest := strings.Repeat("a", 64)
			reviewID := "734a048d-1fdd-479e-a9dd-cdb7104c0cb9"
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/health" {
					_, _ = w.Write([]byte(`{"status":"healthy"}`))
					return
				}
				calls++
				var request workspacev1.PromoteSandboxRequest
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				if command == "promote" {
					err = proto.Unmarshal(body, &request)
				} else {
					err = protojson.Unmarshal(body, &request)
				}
				if err != nil || request.GetExpectedPatchSha256() != digest || request.GetMode() != "all" || !request.GetForce() || request.GetReviewRequestId() != reviewID || request.GetExpectedReviewSha256() != digest {
					t.Errorf("CLI lost reviewed patch precondition: %v, %v", &request, err)
				}
				if command == "promote" {
					w.Header().Set("Content-Type", "application/proto")
					body, err := proto.Marshal(&workspacev1.PromoteSandboxResponse{Success: true, Applied: 1, AppliedPatchSha256: digest})
					if err != nil {
						t.Error(err)
					}
					_, _ = w.Write(body)
					return
				}
				_, _ = w.Write([]byte(`{"success":true,"applied":1,"appliedPatchSha256":"` + digest + `"}`))
			}))
			defer server.Close()
			t.Setenv("WORKSPACE_SANDBOX_API_BASE", server.URL)
			t.Setenv("CLI_CONFIG_DIR_OVERRIDE", t.TempDir())
			app, err := NewApp()
			if err != nil {
				t.Fatal(err)
			}
			args := []string{"change", command, "bd34c03c-e2bf-4f24-97fb-0f078570fb8a", "--force", "--expected-patch-sha256", digest, "--review-request-id", reviewID, "--expected-review-sha256", digest, "--json"}
			if command == "promote" {
				args = append(args, "--confirm")
			}
			if err := app.Run(args); err != nil || calls != 1 {
				t.Fatalf("CLI approval failed: %v, calls %d", err, calls)
			}
		})
	}
}

func TestReviewCommandsPreserveOwnerIdentity(t *testing.T) {
	for _, command := range []string{"review-capture", "review-workspace", "review-show", "review-file"} {
		t.Run(command, func(t *testing.T) {
			sandboxID, requestID := "bd34c03c-e2bf-4f24-97fb-0f078570fb8a", "734a048d-1fdd-479e-a9dd-cdb7104c0cb9"
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/health" {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"status":"healthy"}`))
					return
				}
				calls++
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				var response proto.Message = &workspacev1.ReviewSnapshotResponse{Snapshot: &workspacev1.ReviewSnapshot{SandboxId: sandboxID, RequestId: requestID, Sha256: strings.Repeat("a", 64)}}
				switch command {
				case "review-workspace":
					var request workspacev1.MaterializeReviewSnapshotRequest
					if err := proto.Unmarshal(body, &request); err != nil || request.GetSandboxId() != sandboxID || request.GetRequestId() != requestID || request.GetExpectedSha256() != strings.Repeat("a", 64) {
						t.Errorf("workspace request changed: %v, %v", &request, err)
					}
					response = &workspacev1.ReviewWorkspace{Root: "/retained/review-tree", Sha256: strings.Repeat("a", 64)}
				case "review-capture":
					var request workspacev1.CaptureReviewSnapshotRequest
					if err := proto.Unmarshal(body, &request); err != nil || request.GetSandboxId() != sandboxID || request.GetRequestId() != requestID || strings.Join(request.GetPaths(), ",") != "src,tests" {
						t.Errorf("capture request changed: %v, %v", &request, err)
					}
				case "review-show":
					var request workspacev1.GetReviewSnapshotRequest
					if err := proto.Unmarshal(body, &request); err != nil || request.GetSandboxId() != sandboxID || request.GetRequestId() != requestID {
						t.Errorf("read request changed: %v, %v", &request, err)
					}
				case "review-file":
					var request workspacev1.GetReviewFileRequest
					if err := proto.Unmarshal(body, &request); err != nil || request.GetSandboxId() != sandboxID || request.GetRequestId() != requestID || request.GetSide() != "after" || request.GetPath() != "src/file" {
						t.Errorf("file request changed: %v, %v", &request, err)
					}
					response = &workspacev1.GetReviewFileResponse{Content: []byte{0, 255, 10}}
				}
				w.Header().Set("Content-Type", "application/proto")
				body, err = proto.Marshal(response)
				if err != nil {
					t.Error(err)
				}
				_, _ = w.Write(body)
			}))
			defer server.Close()
			t.Setenv("WORKSPACE_SANDBOX_API_BASE", server.URL)
			t.Setenv("CLI_CONFIG_DIR_OVERRIDE", t.TempDir())
			app, err := NewApp()
			if err != nil {
				t.Fatal(err)
			}
			args := []string{"change", command, sandboxID, requestID, "--json"}
			if command == "review-workspace" {
				args = append(args, "--expected-sha256", strings.Repeat("a", 64))
			}
			if command == "review-capture" {
				args = append(args, "--path", "src", "--path", "tests")
			}
			if command == "review-file" {
				args = append(args, "--side", "after", "--path", "src/file")
			}
			if err := app.Run(args); err != nil || calls != 1 {
				t.Fatalf("review command failed: %v, calls %d", err, calls)
			}
		})
	}
}

func TestAppConstants(t *testing.T) {
	if appName != "workspace-sandbox" {
		t.Fatalf("appName = %q, want workspace-sandbox", appName)
	}
	if appVersion != "0.1.0" {
		t.Fatalf("appVersion = %q, want 0.1.0", appVersion)
	}
	if defaultAPIBase != "" {
		t.Fatalf("defaultAPIBase = %q, want empty", defaultAPIBase)
	}
}

func TestNewApp(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Skipf("NewApp requires environment setup: %v", err)
	}
	if app == nil || app.core == nil {
		t.Fatalf("NewApp returned nil wiring")
	}
}

func TestWorkspaceSandboxCLIUsesDomainArchitecture(t *testing.T) {
	app := &App{}
	if groups := app.commandGroups(); len(groups) != 0 {
		t.Fatalf("commandGroups len = %d, want 0 (status served by cli-core)", len(groups))
	}

	subcommands := map[string]bool{}
	for _, group := range app.subcommandGroups() {
		subcommands[group.Name] = true
	}

	for _, expected := range []string{"sandbox", "process", "change", "maintenance", "provenance"} {
		if !subcommands[expected] {
			t.Fatalf("missing subcommand group %q", expected)
		}
	}
}
