//go:build !windows

package platform

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func darwinMachOPathCommand(command uint32, name string, minimum int) []byte {
	size := (minimum + len(name) + 1 + 7) &^ 7
	raw := make([]byte, size)
	binary.LittleEndian.PutUint32(raw, command)
	binary.LittleEndian.PutUint32(raw[4:], uint32(size))
	binary.LittleEndian.PutUint32(raw[8:], uint32(minimum))
	copy(raw[minimum:], name)
	return raw
}
func darwinMachOCommands(commands ...[]byte) []byte {
	size := 0
	for _, command := range commands {
		size += len(command)
	}
	image := make([]byte, 32+size)
	for index, value := range []uint32{0xfeedfacf, 0x100000c, 0, 2, uint32(len(commands)), uint32(size), 0, 0} {
		binary.LittleEndian.PutUint32(image[index*4:], value)
	}
	at := 32
	for _, command := range commands {
		copy(image[at:], command)
		at += len(command)
	}
	return image
}
func TestDarwinMachOClosureEveryDylibLoaderForm(t *testing.T) {
	for _, command := range []uint32{0xc, 0x80000018, 0x8000001f, 0x80000023, 0x20} {
		for _, name := range []string{"/usr/lib/libSystem.B.dylib", "/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation"} {
			if err := validateDarwinBrokerMachO(bytes.NewReader(darwinMachOCommands(darwinMachOPathCommand(command, name, 24))), "arm64"); err != nil {
				t.Fatalf("system loader %#x refused: %v", command, err)
			}
		}
		for _, name := range []string{"/Users/operator/evil.dylib", "@rpath/evil.dylib", "@loader_path/evil.dylib", "/usr/lib/../local/evil.dylib"} {
			if err := validateDarwinBrokerMachO(bytes.NewReader(darwinMachOCommands(darwinMachOPathCommand(command, name, 24))), "arm64"); err == nil {
				t.Fatalf("foreign loader %#x admitted %s", command, name)
			}
		}
	}
}
func TestDarwinMachOClosureDynamicLinkerRPathAndEnvironment(t *testing.T) {
	if err := validateDarwinBrokerMachO(bytes.NewReader(darwinMachOCommands(darwinMachOPathCommand(0xe, "/usr/lib/dyld", 12), darwinMachOPathCommand(0x8000001c, "/System/Library/Frameworks", 12))), "arm64"); err != nil {
		t.Fatal(err)
	}
	for _, command := range [][]byte{darwinMachOPathCommand(0xe, "/Users/operator/dyld", 12), darwinMachOPathCommand(0x8000001c, "@executable_path/Frameworks", 12), darwinMachOPathCommand(0x27, "DYLD_LIBRARY_PATH=/usr/lib", 12), darwinMachOPathCommand(0x27, "DYLD_INSERT_LIBRARIES=/Users/operator/evil.dylib", 12)} {
		if err := validateDarwinBrokerMachO(bytes.NewReader(darwinMachOCommands(command)), "arm64"); err == nil {
			t.Fatal("loader/environment override admitted")
		}
	}
	if err := validateDarwinBrokerMachO(bytes.NewReader(darwinMachOCommands(darwinMachOPathCommand(0xe, "/usr/lib/dyld", 12), darwinMachOPathCommand(0xe, "/usr/lib/dyld", 12))), "arm64"); err == nil {
		t.Fatal("ambiguous multiple dynamic linkers admitted")
	}
}
func TestDarwinMachOClosureRejectsMalformedAndUnparsedCommands(t *testing.T) {
	for _, kind := range []string{"offset-header", "offset-end", "unterminated", "nonzero-padding", "count-overflow", "size-overflow", "zero-command", "trailing-command-bytes", "unknown-required", "obsolete-loader", "fileset", "lazy-load-info"} {
		t.Run(kind, func(t *testing.T) {
			raw := darwinMachOCommands(darwinMachOPathCommand(0xc, "/usr/lib/ok", 24))
			switch kind {
			case "offset-header":
				binary.LittleEndian.PutUint32(raw[40:], 8)
			case "offset-end":
				binary.LittleEndian.PutUint32(raw[40:], uint32(len(raw)-32))
			case "unterminated":
				for n := 56; n < len(raw); n++ {
					raw[n] = 'x'
				}
			case "nonzero-padding":
				raw[len(raw)-1] = 1
			case "count-overflow":
				binary.LittleEndian.PutUint32(raw[16:], 0xffffffff)
			case "size-overflow":
				binary.LittleEndian.PutUint32(raw[20:], 0xffffffff)
			case "zero-command":
				binary.LittleEndian.PutUint32(raw[36:], 0)
			case "trailing-command-bytes":
				binary.LittleEndian.PutUint32(raw[16:], 0)
			case "unknown-required":
				binary.LittleEndian.PutUint32(raw[32:], 0x80007777)
			case "obsolete-loader":
				binary.LittleEndian.PutUint32(raw[32:], 0x6)
			case "fileset":
				binary.LittleEndian.PutUint32(raw[32:], 0x80000035)
			case "lazy-load-info":
				binary.LittleEndian.PutUint32(raw[32:], 0x3a)
			}
			if err := validateDarwinBrokerMachO(bytes.NewReader(raw), "arm64"); err == nil {
				t.Fatal("unqualified command/form admitted", kind)
			}
		})
	}
}
func darwinMachOFatFixture(fat64 bool, images ...[]byte) []byte {
	stride := 20
	magic := uint32(0xcafebabe)
	if fat64 {
		stride = 32
		magic = 0xcafebabf
	}
	result := make([]byte, 4096*(len(images)+1))
	binary.BigEndian.PutUint32(result, magic)
	binary.BigEndian.PutUint32(result[4:], uint32(len(images)))
	for n, image := range images {
		entry := result[8+n*stride : 8+(n+1)*stride]
		binary.BigEndian.PutUint32(entry, binary.LittleEndian.Uint32(image[4:]))
		if fat64 {
			binary.BigEndian.PutUint64(entry[8:], uint64((n+1)*4096))
			binary.BigEndian.PutUint64(entry[16:], uint64(len(image)))
			binary.BigEndian.PutUint32(entry[24:], 12)
		} else {
			binary.BigEndian.PutUint32(entry[8:], uint32((n+1)*4096))
			binary.BigEndian.PutUint32(entry[12:], uint32(len(image)))
			binary.BigEndian.PutUint32(entry[16:], 12)
		}
		copy(result[(n+1)*4096:], image)
	}
	return result
}
func TestDarwinMachOClosureChecksEveryUniversalSlice(t *testing.T) {
	arm := darwinMachOCommands(darwinMachOPathCommand(0xc, "/usr/lib/libSystem.B.dylib", 24))
	amd := append([]byte(nil), arm...)
	binary.LittleEndian.PutUint32(amd[4:], 0x1000007)
	for _, fat64 := range []bool{false, true} {
		if err := validateDarwinBrokerMachO(bytes.NewReader(darwinMachOFatFixture(fat64, amd, arm)), "arm64"); err != nil {
			t.Fatal("qualified universal image refused", fat64, err)
		}
		foreign := darwinMachOCommands(darwinMachOPathCommand(0x80000018, "/Users/operator/evil.dylib", 24))
		binary.LittleEndian.PutUint32(foreign[4:], 0x1000007)
		if err := validateDarwinBrokerMachO(bytes.NewReader(darwinMachOFatFixture(fat64, foreign, arm)), "arm64"); err == nil {
			t.Fatal("non-selected universal slice escaped closure policy", fat64)
		}
	}
	bad := darwinMachOFatFixture(false, amd, arm)
	binary.BigEndian.PutUint32(bad[36:], 4096)
	if err := validateDarwinBrokerMachO(bytes.NewReader(bad), "arm64"); err == nil {
		t.Fatal("overlapping universal slices admitted")
	}
}
func TestDarwinMachOLoaderMetadataReportsVendorFactsWithoutHostPolicy(t *testing.T) {
	for _, fileType := range []uint32{2, 6, 8} {
		commands := [][]byte{darwinMachOPathCommand(0x80000018, "@rpath/QtCore.framework/Versions/5/QtCore", 24), darwinMachOPathCommand(0x8000001c, "@loader_path/../Frameworks", 12), darwinMachOPathCommand(0x27, "DYLD_FRAMEWORK_PATH=/sealed/frameworks", 12)}
		if fileType == 6 {
			commands = append(commands, darwinMachOPathCommand(0xd, "@rpath/QtCore.framework/Versions/5/QtCore", 24))
		}
		image := darwinMachOCommands(commands...)
		binary.LittleEndian.PutUint32(image[12:], fileType)
		metadata, err := ReadMachOLoaderMetadata(bytes.NewReader(image), int64(len(image)))
		if err != nil {
			t.Fatal("metadata imposed maintained-host path policy", fileType, err)
		}
		if len(metadata.Slices) != 1 {
			t.Fatal(metadata)
		}
		slice := metadata.Slices[0]
		if slice.CPU != 0x100000c || slice.FileType != fileType || len(slice.Dylibs) != 1 || slice.Dylibs[0].Command != 0x80000018 || slice.Dylibs[0].Path != "@rpath/QtCore.framework/Versions/5/QtCore" || len(slice.RPaths) != 1 || slice.RPaths[0] != "@loader_path/../Frameworks" || len(slice.DyldEnvironment) != 1 || slice.DyldEnvironment[0] != "DYLD_FRAMEWORK_PATH=/sealed/frameworks" {
			t.Fatalf("loader facts lost: %+v", slice)
		}
		if fileType == 6 && slice.InstallName != "@rpath/QtCore.framework/Versions/5/QtCore" {
			t.Fatal("install identity lost", slice)
		}
		if err = validateDarwinBrokerMachO(bytes.NewReader(image), "arm64"); err == nil {
			t.Fatal("maintained-host policy stopped rejecting vendor/embedded-env image")
		}
	}
}
func TestDarwinMachOLoaderMetadataHonorsExactReaderExtent(t *testing.T) {
	image := darwinMachOCommands(darwinMachOPathCommand(0xc, "/usr/lib/libSystem.B.dylib", 24))
	for _, size := range []int64{-1, 0, 31, int64(len(image) - 1), int64(len(image) + 1), maxNativeSystemBrokerImageBytes + 1} {
		if _, err := ReadMachOLoaderMetadata(bytes.NewReader(image), size); err == nil {
			t.Fatal("invalid reader extent admitted", size)
		}
	}
}
func TestDarwinMachOLoaderMetadataDylibUseEncoding(t *testing.T) {
	for _, command := range []uint32{0xc, 0x80000018} {
		for _, flags := range []uint32{0, 1, 2, 4, 8, 15} {
			raw := darwinMachOPathCommand(command, "@rpath/QtCore", 28)
			binary.LittleEndian.PutUint32(raw[12:], 0x1a741800)
			binary.LittleEndian.PutUint32(raw[24:], flags)
			image := darwinMachOCommands(raw)
			metadata, err := ReadMachOLoaderMetadata(bytes.NewReader(image), int64(len(image)))
			if err != nil || metadata.Slices[0].Dylibs[0].Flags != flags {
				t.Fatalf("dylib-use facts lost %#x %#x: %+v %v", command, flags, metadata, err)
			}
		}
	}
	raw := darwinMachOPathCommand(0xc, "/usr/lib/libSystem.B.dylib", 28)
	binary.LittleEndian.PutUint32(raw[12:], 0x1a741800)
	binary.LittleEndian.PutUint32(raw[24:], 16)
	image := darwinMachOCommands(raw)
	if _, err := ReadMachOLoaderMetadata(bytes.NewReader(image), int64(len(image))); err == nil {
		t.Fatal("unknown dylib-use flags admitted")
	}
}
