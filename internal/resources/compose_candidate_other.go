//go:build !linux

package resources

import "fmt"

func publishComposeCandidate(stage, destination string) error {
	return fmt.Errorf("compose-candidate atomic publication is currently supported on Linux only")
}
func composeCandidatePlatformSupported() bool { return false }
