package resources

import (
	"context"
	"errors"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"os"
	"path/filepath"
)

// The privileged executor already holds the lighting intent fence. This
// resource fence also serializes normal resource start/restart through the
// stopped observation, permission mutation, and resulting policy publication.
func (c *Controller) withOpenRGBDevicePermissionFence(ctx context.Context, change func() error) error {
	invokingHome, err := hostreqkit.InvokingUserHomeDir()
	if err != nil {
		return err
	}
	if c == nil || filepath.Clean(c.Home) != filepath.Clean(invokingHome) {
		return errors.New("OpenRGB permission executor differs from the resource owner home")
	}
	return withManagedServiceLifecycleLock(ctx, openRGBResourceName, func() error {
		state, err := c.operatorStateService().Load(ctx)
		if err != nil {
			return err
		}
		if choice := state.Resources[openRGBResourceName]; choice.Enabled != nil && *choice.Enabled {
			return errors.New("disable the OpenRGB resource before changing device permissions")
		}
		if err := ensureManagedUpdateLifecycleAvailable(openRGBResourceName, "device permission changes"); err != nil {
			return err
		}
		if err := ensureOpenRGBUpdateFencesRetired(); err != nil {
			return err
		}

		store, err := newOpenRGBInitialStore()
		if err != nil {
			return err
		}
		if id, err := store.current(); err == nil {
			a, err := store.read(id)
			if err != nil {
				return err
			}
			if a.State != openRGBInitialStopped {
				return errors.New("finish pending OpenRGB native cleanup before changing device permissions")
			}
			if a.Runtime.Home != c.Home || a.Runtime.Root != c.Root {
				return errors.New("retained OpenRGB initial owner differs from permission executor")
			}
			fences, err := newManagedInitialExecStore(a.ID)
			if err != nil {
				return err
			}
			f, err := fences.read()
			if err != nil {
				return err
			}
			if !sameManagedExecIdentity(f, a.fence()) || f.State != managedUpdateExecFenceRevoked {
				return errors.New("retire the initial OpenRGB exec fence before changing device permissions")
			}

		} else if !os.IsNotExist(err) {
			return err
		}
		supervisor, _, err := managedServiceSupervisorFor(openRGBResourceName)
		if err != nil {
			return err
		}
		if _, running, err := supervisor.Status(); err != nil || running {
			return errors.Join(errors.New("stop the owned lighting backend before changing device permissions"), err)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		return change()
	})
}
