package resources

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// writeELFWithNeeded writes a minimal but format-accurate ELF64 whose dynamic
// section declares the given DT_NEEDED libraries.
//
// It is hand-assembled rather than compiled so the fixture is deterministic and
// needs no C toolchain: the check reads DT_NEEDED entries, so a fixture that
// carries exactly those is the honest thing to test against. Passing no
// libraries produces an ELF with no dynamic section at all, which is what a
// statically linked artifact looks like.
func writeELFWithNeeded(t *testing.T, dir string, libraries ...string) string {
	t.Helper()

	// .dynstr: a NUL-terminated string table, offset 0 reserved for "".
	var dynstr bytes.Buffer
	dynstr.WriteByte(0)
	offsets := make([]uint64, 0, len(libraries))
	for _, library := range libraries {
		offsets = append(offsets, uint64(dynstr.Len()))
		dynstr.WriteString(library)
		dynstr.WriteByte(0)
	}

	// .dynamic: one Elf64_Dyn per DT_NEEDED, terminated by DT_NULL.
	var dynamic bytes.Buffer
	for _, offset := range offsets {
		_ = binary.Write(&dynamic, binary.LittleEndian, uint64(1)) // DT_NEEDED
		_ = binary.Write(&dynamic, binary.LittleEndian, offset)
	}
	_ = binary.Write(&dynamic, binary.LittleEndian, uint64(0)) // DT_NULL
	_ = binary.Write(&dynamic, binary.LittleEndian, uint64(0))

	const (
		headerSize      = 64
		sectionSize     = 64
		sectionCount    = 4 // null, .shstrtab, .dynstr, .dynamic
		sectionStrIndex = 1
		dynstrIndex     = 2
		dynamicIndex    = 3
	)

	// .shstrtab: section header names.
	var shstrtab bytes.Buffer
	shstrtab.WriteByte(0)
	nameOffset := func(name string) uint32 {
		offset := uint32(shstrtab.Len())
		shstrtab.WriteString(name)
		shstrtab.WriteByte(0)
		return offset
	}
	shstrtabName := nameOffset(".shstrtab")
	dynstrName := nameOffset(".dynstr")
	dynamicName := nameOffset(".dynamic")

	shstrtabOffset := uint64(headerSize)
	dynstrOffset := shstrtabOffset + uint64(shstrtab.Len())
	dynamicOffset := dynstrOffset + uint64(dynstr.Len())
	sectionTableOffset := dynamicOffset + uint64(dynamic.Len())

	var out bytes.Buffer

	// Elf64_Ehdr: a little-endian x86-64 executable.
	out.Write([]byte{0x7f, 'E', 'L', 'F', 2, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	_ = binary.Write(&out, binary.LittleEndian, uint16(2))  // ET_EXEC
	_ = binary.Write(&out, binary.LittleEndian, uint16(62)) // EM_X86_64
	_ = binary.Write(&out, binary.LittleEndian, uint32(1))  // EV_CURRENT
	_ = binary.Write(&out, binary.LittleEndian, uint64(0))  // e_entry
	_ = binary.Write(&out, binary.LittleEndian, uint64(0))  // e_phoff
	_ = binary.Write(&out, binary.LittleEndian, sectionTableOffset)
	_ = binary.Write(&out, binary.LittleEndian, uint32(0))               // e_flags
	_ = binary.Write(&out, binary.LittleEndian, uint16(headerSize))      // e_ehsize
	_ = binary.Write(&out, binary.LittleEndian, uint16(0))               // e_phentsize
	_ = binary.Write(&out, binary.LittleEndian, uint16(0))               // e_phnum
	_ = binary.Write(&out, binary.LittleEndian, uint16(sectionSize))     // e_shentsize
	_ = binary.Write(&out, binary.LittleEndian, uint16(sectionCount))    // e_shnum
	_ = binary.Write(&out, binary.LittleEndian, uint16(sectionStrIndex)) // e_shstrndx

	out.Write(shstrtab.Bytes())
	out.Write(dynstr.Bytes())
	out.Write(dynamic.Bytes())

	writeSection := func(name uint32, kind, flags, offset, size, link, entsize uint64) {
		_ = binary.Write(&out, binary.LittleEndian, name)
		_ = binary.Write(&out, binary.LittleEndian, uint32(kind))
		_ = binary.Write(&out, binary.LittleEndian, flags)
		_ = binary.Write(&out, binary.LittleEndian, uint64(0)) // sh_addr
		_ = binary.Write(&out, binary.LittleEndian, offset)
		_ = binary.Write(&out, binary.LittleEndian, size)
		_ = binary.Write(&out, binary.LittleEndian, uint32(link))
		_ = binary.Write(&out, binary.LittleEndian, uint32(0)) // sh_info
		_ = binary.Write(&out, binary.LittleEndian, uint64(8)) // sh_addralign
		_ = binary.Write(&out, binary.LittleEndian, entsize)
	}

	writeSection(0, 0, 0, 0, 0, 0, 0)                                                      // SHT_NULL
	writeSection(shstrtabName, 3, 0, shstrtabOffset, uint64(shstrtab.Len()), 0, 0)         // SHT_STRTAB
	writeSection(dynstrName, 3, 2, dynstrOffset, uint64(dynstr.Len()), 0, 0)               // SHT_STRTAB
	writeSection(dynamicName, 6, 3, dynamicOffset, uint64(dynamic.Len()), dynstrIndex, 16) // SHT_DYNAMIC

	path := filepath.Join(dir, "fixture-artifact")
	if err := os.WriteFile(path, out.Bytes(), 0o755); err != nil {
		t.Fatalf("write ELF fixture: %v", err)
	}
	return path
}
