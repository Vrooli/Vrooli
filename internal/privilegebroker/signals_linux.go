//go:build linux

package privilegebroker

import (
	"os"
	"syscall"
)

func serviceSignals() []os.Signal { return []os.Signal{os.Interrupt, syscall.SIGTERM} }
