package resources

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenRGBNativeConstructorsRemainPrivate(t *testing.T) {
	for _, goos := range []string{"linux", "darwin", "windows"} {
		t.Run(goos, func(t *testing.T) {
			root := t.TempDir()
			ambient := filepath.Join(root, "ambient", "OpenRGB", "OpenRGB.json")
			if err := os.MkdirAll(filepath.Dir(ambient), 0o700); err != nil {
				t.Fatal(err)
			}
			before := []byte(`{"Client":{"clients":["ambient"]}}`)
			if err := os.WriteFile(ambient, before, 0o600); err != nil {
				t.Fatal(err)
			}
			private := filepath.Join(root, "private")
			env := map[string]string{"XDG_CONFIG_HOME": private, "APPDATA": private, "QT_QPA_PLATFORM": "offscreen"}
			if err := verifyOpenRGBConstructorEnvironment(goos, private, env); err != nil {
				t.Fatal(err)
			}
			for _, key := range []string{"XDG_CONFIG_HOME", "APPDATA"} {
				saved := env[key]
				env[key] = filepath.Dir(filepath.Dir(ambient))
				if err := verifyOpenRGBConstructorEnvironment(goos, private, env); err == nil {
					t.Fatalf("accepted ambient %s", key)
				}
				env[key] = saved
			}
			after, err := os.ReadFile(ambient)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("constructor validation changed ambient settings")
			}
		})
	}
}
