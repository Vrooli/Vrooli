package msvcruntime

import (
	"slices"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestMicrosoftRuntimeDoesNotInstallOnOtherPlatforms(t *testing.T) {
	h := handler{hostreqkit.ToolManifest{Name: "msvc-runtime"}}
	for _, os := range []string{"linux", "darwin"} {
		s := h.Inspect(hostreqkit.Host{OS: os}, hostreqspec.ResolvedRequirement{})
		if s.Installed || s.InstallSupported || s.SupportClass != hostreqkit.SupportUnsupported {
			t.Fatalf("unexpected support on %s: %+v", os, s)
		}
	}
}

func TestMicrosoftRuntimeExplicitInstallUsesClosedWingetSource(t *testing.T) {
	old := hostreqkit.RunCommandFn
	defer func() { hostreqkit.RunCommandFn = old }()
	calls := 0
	hostreqkit.RunCommandFn = func(command string, args []string, opts hostreqkit.EnsureOptions) error {
		calls++
		if command != "winget" || !opts.DryRun {
			t.Fatalf("unexpected installer execution %s %+v", command, opts)
		}
		expected := []string{"install", "--id", "Microsoft.VCRedist.2015+.x64", "--accept-package-agreements", "--accept-source-agreements", "--exact", "--source", "winget"}
		if !slices.Equal(args, expected) {
			t.Fatalf("unbounded package source: %q", args)
		}
		return nil
	}
	h := handler{hostreqkit.ToolManifest{Name: "msvc-runtime", Packages: map[string]string{"winget": "Microsoft.VCRedist.2015+.x64"}}}
	s := hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportSupported, InstallSupported: true}
	if _, err := h.Apply(hostreqkit.Host{OS: "windows", PackageManager: "winget"}, s, hostreqkit.EnsureOptions{DryRun: true}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal("explicit preview did not reach native governed owner")
	}
	s.Installed = true
	if _, err := h.Apply(hostreqkit.Host{OS: "windows", PackageManager: "winget"}, s, hostreqkit.EnsureOptions{DryRun: true}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal("adopted runtime was unnecessarily installed again")
	}
}
