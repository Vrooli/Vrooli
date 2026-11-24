package driveprofile

import (
	"reflect"
	"strings"
	"testing"
)

func TestLookupResolvesAliasesToCanonicalProfiles(t *testing.T) {
	cases := map[string]Family{
		"ext4": FamilyExt, "ext3": FamilyExt, "ext2": FamilyExt,
		"ntfs": FamilyNTFS, "ntfs3": FamilyNTFS,
		"exfat": FamilyExFAT,
		"vfat":  FamilyFAT, "fat32": FamilyFAT, "msdos": FamilyFAT,
		"apfs": FamilyAPFS, "hfs+": FamilyHFS,
	}
	for name, want := range cases {
		got, ok := Lookup(name)
		if !ok {
			t.Fatalf("Lookup(%q) was not found", name)
		}
		if got.Family != want {
			t.Fatalf("Lookup(%q).Family = %q, want %q", name, got.Family, want)
		}
	}
	if _, ok := Lookup("btrfs-not-yet-profiled"); ok {
		t.Fatal("an unprofiled filesystem must not be guessed at")
	}
}

func TestRepairCommandPreservesTheFilesystemSpecificTool(t *testing.T) {
	cases := []struct {
		fs     string
		device string
		want   []string
	}{
		{"ntfs", "/dev/sda1", []string{"ntfsfix", "-d", "/dev/sda1"}},
		{"ntfs3", "/dev/sdb1", []string{"ntfsfix", "-d", "/dev/sdb1"}},
		{"ext4", "/dev/nvme0n1p2", []string{"e2fsck", "-f", "-y", "/dev/nvme0n1p2"}},
		{"exfat", "/dev/sdc1", []string{"fsck.exfat", "-y", "/dev/sdc1"}},
		{"vfat", "/dev/sdd1", []string{"fsck.fat", "-a", "/dev/sdd1"}},
	}
	for _, tc := range cases {
		got, ok := RepairCommand(tc.fs, tc.device)
		if !ok {
			t.Fatalf("RepairCommand(%q) unsupported", tc.fs)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("RepairCommand(%q) = %v, want %v", tc.fs, got, tc.want)
		}
	}
	if _, ok := RepairCommand("apfs", "/dev/sda1"); ok {
		t.Fatal("APFS has no Linux repair command and must not invent one")
	}
}

func TestNTFSRepairIsHonestlyNotAMetadataRepair(t *testing.T) {
	p, ok := Lookup("ntfs")
	if !ok {
		t.Fatal("ntfs profile missing")
	}
	if p.RepairIsMetadata {
		t.Fatal("ntfsfix must not be presented as a metadata repair")
	}
	if p.Suitability != Caution {
		t.Fatalf("ntfs suitability = %q, want caution", p.Suitability)
	}
	if p.RepairLimitation == "" {
		t.Fatal("a tool that cannot repair metadata must carry the limitation it must show the operator")
	}
}

// A real metadata repair tool has no limitation to warn about. If one ever
// gains a limitation, it belongs in the table rather than in a caller switch.
func TestMetadataRepairProfilesCarryNoFalseLimitation(t *testing.T) {
	for _, p := range All() {
		if p.RepairIsMetadata && p.RepairLimitation != "" {
			t.Fatalf("%s is a metadata repair but declares a limitation: %q", p.Name, p.RepairLimitation)
		}
	}
}

func TestExt4IsRecommendedWithKernelStateSignal(t *testing.T) {
	p, _ := Lookup("ext4")
	if p.StateSignal != StateSignalKernel {
		t.Fatalf("ext4 state signal = %q, want kernel", p.StateSignal)
	}
	if p.Suitability != Recommended {
		t.Fatalf("ext4 suitability = %q, want recommended", p.Suitability)
	}
	if !p.RepairIsMetadata {
		t.Fatal("e2fsck is a real metadata repair")
	}
}

// The same filesystem can be first-class on one platform and unsuitable on
// another. That difference belongs in the table, not in a caller switch.
func TestSuitabilityIsPlatformSpecific(t *testing.T) {
	cases := []struct {
		fs, goos string
		want     Suitability
	}{
		{"ext4", "linux", Recommended},
		{"ext4", "windows", Avoid},
		{"ext4", "darwin", Avoid},
		{"ntfs", "linux", Caution},
		{"ntfs", "windows", Recommended},
		{"apfs", "darwin", Recommended},
		{"apfs", "linux", Avoid},
		{"exfat", "windows", Usable},
		{"macos", "darwin", Avoid}, // an unprofiled label stays unprofiled
	}
	for _, tc := range cases {
		p, ok := Lookup(tc.fs)
		if tc.fs == "macos" {
			if ok {
				t.Fatalf("%q must not resolve to a profile", tc.fs)
			}
			continue
		}
		if !ok {
			t.Fatalf("Lookup(%q) not found", tc.fs)
		}
		if got := p.SuitabilityOn(tc.goos); got != tc.want {
			t.Errorf("%s on %s = %q, want %q", tc.fs, tc.goos, got, tc.want)
		}
	}
}

func TestPlatformNoteExplainsTheDifference(t *testing.T) {
	ntfs, _ := Lookup("ntfs")
	if note := ntfs.NoteOn("windows"); !strings.Contains(note, "chkdsk") {
		t.Fatalf("Windows NTFS note should name the native repair tool, got %q", note)
	}
	if note := ntfs.NoteOn("linux"); note != "" {
		t.Fatalf("Linux is the profile's own platform and needs no override note, got %q", note)
	}
	if got := NormalizeOS("macOS"); got != "darwin" {
		t.Fatalf("NormalizeOS(macOS) = %q", got)
	}
}
