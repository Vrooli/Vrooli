//go:build !unix && !windows

package process

import (
	"fmt"
)

func pidIsAlive(pid int) bool {
	// No liveness primitive on this platform. Fail toward alive: callers wrap
	// the answer as affirmative evidence (Known:true), and false-dead evidence
	// expires valid registry claims downstream. An unverifiable process must
	// read as alive, never dead.
	return true
}

func readProcessEnvironment(pid int) (map[string]string, error) {
	return nil, fmt.Errorf("process environment inspection is not supported on this platform")
}
