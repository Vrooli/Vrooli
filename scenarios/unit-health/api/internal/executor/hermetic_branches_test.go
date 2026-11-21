package executor

import (
	"reflect"
	"testing"
)

func TestHermeticCommandAllowsPermissiveExecutionAndBuildsReadonlyArguments(t *testing.T) {
	wantArgs := []string{"-test.run=TestCase"}
	path, args, err := hermeticCommand("go", wantArgs, HermeticPolicy{Network: "allow", Filesystem: "workspace"}, "/work", "")
	if err != nil || path != "go" || !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("permissive command = %q,%v,%v", path, args, err)
	}
	wrapped := bwrapReadonlyCommand("go", wantArgs, "/work", "/tmp/root", true)
	joined := formatCommand("bwrap", wrapped)
	for _, flag := range []string{"--ro-bind", "--unshare-net", "--bind", "--chdir", "--", "go"} {
		if !containsToken(wrapped, flag) {
			t.Fatalf("readonly command %q lacks %s", joined, flag)
		}
	}
}

func containsToken(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
