package resources

import (
	"bytes"
	"debug/macho"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/binaryfetch"
)

func TestOpenRGBMacABIRequiresNativeWordSizeByteOrderAndSubtype(t *testing.T) {
	profile, _ := openRGBReleaseProfileFor("darwin", "amd64")
	for _, tc := range []struct {
		name           string
		magic, subtype uint32
		order          binary.ByteOrder
		valid          bool
	}{{"native", macho.Magic64, 3, binary.LittleEndian, true}, {"32-bit-header", macho.Magic32, 3, binary.LittleEndian, false}, {"big-endian", macho.Magic64, 3, binary.BigEndian, false}, {"unqualified-cpu-subtype", macho.Magic64, 8, binary.LittleEndian, false}} {
		t.Run(tc.name, func(t *testing.T) {
			fields := []uint32{tc.magic, uint32(macho.CpuAmd64), tc.subtype, uint32(macho.TypeExec), 1, 24, 0}
			if tc.magic == macho.Magic64 {
				fields = append(fields, 0)
			}
			fields = append(fields, 0x32, 24, 1, 15<<16, 15<<16, 0)
			data := make([]byte, len(fields)*4)
			for index, value := range fields {
				tc.order.PutUint32(data[index*4:], value)
			}
			path := filepath.Join(t.TempDir(), "image")
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
			err := profile.verifyPayloadABI(path)
			if (err == nil) != tc.valid {
				t.Fatalf("native ABI admission valid=%v: %v", tc.valid, err)
			}
		})
	}
}

func TestOpenRGBWindowsBaselineBindsBundledQtRequirements(t *testing.T) {
	p, err := openRGBReleaseProfileFor("windows", "amd64")
	if err != nil || p.MinimumWindowsBuild != 17763 {
		t.Fatalf("missing Qt6.8 Windows baseline: %+v %v", p, err)
	}
	for _, tc := range []struct {
		major, minor, build uint32
		valid               bool
	}{{0, 0, 0, false}, {6, 3, 9600, false}, {10, 0, 10240, false}, {10, 0, 16299, false}, {10, 0, 17762, false}, {10, 0, 17763, true}, {10, 0, 22000, true}, {11, 0, 1, true}} {
		if got := openRGBWindowsVersionAtLeast(tc.major, tc.minor, tc.build, p.MinimumWindowsBuild); got != tc.valid {
			t.Fatalf("OS %d.%d build%d: got%v want%v", tc.major, tc.minor, tc.build, got, tc.valid)
		}
	}
}

func TestOpenRGBReleaseProfileMeasurement(t *testing.T) {
	root := os.Getenv("VROOLI_OPENRGB_PROFILE_MEASUREMENT_ROOT")
	if root == "" {
		t.Skip("explicit offline artifact qualification only")
	}
	for _, tc := range []struct{ os, arch, name, extracted string }{{"linux", "arm64", "OpenRGB_1.1_arm64_8e16de2.AppImage", "linux-arm64"}, {"darwin", "amd64", "OpenRGB_1.1_MacOS_Intel_8e16de2.zip", ""}, {"darwin", "arm64", "OpenRGB_1.1_MacOS_ARM64_8e16de2.zip", ""}, {"windows", "amd64", "OpenRGB_1.1_Windows_64_8e16de2.zip", ""}} {
		p, err := openRGBReleaseProfileFor(tc.os, tc.arch)
		if err != nil {
			t.Fatal(err)
		}
		archive := filepath.Join(root, tc.name)
		if err := binaryfetch.VerifyArtifactContext(t.Context(), archive, "file", p.EnvelopeSHA256); err != nil {
			t.Fatal(err)
		}
		var dir string
		if p.Format == "zip" {
			dir = filepath.Join(t.TempDir(), "bundle")
			if err := os.Mkdir(dir, 0o700); err != nil {
				t.Fatal(err)
			}
			f, err := os.Open(archive)
			if err != nil {
				t.Fatal(err)
			}
			err = extractOpenRGBZip(t.Context(), f, dir, p)
			f.Close()
			if err != nil {
				t.Fatal(err)
			}
		} else {
			dir = filepath.Join(root, tc.extracted)
		}
		payload := filepath.Join(dir, filepath.FromSlash(p.Payload))
		if err := p.verifyVendorStartupMode(payload); err != nil {
			t.Fatal(err)
		}
		if err := p.verifyPayloadABI(payload); err != nil {
			t.Fatal(err)
		}
		if err := p.verifyBundleABI(t.Context(), dir); err != nil {
			t.Fatal(err)
		}
		tree, err := binaryfetch.TreeDigest(dir)
		if err != nil {
			t.Fatal(err)
		}
		hash, err := binaryfetch.FileDigestContext(t.Context(), payload)
		if err != nil {
			t.Fatal(err)
		}
		if tree != p.TreeSHA256 || hash != p.PayloadSHA256 {
			t.Fatal("measured official closure differs from frozen profile")
		}
		for candidate, expected := range map[string]string{archive: p.EnvelopeSHA256, payload: p.PayloadSHA256} {
			adopted, err := inspectOpenRGBAdoptedSourceForProfile(t.Context(), candidate, p)
			if err != nil || adopted != expected {
				t.Fatalf("exact native profile adoption refused: %s: digest=%s err=%v", candidate, adopted, err)
			}
		}
		data, err := os.ReadFile(payload)
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"ASUS Aura Motherboard", "ASUS ROG AURA Terminal"} {
			if !bytes.Contains(data, []byte(name)) {
				t.Fatalf("selected detector %q registration absent from exact native payload", name)
			}
		}
		wrong := p
		if p.Arch == "amd64" {
			wrong.Arch = "arm64"
		} else {
			wrong.Arch = "amd64"
		}
		if p.OS != "windows" && wrong.verifyPayloadABI(payload) == nil {
			t.Fatal("foreign architecture accepted")
		}
		t.Logf("%s-%s TREE=%s PAYLOAD=%s", tc.os, tc.arch, tree, hash)
	}
}

func TestOpenRGBNativeProfileMatrixIsClosed(t *testing.T) {
	for _, target := range [][2]string{{"linux", "amd64"}, {"linux", "arm64"}, {"darwin", "amd64"}, {"darwin", "arm64"}, {"windows", "amd64"}} {
		p, err := openRGBReleaseProfileFor(target[0], target[1])
		if err != nil || p.validate() != nil {
			t.Fatalf("missing profile %v: %+v %v", target, p, err)
		}
		if p.Payload == "" || p.DownloadBytes > 64<<20 || p.EnvelopeSHA256 == p.PayloadSHA256 {
			t.Fatalf("invalid closed profile %+v", p)
		}
		if !p.acceptsArtifactSHA256(p.PayloadSHA256) || !p.acceptsArtifactSHA256(p.EnvelopeSHA256) || p.acceptsArtifactSHA256(updateTestDigest("other")) {
			t.Fatal("profile identity acceptance is not exact")
		}
	}
	for _, target := range [][2]string{{"windows", "arm64"}, {"linux", "386"}, {"darwin", "386"}, {"freebsd", "amd64"}} {
		if _, err := openRGBReleaseProfileFor(target[0], target[1]); err == nil {
			t.Fatalf("accepted unqualified target %v", target)
		}
	}
}

func TestOpenRGBWindowsStageAcceptsDrivePrefixOnly(t *testing.T) {
	for _, path := range []string{`C:\ProgramData\Vrooli\resources\openrgb\artifacts\AppDir`, `D:/Vrooli/OpenRGB.app`} {
		if openRGBStagePathHasUnsafeCharacters("windows", path) {
			t.Fatalf("rejected canonical native drive path %q", path)
		}
	}
	for _, path := range []string{`C:\stage:stream`, `C:\stage;other`, `\\server\share\stage`, `C:relative`} {
		if !openRGBStagePathHasUnsafeCharacters("windows", path) {
			t.Fatalf("accepted unsafe native stage path %q", path)
		}
	}
	if !openRGBStagePathHasUnsafeCharacters("linux", "/tmp/path:other") {
		t.Fatal("Linux library path separator accepted")
	}
}
