package platform

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
)

// Reuse the boot definition/immutable descriptor/image validator, without the
// current-manager-only runtime reader. Only absence of the fixed unit is absence.
func (e darwinSystemBrokerEngine) enrolledResourceStorageOptions(ctx context.Context) (NativeSystemServiceBrokerOptions, bool, error) {
	var zero NativeSystemServiceBrokerOptions
	unit, err := e.bounded(e.roots.Unit, 1<<20, false)
	if os.IsNotExist(err) {
		return e.retainedResourceStorageOptions(ctx)
	}
	if err != nil {
		return zero, false, err
	}
	descriptor, err := darwinBrokerDescriptorFromPlist(unit)
	if err != nil {
		return zero, false, err
	}
	plan, err := e.descriptor(descriptor)
	if err != nil {
		return zero, false, err
	}
	rendered, err := RenderLaunchd(plan.Definition)
	if err != nil || !bytes.Equal(unit, []byte(rendered.Primary().Content)) {
		return zero, false, ErrOperationIdentity
	}
	confirmed, err := e.bounded(e.roots.Unit, 1<<20, false)
	if err != nil {
		return zero, false, err
	}
	if !bytes.Equal(unit, confirmed) {
		return zero, false, ErrOperationIdentity
	}
	if err = ctx.Err(); err != nil {
		return zero, false, err
	}
	return plan.Options, true, nil
}

// The root-owned existing journal outlives boot removal. Only its own absence
// is unenrolled; a journal naming missing/corrupt immutable authority is unknown.
func (e darwinSystemBrokerEngine) retainedResourceStorageOptions(ctx context.Context) (NativeSystemServiceBrokerOptions, bool, error) {
	var zero NativeSystemServiceBrokerOptions
	path := filepath.Join(e.roots.State, "installed-descriptor.json")
	before, err := e.bounded(path, 8192, true)
	if os.IsNotExist(err) {
		return zero, false, nil
	}
	if err != nil {
		return zero, false, err
	}
	if err = e.existingDirectory(e.roots.State); err != nil {
		return zero, false, err
	}
	plan, err := e.journalWithImageProof(false)
	if err != nil {
		return zero, false, err
	}
	after, err := e.bounded(path, 8192, true)
	if err != nil {
		return zero, false, err
	}
	if !bytes.Equal(before, after) {
		return zero, false, ErrOperationIdentity
	}
	if _, err = e.bounded(e.roots.Unit, 1<<20, false); !os.IsNotExist(err) {
		if err != nil {
			return zero, false, err
		}
		return zero, false, ErrOperationIdentity
	}
	if err = ctx.Err(); err != nil {
		return zero, false, err
	}
	return plan.Options, true, nil
}
