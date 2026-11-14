package resources

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

// Native OpenRGB uses the actual operator principal. No per-resource account
// is created or impersonated. Backend access is verified under its actual
// runtime token immediately before execution.
func (c *Controller) openRGBNativeServiceAccount(ctx context.Context) (platform.OwnedServiceAccount, error) {
	if err := ctx.Err(); err != nil {
		return platform.OwnedServiceAccount{}, err
	}
	principal, err := openRGBNativeOperatorPrincipal()
	if err != nil {
		return platform.OwnedServiceAccount{}, err
	}
	if err := hostreqkit.ValidateNativeOpenRGBAccount(runtime.GOOS, principal); err != nil {
		return platform.OwnedServiceAccount{}, err
	}
	if c == nil || !filepath.IsAbs(c.Home) || filepath.Clean(c.Home) != c.Home {
		return platform.OwnedServiceAccount{}, errors.New("native OpenRGB requires the actual operator home")
	}
	info, err := os.Lstat(c.Home)
	if err != nil {
		return platform.OwnedServiceAccount{}, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return platform.OwnedServiceAccount{}, errors.New("native OpenRGB operator home is an alias or non-directory")
	}
	owner, err := config.PrivatePathOwner(c.Home, info)
	if err != nil || owner != principal {
		return platform.OwnedServiceAccount{}, errors.Join(errors.New("native OpenRGB home differs from invoking operator"), err)
	}
	return platform.OwnedServiceAccount{Name: principal, Principal: principal}, nil
}

// PrepareOpenRGBNativeDeviceSetup is the explicit privileged setup boundary.
// It creates no backend process and opens no device. Discovery, enrollment
// planning and normal resource start never call it implicitly.
func (c *Controller) PrepareOpenRGBNativeDeviceSetup(ctx context.Context) error {
	if c == nil || c.Root == "" || c.Home == "" {
		return errors.New("native OpenRGB setup requires an operator resource controller")
	}
	if err := c.ValidateOpenRGBNativeResourceStorage(ctx); err != nil {
		return err
	}
	home, err := hostreqkit.InvokingUserHomeDir()
	if err != nil || filepath.Clean(home) != filepath.Clean(c.Home) {
		return errors.Join(errors.New("native OpenRGB setup owner home differs from invoking operator"), err)
	}
	if err := ValidateOpenRGBNativeHostProfile(ctx); err != nil {
		return err
	}
	if err := validateOpenRGBNativeSetupStorage(); err != nil {
		return err
	}
	account, err := c.openRGBNativeServiceAccount(ctx)
	if err != nil {
		return err
	}
	if _, err = hostreqkit.NativeOpenRGBMachineIdentity(); err != nil {
		return err
	}
	manifest, err := c.managedServiceArtifactManifest(openRGBResourceName)
	if err != nil {
		return err
	}
	artifact, err := manifest.ManagedService.Artifact.ForCurrentPlatform()
	if err != nil {
		return err
	}
	profile, err := openRGBReleaseProfileFor(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}
	if artifact.SHA256 != profile.EnvelopeSHA256 || manifest.ManagedService.Artifact.Version != openRGBAppImageVersion {
		return errors.New("native OpenRGB setup requires the exact supported release profile before provisioning")
	}
	state, err := c.operatorStateService().Load(ctx)
	if err != nil {
		return err
	}
	if choice := state.Resources[openRGBResourceName]; choice.Enabled != nil && *choice.Enabled {
		return errors.New("disable OpenRGB before native preparation")
	}
	ctx, err = c.openRGBNativePreparationLockContext(ctx)
	if err != nil {
		return err
	}
	return c.withOpenRGBDevicePermissionFence(ctx, func() error {
		root, err := OpenRGBNativeResourceRoot()
		if err != nil {
			return err
		}
		if err = prepareOpenRGBNativeRootParents(root); err != nil {
			return err
		}
		owner, err := prepareOpenRGBNativeOwner(ctx, account)
		if err != nil {
			return err
		}
		for _, entry := range []string{"state", "config", "data", "cache", "logs", "launch", "runtime"} {
			path := filepath.Join(root, entry)
			policy, err := owner.policyFor(path)
			if err != nil {
				return err
			}
			if err = config.ConfigurePrivateDirectoryAccess(path, policy); err != nil {
				return fmt.Errorf("provision exact OpenRGB %s directory: %w", entry, err)
			}
		}
		return nil
	})
}
func (c *Controller) openRGBNativeRuntimeAccount(ctx context.Context) (string, error) {
	account, err := c.openRGBNativeServiceAccount(ctx)
	return account.Principal, err
}

// ValidateNativeOpenRGBManagementHome binds native operations to the enrolled
// management principal. A later change of home ownership cannot rebind it.
func (c *Controller) ValidateNativeOpenRGBManagementHome(ctx context.Context) error {
	if c == nil || !filepath.IsAbs(c.Home) || filepath.Clean(c.Home) != c.Home {
		return errors.New("native OpenRGB setup requires the original logical operator home")
	}
	owner, err := readOpenRGBNativeOwnerContext(ctx)
	if err != nil {
		return err
	}
	account, err := c.openRGBNativeServiceAccount(ctx)
	if err != nil {
		return err
	}
	if owner.NativeAccount != account.Name || owner.RuntimePrincipal != account.Principal {
		return errors.New("native OpenRGB setup account differs from maintained owner")
	}
	info, err := os.Lstat(c.Home)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("native OpenRGB operator home is an alias or non-directory")
	}
	principal, err := config.PrivatePathOwner(c.Home, info)
	if err != nil {
		return err
	}
	if principal != owner.OwnerPrincipal {
		return errors.New("native OpenRGB setup belongs to another operator")
	}
	return nil
}
