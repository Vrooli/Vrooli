package resources

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/config"
)

// prepareManagedUpdateLaunch runs under the lifecycle owner before role-intent
// admission. A retry retains exactly the same native invocation and token. It
// performs no dispatch, and its fence alone grants no resource execution.
func (c *Controller) prepareManagedUpdateLaunch(ctx context.Context, u ManagedServiceUpdate, contract ManagedUpdateContract, instance string, port int) (ManagedUpdateLaunchIntent, error) {
	if err := reserveManagedUpdateCapacity(ctx, u); err != nil {
		return ManagedUpdateLaunchIntent{}, err
	}
	if err := u.Validate(); err != nil {
		return ManagedUpdateLaunchIntent{}, err
	}
	if contract.Digest != u.Target.Digest && contract.Digest != u.Rollback.Digest {
		return ManagedUpdateLaunchIntent{}, fmt.Errorf("role contract is outside the admitted update")
	}
	fences, err := newManagedUpdateExecStore(u.Resource, u.ID, instance)
	if err != nil {
		return ManagedUpdateLaunchIntent{}, err
	}
	paths, err := resourceStoragePaths(u.Resource)
	if err != nil {
		return ManagedUpdateLaunchIntent{}, err
	}
	if old, err := fences.read(); err == nil {
		i := old.Intent
		if i.ContractDigest != contract.Digest || i.Runtime.Root != c.Root || i.Runtime.Home != c.Home || i.Runtime.Port != port || i.Runtime.StateDir != paths.StateDir || i.Runtime.StateHome != os.Getenv("XDG_STATE_HOME") {
			return ManagedUpdateLaunchIntent{}, ErrManagedUpdateConflict
		}
		if err := i.Runtime.validate(); err != nil {
			return ManagedUpdateLaunchIntent{}, err
		}
		if err := i.NativeIntent.ValidateSpec(managedUpdateWrapperSpec(u, i)); err != nil {
			return ManagedUpdateLaunchIntent{}, err
		}
		root, _ := i.Runtime.roleRoot(u.ID, instance)
		if err := ensureManagedUpdatePrivateDirectory(i.Runtime.StateDir, root); err != nil {
			return ManagedUpdateLaunchIntent{}, err
		}
		token, err := readManagedUpdateToken(filepath.Join(root, "ownership-token"))
		if err != nil {
			return ManagedUpdateLaunchIntent{}, err
		}
		if managedServiceTokenHash(token) != i.OwnershipTokenHash {
			return ManagedUpdateLaunchIntent{}, fmt.Errorf("prepared role token changed")
		}
		_, err = fences.Prepare(ctx, managedUpdateExecFence{SchemaVersion: managedUpdateExecFenceSchemaVersion, Resource: u.Resource, OperationID: u.ID, Intent: i, State: managedUpdateExecFencePrepared})
		return i, err
	} else if !os.IsNotExist(err) {
		return ManagedUpdateLaunchIntent{}, err
	}
	image, err := platform.OpenProcessExecutable(os.Getpid())
	if err != nil {
		return ManagedUpdateLaunchIntent{}, err
	}
	defer image.Close()
	digest, err := managedUpdateImageDigest(ctx, image)
	if err != nil {
		return ManagedUpdateLaunchIntent{}, err
	}
	launcher := filepath.Join(paths.StateDir, "managed-update-exec", u.ID, "launcher", digest, "vrooli")
	if err := ensureManagedUpdatePrivateDirectory(paths.StateDir, filepath.Dir(launcher)); err != nil {
		return ManagedUpdateLaunchIntent{}, err
	}
	if _, err := image.Seek(0, io.SeekStart); err != nil {
		return ManagedUpdateLaunchIntent{}, err
	}
	if err := config.WriteOwnedFileDurableStream(launcher, 0o755, func(dst io.Writer) error {
		_, err := io.Copy(dst, &managedUpdateContextReader{ctx: ctx, reader: image})
		return err
	}); err != nil {
		return ManagedUpdateLaunchIntent{}, err
	}
	r := ManagedUpdateRuntime{
		Root: c.Root, Home: c.Home, StateHome: os.Getenv("XDG_STATE_HOME"), StateDir: paths.StateDir, Launcher: launcher, LauncherSHA256: digest, Port: port,
		Limits: platform.ServiceLimits{MemoryMaxBytes: managedUpdateRoleMemory, TasksMax: 128, CPUWeight: 100},
	}
	root, err := r.roleRoot(u.ID, instance)
	if err != nil {
		return ManagedUpdateLaunchIntent{}, err
	}
	if err := ensureManagedUpdatePrivateDirectory(paths.StateDir, root); err != nil {
		return ManagedUpdateLaunchIntent{}, err
	}
	tokenPath := filepath.Join(root, "ownership-token")
	token, err := readManagedUpdateToken(tokenPath)
	if os.IsNotExist(err) {
		token, err = newManagedServiceToken()
		if err == nil {
			err = config.WriteOwnedFileDurable(tokenPath, []byte(token), 0o600)
		}
	}
	if err != nil {
		return ManagedUpdateLaunchIntent{}, err
	}
	// Validate templates before persisting any native launch permission.
	if _, err := renderManagedUpdateInvocation(contract, r, u.ID, instance, token); err != nil {
		return ManagedUpdateLaunchIntent{}, err
	}
	i := ManagedUpdateLaunchIntent{InstanceID: instance, OwnershipTokenHash: managedServiceTokenHash(token), ContractDigest: contract.Digest, Runtime: r}
	i.NativeIntent, err = platform.NewServiceIntent("resource-update", instance, managedUpdateWrapperSpec(u, i))
	if err != nil {
		return ManagedUpdateLaunchIntent{}, err
	}
	_, err = fences.Prepare(ctx, managedUpdateExecFence{SchemaVersion: managedUpdateExecFenceSchemaVersion, Resource: u.Resource, OperationID: u.ID, Intent: i, State: managedUpdateExecFencePrepared})
	return i, err
}

func readManagedUpdateToken(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 257))
	if err != nil {
		return "", err
	}
	if len(data) == 0 || len(data) > 256 {
		return "", fmt.Errorf("invalid private role token length")
	}
	return string(data), nil
}
