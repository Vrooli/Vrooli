package lifecycle

import (
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"fmt"
	"runtime"
)

// validateRetainedNativeExecutable excludes interpreter wrappers disguised as
// component binaries. Dynamic system libraries remain an owner compatibility
// obligation; this is format/architecture evidence, not a sandbox or health test.
func validateRetainedNativeExecutable(path string) error {
	switch runtime.GOOS {
	case "linux":
		f, err := elf.Open(path)
		if err != nil {
			return fmt.Errorf("retained executable is not native ELF: %w", err)
		}
		defer f.Close()
		machine := map[string]elf.Machine{"amd64": elf.EM_X86_64, "arm64": elf.EM_AARCH64, "386": elf.EM_386, "arm": elf.EM_ARM, "riscv64": elf.EM_RISCV}[runtime.GOARCH]
		if machine == elf.EM_NONE || f.Machine != machine || (f.Type != elf.ET_EXEC && f.Type != elf.ET_DYN) || f.Entry == 0 {
			return fmt.Errorf("retained ELF is not a host executable")
		}
		for _, p := range f.Progs {
			if p.Type == elf.PT_LOAD && p.Flags&elf.PF_X != 0 && p.Filesz > 0 {
				return nil
			}
		}
		return fmt.Errorf("retained ELF lacks an executable load segment")
	case "darwin":
		f, err := macho.Open(path)
		if err != nil {
			return fmt.Errorf("retained executable is not a supported native Mach-O: %w", err)
		}
		defer f.Close()
		cpu := map[string]macho.Cpu{"amd64": macho.CpuAmd64, "arm64": macho.CpuArm64}[runtime.GOARCH]
		if cpu == 0 || f.Cpu != cpu || f.Type != macho.TypeExec {
			return fmt.Errorf("retained Mach-O is not a host executable")
		}
		return nil
	case "windows":
		f, err := pe.Open(path)
		if err != nil {
			return fmt.Errorf("retained executable is not native PE: %w", err)
		}
		defer f.Close()
		machine := map[string]uint16{"amd64": pe.IMAGE_FILE_MACHINE_AMD64, "arm64": pe.IMAGE_FILE_MACHINE_ARM64, "386": pe.IMAGE_FILE_MACHINE_I386}[runtime.GOARCH]
		if machine == 0 || f.Machine != machine || f.Characteristics&pe.IMAGE_FILE_EXECUTABLE_IMAGE == 0 || f.Characteristics&pe.IMAGE_FILE_DLL != 0 {
			return fmt.Errorf("retained PE is not a host executable")
		}
		return nil
	default:
		return fmt.Errorf("native retained executable qualification is unsupported on %s", runtime.GOOS)
	}
}
