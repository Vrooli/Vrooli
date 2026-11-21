package lightingruntime

import (
	"context"
	"errors"
	"fmt"
)

// WithResourceUpdate serializes an explicit resource update or rollback with
// lighting intent and reconciliation. Call it before acquiring the resource
// lifecycle lock, following the same intent-before-resource order as Reconcile.
// The callback receives a private snapshot and must not reenter the controller.
//
// A non-nil result certifies the exact verified serving selection, including
// when the callback also reports an error (for example, a lost publication
// acknowledgement). Enabled intent converges to that selection before the
// error is returned. A nil result leaves the selection unchanged. Disabled
// intent is never enabled or changed by this operation. A pending disable
// rejects the callback entirely.
func (c *Controller) WithResourceUpdate(ctx context.Context, update func(Intent) (*Selection, error)) error {
	if update == nil {
		return errors.New("lighting resource update requires a callback")
	}
	release, err := c.lock(ctx)
	if err != nil {
		return err
	}
	defer release()
	current, err := c.Current()
	if err != nil {
		return err
	}
	if current.Phase == Disabling {
		return errors.New("headless lighting disable is pending; finish it before updating")
	}
	// Intent contains pointers: the resource owner must not be able to mutate
	// the transaction's captured intent through its snapshot.
	snapshot := current
	if current.Selection != nil {
		selection := *current.Selection
		snapshot.Selection = &selection
	}
	if current.Preparation != nil {
		preparation := *current.Preparation
		snapshot.Preparation = &preparation
	}
	selection, updateErr := update(snapshot)
	if current.Phase != Enabled || selection == nil {
		return updateErr
	}
	if err := selection.validate(); err != nil {
		return errors.Join(updateErr, fmt.Errorf("invalid verified resource update selection: %w", err))
	}
	verified := *selection
	current.Selection = &verified
	if current.Preparation != nil && current.Preparation.Selection != verified {
		// Update does not repeat explicit setup's static qualification. Never
		// carry that qualification forward to different bytes or permissions.
		current.Preparation = nil
	}
	// Even an unchanged result is published again: a previous rename may have
	// succeeded while its directory durability acknowledgement was lost.
	return errors.Join(updateErr, c.write(current))
}
