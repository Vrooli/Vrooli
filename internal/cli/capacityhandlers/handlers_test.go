package capacityhandlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	capacityapp "github.com/vrooli/vrooli/internal/app/capacity"
	engine "github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/cli/capacitycli"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
)

func TestCommandHandlersConstructServiceBeforeParsing(t *testing.T) {
	wantErr := errors.New("output format unavailable")
	deps := rootcli.HandlerDeps[string]{
		Stdout: func(string) io.Writer { return &bytes.Buffer{} },
		OutputFormat: func(string) (cliout.Format, error) {
			return "", wantErr
		},
	}

	for name, handler := range commandtree.BuildHandlerMap(buildCommandTable(deps)) {
		t.Run(name, func(t *testing.T) {
			if err := handler("ctx", []string{"--invalid"}); !errors.Is(err, wantErr) {
				t.Fatalf("error = %v, want %v", err, wantErr)
			}
		})
	}
}

func TestReadPrivateLeaseFileRequiresExactPrivateRegularFile(t *testing.T) {
	root := t.TempDir()
	valid := filepath.Join(root, "lease.json")
	content, err := json.Marshal(diskLeaseFile{ID: "lease-id", Capability: "secret", ProcessID: 17, ProcessStart: "42"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(valid, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readPrivateLeaseFile(valid); err != nil {
		t.Fatalf("read exact 0600 lease: %v", err)
	}

	public := filepath.Join(root, "public.json")
	if err := os.WriteFile(public, content, 0o640); err != nil {
		t.Fatal(err)
	}
	if _, err := readPrivateLeaseFile(public); err == nil {
		t.Fatal("accepted lease credential readable by group")
	}

	link := filepath.Join(root, "lease-link.json")
	if err := os.Symlink(valid, link); err != nil {
		t.Fatal(err)
	}
	if _, err := readPrivateLeaseFile(link); err == nil {
		t.Fatal("accepted a symlink as a private lease file")
	}
}

func TestDiskJobCLIHandlersPersistPrivateLeaseAndActualDeltaReceipt(t *testing.T) {
	ctx := "test"
	root := t.TempDir()
	target := filepath.Join(root, "output")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	ledgerPath := filepath.Join(root, "ledger", "capacity.db")
	if err := os.MkdirAll(filepath.Dir(ledgerPath), 0o700); err != nil {
		t.Fatal(err)
	}
	zeroFloor := int64(0)
	var stdout bytes.Buffer
	deps := rootcli.HandlerDeps[string]{
		Stdout:       func(string) io.Writer { return &stdout },
		OutputFormat: func(string) (cliout.Format, error) { return cliout.FormatJSON, nil },
		ServiceFor: func(string) capacityapp.Service {
			return capacityapp.Service{OpenStore: func(ctx context.Context) (capacityapp.Store, error) {
				return engine.NewSQLiteStore(ctx, engine.Config{DBPath: ledgerPath, MinimumDiskFloorBytes: &zeroFloor})
			}}
		},
	}
	handlers := commandtree.BuildHandlerMap(buildCommandTable(deps))
	call := func(command capacitycli.CommandID, args ...string) string {
		t.Helper()
		stdout.Reset()
		if err := handlers[string(command)](ctx, args); err != nil {
			t.Fatalf("vrooli capacity %s %v: %v", command, args, err)
		}
		return stdout.String()
	}

	start, err := os.ReadFile("/proc/self/stat")
	if err != nil {
		t.Fatal(err)
	}
	closeParen := strings.LastIndexByte(string(start), ')')
	fields := strings.Fields(string(start[closeParen+2:]))
	if len(fields) <= 19 {
		t.Fatal("process stat lacks a start identity")
	}
	startTicks, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	requestPath := filepath.Join(root, "request.json")
	leasePath := filepath.Join(root, "private", "lease.json")
	request, err := json.Marshal(engine.DiskJobRequest{
		OwnerID: "capacity-cli-test", JobID: "cli-receipt-e2e", ProcessID: os.Getpid(), ProcessStart: strconv.FormatUint(startTicks, 10),
		Paths: []engine.DiskJobPath{{Path: target, Role: "output-and-temp", PeakBytes: 2 << 20}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(requestPath, request, 0o600); err != nil {
		t.Fatal(err)
	}
	reserveOutput := call(capacitycli.CommandDiskReserve, "--request-file", requestPath, "--lease-file", leasePath, "--json")
	if strings.Contains(reserveOutput, "capability") {
		t.Fatalf("secret capability leaked to normal receipt output: %s", reserveOutput)
	}
	leaseInfo, err := os.Stat(leasePath)
	if err != nil || leaseInfo.Mode().Perm() != 0o600 {
		t.Fatalf("private lease file stat=%v error=%v", leaseInfo, err)
	}
	var private diskLeaseFile
	leaseBytes, err := os.ReadFile(leasePath)
	if err != nil || json.Unmarshal(leaseBytes, &private) != nil || private.ID == "" || private.Capability == "" {
		t.Fatalf("private lease credential is incomplete: %s (%v)", leaseBytes, err)
	}
	if !strings.Contains(reserveOutput, private.ID) {
		t.Fatalf("reserve receipt %s does not identify private lease %s", reserveOutput, private.ID)
	}
	call(capacitycli.CommandDiskSample, "--lease-file", leasePath, "--path", target, "--bytes", "0", "--json")
	stdout.Reset()
	if err := handlers[string(capacitycli.CommandDiskReconcile)](ctx, []string{"--lease-id", private.ID, "--process-stopped", "--cleanup-verified", "--json"}); err == nil || !strings.Contains(err.Error(), "still running") {
		t.Fatalf("reconcile released a lease bound to the live producer: %v", err)
	}
	activeBeforeHeartbeat := call(capacitycli.CommandDiskList, "--active", "--json")
	if !strings.Contains(activeBeforeHeartbeat, private.ID) {
		t.Fatalf("stale-owner refusal did not retain the active lease: %s", activeBeforeHeartbeat)
	}
	var currentHeadroom engine.DiskHeadroomSnapshot
	headroomOutput := call(capacitycli.CommandDiskList, "--observe", "--json")
	if err := json.Unmarshal([]byte(headroomOutput), &currentHeadroom); err != nil {
		t.Fatalf("decode refreshed filesystem headroom %s: %v", headroomOutput, err)
	}
	if len(currentHeadroom.Filesystems) != 1 {
		t.Fatalf("refreshed filesystem headroom output = %s, parsed=%+v", headroomOutput, currentHeadroom)
	}
	if len(currentHeadroom.ActiveJobs) != 1 || currentHeadroom.ActiveJobs[0].ID != private.ID {
		t.Fatalf("headroom snapshot omitted the active job that contributes to its aggregate: %+v", currentHeadroom.ActiveJobs)
	}
	if currentHeadroom.Filesystems[0].CurrentFreeBytes <= 0 || currentHeadroom.Filesystems[0].ActiveReservedBytes < 2<<20 {
		t.Fatalf("headroom did not refresh free space and include the active lease: %+v", currentHeadroom)
	}

	call(capacitycli.CommandDiskHeartbeat, "--lease-file", leasePath, "--json")
	call(capacitycli.CommandDiskObserve, "--lease-file", leasePath, "--json")
	pathsPath := filepath.Join(root, "extension.json")
	paths, err := json.Marshal([]engine.DiskJobPath{{Path: target, Role: "output-and-temp", PeakBytes: 4 << 20}})
	if err != nil || os.WriteFile(pathsPath, paths, 0o600) != nil {
		t.Fatalf("write extension paths: %v", err)
	}
	call(capacitycli.CommandDiskExtend, "--lease-file", leasePath, "--paths-file", pathsPath, "--json")
	if err := os.WriteFile(filepath.Join(target, "output.bin"), bytes.Repeat([]byte{0x37}, 1<<20), 0o600); err != nil {
		t.Fatal(err)
	}
	sampleOutput := call(capacitycli.CommandDiskSample, "--lease-file", leasePath, "--path", target, "--bytes", strconv.Itoa(1<<20), "--json")
	if !strings.Contains(sampleOutput, `"delta_bytes": 1048576`) {
		t.Fatalf("path sample receipt omitted measured growth: %s", sampleOutput)
	}
	completeOutput := call(capacitycli.CommandDiskComplete, "--lease-file", leasePath, "--cleanup-verified", "--json")
	if !strings.Contains(completeOutput, `"status": "completed"`) || !strings.Contains(completeOutput, `"actual_delta_bytes"`) {
		t.Fatalf("terminal output lacks completion/growth receipt: %s", completeOutput)
	}
	if _, err := os.Stat(leasePath); !os.IsNotExist(err) {
		t.Fatalf("completed private lease credential was retained: %v", err)
	}
	listOutput := call(capacitycli.CommandDiskList, "--active", "--json")
	if !strings.Contains(listOutput, "[]") {
		t.Fatalf("active disk-list after completion = %s", listOutput)
	}
}
