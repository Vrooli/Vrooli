package resources

import (
	"context"
	"errors"
	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/process"
)

func (o openRGBServingObservation) servingPID() (int, error) {
	if o.witness.ExecutionChild == nil {
		return o.ref.RootPID, nil
	}
	child := o.witness.ExecutionChild
	if err := child.validateFor(o.ref); err != nil {
		return 0, err
	}
	incarnation, err := process.Incarnation(child.PID)
	if err != nil || incarnation != child.Incarnation {
		return 0, errors.Join(errors.New("OpenRGB serving child incarnation changed"), err)
	}
	return child.PID, nil
}

// Managed state retains the exact trusted wrapper root. Windows listener and
// image attribution use only its separately claimed child; an arbitrary PID
// supplied by an SDK or state hint never substitutes for that claim.
func (c *Controller) openRGBServingListenerPID(ctx context.Context, state ManagedServiceState) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	var ref platform.ServiceRef
	var witness *openRGBExecWitness
	if state.InitialLaunch != nil {
		_, f, err := readOpenRGBInitialServing(state)
		if err != nil {
			return 0, err
		}
		if f.State != managedUpdateExecFenceClaimed {
			return 0, errors.New("OpenRGB serving claim is revoked")
		}
		ref = state.InitialLaunch.NativeRef
		witness = f.OpenRGBWitness
	} else if state.UpdateLaunch != nil {
		launch := state.UpdateLaunch
		store, err := newManagedUpdateExecStore(openRGBResourceName, launch.OperationID, state.InstanceID)
		if err != nil {
			return 0, err
		}
		f, err := store.read()
		if err != nil {
			return 0, err
		}
		if f.State != managedUpdateExecFenceClaimed || f.Claim == nil || !f.Claim.Equal(launch.NativeRef) || f.validate() != nil {
			return 0, errors.New("OpenRGB serving update claim changed")
		}
		ref = launch.NativeRef
		witness = f.OpenRGBWitness
	} else {
		return state.PID, nil
	}
	if witness == nil {
		return 0, errors.New("OpenRGB serving device witness unavailable")
	}
	return (openRGBServingObservation{ref: ref, witness: *witness}).servingPID()
}
