package dependencygovernance

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
)

func TestExternalWorkspaceInstallGovernsRequestedAndExistingDependencies(t *testing.T) {
	for _, state := range []string{"approved", "denied", "unrecorded", "wrong-surface"} {
		for _, apply := range []bool{false, true} {
			t.Run(state+map[bool]string{true: "/apply", false: "/preview"}[apply], func(t *testing.T) {
				repo, workspace := t.TempDir(), t.TempDir()
				record := `,{"ecosystem":"npm","package_name":"existing","state":"` + state + `","version_range":"1.0.0","allowed_surfaces":["workspace"],"rationale":"Test fixture"}`
				if state == "unrecorded" {
					record = ""
				}
				if state == "wrong-surface" {
					record = `,{"ecosystem":"npm","package_name":"existing","state":"approved","version_range":"1.0.0","allowed_surfaces":["ui"],"rationale":"Test fixture"}`
				}
				writeRegistry(t, repo, `{"records":[{"ecosystem":"npm","package_name":"requested","state":"approved","version_range":"1.0.0","allowed_surfaces":["workspace"],"rationale":"Test fixture"}`+record+`]}`)
				if err := os.WriteFile(filepath.Join(workspace, "package.json"), []byte(`{"packageManager":"pnpm@10.19.0","dependencies":{"existing":"1.0.0"}}`), 0600); err != nil {
					t.Fatal(err)
				}
				installer := &recordingInstaller{}
				h := &connectHandler{scenariosDir: func() string { return filepath.Join(repo, "scenarios") }, installer: installer}
				request := &governancev1.InstallDependencyRequest{WorkspacePath: workspace, Ecosystem: "npm", PackageName: "requested", Version: "1.0.0", Apply: apply}
				resp, err := h.InstallDependency(context.Background(), connect.NewRequest(request))
				if err != nil {
					t.Fatal(err)
				}
				want := state == "approved" && apply
				if installer.called != want || resp.Msg.Installed != want || resp.Msg.Blocked != (state != "approved") {
					t.Fatalf("governance bypassed: called=%t response=%v", installer.called, resp.Msg)
				}
				if request.Scenario != "" || request.Surface != "" {
					t.Fatal("handler mutated caller request")
				}
			})
		}
	}
}

func TestExternalWorkspaceRejectsMixedTargetBeforeEffects(t *testing.T) {
	installer := &recordingInstaller{}
	h := &connectHandler{installer: installer}
	_, err := h.InstallDependency(context.Background(), connect.NewRequest(&governancev1.InstallDependencyRequest{WorkspacePath: "/tmp/external", Scenario: "demo", Apply: true}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument || installer.called {
		t.Fatalf("mixed target admitted: %v", err)
	}
}
