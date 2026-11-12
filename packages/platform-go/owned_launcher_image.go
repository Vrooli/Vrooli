package platform

import (
	"os"
)

// OpenOwnedLauncherImage observes the executable of the exact current owned
// service root. Windows additionally requires the image lock established at
// process birth; an unlocked pathname is never a mapped-image substitute.
func OpenOwnedLauncherImage(ref ServiceRef) (*os.File, error) {
	if ref.ValidateIdentity() != nil || ref.RootPID != os.Getpid() {
		return nil, ErrOperationIdentity
	}
	return openOwnedLauncherImage(ref)
}
