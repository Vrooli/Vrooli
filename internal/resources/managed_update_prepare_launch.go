package resources

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	platform "github.com/vrooli/platform-go"
)

// prepareManagedUpdateLaunch runs under the lifecycle owner before role-intent
// admission. A retry retains exactly the same native invocation and token. It
// performs no dispatch, and its fence alone grants no resource execution.
func (c *Controller) prepareManagedUpdateLaunch(ctx context.Context, u ManagedServiceUpdate, contract ManagedUpdateContract, instance string, port int) (ManagedUpdateLaunchIntent, error) {
	var err error
	if err := u.Validate(); err != nil {
		return ManagedUpdateLaunchIntent{}, err
	}
	if contract.Digest != u.Target.Digest && contract.Digest != u.Rollback.Digest {
		return ManagedUpdateLaunchIntent{}, fmt.Errorf("role contract is outside the admitted update")
	}
	var openRGBPolicy *openRGBExecPolicy
	if u.Resource == openRGBResourceName {
		openRGBPolicy, err = c.openRGBManagedUpdateLaunchPolicy(ctx, u, contract)
		if err != nil {
			return ManagedUpdateLaunchIntent{}, err
		}
	}
	if err := reserveManagedUpdateCapacity(ctx, u); err != nil {
		return ManagedUpdateLaunchIntent{}, err
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
		if i.ContractDigest != contract.Digest || i.Runtime.Root != c.Root || i.Runtime.Home != c.Home || i.Runtime.Port != port || i.Runtime.StateDir != paths.StateDir || (i.Runtime.NativeAccount == "" && i.Runtime.StateHome != os.Getenv("XDG_STATE_HOME")) {
			return ManagedUpdateLaunchIntent{}, ErrManagedUpdateConflict
		}
		if err := i.Runtime.validate(); err != nil {
			return ManagedUpdateLaunchIntent{}, err
		}
		if err := i.NativeIntent.ValidateSpec(managedUpdateWrapperSpec(u, i)); err != nil {
			return ManagedUpdateLaunchIntent{}, err
		}
		tokenPath, e := i.Runtime.tokenPath(u.ID, instance)
		if e != nil {
			return ManagedUpdateLaunchIntent{}, e
		}
		if err := ensureManagedUpdatePrivateDirectory(i.Runtime.StateDir, filepath.Dir(tokenPath)); err != nil {
			return ManagedUpdateLaunchIntent{}, err
		}
		token, err := readManagedUpdateToken(tokenPath)
		if err != nil {
			return ManagedUpdateLaunchIntent{}, err
		}
		if managedServiceTokenHash(token) != i.OwnershipTokenHash {
			return ManagedUpdateLaunchIntent{}, fmt.Errorf("prepared role token changed")
		}
		_, err = fences.Prepare(ctx, managedUpdateExecFence{SchemaVersion: managedUpdateExecFenceSchemaVersion, Resource: u.Resource, OperationID: u.ID, Intent: i, State: managedUpdateExecFencePrepared, OpenRGBPolicy: openRGBPolicy})
		return i, err
	} else if !os.IsNotExist(err) {
		return ManagedUpdateLaunchIntent{}, err
	}
	image, err := c.openOpenRGBLauncherImage(ctx, u.Resource == openRGBResourceName && openRGBPolicy != nil && openRGBPolicy.Platform != "", &u.Original)
	if err != nil {
		return ManagedUpdateLaunchIntent{}, err
	}
	defer image.Close()
	digest, err := managedUpdateImageDigest(ctx, image)
	if err != nil {
		return ManagedUpdateLaunchIntent{}, err
	}
	launcher := filepath.Join(paths.StateDir, "managed-update-exec", u.ID, "launcher", digest, "vrooli")
	nativeAccount := ""
	ownerPrincipal := ""
	if u.Resource == openRGBResourceName && openRGBPolicy != nil && openRGBPolicy.Platform != "" {
		account, e := c.openRGBNativeServiceAccount(ctx)
		if e != nil {
			return ManagedUpdateLaunchIntent{}, e
		}
		owner, e := readOpenRGBNativeOwnerContext(ctx)
		if e != nil {
			return ManagedUpdateLaunchIntent{}, e
		}
		if account.Principal != openRGBPolicy.Account {
			return ManagedUpdateLaunchIntent{}, fmt.Errorf("native update account differs from retained device scope")
		}
		nativeAccount = account.Name
		ownerPrincipal = owner.OwnerPrincipal
	}
	if nativeAccount != "" {
		launcher, err = c.stageOpenRGBNativeLauncher(ctx, image, digest)
		if err != nil {
			return ManagedUpdateLaunchIntent{}, err
		}
	} else {
		if err := ensureManagedUpdatePrivateDirectory(paths.StateDir, filepath.Dir(launcher)); err != nil {
			return ManagedUpdateLaunchIntent{}, err
		}
		if _, err := image.Seek(0, io.SeekStart); err != nil {
			return ManagedUpdateLaunchIntent{}, err
		}
		if err := writeResourceOwnedFileDurableStream(launcher, 0755, func(dst io.Writer) error {
			_, err := io.Copy(dst, &managedUpdateContextReader{ctx: ctx, reader: image})
			return err
		}); err != nil {
			return ManagedUpdateLaunchIntent{}, err
		}
	}
	r := ManagedUpdateRuntime{NativeAccount: nativeAccount, OwnerPrincipal: ownerPrincipal,
		Root: c.Root, Home: c.Home, StateHome: os.Getenv("XDG_STATE_HOME"), StateDir: paths.StateDir, Launcher: launcher, LauncherSHA256: digest, Port: port,
		Limits: platform.ServiceLimits{MemoryMaxBytes: managedUpdateRoleMemory, TasksMax: 128, CPUWeight: 100},
	}
	if nativeAccount != "" {
		// Native state is the frozen canonical resource root; XDG_STATE_HOME
		// belongs only to the Linux wrapper namespace.
		r.StateHome = ""
		r.NativeLauncherProfile = platform.NativeLauncherProfileOpenRGB
		switch runtime.GOOS {
		case "darwin":
			r.Limits.Enforcement = "native-advisory"
		case "windows":
			r.Limits.Enforcement = "native-process-limits"
		}
	}
	tokenPath, err := r.tokenPath(u.ID, instance)
	if err != nil {
		return ManagedUpdateLaunchIntent{}, err
	}
	if err := ensureManagedUpdatePrivateDirectory(paths.StateDir, filepath.Dir(tokenPath)); err != nil {
		return ManagedUpdateLaunchIntent{}, err
	}
	token, err := readManagedUpdateToken(tokenPath)
	if os.IsNotExist(err) {
		token, err = newManagedServiceToken()
		if err == nil {
			err = writeResourceOwnedFileDurable(tokenPath, []byte(token), 0o600)
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
	_, err = fences.Prepare(ctx, managedUpdateExecFence{SchemaVersion: managedUpdateExecFenceSchemaVersion, Resource: u.Resource, OperationID: u.ID, Intent: i, State: managedUpdateExecFencePrepared, OpenRGBPolicy: openRGBPolicy})
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
