package climanifest

import (
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestLoadGroupPrefixesManifestErrorsWithDomain(t *testing.T) {
	_, err := LoadGroup([]byte(`{}`), "audio", map[string]func(cliapp.RunContext) error{})
	if err == nil {
		t.Fatal("expected malformed manifest error")
	}
	if !strings.HasPrefix(err.Error(), "audio: load from manifest:") {
		t.Fatalf("error prefix = %q", err)
	}
}
