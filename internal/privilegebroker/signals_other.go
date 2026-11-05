//go:build !linux

package privilegebroker

import "os"

func serviceSignals() []os.Signal { return []os.Signal{os.Interrupt} }
