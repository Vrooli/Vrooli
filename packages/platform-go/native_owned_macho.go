package platform

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// Command layouts follow Apple's mach-o/loader.h, including LC_REQ_DYLD.
// https://github.com/apple-oss-distributions/cctools/blob/main/include/mach-o/loader.h
// debug/macho.ImportedLibraries only reports LC_LOAD_DYLIB. Do not use that
// convenience method as a complete dyld dependency/loader policy.
const darwinMachOMaxCommands = 4096
const darwinMachOMaxCommandBytes = 4 << 20

func darwinMachORead(r io.ReaderAt, offset, size uint64) ([]byte, error) {
	if offset > uint64(maxNativeSystemBrokerImageBytes) || size > darwinMachOMaxCommandBytes || size > uint64(maxNativeSystemBrokerImageBytes)-offset {
		return nil, ErrOperationIdentity
	}
	if size == 0 {
		return []byte{}, nil
	}
	b := make([]byte, int(size))
	if _, e := r.ReadAt(b, int64(offset)); e != nil {
		return nil, e
	}
	return b, nil
}

// MachOLoaderMetadata describes parsed loader facts, not a trust policy or an
// execution qualification. Consumers must qualify their own dependency roots.
type MachOLoaderMetadata struct{ Slices []MachOLoaderSlice }
type MachOLoaderSlice struct {
	CPU             uint32
	FileType        uint32
	Dylibs          []MachODylibLoad
	Dylinker        string
	RPaths          []string
	DyldEnvironment []string
	InstallName     string
}
type MachODylibLoad struct {
	Command uint32
	Flags   uint32
	Path    string
}

// ReadMachOLoaderMetadata reads only the explicit held-reader extent. It accepts
// 64-bit executable/dylib/bundle images, reports every supported dependency form
// without applying path policy, and refuses malformed or unparsed load forms.
func ReadMachOLoaderMetadata(reader io.ReaderAt, size int64) (MachOLoaderMetadata, error) {
	empty := MachOLoaderMetadata{}
	if reader == nil || size < 32 || size > maxNativeSystemBrokerImageBytes {
		return empty, ErrOperationIdentity
	}
	var last [1]byte
	if _, err := reader.ReadAt(last[:], size-1); err != nil {
		return empty, err
	}
	r := io.NewSectionReader(reader, 0, size)
	first, e := darwinMachORead(r, 0, 8)
	if e != nil {
		return empty, e
	}
	magic := binary.BigEndian.Uint32(first)
	if magic != 0xcafebabe && magic != 0xcafebabf {
		slice, e := readMachOLoaderSlice(r)
		if e != nil {
			return empty, e
		}
		return MachOLoaderMetadata{Slices: []MachOLoaderSlice{slice}}, nil
	}
	count := uint64(binary.BigEndian.Uint32(first[4:]))
	if count == 0 || count > 16 {
		return empty, ErrOperationIdentity
	}
	stride := uint64(20)
	if magic == 0xcafebabf {
		stride = 32
	}
	table, e := darwinMachORead(r, 8, count*stride)
	if e != nil {
		return empty, e
	}
	type span struct{ start, end uint64 }
	var spans []span
	seen := map[uint32]bool{}
	metadata := MachOLoaderMetadata{}
	for n := uint64(0); n < count; n++ {
		entry := table[n*stride : (n+1)*stride]
		cpu := binary.BigEndian.Uint32(entry)
		if cpu != 0x1000007 && cpu != 0x100000c {
			return empty, fmt.Errorf("%w: unsupported universal Mach-O architecture", ErrUnsupported)
		}
		if seen[cpu] {
			return empty, ErrOperationIdentity
		}
		seen[cpu] = true
		var offset, length uint64
		var align uint32
		if stride == 20 {
			offset = uint64(binary.BigEndian.Uint32(entry[8:]))
			length = uint64(binary.BigEndian.Uint32(entry[12:]))
			align = binary.BigEndian.Uint32(entry[16:])
		} else {
			offset = binary.BigEndian.Uint64(entry[8:])
			length = binary.BigEndian.Uint64(entry[16:])
			align = binary.BigEndian.Uint32(entry[24:])
			if binary.BigEndian.Uint32(entry[28:]) != 0 {
				return empty, ErrOperationIdentity
			}
		}
		if align > 31 || offset%(uint64(1)<<align) != 0 || offset < 8+count*stride || length < 32 || offset > uint64(size) || length > uint64(size)-offset {
			return empty, ErrOperationIdentity
		}
		for _, previous := range spans {
			if offset < previous.end && previous.start < offset+length {
				return empty, ErrOperationIdentity
			}
		}
		spans = append(spans, span{offset, offset + length})
		slice, e := readMachOLoaderSlice(io.NewSectionReader(r, int64(offset), int64(length)))
		if e != nil {
			return empty, e
		}
		if slice.CPU != cpu {
			return empty, ErrOperationIdentity
		}
		metadata.Slices = append(metadata.Slices, slice)
	}
	return metadata, nil
}
func validateDarwinBrokerMachO(r io.ReaderAt, architecture string) error {
	var wanted uint32
	switch architecture {
	case "amd64":
		wanted = 0x1000007
	case "arm64":
		wanted = 0x100000c
	default:
		return ErrUnsupported
	}
	var size int64
	switch source := r.(type) {
	case interface{ Size() int64 }:
		size = source.Size()
	case interface{ Stat() (os.FileInfo, error) }:
		st, e := source.Stat()
		if e != nil {
			return e
		}
		size = st.Size()
	default:
		return fmt.Errorf("%w: Mach-O reader extent unavailable", ErrUnsupported)
	}
	metadata, e := ReadMachOLoaderMetadata(r, size)
	if e != nil {
		return e
	}
	found := false
	for _, slice := range metadata.Slices {
		if slice.FileType != 2 {
			return fmt.Errorf("%w: maintained host must be a Mach-O executable", ErrOperationIdentity)
		}
		found = found || slice.CPU == wanted
		for _, library := range slice.Dylibs {
			if !darwinBrokerSystemLibrary(library.Path) {
				return fmt.Errorf("%w: non-system Mach-O dylib %q", ErrOperationIdentity, library.Path)
			}
		}
		if slice.Dylinker != "" && slice.Dylinker != "/usr/lib/dyld" {
			return fmt.Errorf("%w: unsupported Mach-O dynamic linker", ErrOperationIdentity)
		}
		for _, rpath := range slice.RPaths {
			if !darwinBrokerSystemLibrary(rpath) {
				return fmt.Errorf("%w: non-system Mach-O rpath", ErrOperationIdentity)
			}
		}
		if len(slice.DyldEnvironment) != 0 {
			return fmt.Errorf("%w: embedded dyld environment is unsupported", ErrUnsupported)
		}
	}
	if !found {
		return ErrOperationIdentity
	}
	return nil
}
func readMachOLoaderSlice(r io.ReaderAt) (MachOLoaderSlice, error) {
	header, e := darwinMachORead(r, 0, 32)
	if e != nil {
		return MachOLoaderSlice{}, e
	}
	var order binary.ByteOrder
	switch binary.LittleEndian.Uint32(header) {
	case 0xfeedfacf:
		order = binary.LittleEndian
	case 0xcffaedfe:
		order = binary.BigEndian
	default:
		return MachOLoaderSlice{}, fmt.Errorf("%w: only supported 64-bit Mach-O images are parsed", ErrUnsupported)
	}
	cpu := order.Uint32(header[4:])
	fileType := order.Uint32(header[12:])
	if cpu != 0x1000007 && cpu != 0x100000c || fileType != 2 && fileType != 6 && fileType != 8 || order.Uint32(header[28:]) != 0 {
		return MachOLoaderSlice{}, ErrOperationIdentity
	}
	count, size := order.Uint32(header[16:]), order.Uint32(header[20:])
	if count > darwinMachOMaxCommands || size > darwinMachOMaxCommandBytes || uint64(count)*8 > uint64(size) {
		return MachOLoaderSlice{}, ErrOperationIdentity
	}
	commands, e := darwinMachORead(r, 32, uint64(size))
	if e != nil {
		return MachOLoaderSlice{}, e
	}
	position := uint64(0)
	result := MachOLoaderSlice{CPU: cpu, FileType: fileType}
	for n := uint32(0); n < count; n++ {
		if position+8 > uint64(len(commands)) {
			return MachOLoaderSlice{}, ErrOperationIdentity
		}
		command := order.Uint32(commands[position:])
		length := uint64(order.Uint32(commands[position+4:]))
		if length < 8 || length%8 != 0 || length > uint64(len(commands))-position {
			return MachOLoaderSlice{}, ErrOperationIdentity
		}
		raw := commands[position : position+length]
		position += length
		exact := func(want uint64) bool { return length == want }
		switch command {
		case 0xc, 0x80000018, 0x8000001f, 0x80000023, 0x20: // load/weak/reexport/upward/lazy dylib
			minimum := uint32(24)
			flags := uint32(0)
			if len(raw) >= 24 && order.Uint32(raw[12:]) == 0x1a741800 {
				// Apple's macOS15+ dylib_use_command alternate encoding.
				if command != 0xc && command != 0x80000018 || len(raw) < 28 || order.Uint32(raw[8:]) != 28 {
					return MachOLoaderSlice{}, ErrUnsupported
				}
				minimum = 28
				flags = order.Uint32(raw[24:])
				if flags&^uint32(15) != 0 {
					return MachOLoaderSlice{}, fmt.Errorf("%w: unknown Mach-O dylib-use flags", ErrUnsupported)
				}
			}
			name, e := darwinMachOCommandString(raw, order, minimum)
			if e != nil {
				return MachOLoaderSlice{}, e
			}
			result.Dylibs = append(result.Dylibs, MachODylibLoad{Command: command, Flags: flags, Path: name})
		case 0xd: // LC_ID_DYLIB is identity metadata, not an imported dependency.
			name, e := darwinMachOCommandString(raw, order, 24)
			if e != nil {
				return MachOLoaderSlice{}, e
			}
			if fileType != 6 || result.InstallName != "" {
				return MachOLoaderSlice{}, ErrOperationIdentity
			}
			result.InstallName = name

		case 0xe: // LC_LOAD_DYLINKER
			name, e := darwinMachOCommandString(raw, order, 12)
			if e != nil {
				return MachOLoaderSlice{}, e
			}
			if result.Dylinker != "" {
				return MachOLoaderSlice{}, ErrOperationIdentity
			}
			result.Dylinker = name

		case 0x8000001c: // LC_RPATH
			name, e := darwinMachOCommandString(raw, order, 12)
			if e != nil {
				return MachOLoaderSlice{}, e
			}
			result.RPaths = append(result.RPaths, name)
		case 0x27: // LC_DYLD_ENVIRONMENT is reported; each consumer owns its policy.
			value, e := darwinMachOCommandString(raw, order, 12)
			if e != nil {
				return MachOLoaderSlice{}, e
			}
			result.DyldEnvironment = append(result.DyldEnvironment, value)

		case 0x19: // LC_SEGMENT_64, followed only by section_64 records.
			if length < 72 || uint64(order.Uint32(raw[64:])) > 4096 || length != 72+uint64(order.Uint32(raw[64:]))*80 {
				return MachOLoaderSlice{}, ErrOperationIdentity
			}
		case 0x2:
			if !exact(24) {
				return MachOLoaderSlice{}, ErrOperationIdentity
			} // LC_SYMTAB
		case 0xb:
			if !exact(80) {
				return MachOLoaderSlice{}, ErrOperationIdentity
			} // LC_DYSYMTAB
		case 0x4, 0x5: // thread state records, bounded inside this command; no external paths.
			at := uint64(8)
			for at < length {
				if at+8 > length {
					return MachOLoaderSlice{}, ErrOperationIdentity
				}
				words := uint64(order.Uint32(raw[at+4:]))
				if words == 0 || words > (length-at-8)/4 {
					return MachOLoaderSlice{}, ErrOperationIdentity
				}
				at += 8 + words*4
			}
			if at != length {
				return MachOLoaderSlice{}, ErrOperationIdentity
			}
		case 0x1a:
			if !exact(72) {
				return MachOLoaderSlice{}, ErrOperationIdentity
			} // LC_ROUTINES_64
		case 0x1b, 0x80000028:
			if !exact(24) {
				return MachOLoaderSlice{}, ErrOperationIdentity
			} // UUID / MAIN
		case 0x1d, 0x1e, 0x26, 0x29, 0x2a, 0x2b, 0x2e, 0x80000033, 0x80000034, 0x36:
			if !exact(16) {
				return MachOLoaderSlice{}, ErrOperationIdentity
			} // known linkedit data / source version
		case 0x22, 0x80000022:
			if !exact(48) {
				return MachOLoaderSlice{}, ErrOperationIdentity
			} // LC_DYLD_INFO[_ONLY]
		case 0x24:
			if !exact(16) {
				return MachOLoaderSlice{}, ErrOperationIdentity
			} // LC_VERSION_MIN_MACOSX
		case 0x2c:
			if !exact(24) || order.Uint32(raw[16:]) != 0 {
				return MachOLoaderSlice{}, fmt.Errorf("%w: encrypted Mach-O is unsupported", ErrUnsupported)
			}
		case 0x31:
			if !exact(40) {
				return MachOLoaderSlice{}, ErrOperationIdentity
			} // LC_NOTE
		case 0x32: // LC_BUILD_VERSION: macOS platform and bounded build-tool records.
			if length < 24 || order.Uint32(raw[8:]) != 1 || uint64(order.Uint32(raw[20:])) > 1024 || length != 24+uint64(order.Uint32(raw[20:]))*8 {
				return MachOLoaderSlice{}, fmt.Errorf("%w: unsupported Mach-O build platform/form", ErrUnsupported)
			}
		case 0x39: // LC_TARGET_TRIPLE is bounded informational text, not a loader path.
			if _, e := darwinMachOCommandString(raw, order, 12); e != nil {
				return MachOLoaderSlice{}, e
			}
		default:
			// Includes fixed-VM/prebound/sub-library/fileset/embedded lazy-load-info
			// forms and future commands. No unparsed load-affecting command is proof.
			return MachOLoaderSlice{}, fmt.Errorf("%w: unsupported Mach-O load command %#x", ErrUnsupported, command)
		}
	}
	if position != uint64(len(commands)) {
		return MachOLoaderSlice{}, ErrOperationIdentity
	}
	return result, nil
}
func darwinMachOCommandString(raw []byte, order binary.ByteOrder, minimum uint32) (string, error) {
	if len(raw) < int(minimum) {
		return "", ErrOperationIdentity
	}
	offset := order.Uint32(raw[8:])
	if offset < minimum || uint64(offset) >= uint64(len(raw)) {
		return "", ErrOperationIdentity
	}
	for _, value := range raw[minimum:offset] {
		if value != 0 {
			return "", ErrOperationIdentity
		}
	}
	tail := raw[offset:]
	end := bytes.IndexByte(tail, 0)
	if end <= 0 || end > 4096 {
		return "", ErrOperationIdentity
	}
	for _, value := range tail[end+1:] {
		if value != 0 {
			return "", ErrOperationIdentity
		}
	}
	for _, value := range tail[:end] {
		if value < 32 || value == 127 {
			return "", ErrOperationIdentity
		}
	}
	return string(tail[:end]), nil
}
