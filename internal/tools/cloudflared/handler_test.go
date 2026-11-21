package cloudflared

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/resources"
)

func TestCloudflaredRequirementUsesVerifiedResourceArtifact(t *testing.T) {
	info := resources.ManagedServiceArtifact{Resource: "cloudflared", Path: "/verified/cloudflared", Version: "2026.10.7", SHA256: strings.Repeat("a", 64)}
	h := newHandler(hostreqkit.ToolManifest{Name: "cloudflared"}, func(context.Context) (resources.ManagedServiceArtifact, error) { return info, nil }, func(string, []string, hostreqkit.EnsureOptions) error {
		t.Fatal("already verified artifact must not install")
		return nil
	})
	s := h.Inspect(hostreqkit.Host{OS: "linux"}, hostreqspec.ResolvedRequirement{Name: "cloudflared", Required: true})
	if !s.Installed || s.Command != info.Path || s.Version != info.Version || !s.Required {
		t.Fatalf("verified resource not projected: %+v", s)
	}
	if got, err := h.Apply(hostreqkit.Host{}, s, hostreqkit.EnsureOptions{}); err != nil || got.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("valid artifact should be reused: %+v %v", got, err)
	}
}

func TestCloudflaredRequirementStagesThroughGovernedOwnerAndReverifies(t *testing.T) {
	for _, failure := range []string{"", "governance", "post-install integrity"} {
		t.Run(failure, func(t *testing.T) {
			ready := false
			calls := 0
			inspect := func(context.Context) (resources.ManagedServiceArtifact, error) {
				info := resources.ManagedServiceArtifact{Version: "2026.10.7", Path: "/verified/cloudflared"}
				if !ready || failure == "post-install integrity" {
					return info, errors.New("artifact missing or checksum mismatch")
				}
				return info, nil
			}
			h := newHandler(hostreqkit.ToolManifest{Name: "cloudflared"}, inspect, func(cmd string, args []string, _ hostreqkit.EnsureOptions) error {
				calls++
				if cmd != "scenario-dependency-analyzer" || strings.Join(args, " ") != "deps install source/cloudflared@2026.10.7 --scenario cloudflared --surface host-tool --exact --json --apply" {
					t.Fatalf("unexpected acquisition: %s %v", cmd, args)
				}
				if failure == "governance" {
					return errors.New("policy refused")
				}
				ready = true
				return nil
			})
			s := h.Inspect(hostreqkit.Host{OS: "linux"}, hostreqspec.ResolvedRequirement{Name: "cloudflared"})
			if s.Installed || s.Command != "" || s.Version != "2026.10.7" {
				t.Fatalf("unverified artifact accepted: %+v", s)
			}
			dry, err := h.Apply(hostreqkit.Host{}, s, hostreqkit.EnsureOptions{DryRun: true})
			if err != nil || calls != 0 || dry.ExecutionState != hostreqkit.ExecutionWouldInstall {
				t.Fatalf("dry run: %+v calls=%d err=%v", dry, calls, err)
			}
			got, err := h.Apply(hostreqkit.Host{}, s, hostreqkit.EnsureOptions{})
			if (err != nil) != (failure != "") || calls != 1 {
				t.Fatalf("outcome: %+v calls=%d err=%v", got, calls, err)
			}
			if failure == "" && (!got.Installed || got.Command != "/verified/cloudflared") {
				t.Fatalf("verified install not projected: %+v", got)
			}
			if failure != "" && got.Installed {
				t.Fatal("failed install reported installed")
			}
		})
	}
}

func TestCloudflaredRequirementReportsUnresolvedAndUnsupportedTargets(t *testing.T) {
	h := newHandler(hostreqkit.ToolManifest{Name: "cloudflared"}, func(context.Context) (resources.ManagedServiceArtifact, error) {
		return resources.ManagedServiceArtifact{}, errors.New("no pinned platform target")
	}, func(string, []string, hostreqkit.EnsureOptions) error {
		t.Fatal("unresolved target must not install")
		return nil
	})
	s := h.Inspect(hostreqkit.Host{OS: "linux"}, hostreqspec.ResolvedRequirement{Name: "cloudflared"})
	if s.Installed || s.InstallSupported {
		t.Fatalf("unresolved target: %+v", s)
	}
	if _, err := h.Apply(hostreqkit.Host{}, s, hostreqkit.EnsureOptions{}); err == nil {
		t.Fatal("missing target accepted")
	}
	unsupported := h.Inspect(hostreqkit.Host{OS: "plan9"}, hostreqspec.ResolvedRequirement{Name: "cloudflared"})
	if unsupported.SupportClass != hostreqkit.SupportUnsupported {
		t.Fatalf("unsupported target: %+v", unsupported)
	}
}
