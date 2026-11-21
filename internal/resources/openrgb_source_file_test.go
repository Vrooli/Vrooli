package resources

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenRGBSourceOpenPinsRegularObservedObject(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source")
	if err := os.WriteFile(path, []byte("first"), 0o600); err != nil {
		t.Fatal(err)
	}
	expected, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	file, err := openOpenRGBSourceFile(path, expected)
	if err != nil {
		t.Fatal(err)
	}
	file.Close()
	if err = os.Rename(path, path+"-old"); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, []byte("replacement"), 0o600); err != nil {
		t.Fatal(err)
	}
	if f, e := openOpenRGBSourceFile(path, expected); e == nil {
		f.Close()
		t.Fatal("replaced acquired source object accepted")
	}
	if f, e := openOpenRGBSourceFile(filepath.Dir(path), nil); e == nil {
		f.Close()
		t.Fatal("directory accepted as acquired source")
	}
}
