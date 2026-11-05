package dependencygovernance

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
)

func TestDevelopmentInstallIsGovernedBeforeEffects(t *testing.T) {
	for _, state := range []string{"approved", "denied"} {
		for _, apply := range []bool{false, true} {
			t.Run(state+"/apply="+map[bool]string{false: "false", true: "true"}[apply], func(t *testing.T) {
				root := t.TempDir()
				writeRegistry(t, root, `{"records":[{"ecosystem":"npm","package_name":"coverage-fixture","state":"`+state+`","rationale":"Fixture reviewed dependency.","version_range":"1.2.3","allowed_surfaces":["ui"]}]}`)
				dir := filepath.Join(root, "scenarios", "demo", "ui")
				if err := os.MkdirAll(dir, 0o700); err != nil {
					t.Fatal(err)
				}
				for _, name := range []string{"package.json", "pnpm-lock.yaml"} {
					if err := os.WriteFile(filepath.Join(dir, name), []byte(`{}`), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				installer := &recordingInstaller{}
				h := &connectHandler{scenariosDir: func() string { return filepath.Join(root, "scenarios") }, installer: installer}
				resp, err := h.InstallDependency(context.Background(), connect.NewRequest(&governancev1.InstallDependencyRequest{
					Scenario: "demo", Surface: "ui", Ecosystem: "npm", PackageName: "coverage-fixture", Version: "1.2.3", ExactVersion: true, Development: true, Apply: apply,
				}))
				if err != nil {
					t.Fatal(err)
				}
				wantInstall := apply && state == "approved"
				if installer.called != wantInstall || resp.Msg.GetInstalled() != wantInstall || resp.Msg.GetBlocked() != (state == "denied") || resp.Msg.GetSourceUpdated() {
					t.Fatalf("incorrect outcome: called=%v response=%v", installer.called, resp.Msg)
				}
				if !strings.Contains(resp.Msg.GetCommand(), "-D") || !strings.Contains(resp.Msg.GetCommand(), "--ignore-scripts") || !strings.Contains(resp.Msg.GetCommand(), "--save-exact") {
					t.Fatalf("development/exact/script policy missing: %s", resp.Msg.GetCommand())
				}
			})
		}
	}
}

func TestInstallOptionsRejectInvalidCombinationsBeforeResolution(t *testing.T) {
	for _, msg := range []*governancev1.InstallDependencyRequest{
		{Ecosystem: "go", Development: true},
		{Ecosystem: "npm", Development: true, Version: "override:1.2.3"},
		{Ecosystem: "npm", Development: true, Surface: "host-tool"},
		{Ecosystem: "npm", RefreshLock: true, Surface: "ui", PackageName: "hyperframes", ExactVersion: true},
		{Ecosystem: "npm", RefreshLock: true, Surface: "host-tool", PackageName: "hyperframes"},
		{Ecosystem: "go", RefreshLock: true, Surface: "host-tool", PackageName: "hyperframes", ExactVersion: true},
		{Ecosystem: "npm", RefreshLock: true, Surface: "host-tool", PackageName: "other", ExactVersion: true},
		{Ecosystem: "npm", RefreshLock: true, Surface: "host-tool", PackageName: "hyperframes", ExactVersion: true, Development: true},
	} {
		t.Run(msg.String(), func(t *testing.T) {
			installer := &recordingInstaller{}
			h := &connectHandler{installer: installer}
			_, err := h.InstallDependency(context.Background(), connect.NewRequest(msg))
			if connect.CodeOf(err) != connect.CodeInvalidArgument || installer.called || !(strings.Contains(err.Error(), "development placement") || strings.Contains(err.Error(), "refresh_lock requires")) {
				t.Fatalf("invalid options admitted: %v, installer=%v", err, installer.called)
			}
		})
	}
}

func TestHostToolSourceRefreshOutcomeNeverClaimsRuntimeInstalled(t *testing.T) {
	for _, state := range []string{"approved", "denied"} {
		for _, apply := range []bool{false, true} {
			t.Run(state+"/apply="+map[bool]string{false: "false", true: "true"}[apply], func(t *testing.T) {
				root := t.TempDir()
				writeRegistry(t, root, `{"records":[{"ecosystem":"npm","package_name":"hyperframes","state":"`+state+`","rationale":"Fixture reviewed dependency.","version_range":"0.8.112","allowed_surfaces":["host-tool"]}]}`)
				source := filepath.Join(root, "internal", "tools", "hyperframes")
				if err := os.MkdirAll(source, 0o700); err != nil {
					t.Fatal(err)
				}
				for _, name := range []string{"package.json", "package-lock.json"} {
					data, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "internal", "tools", "hyperframes", name))
					if err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(source, name), data, 0o600); err != nil {
						t.Fatal(err)
					}
				}
				installer := &recordingInstaller{}
				h := &connectHandler{scenariosDir: func() string { return filepath.Join(root, "scenarios") }, installer: installer}
				resp, err := h.InstallDependency(context.Background(), connect.NewRequest(&governancev1.InstallDependencyRequest{
					Scenario: "hyperframes", Surface: "host-tool", Ecosystem: "npm", PackageName: "hyperframes", Version: "0.8.112", ExactVersion: true, RefreshLock: true, Apply: apply,
				}))
				if err != nil {
					t.Fatal(err)
				}
				wantRefresh := apply && state == "approved"
				if installer.called != wantRefresh || resp.Msg.GetSourceUpdated() != wantRefresh || resp.Msg.GetInstalled() || resp.Msg.GetBlocked() != (state == "denied") {
					t.Fatalf("source-only operation misreported: called=%v response=%v", installer.called, resp.Msg)
				}
				if !strings.Contains(resp.Msg.GetCommand(), "--package-lock-only") || !strings.Contains(resp.Msg.GetCommand(), "--ignore-scripts") || !strings.Contains(resp.Msg.GetMessage(), "no runtime") && !strings.Contains(resp.Msg.GetMessage(), "no managed runtime") {
					if state == "approved" {
						t.Fatalf("source-only/script policy missing: %v", resp.Msg)
					}
				}
			})
		}
	}
}
