// Package akt owns its explicit pinned release and license policy.
package akt

import (
	"context"
	_ "embed"
	"errors"
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
	ToolName           = "akt"
	Version            = "1.0.4"
	MaxTreeBytes int64 = 256 << 20
)

//go:embed tool.json
var manifestBytes []byte

type Receipt = pinnedrelease.Receipt

func release() pinnedrelease.Profile {
	return pinnedrelease.Profile{Name: ToolName, Version: Version, License: "Apache-2.0", Manifest: manifestBytes, MaxTreeBytes: MaxTreeBytes, VerifyLicense: verifyLicenses}
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

func verifyLicenses(tree string) error {
	if _, err := os.Stat(filepath.Join(tree, "LICENSE")); err != nil {
		return errors.New("akt release license missing")
	}
	return nil
}

func NewHandler(manifest hostreqkit.ToolManifest) hostreqkit.Handler {
	return hostreqkit.InstallerHandler{
		Manifest: manifest, KindValue: hostreqspec.KindTool,
		InspectFunc: func(host hostreqkit.Host, r hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
			s := hostreqkit.BaseStatus(r)
			s.SupportClass, s.InstallSupported = hostreqkit.SupportSupported, true
			if _, err := releaseTarget(host.OS, runtime.GOARCH); err != nil {
				s.SupportClass, s.ExecutionState = hostreqkit.SupportUnsupported, hostreqkit.ExecutionUnsupported
				s.Notes = append(s.Notes, err.Error())
				return s
			}
			if path, _, err := SelectedInstallation(context.Background()); err == nil {
				s.Command, s.Version, s.Installed, s.ExecutionState = path, Version, true, hostreqkit.ExecutionAlreadyPresent
			} else {
				s.ExecutionState = hostreqkit.ExecutionPending
				s.Notes = append(s.Notes, err.Error())
			}
			return s
		}, ApplyFunc: func(s hostreqkit.ItemStatus, o hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
			if s, done := hostreqkit.ApplyStatus(s); done {
				return s, nil
			}
			args := []string{"deps", "install", "source/akt@" + Version, "--scenario", ToolName, "--surface", "host-tool", "--exact", "--json"}
			if o.DryRun {
				s.ExecutionState = hostreqkit.ExecutionWouldInstall
				s.Notes = append(s.Notes, "scenario-dependency-analyzer "+strings.Join(args, " "))
				return s, nil
			}
			if err := hostreqkit.RunAsInvokingUser("scenario-dependency-analyzer", append(args, "--apply"), o); err != nil {
				return s, err
			}
			path, _, err := SelectedInstallation(context.Background())
			if err != nil {
				return s, err
			}
			s.Command, s.Version, s.Installed, s.ExecutionState = path, Version, true, hostreqkit.ExecutionInstalled
			return s, nil
		},
	}
}
