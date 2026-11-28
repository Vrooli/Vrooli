package flow

import (
	"device-control/internal/control/flow/generated"
)

// TransitionDeviceLease is the hand-authored wrapper around the
// generated state machine for the device_control.session.lifecycle flow.
func TransitionDeviceLease(status generated.Status, event generated.Event) (generated.Status, error) {
	return generated.TransitionDeviceLeaseStatus(status, event)
}
