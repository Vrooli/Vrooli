// Package kubo owns its explicit pinned release and license policy.
package kubo

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/tools/pinnedrelease"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	ToolName           = "kubo"
	Version            = "0.43.1"
	MaxTreeBytes int64 = 256 << 20
)

//go:embed tool.json
var manifestBytes []byte

type Receipt = pinnedrelease.Receipt

func release() pinnedrelease.Profile {
	return pinnedrelease.Profile{Name: ToolName, Version: Version, License: "MIT OR Apache-2.0", Manifest: manifestBytes, MaxTreeBytes: MaxTreeBytes, VerifyLicense: verifyLicenses}
}
func ValidateSource(source string) error { return release().ValidateSource(source) }
func releaseTarget(goos, goarch string) (binaryfetch.Target, error) {
	return release().Target(goos, goarch)
}
func SelectedInstallation(ctx context.Context) (string, Receipt, error) {
	return release().SelectedInstallation(ctx)
}
func selectedAt(ctx context.Context, root, home string) (string, Receipt, error) {
	return release().SelectAt(ctx, root, home)
}
func Provision(ctx context.Context) (string, error) { return release().Provision(ctx) }
func provisionAt(ctx context.Context, root, home string, target binaryfetch.Target, fetch pinnedrelease.FetchFunc) (string, error) {
	return release().ProvisionAt(ctx, root, home, target, fetch)
}

// verifyLicenses binds retained release notices to the reviewed upstream bytes.
func verifyLicenses(tree string) error {
	expected := map[string]string{
		"LICENSE-MIT":    "8f71659370c5268d9a1dc962a46232540e8fca63462586d8efaa95aab492a208",
		"LICENSE-APACHE": "30fdfde4dbde73c66123aba24487366771ffc54bba09566b313338199c3eb7dc",
	}
	for name, want := range expected {
		path := filepath.Join(tree, "kubo", name)
		st, err := os.Lstat(path)
		if err != nil || !st.Mode().IsRegular() || st.Size() > 64<<10 {
			return fmt.Errorf("kubo release license %s missing, linked or oversized", name)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(b)
		if hex.EncodeToString(digest[:]) != want {
			return fmt.Errorf("kubo release license %s differs from reviewed notice", name)
		}
	}
	return nil
}

// NewHandler reuses generic manifest inspection but never its PATH adoption or
// direct downloader: the Kubo owner verifies selection and SDA governs writes.
func NewHandler(manifest hostreqkit.ToolManifest, generic hostreqkit.Handler) hostreqkit.Handler {
	return newHandler(manifest, generic, SelectedInstallation)
}

func newHandler(manifest hostreqkit.ToolManifest, generic hostreqkit.Handler, selectInstallation func(context.Context) (string, Receipt, error)) hostreqkit.Handler {
	return hostreqkit.InstallerHandler{
		Manifest: manifest, KindValue: hostreqspec.KindTool,
		InspectFunc: func(host hostreqkit.Host, r hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
			s := generic.Inspect(host, r)
			s.Installed, s.Command, s.Version = false, "", ""
			if _, err := releaseTarget(host.OS, runtime.GOARCH); err != nil {
				s.SupportClass, s.ExecutionState, s.InstallSupported = hostreqkit.SupportUnsupported, hostreqkit.ExecutionUnsupported, false
				s.Installed, s.Command = false, ""
				s.Notes = append(s.Notes, err.Error())
				return s
			}
			if s.SupportClass == hostreqkit.SupportNotApplicable || s.SupportClass == hostreqkit.SupportManualOnly {
				return s
			}
			s.SupportClass, s.InstallSupported = hostreqkit.SupportSupported, true
			if path, _, err := selectInstallation(context.Background()); err == nil {
				s.Command, s.Version, s.Installed, s.ExecutionState = path, Version, true, hostreqkit.ExecutionAlreadyPresent
			} else {
				s.ExecutionState = hostreqkit.ExecutionPending
				s.Notes = append(s.Notes, err.Error())
			}
			return s
		},
		ApplyFunc: func(s hostreqkit.ItemStatus, o hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
			if s, done := hostreqkit.ApplyStatus(s); done {
				return s, nil
			}
			args := []string{"deps", "install", "source/kubo@" + Version, "--scenario", ToolName, "--surface", "host-tool", "--exact", "--json"}
			if o.DryRun {
				s.ExecutionState = hostreqkit.ExecutionWouldInstall
				s.Notes = append(s.Notes, "scenario-dependency-analyzer "+strings.Join(args, " "))
				return s, nil
			}
			if err := hostreqkit.RunAsInvokingUser("scenario-dependency-analyzer", append(args, "--apply"), o); err != nil {
				return s, err
			}
			path, _, err := selectInstallation(context.Background())
			if err != nil {
				return s, err
			}
			s.Command, s.Version, s.Installed, s.ExecutionState = path, Version, true, hostreqkit.ExecutionInstalled
			return s, nil
		},
	}
}
