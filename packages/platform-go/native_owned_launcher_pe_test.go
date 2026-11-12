package platform

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"errors"
	"testing"
)

// A real PE32+ layout with one raw-backed .idata section and a complete import
// descriptor/name/terminator. Fixtures exercise actual PE/RVA parsing, not an
// injected list of DLL names (which hid Go's unimplemented ImportedLibraries).
func windowsLauncherPEFixture(library string) []byte {
	image := make([]byte, 1024)
	copy(image, "MZ")
	binary.LittleEndian.PutUint32(image[0x3c:], 0x80)
	copy(image[0x80:], "PE\x00\x00")
	header := image[0x84:]
	binary.LittleEndian.PutUint16(header, pe.IMAGE_FILE_MACHINE_AMD64)
	binary.LittleEndian.PutUint16(header[2:], 1)
	binary.LittleEndian.PutUint16(header[16:], 240)
	binary.LittleEndian.PutUint16(header[18:], pe.IMAGE_FILE_EXECUTABLE_IMAGE|pe.IMAGE_FILE_LARGE_ADDRESS_AWARE)
	optional := image[0x98:]
	binary.LittleEndian.PutUint16(optional, 0x20b)
	binary.LittleEndian.PutUint32(optional[32:], 4096)
	binary.LittleEndian.PutUint32(optional[36:], 512)
	binary.LittleEndian.PutUint32(optional[56:], 8192)
	binary.LittleEndian.PutUint32(optional[60:], 512)
	binary.LittleEndian.PutUint32(optional[108:], 16)
	section := image[0x188:]
	copy(section, ".idata")
	binary.LittleEndian.PutUint32(section[8:], 512)
	binary.LittleEndian.PutUint32(section[12:], 0x1000)
	binary.LittleEndian.PutUint32(section[16:], 512)
	binary.LittleEndian.PutUint32(section[20:], 512)
	if library != "" {
		binary.LittleEndian.PutUint32(optional[120:], 0x1000)
		binary.LittleEndian.PutUint32(optional[124:], 40)
		descriptor := image[512:]
		binary.LittleEndian.PutUint32(descriptor, 0x1050)
		binary.LittleEndian.PutUint32(descriptor[12:], 0x1080)
		binary.LittleEndian.PutUint32(descriptor[16:], 0x1060)
		copy(image[640:], library)
	}
	return image
}

func TestWindowsMaintainedLauncherPEReadsActualStaticDependencies(t *testing.T) {
	for _, library := range []string{"kernel32.dll", "KERNEL32.DLL", "advapi32.dll", "oleaut32.dll", ""} {
		if e := verifyWindowsMaintainedLauncherPE(bytes.NewReader(windowsLauncherPEFixture(library)), "amd64"); e != nil {
			t.Fatal(library, e)
		}
	}
	for _, library := range []string{"ambient.dll", "..\\kernel32.dll", "C:\\Windows\\System32\\kernel32.dll", "kernel32.dll ", "api-ms-unqualified.dll"} {
		if e := verifyWindowsMaintainedLauncherPE(bytes.NewReader(windowsLauncherPEFixture(library)), "amd64"); e == nil {
			t.Fatal("unqualified loader dependency accepted", library)
		}
	}
}
func TestWindowsMaintainedLauncherPERequiresNativeABIAndClosedLoaderForms(t *testing.T) {
	for name, mutate := range map[string]func([]byte){
		"x86":           func(b []byte) { binary.LittleEndian.PutUint16(b[0x84:], pe.IMAGE_FILE_MACHINE_I386) },
		"foreign arm64": func(b []byte) { binary.LittleEndian.PutUint16(b[0x84:], pe.IMAGE_FILE_MACHINE_ARM64) },
		"DLL instead of executable": func(b []byte) {
			binary.LittleEndian.PutUint16(b[0x84+18:], pe.IMAGE_FILE_EXECUTABLE_IMAGE|pe.IMAGE_FILE_DLL)
		},
		"not executable": func(b []byte) { binary.LittleEndian.PutUint16(b[0x84+18:], 0) },
		"delay import": func(b []byte) {
			binary.LittleEndian.PutUint32(b[0x98+112+13*8:], 0x1100)
			binary.LittleEndian.PutUint32(b[0x98+116+13*8:], 32)
		},
		"managed loader": func(b []byte) {
			binary.LittleEndian.PutUint32(b[0x98+112+14*8:], 0x1100)
			binary.LittleEndian.PutUint32(b[0x98+116+14*8:], 72)
		},
	} {
		t.Run(name, func(t *testing.T) {
			image := windowsLauncherPEFixture("kernel32.dll")
			mutate(image)
			if e := verifyWindowsMaintainedLauncherPE(bytes.NewReader(image), "amd64"); e == nil {
				t.Fatal("unqualified native image accepted")
			}
		})
	}
	arm := windowsLauncherPEFixture("kernel32.dll")
	binary.LittleEndian.PutUint16(arm[0x84:], pe.IMAGE_FILE_MACHINE_ARM64)
	if e := verifyWindowsMaintainedLauncherPE(bytes.NewReader(arm), "arm64"); e != nil {
		t.Fatal(e)
	}
	if e := verifyWindowsMaintainedLauncherPE(bytes.NewReader(arm), "386"); !errors.Is(e, ErrUnsupported) {
		t.Fatal(e)
	}
}
func TestWindowsMaintainedLauncherPERefusesMalformedImportEvidence(t *testing.T) {
	for name, mutate := range map[string]func([]byte) []byte{
		"missing DOS":        func(b []byte) []byte { b[0] = 'x'; return b },
		"short header":       func(b []byte) []byte { return b[:100] },
		"truncated section":  func(b []byte) []byte { return b[:650] },
		"zero table RVA":     func(b []byte) []byte { binary.LittleEndian.PutUint32(b[0x98+120:], 0); return b },
		"missing table size": func(b []byte) []byte { binary.LittleEndian.PutUint32(b[0x98+124:], 0); return b },
		"unbounded table":    func(b []byte) []byte { binary.LittleEndian.PutUint32(b[0x98+124:], 65537); return b },
		"missing terminator": func(b []byte) []byte { binary.LittleEndian.PutUint32(b[0x98+124:], 20); return b },
		"outside name RVA":   func(b []byte) []byte { binary.LittleEndian.PutUint32(b[512+12:], 0x8000); return b },
		"zero name RVA":      func(b []byte) []byte { binary.LittleEndian.PutUint32(b[512+12:], 0); return b },
		"zero thunk RVA":     func(b []byte) []byte { binary.LittleEndian.PutUint32(b[512+16:], 0); return b },
		"unterminated name": func(b []byte) []byte {
			for n := 640; n < 896; n++ {
				b[n] = 'A'
			}
			return b
		},
		"table RVA wraps": func(b []byte) []byte { binary.LittleEndian.PutUint32(b[0x98+120:], 0xfffffff8); return b },
		"overlapping virtual sections": func(b []byte) []byte {
			binary.LittleEndian.PutUint16(b[0x84+2:], 2)
			copy(b[0x188+40:0x188+80], b[0x188:0x188+40])
			return b
		},
	} {
		t.Run(name, func(t *testing.T) {
			image := mutate(windowsLauncherPEFixture("kernel32.dll"))
			if e := verifyWindowsMaintainedLauncherPE(bytes.NewReader(image), "amd64"); e == nil {
				t.Fatal("malformed loader evidence accepted")
			}
		})
	}
}
