package resources

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/process"
)

const managedUpdateExecCommand = "managed-update-exec"

func managedUpdateWrapperSpec(u ManagedServiceUpdate, i ManagedUpdateLaunchIntent) platform.OwnedServiceSpec {
	r := i.Runtime
	env := []string{"HOME=" + r.Home, "PATH=/usr/local/bin:/usr/bin:/bin", "XDG_STATE_HOME=" + r.StateHome}
	return platform.OwnedServiceSpec{
		Path: r.Launcher, Dir: r.Root,
		Args: []string{managedUpdateExecCommand, u.Resource, u.RequestID, i.InstanceID}, Env: env, Limits: r.Limits,
	}
}

func managedUpdateAdmittedRole(u ManagedServiceUpdate, instance string) (ManagedUpdateProcess, ManagedUpdateContract, error) {
	var p ManagedUpdateProcess
	var c ManagedUpdateContract
	switch u.Phase {
	case UpdateRollbackProbeStarting:
		p, c = u.RollbackProbe, u.Rollback
	case UpdateCandidateStarting:
		p, c = u.Candidate, u.Target
	case UpdatePrimaryStarting:
		p, c = u.Primary, u.Target
	case UpdateRestoreStarting:
		p, c = u.Restored, u.Rollback
	default:
		return p, c, fmt.Errorf("update phase does not admit a resource exec")
	}
	if !validUpdateIntent(p.Intent, c) || p.Intent.InstanceID != instance || p.State != nil || p.NativeRef != nil || p.Quiescence != nil || p.Intent.Runtime.validate() != nil {
		return ManagedUpdateProcess{}, ManagedUpdateContract{}, fmt.Errorf("resource exec differs from admitted untouched role")
	}
	if len(c.Manifest.Ports) != 1 {
		return ManagedUpdateProcess{}, ManagedUpdateContract{}, fmt.Errorf("resource role requires exactly one frozen port")
	}
	port := c.Manifest.Ports[0].Host
	shadow := u.Phase == UpdateCandidateStarting || u.Phase == UpdateRollbackProbeStarting
	if port < 1 || (shadow && p.Intent.Runtime.Port == port) || (!shadow && p.Intent.Runtime.Port != port) {
		return ManagedUpdateProcess{}, ManagedUpdateContract{}, fmt.Errorf("resource role port differs from its frozen contract")
	}
	return p, c, nil
}

type managedUpdateExecOwner struct {
	lookup      func(context.Context, platform.ServiceIntent) (platform.ServiceRef, error)
	claim       func(context.Context, managedUpdateExecFence, platform.ServiceRef) (managedUpdateExecFence, error)
	selfImage   func(int) (*os.File, error)
	incarnation func(int) (string, error)
	replace     func(*os.File, []string, []string) error
	chdir       func(string) error
}

// RunManagedUpdateExec is internal native-service plumbing. Command-line
// identifiers are selectors, never permission: the durable checkpoint, exact
// native root, launcher image, runtime specification and once-only fence must
// all agree. It takes no lifecycle lock, which the dispatching owner may hold.
func (c *Controller) RunManagedUpdateExec(ctx context.Context, resource, request, instance string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if !managedUpdateResource.MatchString(resource) || !managedUpdateKey.MatchString(request) || !managedUpdateKey.MatchString(instance) {
		return fmt.Errorf("invalid resource exec selector")
	}
	s, err := newManagedUpdateStore(resource)
	if err != nil {
		return err
	}
	u, err := s.Lookup(request)
	if err != nil {
		return err
	}
	u, err = s.Confirm(ctx, request, u.Revision)
	if err != nil {
		return err
	}
	p, _, err := managedUpdateAdmittedRole(u, instance)
	if err != nil {
		return err
	}
	r := p.Intent.Runtime
	if r.Root != c.Root || r.Home != c.Home || filepath.Dir(s.path) != r.StateDir {
		return fmt.Errorf("resource exec runtime differs from the current owner namespace")
	}
	f, err := newManagedUpdateExecStore(resource, u.ID, instance)
	if err != nil {
		return err
	}
	return executeManagedUpdateRole(ctx, u, instance, managedUpdateExecOwner{
		lookup: platform.LookupOwnedService, claim: f.Claim, selfImage: platform.OpenProcessExecutable,
		incarnation: process.Incarnation, replace: platform.ReplaceProcessImage,
		chdir: os.Chdir,
	})
}

func executeManagedUpdateRole(ctx context.Context, u ManagedServiceUpdate, instance string, owner managedUpdateExecOwner) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := u.Validate(); err != nil {
		return err
	}
	p, c, err := managedUpdateAdmittedRole(u, instance)
	if err != nil {
		return err
	}
	i := *p.Intent
	if err := i.NativeIntent.ValidateSpec(managedUpdateWrapperSpec(u, i)); err != nil {
		return err
	}
	self, err := owner.selfImage(os.Getpid())
	if err != nil {
		return err
	}
	digest, err := managedUpdateImageDigest(ctx, self)
	_ = self.Close()
	if err != nil {
		return err
	}
	if digest != i.Runtime.LauncherSHA256 {
		return fmt.Errorf("resource wrapper is not its frozen launcher image")
	}
	ref, err := owner.lookup(ctx, i.NativeIntent)
	if err != nil {
		return err
	}
	incarnation, err := owner.incarnation(os.Getpid())
	if err != nil {
		return err
	}
	_, ticks, ok := strings.Cut(incarnation, ":")
	if ref.ValidateIdentity() != nil || ref.Intent != platform.OperationIntent(i.NativeIntent) || ref.RootPID != os.Getpid() || !ok || ref.RootStartTime != ticks {
		return fmt.Errorf("resource wrapper is outside its exact admitted native root")
	}
	root, err := i.Runtime.roleRoot(u.ID, instance)
	if err != nil {
		return err
	}
	token, err := readManagedUpdateToken(filepath.Join(root, "ownership-token"))
	if err != nil {
		return err
	}
	if managedServiceTokenHash(token) != i.OwnershipTokenHash {
		return fmt.Errorf("resource role ownership token differs from admission")
	}
	invocation, err := renderManagedUpdateInvocation(c, i.Runtime, u.ID, instance, token)
	if err != nil {
		return err
	}
	image, err := os.Open(invocation.Path)
	if err != nil {
		return err
	}
	defer image.Close()
	digest, err = managedUpdateImageDigest(ctx, image)
	if err != nil {
		return err
	}
	if digest != c.Artifact.SHA256 {
		return fmt.Errorf("resource image differs from its frozen checksum")
	}
	if err := materializeManagedUpdateInvocation(u, c, invocation); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	fence := managedUpdateExecFence{SchemaVersion: managedUpdateExecFenceSchemaVersion, Resource: u.Resource, OperationID: u.ID, Intent: i, State: managedUpdateExecFencePrepared}
	if err := verifyManagedUpdateCapacity(ctx, u); err != nil {
		return err
	}
	if _, err := owner.claim(ctx, fence, ref); err != nil {
		return err
	}
	// A successful claim is consumed even if exec fails or the caller disappears.
	// Recovery reconciles this unit; it never launches this identity again.
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := verifyManagedUpdateCapacity(ctx, u); err != nil {
		return err
	}
	if err := owner.chdir(invocation.Dir); err != nil {
		return err
	}
	return owner.replace(image, append([]string{invocation.Path}, invocation.Args...), invocation.Env)
}
