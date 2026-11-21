package platform

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

// Verify the retained maintained Go host's native PE and finite static DLL
// closure from the SAME held file used for its image digest. debug/pe's
// ImportedLibraries is unimplemented in Go1.25 and must not be used as proof.
// PE import descriptors/RVAs follow Microsoft's PE/COFF specification:
// https://learn.microsoft.com/en-us/windows/win32/debug/pe-format#the-idata-section
func verifyWindowsMaintainedLauncherPE(image io.ReaderAt, architecture string) error {
	executable, e := pe.NewFile(image)
	if e != nil {
		return e
	}
	expected := uint16(pe.IMAGE_FILE_MACHINE_AMD64)
	if architecture == "arm64" {
		expected = pe.IMAGE_FILE_MACHINE_ARM64
	} else if architecture != "amd64" {
		return ErrUnsupported
	}
	optional, ok := executable.OptionalHeader.(*pe.OptionalHeader64)
	if !ok || executable.Machine != expected || executable.Characteristics&pe.IMAGE_FILE_EXECUTABLE_IMAGE == 0 || executable.Characteristics&pe.IMAGE_FILE_DLL != 0 {
		return ErrOperationIdentity
	}
	// The maintained host does not use a delay-load helper or managed CLR entry.
	// Those loader forms need separate closure qualification, not empty success.
	if optional.NumberOfRvaAndSizes > 16 {
		return ErrOperationIdentity
	}
	if optional.NumberOfRvaAndSizes > 13 && (optional.DataDirectory[13].VirtualAddress != 0 || optional.DataDirectory[13].Size != 0) {
		return ErrOperationIdentity
	}
	if optional.NumberOfRvaAndSizes > 14 && (optional.DataDirectory[14].VirtualAddress != 0 || optional.DataDirectory[14].Size != 0) {
		return ErrOperationIdentity
	}
	if optional.NumberOfRvaAndSizes <= 1 {
		return nil
	}
	directory := optional.DataDirectory[1]
	if directory.VirtualAddress == 0 && directory.Size == 0 {
		return nil
	}
	if directory.VirtualAddress == 0 || directory.Size < 20 || directory.Size > 64<<10 {
		return ErrOperationIdentity
	}
	if len(executable.Sections) > 96 {
		return ErrOperationIdentity
	}
	for n, section := range executable.Sections {
		span := uint64(section.VirtualSize)
		if span < uint64(section.Size) {
			span = uint64(section.Size)
		}
		end := uint64(section.VirtualAddress) + span
		if end > 1<<32 {
			return ErrOperationIdentity
		}
		for _, other := range executable.Sections[:n] {
			otherSpan := uint64(other.VirtualSize)
			if otherSpan < uint64(other.Size) {
				otherSpan = uint64(other.Size)
			}
			if span > 0 && otherSpan > 0 && uint64(section.VirtualAddress) < uint64(other.VirtualAddress)+otherSpan && uint64(other.VirtualAddress) < end {
				return ErrOperationIdentity
			}
		}
	}
	// A mapped RVA must identify one unambiguous raw-backed section. Refuse
	// overlapping sections, virtual-only bytes, wraparound and truncated reads.
	readRVA := func(rva uint32, amount uint32) ([]byte, error) {
		if amount == 0 || uint64(rva)+uint64(amount) > 1<<32 {
			return nil, ErrOperationIdentity
		}
		var selected *pe.Section
		for _, section := range executable.Sections {
			if uint64(rva) >= uint64(section.VirtualAddress) && uint64(rva)+uint64(amount) <= uint64(section.VirtualAddress)+uint64(section.Size) {
				if selected != nil {
					return nil, ErrOperationIdentity
				}
				selected = section
			}
		}
		if selected == nil {
			return nil, ErrOperationIdentity
		}
		offset := uint64(selected.Offset) + uint64(rva) - uint64(selected.VirtualAddress)
		if offset+uint64(amount) > uint64(maxNativeSystemBrokerImageBytes) {
			return nil, ErrOperationIdentity
		}
		out := make([]byte, amount)
		if _, err := image.ReadAt(out, int64(offset)); err != nil {
			return nil, err
		}
		return out, nil
	}
	allowed := map[string]bool{"kernel32.dll": true, "ntdll.dll": true, "advapi32.dll": true, "userenv.dll": true, "ws2_32.dll": true, "iphlpapi.dll": true, "dnsapi.dll": true, "bcrypt.dll": true, "crypt32.dll": true, "secur32.dll": true, "netapi32.dll": true, "ole32.dll": true, "oleaut32.dll": true, "shell32.dll": true, "user32.dll": true, "msvcrt.dll": true}
	terminated := false
	for offset := uint32(0); offset+20 <= directory.Size; offset += 20 {
		rva := uint64(directory.VirtualAddress) + uint64(offset)
		if rva > uint64(^uint32(0)) {
			return ErrOperationIdentity
		}
		descriptor, err := readRVA(uint32(rva), 20)
		if err != nil {
			return err
		}
		if bytes.Equal(descriptor, make([]byte, 20)) {
			terminated = true
			break
		}
		nameRVA := binary.LittleEndian.Uint32(descriptor[12:16])
		if nameRVA == 0 || binary.LittleEndian.Uint32(descriptor[16:20]) == 0 {
			return ErrOperationIdentity
		}
		var name []byte
		for n := uint32(0); n < 256; n++ {
			if uint64(nameRVA)+uint64(n) > uint64(^uint32(0)) {
				return ErrOperationIdentity
			}
			b, err := readRVA(nameRVA+n, 1)
			if err != nil {
				return err
			}
			if b[0] == 0 {
				break
			}
			if b[0] < 0x21 || b[0] > 0x7e {
				return ErrOperationIdentity
			}
			name = append(name, b[0])
		}
		if len(name) == 0 || len(name) >= 256 {
			return ErrOperationIdentity
		}
		if !allowed[strings.ToLower(string(name))] {
			return fmt.Errorf("platform: broker dependency %q has no qualified System32 loader closure: %w", name, ErrOperationIdentity)
		}
	}
	if !terminated {
		return ErrOperationIdentity
	}
	return nil
}
