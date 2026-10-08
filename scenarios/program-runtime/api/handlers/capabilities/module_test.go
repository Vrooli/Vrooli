package capabilities

import "testing"

func TestModuleContractIsPresent(t *testing.T) {
	if Module(nil).Name == "" {
		t.Fatal("capabilities module has no name")
	}
}
