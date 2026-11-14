package exec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandEnvironmentPrependsConfiguredBinaryAndKeepsExistingPath(t *testing.T) {
	binDir := t.TempDir()
	oldDir := t.TempDir()
	t.Setenv("PATH", oldDir)

	env := commandEnvironment(filepath.Join(binDir, "vrooli"))
	var pathValue string
	for _, value := range env {
		if strings.HasPrefix(value, "PATH=") {
			pathValue = strings.TrimPrefix(value, "PATH=")
			break
		}
	}

	paths := strings.Split(pathValue, string(os.PathListSeparator))
	if len(paths) < 2 || paths[0] != binDir || paths[len(paths)-1] != oldDir {
		t.Fatalf("PATH = %q, want binary directory first and original PATH retained", pathValue)
	}
}
