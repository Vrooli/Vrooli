// Package n8n owns the task-scoped n8n container and its private, egress-denied
// network. Callers identify a sandbox and a companion runtime; they never
// select Docker objects, images, commands, paths, or networks.
package n8n

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/vrooli/internal/capacity"
)

const (
	ImageRef                       = "n8nio/n8n@sha256:416c29f75a0f9ac65b4339761a372af48f4a7e948de9df0d837b080d2b9d47a8"
	ImageVersion                   = "2.48.0"
	SyntheticPostgresImage         = "postgres:16.4-alpine@sha256:5660c2cbfea50c7a9127d17dc4e48543eedd3d7a41a595a2dfa572471e37e64c"
	SyntheticPostgresVersion       = "16.4"
	SyntheticPostgresName          = "pg"
	SyntheticPostgresDB            = "workflow"
	ImagePeakBytes           int64 = 3 * 1024 * 1024 * 1024
	DiskFloorBytes           int64 = 120 * 1024 * 1024 * 1024
	imageOwnerID                   = "resource:n8n:image:2.48.0"
	readinessTimeout               = 90 * time.Second
	containerStopWait              = 10
	// MaxSyntheticWorkflowBytes is also enforced by the control-plane stdin boundary.
	MaxSyntheticWorkflowBytes  = 10 << 20
	maxSyntheticWorkflowJSON   = MaxSyntheticWorkflowBytes
	maxWorkflowExecutionOut    = 16 << 20
	maxCredentialImportOutput  = 256 << 10
	syntheticWorkflowRunTTL    = 5 * time.Minute
	maxSyntheticCredentialJSON = 1 << 20
	stageWorkflowFileProgram   = `const fs=require("fs");const p=process.argv[1];const flags=fs.constants.O_WRONLY|fs.constants.O_CREAT|fs.constants.O_TRUNC|fs.constants.O_NOFOLLOW;const fd=fs.openSync(p,flags,0o600);process.stdin.on("data",b=>{for(let o=0;o<b.length;)o+=fs.writeSync(fd,b,o,b.length-o)});process.stdin.on("error",()=>{try{fs.closeSync(fd)}catch{};process.exitCode=1});process.stdin.on("end",()=>fs.closeSync(fd));`
)

const (
	labelResource          = "com.vrooli.resource"
	labelSandboxID         = "com.vrooli.sandbox-id"
	labelRuntimeID         = "com.vrooli.runtime-id"
	labelObjectKind        = "com.vrooli.object-kind"
	labelSandboxOwner      = "com.vrooli.workspace-sandbox.sandbox-id"
	labelSandboxRunID      = "com.vrooli.workspace-sandbox.runtime-id"
	labelSandboxDecl       = "com.vrooli.workspace-sandbox.companion-declared"
	labelCompanionResource = "com.vrooli.resource-companion"
	privateNetworkName     = "bridge"
)

var companionRuntimePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$`)

var (
	ErrDockerUnavailable = errors.New("n8n runtime: Docker daemon unavailable")
	ErrImageMissing      = errors.New("n8n runtime: pinned image is not installed")
	ErrForeignObject     = errors.New("n8n runtime: refusing to modify an object without matching ownership labels")
	ErrCompanionDenied   = errors.New("n8n runtime: companion is not declared for this sandbox")
)

// CommandRunner is the only Docker seam. Production supplies the Docker CLI;
// focused owner tests use a fake and never contact a daemon.
type CommandRunner interface {
	Run(context.Context, ...string) ([]byte, error)
}

type quietCommandRunner interface {
	RunQuiet(context.Context, ...string) ([]byte, error)
	RunInputQuiet(context.Context, []byte, ...string) ([]byte, error)
	RunStdoutLimitQuiet(context.Context, int, ...string) ([]byte, bool, error)
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("docker %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func (execRunner) RunQuiet(ctx context.Context, args ...string) ([]byte, error) {
	return runDocker(ctx, nil, args...)
}

func (execRunner) RunInputQuiet(ctx context.Context, input []byte, args ...string) ([]byte, error) {
	return runDocker(ctx, bytes.NewReader(input), args...)
}

func (execRunner) RunStdoutLimitQuiet(ctx context.Context, limit int, args ...string) ([]byte, bool, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	stdout := &limitedCapture{limit: limit}
	cmd.Stdout = stdout
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return nil, stdout.exceeded, fmt.Errorf("docker %s failed: %w", strings.Join(args, " "), err)
	}
	return stdout.buffer.Bytes(), stdout.exceeded, nil
}

type limitedCapture struct {
	limit    int
	buffer   bytes.Buffer
	exceeded bool
}

func (w *limitedCapture) Write(data []byte) (int, error) {
	remaining := w.limit - w.buffer.Len()
	if remaining > len(data) {
		remaining = len(data)
	}
	if remaining > 0 {
		_, _ = w.buffer.Write(data[:remaining])
	}
	if remaining < len(data) {
		w.exceeded = true
	}
	// Keep draining stdout while retaining no more than the configured limit.
	return len(data), nil
}

func runDocker(ctx context.Context, input io.Reader, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	if input != nil {
		cmd.Stdin = input
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker %s failed: %w", strings.Join(args, " "), err)
	}
	return output, nil
}

// DiskLease keeps the host admission owner's reservation alive throughout a
// complete OCI acquisition and makes the release part of the terminal path.
type DiskLease interface {
	Heartbeat(context.Context) error
	Release(context.Context) error
}

type DiskReservationDetails struct {
	ID                 string
	Path               string
	PeakBytes          int64
	FreeBytes          int64
	OtherReservedBytes int64
	OtherReservations  []DiskReservationSummary
}

type DiskReservationSummary struct {
	OwnerID    string `json:"ownerId"`
	Path       string `json:"path"`
	PeakBytes  int64  `json:"peakBytes"`
	FloorBytes int64  `json:"floorBytes"`
	Status     string `json:"status"`
}

type leaseDetails interface {
	Details(context.Context) (DiskReservationDetails, error)
}

type DiskAdmission interface {
	Reserve(context.Context, string, string, int64, int64) (DiskLease, error)
}

type capacityAdmission struct{ home string }

func (a capacityAdmission) Reserve(ctx context.Context, ownerID, path string, peak, floor int64) (DiskLease, error) {
	store, err := capacity.NewSQLiteStore(ctx, capacity.Config{HomeDir: a.home})
	if err != nil {
		return nil, fmt.Errorf("open host capacity admission owner: %w", err)
	}
	reservation, err := store.ReserveDisk(ctx, ownerID, path, peak, floor)
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	return &capacityLease{store: store, id: reservation.ID, reservation: reservation}, nil
}

type capacityLease struct {
	store       *capacity.SQLiteStore
	id          string
	reservation capacity.DiskReservation
}

func (l *capacityLease) Heartbeat(ctx context.Context) error { return l.store.HeartbeatDisk(ctx, l.id) }
func (l *capacityLease) Details(ctx context.Context) (DiskReservationDetails, error) {
	details := DiskReservationDetails{ID: l.id, Path: l.reservation.Path, PeakBytes: l.reservation.PeakBytes, FreeBytes: l.reservation.Available, OtherReservedBytes: l.reservation.Reserved}
	rows, err := l.store.ListDiskReservations(ctx, capacity.DiskReservationFilter{FilesystemID: l.reservation.FilesystemID, ActiveOnly: true})
	if err != nil {
		return details, err
	}
	for _, row := range rows {
		if row.ID == l.id {
			continue
		}
		details.OtherReservations = append(details.OtherReservations, DiskReservationSummary{OwnerID: row.OwnerID, Path: row.Path, PeakBytes: row.PeakBytes, FloorBytes: row.FloorBytes, Status: row.Status})
	}
	return details, nil
}

func (l *capacityLease) Release(ctx context.Context) error {
	releaseErr := l.store.ReleaseDisk(ctx, l.id)
	closeErr := l.store.Close()
	return errors.Join(releaseErr, closeErr)
}

// Owner owns only containers and networks whose labels match the F7 resource
// contract. The caller should use NewOwner so the pinned manifest image and
// capacity admission owner are authoritative.
type Owner struct {
	Image      string
	Home       string
	Runner     CommandRunner
	Admission  DiskAdmission
	HTTPClient *http.Client
	PollEvery  time.Duration
	ImagePeak  int64
	DiskFloor  int64
	// WebhookPublishWait bounds testable publication-service reconciliation.
	WebhookPublishWait time.Duration
}

func NewOwner(home string) *Owner {
	return &Owner{
		Image:      ImageRef,
		Home:       home,
		Runner:     execRunner{},
		Admission:  capacityAdmission{home: home},
		HTTPClient: &http.Client{Timeout: 2 * time.Second},
		PollEvery:  2 * time.Second,
		ImagePeak:  ImagePeakBytes,
		DiskFloor:  DiskFloorBytes,
	}
}

type ImageReceipt struct {
	Image              string                   `json:"image"`
	Version            string                   `json:"version"`
	Acquired           bool                     `json:"acquired"`
	AlreadyPresent     bool                     `json:"alreadyPresent"`
	DiskPath           string                   `json:"diskPath,omitempty"`
	PeakBytesReserved  int64                    `json:"peakBytesReserved,omitempty"`
	FreeBytesAtStart   int64                    `json:"freeBytesAtStart,omitempty"`
	DiskReservationID  string                   `json:"diskReservationId,omitempty"`
	ImageSizeBytes     int64                    `json:"imageSizeBytes,omitempty"`
	ReusePolicy        string                   `json:"reusePolicy,omitempty"`
	EvictionRule       string                   `json:"evictionRule,omitempty"`
	OtherReservedBytes int64                    `json:"otherReservedBytes,omitempty"`
	ActiveReservations []DiskReservationSummary `json:"activeReservations,omitempty"`
}

type Runtime struct {
	SandboxID   string `json:"sandboxId"`
	RuntimeID   string `json:"runtimeId"`
	Endpoint    string `json:"endpoint"`
	Alias       string `json:"networkAlias"`
	Status      string `json:"status"`
	HealthError string `json:"healthError,omitempty"`
}

type CompanionLink struct {
	SandboxID          string `json:"sandboxId"`
	RuntimeID          string `json:"runtimeId"`
	CompanionRuntimeID string `json:"companionRuntimeId"`
	NetworkAlias       string `json:"networkAlias"`
	Status             string `json:"status"`
}

type StopResult struct {
	SandboxID string   `json:"sandboxId"`
	RuntimeID string   `json:"runtimeId"`
	Status    string   `json:"status"`
	Retained  []string `json:"retained,omitempty"`
}

// SyntheticWorkflowImport is one workflow and its owner idempotency key.
// Credential references are accepted only for credential types with an
// explicit synthetic profile below; exported credential data is never accepted.
type SyntheticWorkflowImport struct {
	OperationID                uuid.UUID
	JSON                       []byte
	PostgresCompanionRuntimeID string
}

type WorkflowImportReceipt struct {
	SandboxID       string `json:"sandboxId"`
	RuntimeID       string `json:"runtimeId"`
	OperationID     string `json:"operationId"`
	WorkflowID      string `json:"workflowId"`
	Status          string `json:"status"`
	CredentialCount int    `json:"credentialCount,omitempty"`
	InputRetained   bool   `json:"inputRetained,omitempty"`
}

type WorkflowExecutionReceipt struct {
	SandboxID   string                    `json:"sandboxId"`
	RuntimeID   string                    `json:"runtimeId"`
	OperationID string                    `json:"operationId"`
	WorkflowID  string                    `json:"workflowId"`
	Status      string                    `json:"status"`
	Evidence    WorkflowExecutionEvidence `json:"evidence"`
}

// WorkflowExecutionEvidence contains only content-free execution measurements.
// n8n output values and node names never cross the owner boundary.
type WorkflowExecutionEvidence struct {
	OutputSHA256             string `json:"outputSha256"`
	OutputBytes              int    `json:"outputBytes"`
	NodeCount                int    `json:"nodeCount"`
	ExecutionCount           int    `json:"executionCount"`
	FailedNodeExecutionCount int    `json:"failedNodeExecutionCount"`
	ItemCount                int    `json:"itemCount"`
}

func (o *Owner) runner() CommandRunner {
	if o.Runner != nil {
		return o.Runner
	}
	return execRunner{}
}

func (o *Owner) client() *http.Client {
	if o.HTTPClient != nil {
		return o.HTTPClient
	}
	return &http.Client{Timeout: 2 * time.Second}
}

// AcquireImage is the sole pull path. It reserves a peak on the daemon's
// actual DockerRootDir filesystem with a 120 GiB post-peak floor, heartbeats
// for the full pull, and never broadens to a floating tag.
func (o *Owner) AcquireImage(ctx context.Context) (ImageReceipt, error) {
	receipt := ImageReceipt{Image: o.image(), Version: ImageVersion}
	root, err := o.dockerRoot(ctx)
	if err != nil {
		return receipt, err
	}
	if ok, err := o.imagePresent(ctx); err != nil {
		return receipt, err
	} else if ok {
		receipt.AlreadyPresent = true
		receipt.ReusePolicy, receipt.EvictionRule = retainedImagePolicy()
		metadata, inspectErr := o.runner().Run(ctx, "image", "inspect", "--format", "{{.Size}}", o.image())
		if inspectErr != nil {
			return receipt, fmt.Errorf("inspect retained pinned n8n image size: %w", inspectErr)
		}
		receipt.ImageSizeBytes, err = strconv.ParseInt(strings.TrimSpace(string(metadata)), 10, 64)
		if err != nil || receipt.ImageSizeBytes <= 0 {
			return receipt, fmt.Errorf("Docker returned an invalid retained n8n image size %q", strings.TrimSpace(string(metadata)))
		}
		return receipt, nil
	}
	if o.Admission == nil {
		return receipt, fmt.Errorf("disk admission owner is unavailable; refusing image acquisition")
	}
	peak, floor := o.ImagePeak, o.DiskFloor
	if peak <= 0 {
		peak = ImagePeakBytes
	}
	if floor <= 0 {
		floor = DiskFloorBytes
	}
	lease, err := o.Admission.Reserve(ctx, imageOwnerID, root, peak, floor)
	if err != nil {
		return receipt, fmt.Errorf("admit n8n image acquisition on %s: %w", root, err)
	}
	leaseReleased := false
	defer func() {
		if !leaseReleased {
			_ = lease.Release(context.Background())
		}
	}()
	receipt.DiskPath, receipt.PeakBytesReserved = root, peak
	provider, ok := lease.(leaseDetails)
	if !ok {
		return receipt, fmt.Errorf("disk admission owner cannot list the active reservations; refusing image acquisition")
	}
	admission, err := provider.Details(ctx)
	if err != nil {
		return receipt, fmt.Errorf("reconcile active disk reservations on %s: %w", root, err)
	}
	receipt.DiskReservationID = admission.ID
	receipt.FreeBytesAtStart = admission.FreeBytes
	receipt.OtherReservedBytes = admission.OtherReservedBytes
	receipt.ActiveReservations = admission.OtherReservations
	if metadata, err := o.runner().Run(ctx, "image", "inspect", "--format", "{{.Size}}", o.image()); err == nil {
		receipt.AlreadyPresent = true
		receipt.ImageSizeBytes, err = strconv.ParseInt(strings.TrimSpace(string(metadata)), 10, 64)
		if err != nil || receipt.ImageSizeBytes <= 0 {
			return receipt, fmt.Errorf("Docker returned an invalid retained n8n image size %q", strings.TrimSpace(string(metadata)))
		}
		receipt.ReusePolicy, receipt.EvictionRule = retainedImagePolicy()
		if releaseErr := lease.Release(ctx); releaseErr != nil {
			return receipt, fmt.Errorf("release n8n image admission after concurrent image acquisition: %w", releaseErr)
		}
		leaseReleased = true
		return receipt, nil
	}
	if err := o.pullWithLease(ctx, lease); err != nil {
		return receipt, err
	}
	metadata, err := o.runner().Run(ctx, "image", "inspect", "--format", "{{.Size}}", o.image())
	if err != nil {
		return receipt, fmt.Errorf("pinned n8n image pull completed but verification failed: %w", err)
	}
	receipt.ImageSizeBytes, err = strconv.ParseInt(strings.TrimSpace(string(metadata)), 10, 64)
	if err != nil || receipt.ImageSizeBytes <= 0 {
		return receipt, fmt.Errorf("Docker returned an invalid acquired n8n image size %q", strings.TrimSpace(string(metadata)))
	}
	receipt.ReusePolicy, receipt.EvictionRule = retainedImagePolicy()
	receipt.Acquired = true
	if releaseErr := lease.Release(ctx); releaseErr != nil {
		return receipt, fmt.Errorf("release n8n image disk admission: %w", releaseErr)
	}
	leaseReleased = true
	return receipt, nil
}

func (o *Owner) pullWithLease(ctx context.Context, lease DiskLease) error {
	pullCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	interval := 5 * time.Minute
	if o.PollEvery > 0 && o.PollEvery < interval {
		interval = o.PollEvery
	}
	beatDone := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-pullCtx.Done():
				beatDone <- nil
				return
			case <-ticker.C:
				if err := lease.Heartbeat(pullCtx); err != nil {
					beatDone <- err
					cancel()
					return
				}
			}
		}
	}()
	_, pullErr := o.runner().Run(pullCtx, "pull", o.image())
	cancel()
	beatErr := <-beatDone
	if beatErr != nil {
		return fmt.Errorf("disk admission heartbeat lost during n8n image acquisition: %w", beatErr)
	}
	if pullErr != nil {
		return fmt.Errorf("pull pinned n8n image: %w", pullErr)
	}
	return nil
}

func (o *Owner) Start(ctx context.Context, sandboxID string) (Runtime, error) {
	canonical, err := normalizeSandboxID(sandboxID)
	if err != nil {
		return Runtime{}, err
	}
	if _, err := o.dockerRoot(ctx); err != nil {
		return Runtime{}, err
	}
	if ok, err := o.imagePresent(ctx); err != nil {
		return Runtime{}, err
	} else if !ok {
		return Runtime{}, fmt.Errorf("%w: run `vrooli resource n8n runtime acquire --json` after host storage admission", ErrImageMissing)
	}
	runtimeID := runtimeIDFor(canonical)
	containerName, networkName := objectNames(canonical)
	if existing, found, err := o.inspectContainer(ctx, containerName); err != nil {
		return Runtime{}, err
	} else if found {
		if err := verifyN8nLabels(existing.Config.Labels, canonical, runtimeID, "container"); err != nil {
			return Runtime{}, err
		}
		if !existing.State.Running {
			if _, err := o.runner().Run(ctx, "start", containerName); err != nil {
				return Runtime{}, fmt.Errorf("restart task-owned n8n container: %w", err)
			}
		}
		result, err := o.runtimeFromContainer(ctx, canonical, runtimeID, existing)
		if err != nil {
			return Runtime{}, err
		}
		if err := o.waitReady(ctx, result.Endpoint); err != nil {
			return Runtime{}, err
		}
		result.Status = "running"
		return result, nil
	}
	if _, found, err := o.inspectNetwork(ctx, networkName, canonical, runtimeID); err != nil {
		return Runtime{}, err
	} else if found {
		return Runtime{}, fmt.Errorf("task-owned n8n network exists without its matching container; run stop to reconcile before retrying")
	}
	if _, err := o.runner().Run(ctx, "network", "create", "--driver", privateNetworkName, "--internal",
		"--label", labelResource+"=n8n",
		"--label", labelSandboxID+"="+canonical,
		"--label", labelRuntimeID+"="+runtimeID,
		"--label", labelObjectKind+"=network", networkName); err != nil {
		return Runtime{}, fmt.Errorf("create private n8n sandbox network: %w", err)
	}
	endpoint, hostPort, err := chooseLoopbackPort()
	if err != nil {
		_ = o.removeOwnedNetwork(context.Background(), networkName, canonical, runtimeID)
		return Runtime{}, err
	}
	alias := n8nAlias(canonical)
	args := []string{
		"run", "--detach", "--name", containerName,
		"--network", networkName, "--network-alias", alias,
		"--publish", "127.0.0.1:" + hostPort + ":5678/tcp",
		"--read-only", "--cap-drop", "ALL", "--security-opt", "no-new-privileges:true",
		"--pids-limit", "512", "--memory", "2g", "--memory-swap", "2g", "--cpus", "2",
		"--tmpfs", "/home/node/.n8n:rw,noexec,nosuid,nodev,size=1g,uid=1000,gid=1000,mode=0700",
		"--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=256m,uid=1000,gid=1000,mode=0700",
		"--label", labelResource + "=n8n",
		"--label", labelSandboxID + "=" + canonical,
		"--label", labelRuntimeID + "=" + runtimeID,
		"--label", labelObjectKind + "=container",
		"--env", "N8N_HOST=127.0.0.1",
		"--env", "N8N_LISTEN_ADDRESS=0.0.0.0",
		"--env", "N8N_PORT=5678",
		"--env", "N8N_PROTOCOL=http",
		"--env", "N8N_EDITOR_BASE_URL=" + endpoint + "/",
		"--env", "WEBHOOK_URL=http://" + alias + ":5678/",
		"--env", "N8N_USER_FOLDER=/home/node/.n8n",
		"--env", "N8N_DEFAULT_BINARY_DATA_MODE=filesystem",
		"--env", "N8N_DIAGNOSTICS_ENABLED=false",
		"--env", "N8N_PERSONALIZATION_ENABLED=false",
		"--env", "N8N_VERSION_NOTIFICATIONS_ENABLED=false",
		"--env", "N8N_USE_WORKFLOW_PUBLICATION_SERVICE=true",
		"--env", "N8N_TEMPLATES_ENABLED=false",
		"--env", "N8N_ENFORCE_SETTINGS_FILE_PERMISSIONS=true",
		"--env", "N8N_BLOCK_FILE_ACCESS_TO_N8N_FILES=true",
		"--env", "N8N_BLOCK_ENV_ACCESS_IN_NODE=true",
		"--env", "GENERIC_TIMEZONE=UTC",
		"--env", "TZ=UTC",
		o.image(),
	}
	if _, err := o.runner().Run(ctx, args...); err != nil {
		cleanupErr := o.cleanupAfterStartFailure(context.Background(), containerName, networkName, canonical, runtimeID)
		return Runtime{}, errors.Join(fmt.Errorf("create task-owned n8n container: %w", err), cleanupErr)
	}
	result := Runtime{SandboxID: canonical, RuntimeID: runtimeID, Endpoint: endpoint, Alias: alias, Status: "starting"}
	if err := o.waitReady(ctx, endpoint); err != nil {
		_, cleanupErr := o.Stop(context.Background(), canonical, runtimeID)
		return Runtime{}, errors.Join(err, cleanupErr)
	}
	result.Status = "running"
	return result, nil
}

func (o *Owner) Status(ctx context.Context, sandboxID string) (Runtime, error) {
	canonical, err := normalizeSandboxID(sandboxID)
	if err != nil {
		return Runtime{}, err
	}
	if _, err := o.dockerRoot(ctx); err != nil {
		return Runtime{}, err
	}
	runtimeID := runtimeIDFor(canonical)
	containerName, networkName := objectNames(canonical)
	container, found, err := o.inspectContainer(ctx, containerName)
	if err != nil {
		return Runtime{}, err
	}
	if !found {
		if _, networkFound, err := o.inspectNetwork(ctx, networkName, canonical, runtimeID); err != nil {
			return Runtime{}, err
		} else if networkFound {
			return Runtime{SandboxID: canonical, RuntimeID: runtimeID, Status: "cleanup-required"}, nil
		}
		return Runtime{SandboxID: canonical, RuntimeID: runtimeID, Status: "stopped"}, nil
	}
	if err := verifyN8nLabels(container.Config.Labels, canonical, runtimeID, "container"); err != nil {
		return Runtime{}, err
	}
	result, err := o.runtimeFromContainer(ctx, canonical, runtimeID, container)
	if err != nil {
		return Runtime{}, err
	}
	if !container.State.Running {
		result.Status = "stopped"
		return result, nil
	}
	if err := o.probeReady(ctx, result.Endpoint); err != nil {
		result.Status = "unhealthy"
		result.HealthError = err.Error()
		return result, nil
	}
	result.Status = "running"
	return result, nil
}

// ImportSyntheticWorkflow imports exactly one inactive workflow into a
// verified Sandbox-owned task runtime. The JSON is staged only into the
// runtime's bounded tmpfs and removed after the Server CLI consumes it.
func (o *Owner) ImportSyntheticWorkflow(ctx context.Context, sandboxID, runtimeID string, request SyntheticWorkflowImport) (WorkflowImportReceipt, error) {
	canonical, err := normalizeSandboxID(sandboxID)
	if err != nil {
		return WorkflowImportReceipt{}, err
	}
	if runtimeID != runtimeIDFor(canonical) {
		return WorkflowImportReceipt{}, ErrForeignObject
	}
	if request.OperationID == uuid.Nil {
		return WorkflowImportReceipt{}, fmt.Errorf("workflow import operation ID must be a nonzero UUID")
	}
	workflowID := workflowIDFor(canonical, runtimeID, request.OperationID)
	workflowJSON, credentials, err := prepareSyntheticWorkflow(request.JSON, workflowID, canonical, runtimeID, request.PostgresCompanionRuntimeID)
	if err != nil {
		return WorkflowImportReceipt{}, err
	}
	canonical, containerName, err := o.ownedRunningContainer(ctx, canonical, runtimeID)
	if err != nil {
		return WorkflowImportReceipt{}, err
	}
	for _, credential := range credentials {
		if credential.Type == "postgres" {
			if err := o.verifyAttachedSyntheticCompanion(ctx, canonical, request.PostgresCompanionRuntimeID); err != nil {
				return WorkflowImportReceipt{}, err
			}
			break
		}
	}
	runner, ok := o.runner().(quietCommandRunner)
	if !ok {
		return WorkflowImportReceipt{}, fmt.Errorf("runtime command runner cannot guarantee workflow output redaction")
	}
	path := "/tmp/.vrooli-workflow-" + workflowID + ".json"
	receipt := WorkflowImportReceipt{SandboxID: canonical, RuntimeID: runtimeID, OperationID: request.OperationID.String(), WorkflowID: workflowID, Status: "importing", CredentialCount: len(credentials)}
	if len(credentials) > 0 {
		credentialPath := "/tmp/.vrooli-credentials-" + workflowID + ".json"
		credentialJSON, marshalErr := json.Marshal(credentials)
		if marshalErr != nil || len(credentialJSON) > maxSyntheticCredentialJSON {
			return WorkflowImportReceipt{}, fmt.Errorf("synthetic credentials exceed the bounded import format")
		}
		if _, err := stageWorkflowJSON(ctx, runner, containerName, credentialPath, credentialJSON); err != nil {
			cleanupErr := removeWorkflowInput(ctx, runner, containerName, credentialPath)
			receipt.Status = "credential-stage-failed"
			if cleanupErr != nil {
				receipt.Status, receipt.InputRetained = "import-failed-cleanup-required", true
			}
			return receipt, errors.Join(fmt.Errorf("stage synthetic credentials for workflow %s: %w", workflowID, err), cleanupErr)
		}
		credentialOutput, outputExceeded, err := runner.RunStdoutLimitQuiet(ctx, maxCredentialImportOutput, "exec", "--user", "node", containerName, "n8n", "import:credentials", "--input="+credentialPath)
		cleanupErr := removeWorkflowInput(ctx, runner, containerName, credentialPath)
		if err != nil || outputExceeded || !strings.Contains(string(credentialOutput), fmt.Sprintf("Successfully imported %d credential", len(credentials))) || strings.Contains(string(credentialOutput), "Skipped ") {
			receipt.Status = "credential-import-failed"
			if cleanupErr != nil {
				receipt.Status, receipt.InputRetained = "import-failed-cleanup-required", true
			}
			return receipt, errors.Join(fmt.Errorf("n8n did not confirm importing all synthetic credentials for workflow %s", workflowID), cleanupErr)
		}
		if cleanupErr != nil {
			receipt.Status, receipt.InputRetained = "credentials-imported-input-retained", true
			return receipt, fmt.Errorf("synthetic credentials imported but their tmpfs input could not be removed: %w", cleanupErr)
		}
	}
	if _, err := stageWorkflowJSON(ctx, runner, containerName, path, workflowJSON); err != nil {
		cleanupErr := removeWorkflowInput(ctx, runner, containerName, path)
		receipt.Status = "stage-failed"
		if cleanupErr != nil {
			receipt.Status, receipt.InputRetained = "import-failed-cleanup-required", true
		}
		return receipt, errors.Join(fmt.Errorf("stage synthetic workflow %s in task tmpfs: %w", workflowID, err), cleanupErr)
	}
	if _, err := runner.RunQuiet(ctx, "exec", "--user", "node", containerName, "n8n", "import:workflow", "--input="+path, "--activeState=false"); err != nil {
		cleanupErr := removeWorkflowInput(ctx, runner, containerName, path)
		receipt.Status = "import-failed"
		if cleanupErr != nil {
			receipt.Status, receipt.InputRetained = "import-failed-cleanup-required", true
		}
		return receipt, errors.Join(fmt.Errorf("import synthetic workflow %s through n8n Server CLI: %w", workflowID, err), cleanupErr)
	}
	if err := removeWorkflowInput(ctx, runner, containerName, path); err != nil {
		receipt.Status, receipt.InputRetained = "imported-input-retained", true
		return receipt, fmt.Errorf("workflow %s imported but its tmpfs input could not be removed: %w", workflowID, err)
	}
	receipt.Status = "imported-inactive"
	return receipt, nil
}

func stageWorkflowJSON(ctx context.Context, runner quietCommandRunner, containerName, path string, content []byte) ([]byte, error) {
	if !strings.HasPrefix(path, "/tmp/.vrooli-") || strings.Contains(path, "..") || !strings.HasSuffix(path, ".json") {
		return nil, fmt.Errorf("invalid owner-generated tmpfs staging path")
	}
	return runner.RunInputQuiet(ctx, content, "exec", "--interactive", "--user", "node", containerName, "node", "-e", stageWorkflowFileProgram, path)
}

// ExecuteSyntheticWorkflow selects only the deterministic workflow ID issued
// for this Sandbox and operation key. It never accepts arbitrary n8n commands
// or workflow IDs, and applies a fixed five-minute execution deadline.
func (o *Owner) ExecuteSyntheticWorkflow(ctx context.Context, sandboxID, runtimeID string, operationID uuid.UUID) (WorkflowExecutionReceipt, error) {
	canonical, err := normalizeSandboxID(sandboxID)
	if err != nil {
		return WorkflowExecutionReceipt{}, err
	}
	if runtimeID != runtimeIDFor(canonical) {
		return WorkflowExecutionReceipt{}, ErrForeignObject
	}
	if operationID == uuid.Nil {
		return WorkflowExecutionReceipt{}, fmt.Errorf("workflow execution operation ID must be a nonzero UUID")
	}
	canonical, containerName, err := o.ownedRunningContainer(ctx, canonical, runtimeID)
	if err != nil {
		return WorkflowExecutionReceipt{}, err
	}
	runner, ok := o.runner().(quietCommandRunner)
	if !ok {
		return WorkflowExecutionReceipt{}, fmt.Errorf("runtime command runner cannot guarantee workflow output redaction")
	}
	workflowID := workflowIDFor(canonical, runtimeID, operationID)
	runCtx, cancel := context.WithTimeout(ctx, syntheticWorkflowRunTTL)
	defer cancel()
	output, outputExceeded, err := runner.RunStdoutLimitQuiet(runCtx, maxWorkflowExecutionOut, "exec", "--user", "node", containerName, "n8n", "execute", "--id", workflowID, "--rawOutput")
	receipt := WorkflowExecutionReceipt{SandboxID: canonical, RuntimeID: runtimeID, OperationID: operationID.String(), WorkflowID: workflowID, Status: "failed"}
	if err != nil {
		return receipt, fmt.Errorf("synthetic workflow execution failed")
	}
	if outputExceeded || len(output) > maxWorkflowExecutionOut {
		return receipt, fmt.Errorf("synthetic workflow %s output exceeded %d bytes", workflowID, maxWorkflowExecutionOut)
	}
	evidence, err := summarizeWorkflowExecution(output)
	if err != nil {
		return receipt, fmt.Errorf("synthetic workflow %s returned an unsupported result shape", workflowID)
	}
	receipt.Status = "completed"
	if evidence.FailedNodeExecutionCount > 0 {
		receipt.Status = "completed-with-node-errors"
	}
	receipt.Evidence = evidence
	return receipt, nil
}

func summarizeWorkflowExecution(output []byte) (WorkflowExecutionEvidence, error) {
	if len(output) == 0 || !json.Valid(output) {
		return WorkflowExecutionEvidence{}, errors.New("invalid execution JSON")
	}
	var envelope struct {
		Data struct {
			ResultData struct {
				Error   json.RawMessage `json:"error"`
				RunData map[string][]struct {
					Error           json.RawMessage                `json:"error"`
					ExecutionStatus string                         `json:"executionStatus"`
					Data            map[string][][]json.RawMessage `json:"data"`
				} `json:"runData"`
			} `json:"resultData"`
		} `json:"data"`
	}
	if err := json.Unmarshal(output, &envelope); err != nil || envelope.Data.ResultData.RunData == nil || len(envelope.Data.ResultData.Error) > 0 && string(envelope.Data.ResultData.Error) != "null" {
		return WorkflowExecutionEvidence{}, errors.New("missing n8n runData")
	}
	evidence := WorkflowExecutionEvidence{OutputSHA256: fmt.Sprintf("%x", sha256.Sum256(output)), OutputBytes: len(output), NodeCount: len(envelope.Data.ResultData.RunData)}
	for _, executions := range envelope.Data.ResultData.RunData {
		for _, execution := range executions {
			evidence.ExecutionCount++
			if len(execution.Error) > 0 && string(execution.Error) != "null" || execution.ExecutionStatus == "error" || execution.ExecutionStatus == "crashed" {
				evidence.FailedNodeExecutionCount++
			}
			for _, outputPorts := range execution.Data {
				for _, outputItems := range outputPorts {
					evidence.ItemCount += len(outputItems)
				}
			}
		}
	}
	return evidence, nil
}

func (o *Owner) ownedRunningContainer(ctx context.Context, sandboxID, runtimeID string) (canonical, containerName string, err error) {
	canonical, err = normalizeSandboxID(sandboxID)
	if err != nil {
		return "", "", err
	}
	if runtimeID != runtimeIDFor(canonical) {
		return "", "", ErrForeignObject
	}
	if o.image() != ImageRef {
		return "", "", fmt.Errorf("workflow operations require the exact pinned n8n image")
	}
	if _, err = o.dockerRoot(ctx); err != nil {
		return "", "", err
	}
	containerName, _ = objectNames(canonical)
	container, found, inspectErr := o.inspectContainer(ctx, containerName)
	if inspectErr != nil {
		return "", "", inspectErr
	}
	if !found || !container.State.Running {
		return "", "", fmt.Errorf("n8n task runtime is not running")
	}
	if err = verifyN8nLabels(container.Config.Labels, canonical, runtimeID, "container"); err != nil {
		return "", "", err
	}
	if container.Config.Image != ImageRef {
		return "", "", fmt.Errorf("n8n task runtime does not use the exact pinned image")
	}
	if _, err = o.runtimeFromContainer(ctx, canonical, runtimeID, container); err != nil {
		return "", "", err
	}
	_, networkName := objectNames(canonical)
	if _, found, err = o.inspectNetwork(ctx, networkName, canonical, runtimeID); err != nil {
		return "", "", err
	} else if !found {
		return "", "", fmt.Errorf("n8n task runtime network is missing")
	}
	return canonical, containerName, nil
}

func (o *Owner) verifyAttachedSyntheticCompanion(ctx context.Context, sandboxID, companionRuntimeID string) error {
	if !companionRuntimePattern.MatchString(companionRuntimeID) {
		return fmt.Errorf("Postgres synthetic credentials require a valid attached companion runtime ID")
	}
	containerID, labels, err := o.findCompanion(ctx, sandboxID, companionRuntimeID)
	if err != nil {
		return fmt.Errorf("Postgres synthetic credentials require an owned companion: %w", err)
	}
	if labels[labelSandboxDecl] != "true" {
		return ErrCompanionDenied
	}
	container, err := o.inspectContainerByID(ctx, containerID)
	if err != nil {
		return err
	}
	_, networkName := objectNames(sandboxID)
	if !contains(container.NetworkSettings.Networks[networkName].Aliases, companionAlias(sandboxID, companionRuntimeID)) {
		return fmt.Errorf("Postgres synthetic credential companion is not attached to the task network")
	}
	for attachedNetwork := range container.NetworkSettings.Networks {
		if attachedNetwork != networkName && attachedNetwork != "none" {
			return fmt.Errorf("%w: Postgres synthetic credential companion has egress-capable network %q", ErrCompanionDenied, attachedNetwork)
		}
	}
	return nil
}

func removeWorkflowInput(ctx context.Context, runner quietCommandRunner, containerName, path string) error {
	_ = ctx // Cleanup remains bounded and runs even if the caller canceled its request.
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := runner.RunQuiet(cleanupCtx, "exec", "--user", "node", containerName, "rm", "-f", "--", path); err != nil && !isNotFound(err) {
		return fmt.Errorf("remove staged synthetic workflow from task tmpfs: %w", err)
	}
	return nil
}

func workflowIDFor(sandboxID, runtimeID string, operationID uuid.UUID) string {
	digest := sha256.Sum256([]byte("vrooli-n8n-workflow-v1\x00" + sandboxID + "\x00" + runtimeID + "\x00" + operationID.String()))
	return hex.EncodeToString(digest[:10])
}

type syntheticCredentialRecord struct {
	ID   string         `json:"id"`
	Name string         `json:"name"`
	Type string         `json:"type"`
	Data map[string]any `json:"data"`
}

func prepareSyntheticWorkflow(input []byte, workflowID, sandboxID, runtimeID, postgresCompanionRuntimeID string) ([]byte, []syntheticCredentialRecord, error) {
	if len(input) == 0 || len(input) > maxSyntheticWorkflowJSON {
		return nil, nil, fmt.Errorf("synthetic workflow JSON must contain 1..%d bytes", maxSyntheticWorkflowJSON)
	}
	var workflow map[string]any
	if err := json.Unmarshal(input, &workflow); err != nil || workflow == nil {
		return nil, nil, fmt.Errorf("synthetic workflow must be one JSON object")
	}
	allowed := map[string]bool{"name": true, "nodes": true, "connections": true, "settings": true, "id": true, "active": true, "versionId": true, "versionCounter": true, "triggerCount": true, "createdAt": true, "updatedAt": true, "tags": true, "meta": true}
	for key := range workflow {
		if !allowed[key] {
			return nil, nil, fmt.Errorf("synthetic workflow contains unsupported or client-state field %q", key)
		}
	}
	var name string
	var ok bool
	name, ok = workflow["name"].(string)
	if !ok || strings.TrimSpace(name) == "" || len(name) > 128 {
		return nil, nil, fmt.Errorf("synthetic workflow name must contain 1..128 characters")
	}
	nodes, ok := workflow["nodes"].([]any)
	if !ok || len(nodes) == 0 {
		return nil, nil, fmt.Errorf("synthetic workflow must contain at least one node")
	}
	credentials := map[string]syntheticCredentialRecord{}
	for _, value := range nodes {
		nodeObject, ok := value.(map[string]any)
		if !ok || nodeObject == nil {
			return nil, nil, fmt.Errorf("synthetic workflow nodes must be JSON objects")
		}
		if rawReferences, hasReferences := nodeObject["credentials"]; hasReferences {
			references, ok := rawReferences.(map[string]any)
			if !ok {
				return nil, nil, fmt.Errorf("workflow credential references must be objects")
			}
			syntheticReferences := map[string]any{}
			for credentialType, rawReference := range references {
				reference, ok := rawReference.(map[string]any)
				if !ok || len(reference) == 0 || len(reference) > 2 {
					return nil, nil, fmt.Errorf("workflow credential reference has unsupported fields")
				}
				for key := range reference {
					if key != "id" && key != "name" {
						return nil, nil, fmt.Errorf("workflow credential reference contains unsupported field %q", key)
					}
				}
				sourceID, _ := reference["id"].(string)
				sourceName, _ := reference["name"].(string)
				if (sourceID == "" && sourceName == "") || len(sourceID) > 256 || len(sourceName) > 128 {
					return nil, nil, fmt.Errorf("workflow credential reference needs a bounded source id or name")
				}
				profile, supported := syntheticCredentialProfile(credentialType)
				if !supported {
					return nil, nil, fmt.Errorf("workflow credential type %q has no synthetic profile", credentialType)
				}
				credentialID := syntheticCredentialID(sandboxID, runtimeID, credentialType, sourceID, sourceName)
				credentialName := "Vrooli synthetic " + credentialType + " " + credentialID[:8]
				data, err := profile(sandboxID, postgresCompanionRuntimeID)
				if err != nil {
					return nil, nil, err
				}
				credentials[credentialID] = syntheticCredentialRecord{ID: credentialID, Name: credentialName, Type: credentialType, Data: data}
				syntheticReferences[credentialType] = map[string]any{"id": credentialID, "name": credentialName}
			}
			nodeObject["credentials"] = syntheticReferences
		}
		if parameters, ok := nodeObject["parameters"].(map[string]any); ok {
			if rawType, found := parameters["nodeCredentialType"]; found {
				credentialType, ok := rawType.(string)
				if !ok {
					return nil, nil, fmt.Errorf("nodeCredentialType must be an allowlisted credential type string")
				}
				profile, supported := syntheticCredentialProfile(credentialType)
				if !supported {
					return nil, nil, fmt.Errorf("nodeCredentialType %q has no synthetic profile", credentialType)
				}
				refs, _ := nodeObject["credentials"].(map[string]any)
				if refs == nil {
					refs = map[string]any{}
					nodeObject["credentials"] = refs
				}
				if _, exists := refs[credentialType]; !exists {
					credentialID := syntheticCredentialID(sandboxID, runtimeID, credentialType, "node-type:"+credentialType, "")
					credentialName := "Vrooli synthetic " + credentialType + " " + credentialID[:8]
					data, err := profile(sandboxID, postgresCompanionRuntimeID)
					if err != nil {
						return nil, nil, err
					}
					credentials[credentialID] = syntheticCredentialRecord{ID: credentialID, Name: credentialName, Type: credentialType, Data: data}
					refs[credentialType] = map[string]any{"id": credentialID, "name": credentialName}
				}
				delete(parameters, "nodeCredentialType")
				if err := rejectSensitiveWorkflowFields(map[string]any{"parameters": parameters}); err != nil {
					return nil, nil, err
				}
				parameters["nodeCredentialType"] = credentialType
			}
		}
		if nodeType, ok := nodeObject["type"].(string); ok {
			if requiredType := syntheticCredentialTypeForNode(nodeType); requiredType != "" {
				refs, _ := nodeObject["credentials"].(map[string]any)
				if refs == nil {
					refs = map[string]any{}
					nodeObject["credentials"] = refs
				}
				if _, exists := refs[requiredType]; !exists {
					profile, supported := syntheticCredentialProfile(requiredType)
					if !supported {
						return nil, nil, fmt.Errorf("pinned node credential type %q has no synthetic profile", requiredType)
					}
					credentialID := syntheticCredentialID(sandboxID, runtimeID, requiredType, "node-type:"+nodeType, "")
					credentialName := "Vrooli synthetic " + requiredType + " " + credentialID[:8]
					data, err := profile(sandboxID, postgresCompanionRuntimeID)
					if err != nil {
						return nil, nil, err
					}
					credentials[credentialID] = syntheticCredentialRecord{ID: credentialID, Name: credentialName, Type: requiredType, Data: data}
					refs[requiredType] = map[string]any{"id": credentialID, "name": credentialName}
				}
			}
		}
		for key, child := range nodeObject {
			if key == "credentials" {
				continue
			}
			if key == "parameters" {
				parameters, isObject := child.(map[string]any)
				if _, hasType := parameters["nodeCredentialType"]; isObject && hasType {
					continue
				}
			}
			if err := rejectSensitiveWorkflowFields(map[string]any{key: child}); err != nil {
				return nil, nil, err
			}
		}
	}
	if connections, ok := workflow["connections"].(map[string]any); !ok || connections == nil {
		return nil, nil, fmt.Errorf("synthetic workflow connections must be a JSON object")
	}
	if rawSettings, exists := workflow["settings"]; exists {
		settings, ok := rawSettings.(map[string]any)
		if !ok || settings == nil {
			return nil, nil, fmt.Errorf("synthetic workflow settings must be a JSON object")
		}
		if err := rejectSensitiveWorkflowFields(settings); err != nil {
			return nil, nil, err
		}
		allowedSettings := map[string]bool{"executionOrder": true, "timezone": true, "saveDataErrorExecution": true, "saveDataSuccessExecution": true, "saveManualExecutions": true, "saveExecutionProgress": true, "executionTimeout": true, "callerPolicy": true, "availableInMCP": true}
		for key := range settings {
			if !allowedSettings[key] {
				return nil, nil, fmt.Errorf("synthetic workflow contains unsupported settings field %q", key)
			}
		}
	}
	// Caller credential references now point only to owner-issued stand-ins.
	// Imported IDs/state are stripped so this workflow cannot overwrite another
	// entity or activate a trigger when the server CLI imports it.
	workflow["id"] = workflowID
	workflow["active"] = false
	delete(workflow, "versionId")
	delete(workflow, "versionCounter")
	delete(workflow, "triggerCount")
	delete(workflow, "createdAt")
	delete(workflow, "updatedAt")
	delete(workflow, "tags")
	delete(workflow, "meta")
	workflowJSON, err := json.Marshal(workflow)
	if err != nil {
		return nil, nil, err
	}
	credentialRecords := make([]syntheticCredentialRecord, 0, len(credentials))
	for _, record := range credentials {
		credentialRecords = append(credentialRecords, record)
	}
	return workflowJSON, credentialRecords, nil
}

type syntheticCredentialFactory func(sandboxID, companionRuntimeID string) (map[string]any, error)

func syntheticCredentialProfile(credentialType string) (syntheticCredentialFactory, bool) {
	const inertSyntheticValue = "vrooli-synthetic-no-egress"
	switch credentialType {
	case "githubApi":
		return func(string, string) (map[string]any, error) {
			return map[string]any{"server": "https://api.github.com", "user": "synthetic", "accessToken": inertSyntheticValue}, nil
		}, true
	case "notionApi":
		return func(string, string) (map[string]any, error) {
			return map[string]any{"apiKey": inertSyntheticValue}, nil
		}, true
	case "supabaseApi":
		return func(string, string) (map[string]any, error) {
			return map[string]any{"host": "https://supabase.invalid", "serviceRole": inertSyntheticValue}, nil
		}, true
	case "wooCommerceApi":
		return func(string, string) (map[string]any, error) {
			return map[string]any{"url": "https://woocommerce.invalid", "consumerKey": inertSyntheticValue, "consumerSecret": inertSyntheticValue, "includeCredentialsInQuery": false}, nil
		}, true
	case "openAiApi":
		return func(string, string) (map[string]any, error) {
			return map[string]any{"apiKey": inertSyntheticValue, "organizationId": "", "url": "https://openai.invalid/v1", "header": false}, nil
		}, true
	case "pineconeApi":
		return func(string, string) (map[string]any, error) {
			return map[string]any{"apiKey": inertSyntheticValue}, nil
		}, true
	case "httpHeaderAuth":
		return func(string, string) (map[string]any, error) {
			return map[string]any{"name": "X-Vrooli-Synthetic", "value": inertSyntheticValue}, nil
		}, true
	case "postgres":
		return func(sandboxID, companionRuntimeID string) (map[string]any, error) {
			if !companionRuntimePattern.MatchString(companionRuntimeID) {
				return nil, fmt.Errorf("Postgres synthetic credentials require a declared companion runtime ID")
			}
			return map[string]any{
				"host": companionAlias(sandboxID, companionRuntimeID), "database": SyntheticPostgresDB,
				"user": "synthetic", "password": "vrooli-workflow-rehearsal-only", "port": 5432,
				"maxConnections": 2, "allowUnauthorizedCerts": false, "ssl": "disable",
			}, nil
		}, true
	default:
		return nil, false
	}
}

func syntheticCredentialTypeForNode(nodeType string) string {
	switch nodeType {
	case "@n8n/n8n-nodes-langchain.lmChatOpenAi", "@n8n/n8n-nodes-langchain.embeddingsOpenAi":
		return "openAiApi"
	case "@n8n/n8n-nodes-langchain.vectorStorePinecone":
		return "pineconeApi"
	default:
		return ""
	}
}

func syntheticCredentialID(sandboxID, runtimeID, credentialType, sourceID, sourceName string) string {
	identity := sourceID
	if identity == "" {
		identity = "name:" + sourceName
	}
	digest := sha256.Sum256([]byte("vrooli-n8n-credential-v1\x00" + sandboxID + "\x00" + runtimeID + "\x00" + credentialType + "\x00" + identity))
	return hex.EncodeToString(digest[:10])
}

func rejectSensitiveWorkflowFields(value any) error {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			normalized := strings.ToLower(strings.Map(func(r rune) rune {
				if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
					return r
				}
				return -1
			}, key))
			for _, sensitive := range []string{"credential", "authorization", "password", "secret", "token", "apikey", "cookie", "privatekey"} {
				if strings.Contains(normalized, sensitive) {
					return fmt.Errorf("synthetic workflow contains sensitive field %q", key)
				}
			}
			if err := rejectSensitiveWorkflowFields(child); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range typed {
			if err := rejectSensitiveWorkflowFields(child); err != nil {
				return err
			}
		}
	}
	return nil
}

func (o *Owner) AttachCompanion(ctx context.Context, sandboxID, runtimeID, companionRuntimeID string) (CompanionLink, error) {
	canonical, err := normalizeSandboxID(sandboxID)
	if err != nil {
		return CompanionLink{}, err
	}
	if runtimeID != runtimeIDFor(canonical) {
		return CompanionLink{}, ErrForeignObject
	}
	if !companionRuntimePattern.MatchString(companionRuntimeID) {
		return CompanionLink{}, fmt.Errorf("invalid companion runtime ID")
	}
	_, networkName := objectNames(canonical)
	if _, found, err := o.inspectNetwork(ctx, networkName, canonical, runtimeID); err != nil {
		return CompanionLink{}, err
	} else if !found {
		return CompanionLink{}, fmt.Errorf("n8n task runtime is not running")
	}
	containerName, _ := objectNames(canonical)
	n8nContainer, found, err := o.inspectContainer(ctx, containerName)
	if err != nil || !found || !n8nContainer.State.Running {
		if err != nil {
			return CompanionLink{}, err
		}
		return CompanionLink{}, fmt.Errorf("n8n task runtime is not running")
	}
	if err := verifyN8nLabels(n8nContainer.Config.Labels, canonical, runtimeID, "container"); err != nil {
		return CompanionLink{}, err
	}
	if _, err := o.runtimeFromContainer(ctx, canonical, runtimeID, n8nContainer); err != nil {
		return CompanionLink{}, err
	}
	containerID, labels, err := o.findCompanion(ctx, canonical, companionRuntimeID)
	if err != nil {
		return CompanionLink{}, err
	}
	if labels[labelSandboxDecl] != "true" {
		return CompanionLink{}, ErrCompanionDenied
	}
	companion, err := o.inspectContainerByID(ctx, containerID)
	if err != nil {
		return CompanionLink{}, err
	}
	for attachedNetwork := range companion.NetworkSettings.Networks {
		if attachedNetwork != networkName && attachedNetwork != "none" {
			return CompanionLink{}, fmt.Errorf("%w: companion %s has additional network attachment %q; isolate it before attach", ErrCompanionDenied, companionRuntimeID, attachedNetwork)
		}
	}
	alias := companionAlias(canonical, companionRuntimeID)
	if _, err := o.runner().Run(ctx, "network", "connect", "--alias", alias, networkName, containerID); err != nil {
		// Docker reports already-connected as an error. Treat it as idempotent
		// only after an inspect confirms the expected endpoint alias.
		companion, inspectErr := o.inspectContainerByID(ctx, containerID)
		networkState := companion.NetworkSettings.Networks[networkName]
		if inspectErr != nil || !contains(networkState.Aliases, alias) {
			return CompanionLink{}, fmt.Errorf("attach declared companion to private n8n network: %w", err)
		}
	}
	return CompanionLink{SandboxID: canonical, RuntimeID: runtimeID, CompanionRuntimeID: companionRuntimeID, NetworkAlias: alias, Status: "attached"}, nil
}

func (o *Owner) DetachCompanion(ctx context.Context, sandboxID, runtimeID, companionRuntimeID string) (CompanionLink, error) {
	canonical, err := normalizeSandboxID(sandboxID)
	if err != nil {
		return CompanionLink{}, err
	}
	if runtimeID != runtimeIDFor(canonical) {
		return CompanionLink{}, ErrForeignObject
	}
	if !companionRuntimePattern.MatchString(companionRuntimeID) {
		return CompanionLink{}, fmt.Errorf("invalid companion runtime ID")
	}
	_, networkName := objectNames(canonical)
	if _, found, err := o.inspectNetwork(ctx, networkName, canonical, runtimeID); err != nil {
		return CompanionLink{}, err
	} else if !found {
		return CompanionLink{SandboxID: canonical, RuntimeID: runtimeID, CompanionRuntimeID: companionRuntimeID, Status: "detached"}, nil
	}
	containerID, labels, err := o.findCompanion(ctx, canonical, companionRuntimeID)
	if err != nil {
		if errors.Is(err, errCompanionMissing) {
			return CompanionLink{SandboxID: canonical, RuntimeID: runtimeID, CompanionRuntimeID: companionRuntimeID, Status: "detached"}, nil
		}
		return CompanionLink{}, err
	}
	if labels[labelSandboxDecl] != "true" {
		return CompanionLink{}, ErrCompanionDenied
	}
	container, err := o.inspectContainerByID(ctx, containerID)
	if err != nil {
		return CompanionLink{}, err
	}
	if container.Config.Labels[labelSandboxOwner] != canonical || container.Config.Labels[labelSandboxRunID] != companionRuntimeID || container.Config.Labels[labelSandboxDecl] != "true" {
		return CompanionLink{}, ErrForeignObject
	}
	if _, attached := container.NetworkSettings.Networks[networkName]; !attached {
		return CompanionLink{SandboxID: canonical, RuntimeID: runtimeID, CompanionRuntimeID: companionRuntimeID, Status: "detached"}, nil
	}
	if _, err := o.runner().Run(ctx, "network", "disconnect", networkName, containerID); err != nil && !isNotFound(err) {
		return CompanionLink{}, fmt.Errorf("detach companion from private n8n network: %w", err)
	}
	return CompanionLink{SandboxID: canonical, RuntimeID: runtimeID, CompanionRuntimeID: companionRuntimeID, NetworkAlias: companionAlias(canonical, companionRuntimeID), Status: "detached"}, nil
}

// StartSyntheticPostgres creates the one supported synthetic DB companion for
// a Sandbox. It never pulls: the exact digest must already be present. The
// caller may echo the fixed profile metadata, but cannot select an image,
// command, network, mount, password, or runtime identity.
func (o *Owner) StartSyntheticPostgres(ctx context.Context, sandboxID, name, version, database string) (SyntheticPostgresReceipt, error) {
	canonical, err := normalizeSandboxID(sandboxID)
	if err != nil {
		return SyntheticPostgresReceipt{}, err
	}
	if name != SyntheticPostgresName || version != SyntheticPostgresVersion || database != SyntheticPostgresDB {
		return SyntheticPostgresReceipt{}, fmt.Errorf("unsupported synthetic Postgres profile")
	}
	runtimeID := postgresRuntimeIDFor(canonical)
	containerName := postgresContainerName(canonical)
	if _, err := o.dockerRoot(ctx); err != nil {
		return SyntheticPostgresReceipt{}, err
	}
	if _, err := o.runner().Run(ctx, "image", "inspect", SyntheticPostgresImage); err != nil {
		if isNotFound(err) {
			return SyntheticPostgresReceipt{}, fmt.Errorf("synthetic Postgres image is not installed; owner never pulls companion images")
		}
		return SyntheticPostgresReceipt{}, fmt.Errorf("inspect pinned synthetic Postgres image: %w", err)
	}
	container, found, err := o.inspectContainer(ctx, containerName)
	if err != nil {
		return SyntheticPostgresReceipt{}, err
	}
	_, taskNetwork := objectNames(canonical)
	if found {
		if err := verifyPostgresLabels(container.Config.Labels, canonical, runtimeID); err != nil {
			return SyntheticPostgresReceipt{}, err
		}
		if err := o.verifySyntheticPostgresNetworks(ctx, canonical, container); err != nil {
			return SyntheticPostgresReceipt{}, err
		}
		if !container.State.Running {
			if _, err := o.runner().Run(ctx, "start", containerName); err != nil {
				return SyntheticPostgresReceipt{}, fmt.Errorf("start owned synthetic Postgres companion: %w", err)
			}
			container, err = o.inspectContainerByID(ctx, container.ID)
			if err != nil {
				return SyntheticPostgresReceipt{}, err
			}
		}
	} else {
		args := []string{"run", "--detach", "--name", containerName, "--network", "none", "--read-only", "--cap-drop", "ALL", "--security-opt", "no-new-privileges:true", "--pids-limit", "128", "--memory", "512m", "--memory-swap", "512m", "--cpus", "1", "--tmpfs", "/var/lib/postgresql/data:rw,noexec,nosuid,nodev,size=256m,uid=70,gid=70,mode=0700", "--tmpfs", "/var/run/postgresql:rw,noexec,nosuid,nodev,size=16m,uid=70,gid=70,mode=0700", "--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=16m,uid=70,gid=70,mode=0700", "--label", labelCompanionResource + "=n8n-postgres", "--label", labelSandboxOwner + "=" + canonical, "--label", labelSandboxRunID + "=" + runtimeID, "--label", labelSandboxDecl + "=true", "--env", "POSTGRES_USER=synthetic", "--env", "POSTGRES_PASSWORD=vrooli-workflow-rehearsal-only", "--env", "POSTGRES_DB=" + SyntheticPostgresDB, SyntheticPostgresImage}
		if _, err := o.runner().Run(ctx, args...); err != nil {
			cleanupErr := o.removePartialPostgres(context.Background(), canonical, runtimeID, containerName)
			return SyntheticPostgresReceipt{}, errors.Join(fmt.Errorf("create isolated synthetic Postgres companion: %w", err), cleanupErr)
		}
		container, found, err = o.inspectContainer(ctx, containerName)
		if err != nil || !found {
			cleanupErr := o.removePartialPostgres(context.Background(), canonical, runtimeID, containerName)
			if err == nil {
				err = fmt.Errorf("created synthetic Postgres companion cannot be inspected")
			}
			return SyntheticPostgresReceipt{}, errors.Join(err, cleanupErr)
		}
		if err := verifyPostgresLabels(container.Config.Labels, canonical, runtimeID); err != nil {
			return SyntheticPostgresReceipt{}, err
		}
	}
	if !container.State.Running {
		return SyntheticPostgresReceipt{}, fmt.Errorf("synthetic Postgres companion is not running")
	}
	if err := verifySyntheticPostgresTmpfs(container); err != nil {
		cleanupErr := o.removePartialPostgres(context.Background(), canonical, runtimeID, containerName)
		return SyntheticPostgresReceipt{}, errors.Join(err, cleanupErr)
	}
	if err := o.verifySyntheticPostgresNetworks(ctx, canonical, container); err != nil {
		return SyntheticPostgresReceipt{}, err
	}
	if _, err := o.runner().Run(ctx, "exec", container.ID, "pg_isready", "-U", "synthetic", "-d", SyntheticPostgresDB); err != nil {
		cleanupErr := o.removePartialPostgres(context.Background(), canonical, runtimeID, containerName)
		return SyntheticPostgresReceipt{}, errors.Join(fmt.Errorf("synthetic Postgres is not ready: %w", err), cleanupErr)
	}
	link, err := o.AttachCompanion(ctx, canonical, runtimeIDFor(canonical), runtimeID)
	if err != nil {
		cleanupErr := o.removePartialPostgres(context.Background(), canonical, runtimeID, containerName)
		return SyntheticPostgresReceipt{}, errors.Join(err, cleanupErr)
	}
	container, err = o.inspectContainerByID(ctx, container.ID)
	if err != nil {
		return SyntheticPostgresReceipt{}, err
	}
	if err := verifyPostgresLabels(container.Config.Labels, canonical, runtimeID); err != nil {
		return SyntheticPostgresReceipt{}, err
	}
	if err := verifyNoEgressExceptTask(container, taskNetwork); err != nil {
		return SyntheticPostgresReceipt{}, err
	}
	return SyntheticPostgresReceipt{SandboxID: canonical, CompanionRuntimeID: runtimeID, NetworkAlias: link.NetworkAlias, Status: "attached", NoEgressVerified: true}, nil
}

type SyntheticPostgresReceipt struct {
	SandboxID          string `json:"sandboxId"`
	CompanionRuntimeID string `json:"companionRuntimeId"`
	NetworkAlias       string `json:"networkAlias"`
	Status             string `json:"status"`
	NoEgressVerified   bool   `json:"noEgressVerified"`
}

// StopSyntheticPostgres detaches and removes only the deterministic container
// whose labels attest the requested Sandbox. The owner never removes volumes
// or networks for this companion.
func (o *Owner) StopSyntheticPostgres(ctx context.Context, sandboxID, companionRuntimeID string) error {
	canonical, err := normalizeSandboxID(sandboxID)
	if err != nil {
		return err
	}
	if companionRuntimeID != postgresRuntimeIDFor(canonical) {
		return ErrForeignObject
	}
	containerName := postgresContainerName(canonical)
	container, found, err := o.inspectContainer(ctx, containerName)
	if err != nil || !found {
		return err
	}
	if err := verifyPostgresLabels(container.Config.Labels, canonical, companionRuntimeID); err != nil {
		return err
	}
	if err := o.verifySyntheticPostgresNetworks(ctx, canonical, container); err != nil {
		return err
	}
	if _, err := o.DetachCompanion(ctx, canonical, runtimeIDFor(canonical), companionRuntimeID); err != nil {
		return err
	}
	container, found, err = o.inspectContainer(ctx, containerName)
	if err != nil || !found {
		return err
	}
	if err := verifyPostgresLabels(container.Config.Labels, canonical, companionRuntimeID); err != nil {
		return err
	}
	if err := verifyNoEgressExceptTask(container, "none"); err != nil {
		return err
	}
	if container.State.Running {
		if _, err := o.runner().Run(ctx, "stop", "--time", "5", container.ID); err != nil {
			return fmt.Errorf("stop owned synthetic Postgres companion: %w", err)
		}
	}
	container, found, err = o.inspectContainer(ctx, containerName)
	if err != nil || !found {
		return err
	}
	if err := verifyPostgresLabels(container.Config.Labels, canonical, companionRuntimeID); err != nil {
		return err
	}
	if _, err := o.runner().Run(ctx, "rm", "--volumes", container.ID); err != nil && !isNotFound(err) {
		return fmt.Errorf("remove owned synthetic Postgres companion: %w", err)
	}
	return nil
}

func (o *Owner) removePartialPostgres(ctx context.Context, sandboxID, runtimeID, name string) error {
	container, found, err := o.inspectContainer(ctx, name)
	if err != nil || !found {
		return err
	}
	if err := verifyPostgresLabels(container.Config.Labels, sandboxID, runtimeID); err != nil {
		return err
	}
	_, taskNetwork := objectNames(sandboxID)
	if err := verifyNoEgressExceptTask(container, taskNetwork); err != nil {
		return err
	}
	if container.State.Running {
		if _, err := o.runner().Run(ctx, "stop", "--time", "5", container.ID); err != nil {
			return err
		}
	}
	container, found, err = o.inspectContainer(ctx, name)
	if err != nil || !found {
		return err
	}
	if err := verifyPostgresLabels(container.Config.Labels, sandboxID, runtimeID); err != nil {
		return err
	}
	_, err = o.runner().Run(ctx, "rm", "--volumes", container.ID)
	return err
}

func (o *Owner) verifySyntheticPostgresNetworks(ctx context.Context, sandboxID string, container containerInspect) error {
	_, taskNetwork := objectNames(sandboxID)
	for network := range container.NetworkSettings.Networks {
		if network != "none" && network != taskNetwork {
			return fmt.Errorf("%w: synthetic Postgres has egress-capable network attachment %q", ErrCompanionDenied, network)
		}
		if network == taskNetwork {
			if _, found, err := o.inspectNetwork(ctx, taskNetwork, sandboxID, runtimeIDFor(sandboxID)); err != nil {
				return err
			} else if !found {
				return fmt.Errorf("synthetic Postgres task network is missing")
			}
		}
	}
	if len(container.NetworkSettings.Networks) == 0 {
		return fmt.Errorf("synthetic Postgres has no verified network attachment")
	}
	return nil
}

func (o *Owner) Stop(ctx context.Context, sandboxID, runtimeID string) (StopResult, error) {
	canonical, err := normalizeSandboxID(sandboxID)
	if err != nil {
		return StopResult{}, err
	}
	if runtimeID != runtimeIDFor(canonical) {
		return StopResult{}, ErrForeignObject
	}
	containerName, networkName := objectNames(canonical)
	result := StopResult{SandboxID: canonical, RuntimeID: runtimeID, Status: "stopped"}
	container, found, err := o.inspectContainer(ctx, containerName)
	if err != nil {
		return result, err
	}
	if found {
		if err := verifyN8nLabels(container.Config.Labels, canonical, runtimeID, "container"); err != nil {
			return result, err
		}
		if container.State.Running {
			if _, err := o.runner().Run(ctx, "stop", "--time", strconv.Itoa(containerStopWait), containerName); err != nil {
				stopped, found, inspectErr := o.inspectContainer(ctx, containerName)
				if inspectErr != nil || (found && stopped.State.Running) {
					return result, errors.Join(fmt.Errorf("stop owned n8n container: %w", err), inspectErr)
				}
			}
		}
		if _, err := o.runner().Run(ctx, "rm", containerName); err != nil && !isNotFound(err) {
			return result, fmt.Errorf("remove owned n8n container: %w", err)
		}
	}
	network, found, err := o.inspectNetwork(ctx, networkName, canonical, runtimeID)
	if err != nil {
		return result, err
	}
	if !found {
		return result, nil
	}
	for containerID := range network.Containers {
		container, err := o.inspectContainerByID(ctx, containerID)
		if err != nil {
			result.Retained = append(result.Retained, "network:"+networkName)
			return result, fmt.Errorf("inspect attached container before cleanup: %w", err)
		}
		labels := container.Config.Labels
		if labels[labelResource] == "n8n" && labels[labelSandboxID] == canonical && labels[labelRuntimeID] == runtimeID {
			// A failed `docker rm` can leave a stopped owner container attached.
			if _, err := o.runner().Run(ctx, "network", "disconnect", networkName, containerID); err != nil && !isNotFound(err) {
				result.Retained = append(result.Retained, "network:"+networkName)
				return result, fmt.Errorf("disconnect owned n8n container: %w", err)
			}
			continue
		}
		if labels[labelSandboxOwner] != canonical || labels[labelSandboxRunID] == "" || labels[labelSandboxDecl] != "true" {
			result.Retained = append(result.Retained, "network:"+networkName, "container:"+containerID)
			return result, fmt.Errorf("%w: network %s has an unattested attachment", ErrForeignObject, networkName)
		}
		if _, err := o.runner().Run(ctx, "network", "disconnect", networkName, containerID); err != nil && !isNotFound(err) {
			result.Retained = append(result.Retained, "network:"+networkName)
			return result, fmt.Errorf("detach sandbox companion %s during cleanup: %w", containerID, err)
		}
	}
	if err := o.removeOwnedNetwork(ctx, networkName, canonical, runtimeID); err != nil {
		result.Retained = append(result.Retained, "network:"+networkName)
		return result, err
	}
	return result, nil
}

var errCompanionMissing = errors.New("n8n runtime: declared companion runtime not found")

func (o *Owner) findCompanion(ctx context.Context, sandboxID, runtimeID string) (string, map[string]string, error) {
	filters := []string{"ps", "--quiet", "--no-trunc", "--filter", "label=" + labelSandboxOwner + "=" + sandboxID, "--filter", "label=" + labelSandboxRunID + "=" + runtimeID}
	output, err := o.runner().Run(ctx, filters...)
	if err != nil {
		return "", nil, fmt.Errorf("find Sandbox-owned companion runtime: %w", err)
	}
	ids := strings.Fields(string(output))
	if len(ids) == 0 {
		return "", nil, errCompanionMissing
	}
	if len(ids) != 1 {
		return "", nil, fmt.Errorf("companion runtime ID is ambiguous for sandbox")
	}
	container, err := o.inspectContainerByID(ctx, ids[0])
	if err != nil {
		return "", nil, err
	}
	labels := container.Config.Labels
	if labels[labelSandboxOwner] != sandboxID || labels[labelSandboxRunID] != runtimeID {
		return "", nil, ErrForeignObject
	}
	return container.ID, labels, nil
}

type containerInspect struct {
	ID     string `json:"Id"`
	Mounts []struct {
		Type        string `json:"Type"`
		Destination string `json:"Destination"`
	} `json:"Mounts"`
	Config struct {
		Image  string            `json:"Image"`
		Labels map[string]string `json:"Labels"`
	} `json:"Config"`
	State struct {
		Running bool   `json:"Running"`
		Status  string `json:"Status"`
	} `json:"State"`
	NetworkSettings struct {
		Ports map[string][]struct {
			HostIP   string `json:"HostIp"`
			HostPort string `json:"HostPort"`
		} `json:"Ports"`
		Networks map[string]struct {
			Aliases []string `json:"Aliases"`
		} `json:"Networks"`
	} `json:"NetworkSettings"`
}

type networkInspect struct {
	Name       string            `json:"Name"`
	Driver     string            `json:"Driver"`
	Internal   bool              `json:"Internal"`
	Labels     map[string]string `json:"Labels"`
	Containers map[string]struct {
		Name string `json:"Name"`
	} `json:"Containers"`
}

func (o *Owner) inspectContainer(ctx context.Context, name string) (containerInspect, bool, error) {
	output, err := o.runner().Run(ctx, "container", "inspect", name)
	if err != nil {
		if isNotFound(err) {
			return containerInspect{}, false, nil
		}
		return containerInspect{}, false, fmt.Errorf("inspect Docker container %s: %w", name, err)
	}
	return parseContainerInspect(output)
}

func (o *Owner) inspectContainerByID(ctx context.Context, id string) (containerInspect, error) {
	if !companionRuntimePattern.MatchString(id) {
		return containerInspect{}, fmt.Errorf("invalid Docker container ID returned by daemon")
	}
	output, err := o.runner().Run(ctx, "container", "inspect", id)
	if err != nil {
		return containerInspect{}, fmt.Errorf("inspect Docker container: %w", err)
	}
	container, found, err := parseContainerInspect(output)
	if err != nil {
		return containerInspect{}, err
	}
	if !found {
		return containerInspect{}, errCompanionMissing
	}
	return container, nil
}

func parseContainerInspect(output []byte) (containerInspect, bool, error) {
	var items []containerInspect
	if err := json.Unmarshal(output, &items); err != nil {
		return containerInspect{}, false, fmt.Errorf("decode Docker container inspection: %w", err)
	}
	if len(items) == 0 {
		return containerInspect{}, false, nil
	}
	if len(items) != 1 {
		return containerInspect{}, false, fmt.Errorf("Docker container inspection returned an ambiguous result")
	}
	return items[0], true, nil
}

func (o *Owner) inspectNetwork(ctx context.Context, name, sandboxID, runtimeID string) (networkInspect, bool, error) {
	output, err := o.runner().Run(ctx, "network", "inspect", name)
	if err != nil {
		if isNotFound(err) {
			return networkInspect{}, false, nil
		}
		return networkInspect{}, false, fmt.Errorf("inspect Docker network %s: %w", name, err)
	}
	var items []networkInspect
	if err := json.Unmarshal(output, &items); err != nil {
		return networkInspect{}, false, fmt.Errorf("decode Docker network inspection: %w", err)
	}
	if len(items) == 0 {
		return networkInspect{}, false, nil
	}
	if len(items) != 1 {
		return networkInspect{}, false, fmt.Errorf("Docker network inspection returned an ambiguous result")
	}
	item := items[0]
	labels := item.Labels
	if labels[labelResource] != "n8n" || labels[labelSandboxID] != sandboxID || labels[labelRuntimeID] != runtimeID || labels[labelObjectKind] != "network" || !item.Internal {
		return networkInspect{}, false, ErrForeignObject
	}
	return item, true, nil
}

func (o *Owner) runtimeFromContainer(ctx context.Context, sandboxID, runtimeID string, container containerInspect) (Runtime, error) {
	if err := verifyN8nLabels(container.Config.Labels, sandboxID, runtimeID, "container"); err != nil {
		return Runtime{}, err
	}
	_, networkName := objectNames(sandboxID)
	portBindings := container.NetworkSettings.Ports["5678/tcp"]
	for _, binding := range portBindings {
		if binding.HostIP != "127.0.0.1" {
			return Runtime{}, fmt.Errorf("n8n host port is not bound to loopback")
		}
		port, err := strconv.Atoi(binding.HostPort)
		if err != nil || port < 1 || port > 65535 {
			return Runtime{}, fmt.Errorf("n8n host port is invalid")
		}
		endpoint := "http://127.0.0.1:" + binding.HostPort
		network := container.NetworkSettings.Networks[networkName]
		if len(container.NetworkSettings.Networks) != 1 {
			return Runtime{}, fmt.Errorf("n8n runtime has an unexpected network attachment")
		}
		alias := n8nAlias(sandboxID)
		if !contains(network.Aliases, alias) {
			return Runtime{}, fmt.Errorf("n8n private network alias is missing")
		}
		return Runtime{SandboxID: sandboxID, RuntimeID: runtimeID, Endpoint: endpoint, Alias: alias, Status: "running"}, nil
	}
	return Runtime{}, fmt.Errorf("n8n loopback host port is unavailable")
}

func (o *Owner) waitReady(ctx context.Context, endpoint string) error {
	deadlineCtx, cancel := context.WithTimeout(ctx, readinessTimeout)
	defer cancel()
	interval := o.PollEvery
	if interval <= 0 {
		interval = 2 * time.Second
	}
	var lastProbeErr error
	for {
		if err := o.probeReady(deadlineCtx, endpoint); err == nil {
			return nil
		} else {
			lastProbeErr = err
		}
		timer := time.NewTimer(interval)
		select {
		case <-deadlineCtx.Done():
			timer.Stop()
			if lastProbeErr != nil {
				return fmt.Errorf("n8n runtime failed readiness at %s/healthz: %w (last probe: %v)", strings.TrimRight(endpoint, "/"), deadlineCtx.Err(), lastProbeErr)
			}
			return fmt.Errorf("n8n runtime failed readiness at %s/healthz: %w", strings.TrimRight(endpoint, "/"), deadlineCtx.Err())
		case <-timer.C:
		}
	}
}

func (o *Owner) probeReady(ctx context.Context, endpoint string) error {
	url := strings.TrimRight(endpoint, "/") + "/healthz"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	response, err := o.client().Do(request)
	if err != nil {
		return err
	}
	_ = response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("n8n readiness returned HTTP %d", response.StatusCode)
	}
	return nil
}

func (o *Owner) removeOwnedNetwork(ctx context.Context, name, sandboxID, runtimeID string) error {
	if _, found, err := o.inspectNetwork(ctx, name, sandboxID, runtimeID); err != nil || !found {
		return err
	}
	if _, err := o.runner().Run(ctx, "network", "rm", name); err != nil && !isNotFound(err) {
		return fmt.Errorf("remove owned n8n network: %w", err)
	}
	return nil
}

func (o *Owner) cleanupAfterStartFailure(ctx context.Context, containerName, networkName, sandboxID, runtimeID string) error {
	container, found, err := o.inspectContainer(ctx, containerName)
	if err != nil {
		return fmt.Errorf("reconcile partial n8n container creation: %w", err)
	}
	if found {
		if err := verifyN8nLabels(container.Config.Labels, sandboxID, runtimeID, "container"); err != nil {
			return err
		}
		if container.State.Running {
			if _, err := o.runner().Run(ctx, "stop", "--time", strconv.Itoa(containerStopWait), containerName); err != nil {
				stopped, stillFound, inspectErr := o.inspectContainer(ctx, containerName)
				if inspectErr != nil || (stillFound && stopped.State.Running) {
					return errors.Join(fmt.Errorf("stop partial n8n container: %w", err), inspectErr)
				}
			}
		}
		if _, err := o.runner().Run(ctx, "rm", containerName); err != nil && !isNotFound(err) {
			return fmt.Errorf("remove partial n8n container: %w", err)
		}
	}
	return o.removeOwnedNetwork(ctx, networkName, sandboxID, runtimeID)
}

func chooseLoopbackPort() (string, string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", "", fmt.Errorf("allocate n8n loopback port: %w", err)
	}
	port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	if err := listener.Close(); err != nil {
		return "", "", fmt.Errorf("release n8n loopback port reservation: %w", err)
	}
	return "http://127.0.0.1:" + port, port, nil
}

func normalizeSandboxID(value string) (string, error) {
	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil || id == uuid.Nil {
		return "", fmt.Errorf("sandbox ID must be a nonzero UUID")
	}
	return id.String(), nil
}

func runtimeIDFor(sandboxID string) string { return "n8n:" + sandboxID }

func postgresRuntimeIDFor(sandboxID string) string { return "sandbox.postgres:" + sandboxID }

func postgresContainerName(sandboxID string) string {
	return "vrooli-n8n-pg-" + strings.ReplaceAll(sandboxID, "-", "")
}

func verifyPostgresLabels(labels map[string]string, sandboxID, runtimeID string) error {
	if labels[labelCompanionResource] != "n8n-postgres" || labels[labelSandboxOwner] != sandboxID || labels[labelSandboxRunID] != runtimeID || labels[labelSandboxDecl] != "true" {
		return ErrForeignObject
	}
	return nil
}

func verifyNoEgressExceptTask(container containerInspect, taskNetwork string) error {
	for network := range container.NetworkSettings.Networks {
		if network != "none" && (taskNetwork == "none" || network != taskNetwork) {
			return fmt.Errorf("%w: companion has egress-capable network attachment %q", ErrCompanionDenied, network)
		}
	}
	if taskNetwork != "none" {
		if _, attached := container.NetworkSettings.Networks[taskNetwork]; !attached {
			return fmt.Errorf("synthetic Postgres is not attached to the expected task network")
		}
	}
	return nil
}

func verifySyntheticPostgresTmpfs(container containerInspect) error {
	want := map[string]bool{
		"/var/lib/postgresql/data": false,
		"/var/run/postgresql":      false,
		"/tmp":                     false,
	}
	for _, mount := range container.Mounts {
		if _, expected := want[mount.Destination]; !expected {
			continue
		}
		if mount.Type != "tmpfs" {
			return fmt.Errorf("synthetic Postgres writable path %s is not tmpfs", mount.Destination)
		}
		want[mount.Destination] = true
	}
	for destination, found := range want {
		if !found {
			return fmt.Errorf("synthetic Postgres tmpfs mount %s is unverified", destination)
		}
	}
	return nil
}

func objectNames(sandboxID string) (containerName, networkName string) {
	compact := strings.ReplaceAll(sandboxID, "-", "")
	return "vrooli-n8n-" + compact, "vrooli-n8n-net-" + compact
}

func n8nAlias(sandboxID string) string {
	return "n8n-" + strings.ReplaceAll(sandboxID, "-", "")
}

func companionAlias(sandboxID, runtimeID string) string {
	// Hash the complete runtime ID so long IDs with a shared prefix cannot
	// collide after truncation, while keeping the alias within DNS limits.
	digest := sha256.Sum256([]byte(runtimeID))
	return "companion-" + strings.ReplaceAll(sandboxID, "-", "")[:8] + "-" + hex.EncodeToString(digest[:6])
}

func verifyN8nLabels(labels map[string]string, sandboxID, runtimeID, kind string) error {
	if labels[labelResource] != "n8n" || labels[labelSandboxID] != sandboxID || labels[labelRuntimeID] != runtimeID || labels[labelObjectKind] != kind {
		return ErrForeignObject
	}
	return nil
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "no such image") || strings.Contains(text, "no such object") || strings.Contains(text, "no such container") || strings.Contains(text, "no such network") || strings.Contains(text, "not found")
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func (o *Owner) image() string {
	if strings.TrimSpace(o.Image) == "" {
		return ImageRef
	}
	return o.Image
}

func retainedImagePolicy() (reuse, eviction string) {
	return "reuse the digest-pinned 2.48.0 image for future n8n task runtimes", "retain while this manifest references the digest; remove only through the n8n owner after no n8n runtimes use it and a newer digest replaces it or the resource is retired"
}

func (o *Owner) imagePresent(ctx context.Context) (bool, error) {
	_, err := o.runner().Run(ctx, "image", "inspect", o.image())
	if err == nil {
		return true, nil
	}
	if isNotFound(err) {
		return false, nil
	}
	return false, fmt.Errorf("inspect pinned n8n image: %w", err)
}

func (o *Owner) dockerRoot(ctx context.Context) (string, error) {
	output, err := o.runner().Run(ctx, "info", "--format", "{{.DockerRootDir}}")
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDockerUnavailable, err)
	}
	root := strings.TrimSpace(string(output))
	if root == "" {
		return "", fmt.Errorf("%w: DockerRootDir is empty", ErrDockerUnavailable)
	}
	return root, nil
}
