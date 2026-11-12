//go:build darwin && !cgo

package hostreqkit

import (
	"errors"
	platform "github.com/vrooli/platform-go"
	"testing"
)

func TestDarwinInvokingAccountWithoutNativeDirectoryFailsExplicitly(t *testing.T) {
	if _, err := lookupDarwinInvokingUser("501"); !errors.Is(err, platform.ErrUnsupported) {
		t.Fatalf("missing native directory must fail explicitly: %v", err)
	}
}
