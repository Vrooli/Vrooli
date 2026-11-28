package resourcedeployment

import (
	"debug/pe"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestWindowsNativeExecutableDoesNotRequirePOSIXBits(t *testing.T) {
	data := make([]byte, 392)
	copy(data, []byte{'M', 'Z'})
	binary.LittleEndian.PutUint32(data[60:], 128)
	copy(data[128:], []byte{'P', 'E', 0, 0})
	binary.LittleEndian.PutUint16(data[132:], 0x8664)
	binary.LittleEndian.PutUint16(data[148:], 240)
	binary.LittleEndian.PutUint16(data[150:], 2)
	binary.LittleEndian.PutUint16(data[152:], 0x20b)
	binary.LittleEndian.PutUint32(data[260:], 16)
	path := filepath.Join(t.TempDir(), "OpenRGB.exe")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if f, err := pe.Open(path); err != nil {
		t.Fatal(err)
	} else {
		f.Close()
	}
	if !nativeExecutableFile(path, info, "windows", "amd64") {
		t.Fatal("native PE32+ was rejected for lacking Unix execute bits")
	}
	if nativeExecutableFile(path, info, "windows", "arm64") {
		t.Fatal("accepted another CPU ABI")
	}
	if nativeExecutableFile(path, info, "linux", "amd64") {
		t.Fatal("Linux lost its executable mode requirement")
	}
	binary.LittleEndian.PutUint16(data[150:], 0x2002)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if nativeExecutableFile(path, info, "windows", "amd64") {
		t.Fatal("DLL was accepted as process image")
	}
	if err := os.WriteFile(path, []byte("echo script"), 0o600); err != nil {
		t.Fatal(err)
	}
	if nativeExecutableFile(path, info, "windows", "amd64") {
		t.Fatal("non-PE script was accepted as native executable")
	}
}
