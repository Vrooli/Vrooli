//go:build !linux

package smoketest

import "fmt"

func evictAndMeasureArtifactPageCache(string) (float64, error) {
	return 0, fmt.Errorf("artifact page-cache eviction and residency measurement are unsupported on this platform")
}
