//go:build linux

package hostwireless

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/klauspost/compress/zstd"
)

const policyFirmwareName = "mediatek/mt7927/image.bin"

var policyCandidate = []byte("candidate firmware")

func policyFixture(t *testing.T) (FirmwarePolicyRequest, func(string, []byte)) {
	t.Helper()
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { root.Close() })
	write := func(name string, data []byte) {
		t.Helper()
		full := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("boot/config-test-kernel", []byte("CONFIG_FW_LOADER=y\nCONFIG_EXTRA_FIRMWARE=\"\"\nCONFIG_FW_LOADER_COMPRESS_ZSTD=y\nCONFIG_FW_LOADER_COMPRESS_XZ=y\n"))
	write("sys/module/firmware_class/parameters/path", []byte("\n"))
	write("lib/modules/test-kernel/modules.builtin", nil)
	write("boot/initrd.img-test-kernel", policyCPIO(policyEntry{"init", 0100755, []byte("fixture only")}))
	return FirmwarePolicyRequest{Root: root, KernelRelease: "test-kernel", Firmware: []FirmwarePolicyTarget{{Name: policyFirmwareName, SHA256: digest(policyCandidate)}}, Modules: []string{"mt7925e", "mt7925_common"}}, write
}
func policyInspect(t *testing.T, r FirmwarePolicyRequest) FirmwarePolicyEvidence {
	t.Helper()
	e, err := InspectFirmwarePolicy(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	return e
}
func policyZstd(t *testing.T, data []byte) []byte {
	t.Helper()
	z, err := zstd.NewWriter(nil, zstd.WithEncoderConcurrency(1), zstd.WithSingleSegment(true))
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	return z.EncodeAll(data, nil)
}
func policyGzip(t *testing.T, data []byte) []byte {
	t.Helper()
	var b bytes.Buffer
	z := gzip.NewWriter(&b)
	if _, err := z.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func policyXZ(t *testing.T) []byte {
	t.Helper()
	b, err := base64.StdEncoding.DecodeString("/Td6WFoAAAFpIt42AgAhARYAAAB0L+WjAQARY2FuZGlkYXRlIGZpcm13YXJlAAAAE5moWwABJhJHR+EQkEKZDQEAAAAAAVla")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

type policyEntry struct {
	name string
	mode uint32
	data []byte
}

func policyCPIO(entries ...policyEntry) []byte {
	var b bytes.Buffer
	entries = append(entries, policyEntry{"TRAILER!!!", 0, nil})
	for _, e := range entries {
		fmt.Fprintf(&b, "070701%08x%08x%08x%08x%08x%08x%08x%08x%08x%08x%08x%08x%08x", 1, e.mode, 0, 0, 1, 0, len(e.data), 0, 0, 0, 0, len(e.name)+1, 0)
		b.WriteString(e.name)
		b.WriteByte(0)
		for b.Len()%4 != 0 {
			b.WriteByte(0)
		}
		b.Write(e.data)
		for b.Len()%4 != 0 {
			b.WriteByte(0)
		}
	}
	return b.Bytes()
}

func TestFirmwarePolicySupportedAbsentAndExact(t *testing.T) {
	r, w := policyFixture(t)
	e := policyInspect(t, r)
	if !e.StageAllowed || !e.Selections[0].StageRequired || e.Selections[0].StagePath != "lib/firmware/"+policyFirmwareName {
		t.Fatalf("absence not admitted: %+v", e)
	}
	w(e.Selections[0].StagePath, policyCandidate)
	e = policyInspect(t, r)
	if !e.StageAllowed || e.Selections[0].StageRequired || e.Selections[0].Encoding != "raw" || !validSHA(e.ConfigSHA256) || !validSHA(e.ParameterSHA256) || !validSHA(e.BuiltinModulesSHA256) || !validSHA(e.InitramfsSHA256) {
		t.Fatalf("exact evidence differs: %+v", e)
	}
}
func TestFirmwarePolicyRawPrecedesEveryCompressedDirectory(t *testing.T) {
	r, w := policyFixture(t)
	w("sys/module/firmware_class/parameters/path", []byte("/custom\nignored"))
	w("custom/"+policyFirmwareName+".zst", policyZstd(t, []byte("foreign")))
	w("lib/firmware/"+policyFirmwareName, policyCandidate)
	e := policyInspect(t, r)
	if !e.StageAllowed || e.Selections[0].Path != "lib/firmware/"+policyFirmwareName || e.Selections[0].StageRequired {
		t.Fatalf("raw lookup order differs: %+v", e)
	}
}
func TestFirmwarePolicyRawDirectoryOrderAndOverrides(t *testing.T) {
	dirs := []string{"custom", "lib/firmware/updates/test-kernel", "lib/firmware/updates", "lib/firmware/test-kernel"}
	for _, dir := range dirs {
		t.Run(strings.ReplaceAll(dir, "/", "_"), func(t *testing.T) {
			r, w := policyFixture(t)
			w("sys/module/firmware_class/parameters/path", []byte("/custom\n"))
			w("lib/firmware/"+policyFirmwareName, policyCandidate)
			w(dir+"/"+policyFirmwareName, []byte("foreign"))
			e := policyInspect(t, r)
			if e.StageAllowed || e.Selections[0].Path != dir+"/"+policyFirmwareName {
				t.Fatalf("foreign override admitted: %+v", e)
			}
			w(dir+"/"+policyFirmwareName, policyCandidate)
			e = policyInspect(t, r)
			if !e.StageAllowed || e.Selections[0].StageRequired {
				t.Fatalf("exact override refused: %+v", e)
			}
		})
	}
	// Multiple levels populated: highest directory wins, even if lower is exact.
	r, w := policyFixture(t)
	for _, dir := range dirs {
		w(dir+"/"+policyFirmwareName, policyCandidate)
	}
	w("sys/module/firmware_class/parameters/path", []byte("/custom\n"))
	w("custom/"+policyFirmwareName, []byte("foreign"))
	if e := policyInspect(t, r); e.Selections[0].Path != "custom/"+policyFirmwareName || e.StageAllowed {
		t.Fatal(e)
	}
}
func TestFirmwarePolicyCompressedSelectionAndStaging(t *testing.T) {
	for _, encoding := range []string{"zstd", "xz"} {
		t.Run(encoding, func(t *testing.T) {
			r, w := policyFixture(t)
			suffix := ".zst"
			data := policyZstd(t, policyCandidate)
			if encoding == "xz" {
				suffix = ".xz"
				data = policyXZ(t)
			}
			w("lib/firmware/"+policyFirmwareName+suffix, data)
			e := policyInspect(t, r)
			s := e.Selections[0]
			if !e.StageAllowed || s.StageRequired || s.Encoding != encoding || s.SHA256 != digest(policyCandidate) || s.StoredSHA256 != digest(data) {
				t.Fatalf("compressed selection differs: %+v", e)
			}
		})
	}
	r, w := policyFixture(t)
	w("lib/firmware/updates/"+policyFirmwareName+".xz", policyXZ(t))
	w("lib/firmware/"+policyFirmwareName+".zst", policyZstd(t, []byte("foreign")))
	e := policyInspect(t, r)
	if e.Selections[0].Encoding != "zstd" || !e.StageAllowed || !e.Selections[0].StageRequired {
		t.Fatalf("zstd then xz order or raw stage differs: %+v", e)
	}
}
func TestFirmwarePolicyBuiltinAndConfiguration(t *testing.T) {
	r, w := policyFixture(t)
	w("boot/config-test-kernel", []byte("CONFIG_FW_LOADER=y\nCONFIG_EXTRA_FIRMWARE=\""+policyFirmwareName+"\"\n"))
	w("lib/firmware/"+policyFirmwareName, policyCandidate)
	e := policyInspect(t, r)
	if e.StageAllowed || !e.Selections[0].Builtin {
		t.Fatal(e)
	}
	w("boot/config-test-kernel", []byte("CONFIG_FW_LOADER=y\n"))
	if _, err := InspectFirmwarePolicy(context.Background(), r); err == nil {
		t.Fatal("unknown builtins admitted")
	}
	w("boot/config-test-kernel", []byte("CONFIG_FW_LOADER=y\nCONFIG_EXTRA_FIRMWARE=\"\"\nCONFIG_FW_LOADER=y\n"))
	if _, err := InspectFirmwarePolicy(context.Background(), r); err == nil {
		t.Fatal("duplicate config admitted")
	}
}
func TestFirmwarePolicyInitramfsAffectedAndBuiltinModules(t *testing.T) {
	for _, name := range []string{"lib/modules/test-kernel/kernel/mt7925e.ko.zst", "usr/lib/modules/test-kernel/mt7925-common.ko", "lib/firmware/" + policyFirmwareName, "usr/lib/firmware/" + policyFirmwareName + ".xz", "custom/" + policyFirmwareName + ".zst", "relocated/image.bin"} {
		t.Run(strings.ReplaceAll(name, "/", "_"), func(t *testing.T) {
			r, w := policyFixture(t)
			w("boot/initrd.img-test-kernel", policyCPIO(policyEntry{name, 0100644, []byte("old bytes")}))
			e := policyInspect(t, r)
			if e.StageAllowed || len(e.InitramfsAffected) != 1 || e.InitramfsAffected[0] != name {
				t.Fatal(e)
			}
		})
	}
	r, w := policyFixture(t)
	w("lib/modules/test-kernel/modules.builtin", []byte("kernel/drivers/mt7925-common.ko\n"))
	e := policyInspect(t, r)
	if e.StageAllowed || len(e.BuiltinModules) != 1 {
		t.Fatal(e)
	}
}
func TestFirmwarePolicyInitramfsSupportedArchiveLayouts(t *testing.T) {
	raw := policyCPIO(policyEntry{"init", 0100755, []byte("fixture")})
	early := policyCPIO(policyEntry{"kernel/x86/microcode/vendor.bin", 0100644, []byte("microcode")})
	xzArchive, _ := base64.StdEncoding.DecodeString("/Td6WFoAAAFpIt42AgAhARYAAAB0L+Wj4AD3ADxdABgN3QRjnHIbdqHA2jmwVfruHqrzy7hCepji+6+zi/IW1qWy4+0s6DVn3Jodi3aHItHnA9x8PzsGDnDwAADf00wlAAFU+AEAAADF4rKUPjANiwIAAAAAAVla")
	for _, data := range [][]byte{raw, xzArchive, policyCPIO(), append(append([]byte{}, early...), raw...), policyGzip(t, raw), policyZstd(t, raw), append(append(early, make([]byte, 512)...), policyGzip(t, raw)...)} {
		r, w := policyFixture(t)
		w("boot/initrd.img-test-kernel", data)
		if e := policyInspect(t, r); !e.StageAllowed {
			t.Fatal(e)
		}
	}
}
func TestFirmwarePolicyArchiveRejectsAmbiguityAndMalformed(t *testing.T) {
	raw := policyCPIO(policyEntry{"init", 0100755, []byte("fixture")})
	bad := [][]byte{nil, []byte("unrecognized codec"), append(raw, []byte("garbage")...), append(append([]byte{}, raw...), raw...), policyCPIO(policyEntry{"../escape", 0100644, nil}), policyCPIO(policyEntry{"/absolute", 0100644, nil}), policyCPIO(policyEntry{"init", 0100644, nil}, policyEntry{"init", 0120777, []byte("elsewhere")}), policyGzip(t, []byte("not cpio")), append(policyGzip(t, raw), policyZstd(t, raw)...)}
	for i, data := range bad {
		if _, err := firmwarePolicyArchive(context.Background(), data); err == nil {
			t.Fatalf("malformed archive %d admitted", i)
		}
	}
	for i := 0; i < len(raw); i++ {
		if _, err := firmwarePolicyArchive(context.Background(), raw[:i]); err == nil {
			t.Fatalf("truncated archive %d admitted", i)
		}
	}
}
func TestFirmwarePolicyRejectsUnsafePathsAndUnreadableCandidates(t *testing.T) {
	for _, name := range []string{"../image", "/image", "foo/../../image", "foo//image", "foo/./image", "image\x00", "foo\\bar"} {
		r, _ := policyFixture(t)
		r.Firmware[0].Name = name
		if _, err := InspectFirmwarePolicy(context.Background(), r); err == nil {
			t.Fatalf("unsafe path %q admitted", name)
		}
	}
	for _, custom := range []string{"relative", "/a/../b", "/a\x00b", "/"} {
		r, w := policyFixture(t)
		w("sys/module/firmware_class/parameters/path", []byte(custom))
		if _, err := InspectFirmwarePolicy(context.Background(), r); err == nil {
			t.Fatalf("unsafe custom path %q admitted", custom)
		}
	}
	r, w := policyFixture(t)
	w("lib/firmware/"+policyFirmwareName+".zst", []byte("broken"))
	if _, err := InspectFirmwarePolicy(context.Background(), r); err == nil {
		t.Fatal("broken compressed firmware admitted")
	}
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(outside, policyCandidate, 0600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := os.Symlink(outside, filepath.Join(dir, "escape")); err != nil {
		t.Fatal(err)
	}
	if _, err := firmwarePolicyRead(context.Background(), root, "escape", 1024); err == nil {
		t.Fatal("root escaping symlink admitted")
	}
	if err := syscall.Mkfifo(filepath.Join(dir, "pipe"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := firmwarePolicyRead(context.Background(), root, "pipe", 1024); err == nil {
		t.Fatal("FIFO admitted")
	}
}
func TestFirmwarePolicyCompressionBoundsAndTruncation(t *testing.T) {
	for _, suffix := range []string{".zst", ".xz"} {
		data := policyZstd(t, policyCandidate)
		if suffix == ".xz" {
			data = policyXZ(t)
		}
		for i := 0; i < len(data); i++ {
			if _, err := firmwarePolicyDecode(context.Background(), data[:i], suffix, 1024); err == nil {
				t.Fatalf("truncated %s at %d admitted", suffix, i)
			}
		}
		if _, err := firmwarePolicyDecode(context.Background(), append(append([]byte{}, data...), data...), suffix, 1024); err == nil {
			t.Fatalf("multiple firmware %s streams admitted", suffix)
		}
		if _, err := firmwarePolicyDecode(context.Background(), data, suffix, 3); err == nil {
			t.Fatal("decompression bound ignored")
		}
	}
	r, _ := policyFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := InspectFirmwarePolicy(ctx, r); err == nil {
		t.Fatal("cancellation ignored")
	}
}
func TestFirmwarePolicyUsrMergeSymlinksStayWithinInjectedRoot(t *testing.T) {
	r, w := policyFixture(t)
	w("usr/lib/firmware/"+policyFirmwareName, policyCandidate)
	// Only the firmware directory is a link; modules inventory remains intact.
	if err := r.Root.Symlink("../usr/lib/firmware", "lib/firmware"); err != nil {
		t.Fatal(err)
	}
	e := policyInspect(t, r)
	if !e.StageAllowed || e.Selections[0].StageRequired {
		t.Fatal(e)
	}
}

func TestFirmwarePolicyLiveCurrentImage(t *testing.T) {
	if os.Getenv("VROOLI_WIRELESS_FIRMWARE_POLICY_LIVE") != "1" {
		t.Skip("explicit read-only live opt-in")
	}
	release := os.Getenv("VROOLI_WIRELESS_FIRMWARE_POLICY_KERNEL")
	if release == "" {
		t.Fatal("explicit kernel release required")
	}
	root, err := os.OpenRoot("/")
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	for _, profile := range []string{NobleProfile, NobleBluetoothProfile} {
		var targets []FirmwarePolicyTarget
		for _, pin := range durableFirmwarePins(profile) {
			targets = append(targets, FirmwarePolicyTarget{Name: strings.TrimPrefix(pin.Path, "lib/firmware/"), SHA256: pin.SHA256})
		}
		modules := []string{"mt76", "mt76_connac_lib", "mt792x_lib", "mt7925_common", "mt7925e"}
		if profile == NobleBluetoothProfile {
			modules = []string{"bluetooth", "btusb", "btmtk"}
		}
		e, err := InspectFirmwarePolicy(context.Background(), FirmwarePolicyRequest{Root: root, KernelRelease: release, Firmware: targets, Modules: modules})
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("profile=%s stage_allowed=%t initramfs_sha256=%s config_sha256=%s parameter_sha256=%s builtin_modules_sha256=%s affected=%v reasons=%v selections=%+v", profile, e.StageAllowed, e.InitramfsSHA256, e.ConfigSHA256, e.ParameterSHA256, e.BuiltinModulesSHA256, e.InitramfsAffected, e.Reasons, e.Selections)
	}
}

func TestFirmwarePolicyDisabledCompressionIsNotProvider(t *testing.T) {
	r, w := policyFixture(t)
	w("boot/config-test-kernel", []byte("CONFIG_FW_LOADER=y\nCONFIG_EXTRA_FIRMWARE=\"\"\n# CONFIG_FW_LOADER_COMPRESS is not set\n"))
	w("lib/firmware/"+policyFirmwareName+".zst", []byte("not even compressed"))
	e := policyInspect(t, r)
	if !e.StageAllowed || !e.Selections[0].StageRequired || e.Selections[0].Path != "" {
		t.Fatal(e)
	}
}
func TestFirmwarePolicyMissingPrerequisitesFailClosed(t *testing.T) {
	for _, name := range []string{"boot/config-test-kernel", "sys/module/firmware_class/parameters/path", "lib/modules/test-kernel/modules.builtin", "boot/initrd.img-test-kernel"} {
		r, _ := policyFixture(t)
		if err := r.Root.Remove(name); err != nil {
			t.Fatal(err)
		}
		e, err := InspectFirmwarePolicy(context.Background(), r)
		if err == nil || e.StageAllowed {
			t.Fatalf("missing %s admitted: %+v %v", name, e, err)
		}
	}
}
func TestFirmwarePolicyReadBounds(t *testing.T) {
	r, w := policyFixture(t)
	w("large", bytes.Repeat([]byte{'x'}, 128))
	if _, err := firmwarePolicyRead(context.Background(), r.Root, "large", 127); err == nil {
		t.Fatal("file bound ignored")
	}
	w("sys/module/firmware_class/parameters/path", bytes.Repeat([]byte{'x'}, 257))
	if _, err := firmwarePolicyReadParameter(context.Background(), r.Root); err == nil {
		t.Fatal("parameter bound ignored")
	}
}
func TestFirmwarePolicyCPIOChecksumAndHeaderErrors(t *testing.T) {
	raw := policyCPIO(policyEntry{"init", 0100644, []byte("payload")})
	copy(raw[:6], "070702")
	if _, err := firmwarePolicyArchive(context.Background(), raw); err == nil {
		t.Fatal("invalid CRC accepted")
	}
	for _, offset := range []int{6, 14, 54, 94, 102} {
		bad := policyCPIO(policyEntry{"init", 0100644, nil})
		bad[offset] = 'z'
		if _, err := firmwarePolicyArchive(context.Background(), bad); err == nil {
			t.Fatalf("invalid numeric field %d accepted", offset)
		}
	}
	crc64, _ := base64.StdEncoding.DecodeString("/Td6WFoAAATm1rRGAgAhARYAAAB0L+WjAQARY2FuZGlkYXRlIGZpcm13YXJlAAAA5td5sWWHOEoAASoSSwhUvB+2830BAAAAAARZWg==")
	if _, err := firmwarePolicyDecode(context.Background(), crc64, ".xz", 1024); err == nil {
		t.Fatal("kernel-unsupported firmware XZ check admitted")
	}
}
