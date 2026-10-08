//go:build !linux

package maintenance

import "fmt"

func readScopeIdentity(processTableEntry) (scopeIdentity, error) {
	return scopeIdentity{}, fmt.Errorf("complete executor identity inspection is unsupported on this host")
}
