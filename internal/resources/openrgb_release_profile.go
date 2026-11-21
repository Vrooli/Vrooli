package resources

import (
	"bytes"
	"context"
	"crypto/sha256"
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	platform "github.com/vrooli/platform-go"
)

// These profiles pin complete official 1.1 release envelopes and their closed
// extracted trees. Artifact verification is independent of runtime or hardware
// qualification, which the native launch/device owners must establish.
type openRGBReleaseProfile struct {
	OS, Arch, Format, ArchiveRoot, Payload, EnvelopeSHA256, TreeSHA256, PayloadSHA256 string
	DownloadBytes                                                                     int64
	SquashFSOffset                                                                    int64
	MinimumDarwinVersion                                                              uint32
	MinimumWindowsBuild                                                               uint32
	ServingTreeSHA256, ServingRecipeID, ServingRecipeSHA256                           string
}

func openRGBReleaseProfileFor(goos, arch string) (openRGBReleaseProfile, error) {
	p := openRGBReleaseProfile{OS: goos, Arch: arch}
	switch goos + "-" + arch {
	case "linux-amd64":
		p.Format = "appimage"
		p.Payload = openRGBAppDirPayload
		p.EnvelopeSHA256 = openRGBAppImageSHA256
		p.TreeSHA256 = openRGBAppDirSHA256
		p.PayloadSHA256 = openRGBPayloadSHA256
		p.DownloadBytes = 37528768
		p.SquashFSOffset = 193728
	case "linux-arm64":
		p.TreeSHA256 = "b9437d81927e8eed7c4e9aa5b5f35018c93313601317c3749ae42006941f2cc7"
		p.PayloadSHA256 = "d947ed41462a6cab99e8689b4ac45ebf32383131a8933ed6a8d3584748913ced"
		p.Format = "appimage"
		p.Payload = openRGBAppDirPayload
		p.EnvelopeSHA256 = "0fd6806c265f9dbd9b0a8847cf4b4cb172c86baf2e39f52f869f8c5325c6751b"
		p.DownloadBytes = 36602240
		p.SquashFSOffset = 196992
	case "darwin-amd64":
		p.ServingTreeSHA256 = "61ead4311d86af4c89fdbfd2868b474dae3a2242216b6a506bd295336f2e2519"
		p.ServingRecipeID = "openrgb1.1-darwin-headless-without-orphan-qtcore5compat-v1"
		p.ServingRecipeSHA256 = "8b03da61b917f96d22c2cbc9c35ba8921f0cdb0303be17cf38dd294c66300ec0"
		p.MinimumDarwinVersion = 15 << 16
		p.TreeSHA256 = "b82fd1306b490d012bce5705986919b599ec7568beb77d11b04c5f956b7540a2"
		p.PayloadSHA256 = "d8ff22e19f7ee320c43bb62373124a025a4511be1e8de5868389c7b9a92040a0"
		p.Format = "zip"
		p.ArchiveRoot = "OpenRGB.app"
		p.Payload = "Contents/MacOS/OpenRGB"
		p.EnvelopeSHA256 = "5b30d3c97bced335a40593319cb4c045f1ad818a0e0ddefd92350a695a7a6fc4"
		p.DownloadBytes = 47025932
	case "darwin-arm64":
		p.ServingTreeSHA256 = "b777a233d74db1250f2a100571bb3880fa8975cc6eca6ee71fc44efe3f2cdab5"
		p.ServingRecipeID = "openrgb1.1-darwin-headless-without-orphan-qtcore5compat-v1"
		p.ServingRecipeSHA256 = "879c9b9a299a8096d7dd968986ab8ca2e31ef4e98ae40ab94991d2f3d385ac81"
		p.MinimumDarwinVersion = 15 << 16
		p.TreeSHA256 = "613ff94ecaf7db3953e1e8262a7d8d013ec72b8c040c8c8844ffde7975585a64"
		p.PayloadSHA256 = "f06aeb3e974576f0b403d0db849e176c048ac0853ec4a3d3f7d13672c479f611"
		p.Format = "zip"
		p.ArchiveRoot = "OpenRGB.app"
		p.Payload = "Contents/MacOS/OpenRGB"
		p.EnvelopeSHA256 = "b370d3365eaae8b6a9217e7d1b1ffd5fb6ddffaaff3395fd3cac3673df83191f"
		p.DownloadBytes = 44436430
	case "windows-amd64":
		p.MinimumWindowsBuild = 17763 // Qt6.8: Windows10 1809 or later.
		p.ServingTreeSHA256 = "c2f9c9df90ef023a39b2d3c5bb06ec0645ebc64d43e21f8333e53e9721f39caf"
		p.ServingRecipeID = "openrgb1.1-windows-amd64-hid-without-pawnio-v1"
		p.ServingRecipeSHA256 = "153e7fb02bc4f89ab317787b6f5411b5dd2b1f2caac2e47fd23f4521e656ecbd"
		p.TreeSHA256 = "062cd360105968221df877550e8e1c6f4a2cf850df2a70b93d6fde9af62bdbe9"
		p.PayloadSHA256 = "e26ae71162795a8abcfb90c05c1572a00ba06ca63b886aca57392a0138963b1e"
		p.Format = "zip"
		p.ArchiveRoot = "OpenRGB Windows 64-bit"
		p.Payload = "OpenRGB.exe"
		p.EnvelopeSHA256 = "63291726c6fc84eb479ef2ee752e97e1187f6ce9c49fd27eee40ded588a8172a"
		p.DownloadBytes = 22019605
	default:
		return openRGBReleaseProfile{}, errOpenRGBReleasePayloadUnqualified
	}
	return p, nil
}

// Parse native headers without executing a release on the qualification host.
// A familiar filename/version is never sufficient evidence of architecture.
func (p openRGBReleaseProfile) verifyPayloadABI(path string) error {
	return p.verifyImageABI(path, true)
}

func (p openRGBReleaseProfile) verifyImageABI(path string, executable bool) error {
	switch p.OS {
	case "linux":
		f, err := elf.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		machine := elf.EM_X86_64
		if p.Arch == "arm64" {
			machine = elf.EM_AARCH64
		}
		if f.Class != elf.ELFCLASS64 || f.Data != elf.ELFDATA2LSB || f.Machine != machine || f.Type != elf.ET_DYN && f.Type != elf.ET_EXEC {
			return errors.New("OpenRGB ELF ABI differs from release target")
		}
	case "darwin":
		image, err := os.Open(path)
		if err != nil {
			return err
		}
		defer image.Close()
		info, err := image.Stat()
		if err != nil {
			return err
		}
		metadata, err := platform.ReadMachOLoaderMetadata(image, info.Size())
		if err != nil || len(metadata.Slices) != 1 {
			return errors.Join(errors.New("OpenRGB Mach-O lacks complete native loader metadata"), err)
		}
		f, err := macho.NewFile(image)
		if err != nil {
			return err
		}
		cpu := macho.CpuAmd64
		subtype := uint32(3) // CPU_SUBTYPE_X86_64_ALL, as frozen in this release.
		if p.Arch == "arm64" {
			cpu = macho.CpuArm64
			subtype = 0 // CPU_SUBTYPE_ARM64_ALL; no arm64e profile is implied.
		}
		if f.Magic != macho.Magic64 || f.ByteOrder != binary.LittleEndian || f.Cpu != cpu || f.SubCpu != subtype || executable && f.Type != macho.TypeExec || !executable && f.Type != macho.TypeExec && f.Type != macho.TypeDylib && f.Type != macho.TypeBundle {
			return errors.New("OpenRGB Mach-O ABI differs from release target")
		}
		minimum, err := openRGBMachMinimumVersion(f)
		if err != nil {
			return err
		}
		if p.MinimumDarwinVersion == 0 || minimum > p.MinimumDarwinVersion {
			return errors.New("OpenRGB Mach-O requires a newer platform baseline than its frozen native profile")
		}

	case "windows":
		f, err := pe.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		if f.Machine != pe.IMAGE_FILE_MACHINE_AMD64 {
			return errors.New("OpenRGB PE ABI differs from release target")
		}
		if _, ok := f.OptionalHeader.(*pe.OptionalHeader64); !ok {
			return errors.New("OpenRGB requires PE32+ executable")
		}
		if executable && (f.Characteristics&pe.IMAGE_FILE_EXECUTABLE_IMAGE == 0 || f.Characteristics&pe.IMAGE_FILE_DLL != 0) {
			return errors.New("native image is not a process executable")
		}
	default:
		return errOpenRGBReleasePayloadUnqualified
	}
	return nil
}

func (p openRGBReleaseProfile) validate() error {
	if p.OS == "darwin" && p.MinimumDarwinVersion == 0 {
		return errors.New("native macOS profile lacks its platform baseline")
	}
	if p.OS == "darwin" {
		if !updateDigest(p.ServingTreeSHA256) || p.ServingTreeSHA256 == p.TreeSHA256 || p.ServingRecipeID != "openrgb1.1-darwin-headless-without-orphan-qtcore5compat-v1" || !updateDigest(p.macOmissionTreeSHA256()) {
			return errors.New("native Mac profile lacks its closed headless serving derivation")
		}
		recipe, err := p.servingRecipeBytes()
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(recipe)) != p.ServingRecipeSHA256 {
			return errors.Join(errors.New("native Mac serving recipe differs from its pinned identity"), err)
		}
	}
	if p.OS == "windows" && p.MinimumWindowsBuild == 0 {
		return errors.New("native Windows profile lacks its platform baseline")
	}
	if p.OS == "windows" {
		if !updateDigest(p.ServingTreeSHA256) || p.ServingTreeSHA256 == p.TreeSHA256 || p.ServingRecipeID != "openrgb1.1-windows-amd64-hid-without-pawnio-v1" {
			return errors.New("native Windows profile lacks its closed HID serving derivation")
		}
		recipe, err := p.servingRecipeBytes()
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(recipe)) != p.ServingRecipeSHA256 {
			return errors.Join(errors.New("native Windows serving recipe differs from its pinned identity"), err)
		}
	}
	if !updateDigest(p.EnvelopeSHA256) || !updateDigest(p.TreeSHA256) || !updateDigest(p.PayloadSHA256) || p.DownloadBytes <= 0 {
		return fmt.Errorf("OpenRGB %s/%s release closure is incomplete", p.OS, p.Arch)
	}
	return nil
}

// Adopted payload identity is admitted only alongside verified whole-bundle
// closure; this predicate never makes a standalone executable sufficient.
func (p openRGBReleaseProfile) acceptsArtifactSHA256(digest string) bool {
	return digest == p.EnvelopeSHA256 || digest == p.PayloadSHA256
}

func (p openRGBReleaseProfile) bundleDirectory() string {
	if p.OS == "darwin" {
		return "OpenRGB.app"
	}
	return openRGBSealedAppDir
}

func (p openRGBReleaseProfile) verifyBundleABI(ctx context.Context, root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		var magic [4]byte
		n, _ := f.Read(magic[:])
		f.Close()
		if n < 4 {
			return nil
		}
		native := p.OS == "linux" && bytes.Equal(magic[:], []byte{0x7f, 'E', 'L', 'F'}) || p.OS == "darwin" && bytes.Equal(magic[:], []byte{0xcf, 0xfa, 0xed, 0xfe}) || p.OS == "windows" && magic[0] == 'M' && magic[1] == 'Z'
		if native {
			return p.verifyImageABI(path, false)
		}
		return nil
	})
}

// LC_BUILD_VERSION platform1 is macOS. Do not confuse an ARM64 iOS/Catalyst
// image with a native macOS ABI just because its CPU header matches.
func openRGBMachMinimumVersion(image *macho.File) (uint32, error) {
	var minimum uint32
	for _, load := range image.Loads {
		raw := load.Raw()
		if len(raw) < 8 {
			continue
		}
		command := image.ByteOrder.Uint32(raw)
		switch command {
		case 0x32:
			if len(raw) < 24 || image.ByteOrder.Uint32(raw[8:]) != 1 {
				return 0, errors.New("OpenRGB Mach-O is not a native macOS image")
			}
			version := image.ByteOrder.Uint32(raw[12:])
			if version > minimum {
				minimum = version
			}
		case 0x24:
			if len(raw) < 16 {
				return 0, errors.New("truncated macOS version command")
			}
			version := image.ByteOrder.Uint32(raw[8:])
			if version > minimum {
				minimum = version
			}
		case 0x25, 0x2f, 0x30:
			return 0, errors.New("OpenRGB Mach-O targets another Apple OS")
		}
	}
	if minimum == 0 || image.ByteOrder != binary.LittleEndian {
		return 0, errors.New("OpenRGB Mach-O lacks a qualified native macOS version/byte order")
	}
	return minimum, nil
}
