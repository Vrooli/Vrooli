//go:build darwin

package platform

import (
	"bytes"
	"context"
	"os"
	"os/user"
	"strconv"
)

func openInstalledNativeSupervisorImage(ctx context.Context) (*os.File, error) {
	e := nativeDarwinSystemBrokerEngine()
	unit, err := e.bounded(e.roots.Unit, 1<<20, false)
	if err != nil {
		return nil, err
	}
	descriptor, err := darwinBrokerDescriptorFromPlist(unit)
	if err != nil {
		return nil, err
	}
	plan, err := e.descriptor(descriptor)
	if err != nil {
		return nil, err
	}
	selected, err := user.Lookup(plan.Definition.Username)
	if err != nil {
		return nil, err
	}
	if os.Geteuid() != 0 && selected.Uid != strconv.Itoa(os.Geteuid()) {
		return nil, ErrOperationIdentity
	}
	rendered, err := RenderLaunchd(plan.Definition)
	if err != nil || !bytes.Equal(unit, []byte(rendered.Primary().Content)) {
		return nil, ErrOperationIdentity
	}
	image, err := os.Open(plan.ImagePath)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*os.File, error) { image.Close(); return nil, err }
	if err = verifyNativeSystemBrokerImage(image, plan.SHA256); err != nil {
		return fail(err)
	}
	confirmed, err := e.bounded(e.roots.Unit, 1<<20, false)
	if err != nil {
		return fail(err)
	}
	if !bytes.Equal(unit, confirmed) {
		return fail(ErrOperationIdentity)
	}
	if err = ctx.Err(); err != nil {
		return fail(err)
	}
	return image, nil
}

func openCurrentNativeSystemSupervisorImage(ctx context.Context) (*os.File, error) {
	if len(os.Args) != 3 || os.Args[1] != NativeSystemSupervisorVerb {
		return nil, ErrOperationIdentity
	}
	pid, err := nativeSystemSupervisorPID(ctx)
	if err != nil || pid != os.Getpid() {
		return nil, ErrOperationIdentity
	}
	options, err := nativeDarwinSystemBrokerEngine().runtime(ctx, os.Args[2], pid)
	if err != nil {
		return nil, err
	}
	file, err := darwinOpenProcessExecutable(pid)
	if err != nil {
		return nil, err
	}
	if err = verifyNativeSystemBrokerImage(file, options.ExecutableSHA256); err != nil {
		file.Close()
		return nil, err
	}
	after, err := nativeSystemSupervisorPID(ctx)
	if err != nil || after != pid || ctx.Err() != nil {
		file.Close()
		return nil, ErrOperationIdentity
	}
	return file, nil
}

func openOwnedServiceRootImage(ctx context.Context, ref ServiceRef) (*os.File, error) {
	if ref.Native == nil || ref.Native.Platform != "darwin" || ref.RootPID <= 0 {
		return nil, ErrOperationIdentity
	}
	check := func() error {
		observation, err := InspectOwnedService(ctx, ref)
		if err != nil {
			return err
		}
		if !observation.Ref.Equal(OperationRef(ref)) || observation.State != OperationRunning || !observation.RootAlive || !observation.Populated {
			return ErrOperationIdentity
		}
		return ctx.Err()
	}
	if err := check(); err != nil {
		return nil, err
	}
	file, err := darwinOpenProcessExecutable(ref.RootPID)
	if err != nil {
		return nil, err
	}
	if err = check(); err != nil {
		file.Close()
		return nil, err
	}
	return file, nil
}

func openOwnedServiceChildImage(context.Context, ServiceRef, OwnedImageChildIdentity) (*os.File, error) {
	// Darwin payload admission is a same-PID exec handoff. A separate child
	// receipt has no valid meaning on this platform.
	return nil, ErrOperationIdentity
}
