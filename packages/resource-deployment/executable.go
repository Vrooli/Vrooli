package resourcedeployment

import (
	"debug/pe"
	"encoding/binary"
	"os"
)

// Windows regular files have no POSIX execute mode bits. Require a native
// executable image for its target instead of treating every file as runnable.
func nativeExecutableFile(path string, info os.FileInfo, goos, arch string) bool {
	if info == nil || !info.Mode().IsRegular() {
		return false
	}
	if goos != "windows" {
		return info.Mode().Perm()&0o111 != 0
	}
	header, err := os.Open(path)
	if err != nil {
		return false
	}
	defer header.Close()
	var dos [64]byte
	if _, err = header.ReadAt(dos[:], 0); err != nil || dos[0] != 'M' || dos[1] != 'Z' {
		return false
	}
	var signature [4]byte
	if _, err = header.ReadAt(signature[:], int64(binary.LittleEndian.Uint32(dos[60:]))); err != nil || signature != [4]byte{'P', 'E', 0, 0} {
		return false
	}
	file, err := pe.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	if file.Characteristics&pe.IMAGE_FILE_EXECUTABLE_IMAGE == 0 || file.Characteristics&pe.IMAGE_FILE_DLL != 0 {
		return false
	}
	machine := uint16(pe.IMAGE_FILE_MACHINE_AMD64)
	if arch == "arm64" {
		machine = pe.IMAGE_FILE_MACHINE_ARM64
	} else if arch != "amd64" {
		return false
	}
	if file.Machine != machine {
		return false
	}
	_, ok := file.OptionalHeader.(*pe.OptionalHeader64)
	return ok
}
