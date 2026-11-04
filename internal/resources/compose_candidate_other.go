//go:build !linux

package resources

import (
	"fmt"
	"os"
)

func publishComposeCandidate(stage, destination string) error {
	return fmt.Errorf("compose-candidate atomic publication is currently supported on Linux only")
}
func composeCandidatePlatformSupported() bool { return false }

func openCandidateInput(root, relative string) (*os.File, error) {
	return nil, fmt.Errorf("compose-candidate supports Linux only")
}
