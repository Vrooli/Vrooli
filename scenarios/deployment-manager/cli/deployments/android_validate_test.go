package deployments

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestAndroidValidateHelpListsRequiredFlags(t *testing.T) {
	previous := os.Stderr
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = write
	callErr := (&Commands{}).AndroidValidate([]string{"--help"})
	_ = write.Close()
	os.Stderr = previous

	var output bytes.Buffer
	if _, err := output.ReadFrom(read); err != nil {
		t.Fatal(err)
	}
	_ = read.Close()
	if callErr == nil {
		t.Fatal("expected help parsing to stop the command")
	}
	for _, name := range []string{"profile", "commit", "artifact-ref", "target", "timeout", "format"} {
		if !strings.Contains(output.String(), name) {
			t.Errorf("help output does not list %s: %s", name, output.String())
		}
	}
}
