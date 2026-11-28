package msvcruntime

import (
	"debug/pe"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// The six external Microsoft providers imported by the exact OpenRGB/Qt
// release closure. Keep provenance/version and named-export admission together.
func requiredRuntimeLibraries() []string {
	return []string{"msvcp140.dll", "msvcp140_1.dll", "msvcp140_2.dll", "msvcp140_codecvt_ids.dll", "vcruntime140.dll", "vcruntime140_1.dll"}
}

func verifyBundleImports(bundle, system string) error {
	libraries := map[string]map[string]bool{}
	for _, name := range requiredRuntimeLibraries() {
		exports, err := nativeExports(filepath.Join(system, name))
		if err != nil {
			return err
		}
		libraries[name] = exports
	}
	return filepath.WalkDir(bundle, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".dll" && ext != ".exe" {
			return nil
		}
		f, err := pe.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		symbols, err := f.ImportedSymbols()
		if err != nil {
			return err
		}
		for _, symbol := range symbols {
			function, dll, ok := strings.Cut(symbol, ":")
			if !ok {
				continue
			}
			name := strings.ToLower(dll)
			if exports, ok := libraries[name]; ok {
				if !exports[function] {
					return fmt.Errorf("Microsoft runtime %s lacks required export %s for %s", dll, function, filepath.Base(path))
				}
			} else if strings.HasPrefix(name, "msvcp") || strings.HasPrefix(name, "msvcr") || strings.HasPrefix(name, "vcruntime") || strings.HasPrefix(name, "concrt") {
				return fmt.Errorf("Microsoft runtime provider %s is outside the verified native prerequisite profile", dll)
			}
		}
		return nil
	})
}

func nativeExports(path string) (map[string]bool, error) {
	f, err := pe.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	header, ok := f.OptionalHeader.(*pe.OptionalHeader64)
	if !ok || f.Machine != pe.IMAGE_FILE_MACHINE_AMD64 {
		return nil, errors.New("runtime export provider is not x64 PE32+")
	}
	table, err := peRVA(f, header.DataDirectory[0].VirtualAddress, 40)
	if err != nil {
		return nil, err
	}
	count := binary.LittleEndian.Uint32(table[24:])
	if count == 0 || count > 100000 {
		return nil, errors.New("runtime export table has invalid bound")
	}
	names, err := peRVA(f, binary.LittleEndian.Uint32(table[32:]), count*4)
	if err != nil {
		return nil, err
	}
	result := map[string]bool{}
	for i := uint32(0); i < count; i++ {
		data, err := peRVA(f, binary.LittleEndian.Uint32(names[i*4:]), 1)
		if err != nil {
			return nil, err
		}
		n := 0
		for n < len(data) && n < 4096 && data[n] != 0 {
			n++
		}
		if n == len(data) || n == 4096 {
			return nil, errors.New("unbounded runtime export name")
		}
		result[string(data[:n])] = true
	}
	return result, nil
}

func peRVA(f *pe.File, rva, size uint32) ([]byte, error) {
	for _, section := range f.Sections {
		if rva < section.VirtualAddress {
			continue
		}
		offset := rva - section.VirtualAddress
		if offset >= section.Size {
			continue
		}
		data, err := section.Data()
		if err != nil {
			return nil, err
		}
		if uint64(offset)+uint64(size) > uint64(len(data)) {
			return nil, errors.New("truncated runtime PE metadata")
		}
		return data[offset:], nil
	}
	return nil, errors.New("runtime PE metadata lies outside image")
}
