package privilegebroker

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func volumeRequest(action string) Request {
	return Request{
		Version:   ProtocolVersion,
		RequestID: "req-1",
		Action:    action,
		Volume: &VolumeSubject{
			Device:     "/dev/sda1",
			Filesystem: "ntfs3",
			UUID:       "E26A883E6A881189",
			Serial:     "WD-WX52A946D6VL",
		},
	}
}

// useVolumeFixture points the broker's evidence sources at a temp tree and
// restores them afterwards.
func useVolumeFixture(t *testing.T, mounts string, uuidToDevice map[string]string) {
	t.Helper()
	root := t.TempDir()
	mountsPath := filepath.Join(root, "proc", "mounts")
	byUUID := filepath.Join(root, "dev", "disk", "by-uuid")
	if err := os.MkdirAll(filepath.Dir(mountsPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(byUUID, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(mountsPath, []byte(mounts), 0o644); err != nil {
		t.Fatalf("write mounts: %v", err)
	}
	for uuid := range uuidToDevice {
		if err := os.WriteFile(filepath.Join(byUUID, uuid), nil, 0o644); err != nil {
			t.Fatalf("write uuid entry: %v", err)
		}
	}

	origMounts, origByUUID, origEval := volumeProcMounts, volumeDevDiskByUUID, volumeEvalSymlinks
	volumeProcMounts, volumeDevDiskByUUID = mountsPath, byUUID
	volumeEvalSymlinks = func(p string) (string, error) {
		if device, ok := uuidToDevice[filepath.Base(p)]; ok && strings.HasPrefix(p, byUUID) {
			return device, nil
		}
		return p, nil
	}
	t.Cleanup(func() {
		volumeProcMounts, volumeDevDiskByUUID, volumeEvalSymlinks = origMounts, origByUUID, origEval
	})
}

type recordingExecutor struct {
	name string
	args []string
	out  []byte
	err  error
}

func (r *recordingExecutor) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.name, r.args = name, append([]string(nil), args...)
	return r.out, r.err
}

func TestValidateVolumeRejectsUnsafeShapes(t *testing.T) {
	cases := map[string]func(*Request){
		"missing volume subject": func(r *Request) { r.Volume = nil },
		"empty device":           func(r *Request) { r.Volume.Device = "" },
		"relative device":        func(r *Request) { r.Volume.Device = "sda1" },
		"traversal device":       func(r *Request) { r.Volume.Device = "/dev/../etc/passwd" },
		"argument in device":     func(r *Request) { r.Volume.Device = "/dev/sda1 --force" },
		"shell in device":        func(r *Request) { r.Volume.Device = "/dev/sda1;reboot" },
		"option as device":       func(r *Request) { r.Volume.Device = "/dev/-n" },
		"unknown filesystem":     func(r *Request) { r.Volume.Filesystem = "zfs" },
		"empty filesystem":       func(r *Request) { r.Volume.Filesystem = "" },
		"no identity":            func(r *Request) { r.Volume.UUID, r.Volume.Serial = "", "" },
		"oversized uuid":         func(r *Request) { r.Volume.UUID = strings.Repeat("a", 129) },
		"union with bridge": func(r *Request) {
			r.Subject = Subject{Scenario: BridgeScenario, Port: BridgePort, CandidateIP: "192.168.1.5"}
		},
		"wrong version":    func(r *Request) { r.Version = "v2" },
		"empty request id": func(r *Request) { r.RequestID = "" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			req := volumeRequest(ActionVolumeFilesystemRepair)
			mutate(&req)
			if err := Validate(req); err == nil {
				t.Fatalf("accepted %s", name)
			}
		})
	}
}

// A bridge request must not be able to carry a volume subject.
func TestValidateBridgeRejectsAVolumeSubject(t *testing.T) {
	req := Request{
		Version: ProtocolVersion, RequestID: "req-1", Action: ActionBridgeUFWAllow,
		Subject: Subject{Scenario: BridgeScenario, Port: BridgePort, CandidateIP: "192.168.1.5"},
		Volume:  &VolumeSubject{Device: "/dev/sda1", Filesystem: "ntfs3", UUID: "u"},
	}
	if err := Validate(req); err == nil {
		t.Fatal("a bridge action accepted a volume subject")
	}
}

func TestVolumeArgsAreFixedPerFilesystem(t *testing.T) {
	cases := []struct {
		filesystem string
		action     string
		wantTool   string
		wantArgs   []string
	}{
		{"ntfs3", ActionVolumeFilesystemCheck, "ntfsfix", []string{"-n", "/dev/sda1"}},
		{"ntfs3", ActionVolumeFilesystemRepair, "ntfsfix", []string{"-d", "/dev/sda1"}},
		{"ntfs", ActionVolumeFilesystemRepair, "ntfsfix", []string{"-d", "/dev/sda1"}},
		{"ext4", ActionVolumeFilesystemCheck, "e2fsck", []string{"-f", "-n", "/dev/sda1"}},
		{"ext4", ActionVolumeFilesystemRepair, "e2fsck", []string{"-f", "-y", "/dev/sda1"}},
		{"exfat", ActionVolumeFilesystemRepair, "fsck.exfat", []string{"-y", "/dev/sda1"}},
		{"vfat", ActionVolumeFilesystemRepair, "fsck.fat", []string{"-a", "/dev/sda1"}},
	}
	for _, tc := range cases {
		t.Run(tc.filesystem+"/"+tc.action, func(t *testing.T) {
			req := volumeRequest(tc.action)
			req.Volume.Filesystem = tc.filesystem
			tool, args, err := VolumeArgs(req)
			if err != nil {
				t.Fatalf("VolumeArgs: %v", err)
			}
			if tool != tc.wantTool || strings.Join(args, " ") != strings.Join(tc.wantArgs, " ") {
				t.Fatalf("got %s %v, want %s %v", tool, args, tc.wantTool, tc.wantArgs)
			}
		})
	}
}

func TestExecuteVolumeRefusesAMountedVolume(t *testing.T) {
	useVolumeFixture(t, "/dev/sda1 /media/user/Elements ntfs3 ro 0 0\n", map[string]string{"E26A883E6A881189": "/dev/sda1"})
	executor := &recordingExecutor{}

	res := executeVolume(context.Background(), executor, volumeRequest(ActionVolumeFilesystemRepair))

	if res.Status != "failed" || res.Code != "volume_mounted" {
		t.Fatalf("result = %+v, want a volume_mounted refusal", res)
	}
	if executor.name != "" {
		t.Fatalf("a refused repair ran %s %v", executor.name, executor.args)
	}
}

func TestExecuteVolumeRefusesSystemVolumes(t *testing.T) {
	for _, mountpoint := range []string{"/", "/boot/efi", "/home", "/var/lib"} {
		t.Run(mountpoint, func(t *testing.T) {
			useVolumeFixture(t, "/dev/sda1 "+mountpoint+" ext4 rw 0 0\n", map[string]string{"E26A883E6A881189": "/dev/sda1"})
			executor := &recordingExecutor{}
			res := executeVolume(context.Background(), executor, volumeRequest(ActionVolumeFilesystemRepair))
			if res.Code != "system_volume_refused" {
				t.Fatalf("code = %q, want system_volume_refused", res.Code)
			}
			if executor.name != "" {
				t.Fatal("a system volume must never be touched")
			}
		})
	}
}

// The broker verifies identity itself; a caller's claim is not evidence.
func TestExecuteVolumeRefusesAnIdentityItCannotConfirm(t *testing.T) {
	t.Run("uuid resolves elsewhere", func(t *testing.T) {
		useVolumeFixture(t, "", map[string]string{"E26A883E6A881189": "/dev/sdb1"})
		executor := &recordingExecutor{}
		res := executeVolume(context.Background(), executor, volumeRequest(ActionVolumeFilesystemRepair))
		if res.Code != "device_identity_mismatch" {
			t.Fatalf("code = %q, want device_identity_mismatch", res.Code)
		}
		if executor.name != "" {
			t.Fatal("nothing may run against an unconfirmed device")
		}
	})
	t.Run("serial only cannot be re-resolved", func(t *testing.T) {
		useVolumeFixture(t, "", map[string]string{})
		req := volumeRequest(ActionVolumeFilesystemRepair)
		req.Volume.UUID = ""
		executor := &recordingExecutor{}
		res := executeVolume(context.Background(), executor, req)
		if res.Code != "device_identity_mismatch" {
			t.Fatalf("code = %q, want device_identity_mismatch", res.Code)
		}
	})
}

// An unreadable mount table means the broker cannot prove the volume is
// detached, and it must refuse rather than assume.
func TestExecuteVolumeRefusesWhenMountStateIsUnprovable(t *testing.T) {
	useVolumeFixture(t, "", map[string]string{"E26A883E6A881189": "/dev/sda1"})
	orig := volumeProcMounts
	volumeProcMounts = filepath.Join(t.TempDir(), "does-not-exist")
	t.Cleanup(func() { volumeProcMounts = orig })

	executor := &recordingExecutor{}
	res := executeVolume(context.Background(), executor, volumeRequest(ActionVolumeFilesystemRepair))

	if res.Status == "changed" {
		t.Fatalf("proceeded without provable mount state: %+v", res)
	}
	if executor.name != "" {
		t.Fatal("nothing may run when mount state is unprovable")
	}
}

func TestExecuteVolumeRepairsAnUnmountedVolume(t *testing.T) {
	useVolumeFixture(t, "/dev/nvme0n1p2 / ext4 rw 0 0\n", map[string]string{"E26A883E6A881189": "/dev/sda1"})
	executor := &recordingExecutor{out: []byte("Volume is dirty.\nClearing the dirty flag.\nNTFS volume version is 3.1.\n")}

	res := executeVolume(context.Background(), executor, volumeRequest(ActionVolumeFilesystemRepair))

	if res.Status != "changed" || !res.Changed {
		t.Fatalf("result = %+v, want changed", res)
	}
	if executor.name != "ntfsfix" || strings.Join(executor.args, " ") != "-d /dev/sda1" {
		t.Fatalf("ran %s %v", executor.name, executor.args)
	}
	if !res.Evidence.IdentityVerified || res.Evidence.Detail == "" {
		t.Fatalf("evidence = %+v", res.Evidence)
	}
}

func TestExecuteVolumeCheckDoesNotReportAChange(t *testing.T) {
	useVolumeFixture(t, "", map[string]string{"E26A883E6A881189": "/dev/sda1"})
	executor := &recordingExecutor{out: []byte("Volume is dirty.")}

	res := executeVolume(context.Background(), executor, volumeRequest(ActionVolumeFilesystemCheck))

	if res.Status != "verified" || res.Changed {
		t.Fatalf("result = %+v, want a non-mutating verification", res)
	}
	if strings.Join(executor.args, " ") != "-n /dev/sda1" {
		t.Fatalf("check must use the no-action mode, ran %v", executor.args)
	}
}

func TestExecuteVolumeReportsAMissingTool(t *testing.T) {
	useVolumeFixture(t, "", map[string]string{"E26A883E6A881189": "/dev/sda1"})
	executor := &recordingExecutor{err: exec.ErrNotFound}

	res := executeVolume(context.Background(), executor, volumeRequest(ActionVolumeFilesystemRepair))

	if res.Status != "unavailable" || res.Code != "filesystem_tool_unavailable" {
		t.Fatalf("result = %+v, want an unavailable tool report", res)
	}
}

// e2fsck returns 1 exactly when it corrected errors. Treating every non-zero
// status as failure would report a successful repair as a failure.
func TestVolumeExitInterpretation(t *testing.T) {
	cases := []struct {
		filesystem string
		exitCode   int
		want       bool
	}{
		{"ext4", 0, true},
		{"ext4", 1, true},
		{"ext4", 2, true},
		{"ext4", 4, false},
		{"ext4", 8, false},
		{"vfat", 1, true},
		{"vfat", 2, false},
		{"ntfs3", 1, false},
		{"exfat", 1, false},
	}
	for _, tc := range cases {
		got := volumeExitAcceptable(tc.filesystem, tc.exitCode, errors.New("exit status"))
		if got != tc.want {
			t.Fatalf("%s exit %d acceptable = %v, want %v", tc.filesystem, tc.exitCode, got, tc.want)
		}
	}
	if !volumeExitAcceptable("ntfs3", 0, nil) {
		t.Fatal("a clean exit must be acceptable")
	}
	if volumeExitAcceptable("ext4", -1, errors.New("spawn failure")) {
		t.Fatal("a non-exit failure must not be acceptable")
	}
}
