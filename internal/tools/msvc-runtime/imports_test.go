package msvcruntime

import (
	"debug/pe"
	"encoding/binary"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// These are PE metadata fixtures only. Nothing in the test is loaded/executed,
// and this never substitutes for native signature/version qualification.
func runtimePEFixture(directory string) []byte {
	data := make([]byte, 0x200+1<<17)
	put16 := func(offset int, value uint16) { binary.LittleEndian.PutUint16(data[offset:], value) }
	put32 := func(offset int, value uint32) { binary.LittleEndian.PutUint32(data[offset:], value) }
	copy(data, "MZ")
	put32(0x3c, 0x80)
	copy(data[0x80:], "PE\x00\x00")
	put16(0x84, pe.IMAGE_FILE_MACHINE_AMD64)
	put16(0x86, 1)
	put16(0x94, 240)
	put16(0x98, 0x20b)
	put32(0x98+108, 16)
	index := 0
	if directory == ".idata" {
		index = 1
	}
	put32(0x98+112+index*8, 0x1000)
	put32(0x98+116+index*8, 40)
	section := 0x98 + 240
	copy(data[section:], directory)
	put32(section+8, 1<<17)
	put32(section+12, 0x1000)
	put32(section+16, 1<<17)
	put32(section+20, 0x200)
	return data
}

func writeRuntimeExports(t *testing.T, path string, names []string) {
	t.Helper()
	data := runtimePEFixture(".edata")
	binary.LittleEndian.PutUint32(data[0x200+24:], uint32(len(names)))
	binary.LittleEndian.PutUint32(data[0x200+32:], 0x1028)
	at := 0x228 + len(names)*4
	for index, name := range names {
		binary.LittleEndian.PutUint32(data[0x228+4*index:], uint32(at-0x200+0x1000))
		copy(data[at:], name)
		at += len(name) + 1
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeRuntimeImport(t *testing.T, path, provider, symbol string) {
	t.Helper()
	data := runtimePEFixture(".idata")
	binary.LittleEndian.PutUint32(data[0x200:], 0x1040)
	binary.LittleEndian.PutUint32(data[0x200+12:], 0x1060)
	binary.LittleEndian.PutUint32(data[0x200+16:], 0x1040)
	binary.LittleEndian.PutUint64(data[0x240:], 0x1080)
	copy(data[0x260:], provider)
	copy(data[0x282:], symbol)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestRuntimeExtensionProvidersMustExistAndSupplyRequiredExports(t *testing.T) {
	for _, extension := range []struct{ library, symbol string }{{"msvcp140_1.dll", "_Aligned_get_default_resource"}, {"msvcp140_2.dll", "__std_smf_hypot3f"}} {
		t.Run(extension.library, func(t *testing.T) {
			bundle, system := t.TempDir(), t.TempDir()
			writeRuntimeImport(t, filepath.Join(bundle, "Qt.dll"), extension.library, extension.symbol)
			for _, library := range requiredRuntimeLibraries() {
				writeRuntimeExports(t, filepath.Join(system, library), []string{"unneeded"})
			}
			if err := os.Remove(filepath.Join(system, extension.library)); err != nil {
				t.Fatal(err)
			}
			if err := verifyBundleImports(bundle, system); err == nil {
				t.Fatal("missing extension provider passed preflight")
			}
			writeRuntimeExports(t, filepath.Join(system, extension.library), []string{"wrong-export"})
			if err := verifyBundleImports(bundle, system); err == nil || !strings.Contains(err.Error(), extension.symbol) {
				t.Fatalf("wrong extension export passed preflight: %v", err)
			}
			writeRuntimeExports(t, filepath.Join(system, extension.library), []string{extension.symbol})
			if err := verifyBundleImports(bundle, system); err != nil {
				t.Fatalf("complete extension provider refused: %v", err)
			}
		})
	}
}

func TestRuntimeImportsRefuseUnverifiedMicrosoftProviderFamily(t *testing.T) {
	bundle, system := t.TempDir(), t.TempDir()
	for _, library := range requiredRuntimeLibraries() {
		writeRuntimeExports(t, filepath.Join(system, library), []string{"unneeded"})
	}
	writeRuntimeImport(t, filepath.Join(bundle, "Qt.dll"), "MSVCP140_NEW.dll", "unverified")
	if err := verifyBundleImports(bundle, system); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Fatalf("unknown provider silently ignored: %v", err)
	}
}

func TestActualOpenRGBMicrosoftProviderInventoryIsClosed(t *testing.T) {
	bundle := os.Getenv("VROOLI_OPENRGB_MSVC_BUNDLE")
	if bundle == "" {
		t.Skip("explicit retained Windows bundle metadata qualification only")
	}
	wanted := map[string]map[string]bool{}
	err := filepath.WalkDir(bundle, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".dll" && ext != ".exe" {
			return nil
		}
		image, err := pe.Open(path)
		if err != nil {
			return err
		}
		defer image.Close()
		imports, err := image.ImportedSymbols()
		if err != nil {
			return err
		}
		for _, imported := range imports {
			symbol, library, ok := strings.Cut(imported, ":")
			library = strings.ToLower(library)
			if !ok || !(strings.HasPrefix(library, "msvcp") || strings.HasPrefix(library, "msvcr") || strings.HasPrefix(library, "vcruntime") || strings.HasPrefix(library, "concrt")) {
				continue
			}
			if wanted[library] == nil {
				wanted[library] = map[string]bool{}
			}
			wanted[library][symbol] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(wanted) != 6 || !wanted["msvcp140_1.dll"]["_Aligned_get_default_resource"] || !wanted["msvcp140_2.dll"]["__std_smf_hypot3f"] {
		t.Fatalf("official Microsoft import inventory differs: %+v", wanted)
	}
	system := t.TempDir()
	for library, symbols := range wanted {
		var names []string
		for name := range symbols {
			names = append(names, name)
		}
		sort.Strings(names)
		writeRuntimeExports(t, filepath.Join(system, library), names)
	}
	if err := verifyBundleImports(bundle, system); err != nil {
		t.Fatalf("complete actual import inventory refused: %v", err)
	}
	for _, library := range []string{"msvcp140_1.dll", "msvcp140_2.dll"} {
		path := filepath.Join(system, library)
		original, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		if err := verifyBundleImports(bundle, system); err == nil {
			t.Fatalf("actual bundle accepted missing %s", library)
		}
		if err := os.WriteFile(path, original, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
