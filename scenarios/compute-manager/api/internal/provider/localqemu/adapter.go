package localqemu

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	cloudprovider "compute-manager/internal/provider"
	"github.com/vrooli/api-core/storage"
)

const metadataName = "compute-manager-qemu.json"

// Adapter maps the shared create/describe/list/destroy contract onto the
// manager-owned disposable QEMU lane. Runtime state is kept under DataDir so
// QEMU processes can be reconciled after an API restart.
type Adapter struct {
	QEMU            LocalQEMUProvider
	DataDir         string
	Admission       CapacityAdmission
	RequireCapacity bool
	Roots           interface {
		Pick(context.Context, storage.Class) (string, error)
		RecordWrite(context.Context)
	}
	Now         func() time.Time
	leaseMu     sync.Mutex
	leases      map[string]*adapterLease
	ready       bool
	readyErr    error
	managerLock *os.File
}

type adapterLease struct {
	capacity *QEMULease
	instance Instance
	cancel   context.CancelFunc
}

func (a *Adapter) Name() string { return "local-qemu" }

func (a *Adapter) Facts() cloudprovider.BillingFacts {
	return cloudprovider.BillingFacts{RoundingUnit: time.Minute, MinimumBillable: time.Minute}
}

func (a *Adapter) Create(ctx context.Context, spec cloudprovider.Spec) (cloudprovider.Instance, error) {
	if err := a.requireReady(); err != nil {
		return cloudprovider.Instance{}, err
	}
	requestKey := strings.TrimSpace(spec.Tags["vrooli-request"])
	if requestKey == "" {
		return cloudprovider.Instance{}, fmt.Errorf("local-qemu requires an idempotency key")
	}
	if strings.TrimSpace(spec.Image) == "" || strings.TrimSpace(spec.SSHUser) == "" || strings.TrimSpace(spec.SSHAuthorizedKey) == "" {
		return cloudprovider.Instance{}, fmt.Errorf("local-qemu requires an image, SSH user, and public key")
	}
	root, err := a.dataRoot(ctx)
	if err != nil {
		return cloudprovider.Instance{}, err
	}
	digest := sha256.Sum256([]byte(requestKey))
	id := "qemu-" + hex.EncodeToString(digest[:12])
	workdir := filepath.Join(root, "local-qemu", id)
	path, err := metadataPath(root, id)
	if err != nil {
		return cloudprovider.Instance{}, err
	}
	if existing, readErr := readOwnedInstance(root, id, path); readErr == nil {
		if a.RequireCapacity {
			if err := a.ensureLease(ctx, existing); err != nil {
				return cloudprovider.Instance{}, err
			}
		}
		return toCloudInstance(existing, spec), nil
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return cloudprovider.Instance{}, readErr
	}
	if err := storage.EnsureDirectory(workdir, 0o700); err != nil {
		return cloudprovider.Instance{}, fmt.Errorf("create local-qemu workdir: %w", err)
	}
	port := spec.SSHPort
	if port == 0 {
		port = 2222
	}
	memory := spec.Memory
	if memory == "" {
		memory = "2048"
	}
	cpus := spec.CPUs
	if cpus <= 0 {
		cpus = 2
	}
	profile := Profile(spec.Profile)
	if profile == "" {
		profile = ProfileHeadlessLinux
	}
	var admitted *QEMULease
	if a.RequireCapacity {
		if a.Admission == nil {
			return cloudprovider.Instance{}, errors.Join(fmt.Errorf("shared capacity admission is unavailable"), cleanupOwnedWorkdir(workdir))
		}
		virtualBytes, measureErr := a.QEMU.ImageVirtualSize(spec.Image)
		if measureErr != nil {
			return cloudprovider.Instance{}, errors.Join(fmt.Errorf("measure QEMU backing image before admission: %w", measureErr), cleanupOwnedWorkdir(workdir))
		}
		if virtualBytes > qemuDiskMax-qemuDiskOverhead {
			return cloudprovider.Instance{}, errors.Join(fmt.Errorf("QEMU backing image virtual size exceeds the admitted 25-GiB growth budget"), cleanupOwnedWorkdir(workdir))
		}
		admitted, err = a.Admission.Acquire(ctx, AdmissionRequest{InstanceID: id, Workdir: workdir, Profile: profile, Memory: memory, CPUs: cpus, DiskPeak: virtualBytes + qemuDiskOverhead})
		if err != nil {
			return cloudprovider.Instance{}, errors.Join(fmt.Errorf("local-QEMU host admission refused: %w", err), cleanupOwnedWorkdir(workdir))
		}
	}
	releaseAdmission := func(cause, cleanupErr error) error {
		if admitted == nil {
			return errors.Join(cause, cleanupErr)
		}
		if cleanupErr != nil {
			return errors.Join(cause, cleanupErr)
		}
		return errors.Join(cause, admitted.Release(context.Background()))
	}
	if admitted != nil {
		receipt := admitted.Receipt
		provisional := Instance{ID: id, Name: id, Provider: "local-qemu", State: StatePlanned,
			Image: filepath.Join(workdir, instanceDiskName), Workdir: workdir, Address: "127.0.0.1",
			SSHUser: spec.SSHUser, SSHPort: port, Profile: profile, CreatedAt: a.now(), UpdatedAt: a.now(), Capacity: &receipt}
		if err := writeLocalInstance(path, provisional); err != nil {
			return cloudprovider.Instance{}, releaseAdmission(fmt.Errorf("persist local-QEMU admission before provider effects: %w", err), cleanupOwnedWorkdir(workdir))
		}
		a.recordWrite(ctx)
	}
	local, err := a.QEMU.Create(ctx, Request{ID: id, Name: id, Image: spec.Image, Workdir: workdir, Memory: memory, CPUs: cpus, Profile: profile, User: spec.SSHUser, AuthorizedKey: spec.SSHAuthorizedKey, SSHPort: port})
	if err != nil {
		return cloudprovider.Instance{}, releaseAdmission(err, cleanupOwnedWorkdir(workdir))
	}
	a.recordWrite(ctx)
	local.SSHUser = spec.SSHUser
	local.Size = spec.Size
	if admitted != nil {
		receipt := admitted.Receipt
		local.Capacity = &receipt
	}
	local.State, local.PID = StateStarting, 0
	if err := writeLocalInstance(path, local); err != nil {
		cleanupErr := a.QEMU.Destroy(context.Background(), local)
		if cleanupErr == nil {
			cleanupErr = cleanupOwnedWorkdir(workdir)
		}
		return cloudprovider.Instance{}, releaseAdmission(err, cleanupErr)
	}
	a.recordWrite(ctx)
	waitCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	qemu := a.QEMU
	qemu.OnStarted = func(pid int) error {
		local.PID, local.State, local.UpdatedAt = pid, StateRunning, a.now()
		if err := writeLocalInstance(path, local); err != nil {
			return err
		}
		a.recordWrite(ctx)
		return nil
	}
	pid, err := qemu.Start(ctx, local)
	if pid > 0 {
		local.PID, local.State, local.UpdatedAt = pid, StateRunning, a.now()
	}
	if err != nil {
		cleanupErr := a.QEMU.Destroy(context.Background(), local)
		if cleanupErr == nil {
			cleanupErr = cleanupOwnedWorkdir(workdir)
		}
		return cloudprovider.Instance{}, releaseAdmission(fmt.Errorf("start local-qemu guest: %w", err), cleanupErr)
	}
	if err := writeLocalInstance(path, local); err != nil {
		cleanupErr := a.QEMU.Destroy(context.Background(), local)
		if cleanupErr == nil {
			cleanupErr = cleanupOwnedWorkdir(workdir)
		}
		return cloudprovider.Instance{}, releaseAdmission(err, cleanupErr)
	}
	a.recordWrite(ctx)
	if err := a.QEMU.WaitForSSH(waitCtx, local); err != nil {
		cleanupErr := a.QEMU.Destroy(context.Background(), local)
		if cleanupErr == nil {
			cleanupErr = cleanupOwnedWorkdir(workdir)
		}
		return cloudprovider.Instance{}, releaseAdmission(fmt.Errorf("wait for local-qemu SSH: %w", err), cleanupErr)
	}
	if admitted != nil {
		a.keepLease(id, local, admitted)
	}
	return toCloudInstance(local, spec), nil
}

func (a *Adapter) Describe(ctx context.Context, id string) (cloudprovider.Instance, error) {
	if err := a.requireReady(); err != nil {
		return cloudprovider.Instance{}, err
	}
	if err := ctx.Err(); err != nil {
		return cloudprovider.Instance{}, err
	}
	root, err := a.dataRoot(ctx)
	if err != nil {
		return cloudprovider.Instance{}, err
	}
	path, err := metadataPath(root, id)
	if err != nil {
		return cloudprovider.Instance{}, err
	}
	local, err := readLocalInstance(path)
	if err != nil {
		return cloudprovider.Instance{}, err
	}
	return toCloudInstance(local, cloudprovider.Spec{}), nil
}

func (a *Adapter) List(ctx context.Context) ([]cloudprovider.Instance, error) {
	if err := a.requireReady(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	dataRoot, err := a.dataRoot(ctx)
	if err != nil {
		return nil, err
	}
	root := filepath.Join(dataRoot, "local-qemu")
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return []cloudprovider.Instance{}, nil
	}
	if err != nil {
		return nil, err
	}
	instances := make([]cloudprovider.Instance, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path, err := metadataPath(dataRoot, entry.Name())
		if err != nil {
			return nil, err
		}
		local, err := readOwnedInstance(dataRoot, entry.Name(), path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		instances = append(instances, toCloudInstance(local, cloudprovider.Spec{}))
	}
	return instances, nil
}

func (a *Adapter) Destroy(ctx context.Context, id string) error {
	root, err := a.dataRoot(ctx)
	if err != nil {
		return err
	}
	path, err := metadataPath(root, id)
	if err != nil {
		return err
	}
	local, err := readOwnedInstance(root, id, path)
	if err != nil {
		return err
	}
	a.leaseMu.Lock()
	item := a.leases[id]
	var lease *QEMULease
	if item != nil {
		lease = item.capacity
	}
	a.leaseMu.Unlock()
	if lease == nil && local.Capacity != nil && a.Admission != nil {
		lease, _ = a.Admission.Reattach(ctx, *local.Capacity)
	}
	if err := a.stopAndClean(ctx, local, lease); err != nil {
		return err
	}
	a.forgetLease(id)
	return nil
}

func (a *Adapter) Snapshot(ctx context.Context, id, name string) error {
	if err := a.requireReady(); err != nil {
		return err
	}
	local, err := a.local(ctx, id)
	if err != nil {
		return err
	}
	return a.withStopped(ctx, local, func(stopped Instance) error { return a.QEMU.Snapshot(ctx, stopped, name) })
}

func (a *Adapter) Reset(ctx context.Context, id, name string) error {
	if err := a.requireReady(); err != nil {
		return err
	}
	local, err := a.local(ctx, id)
	if err != nil {
		return err
	}
	return a.withStopped(ctx, local, func(stopped Instance) error { return a.QEMU.Reset(ctx, stopped, name) })
}

func (a *Adapter) withStopped(ctx context.Context, local Instance, operation func(Instance) error) error {
	root, err := a.dataRoot(ctx)
	if err != nil {
		return err
	}
	path, err := metadataPath(root, local.ID)
	if err != nil {
		return err
	}
	running := local.State == StateRunning
	if running {
		if err := a.QEMU.StopAndWait(ctx, local); err != nil {
			return err
		}
		local.State, local.PID, local.UpdatedAt = StateStopped, 0, a.now()
		if err := writeLocalInstance(path, local); err != nil {
			return err
		}
		a.recordWrite(ctx)
	}
	operationErr := operation(local)
	if operationErr == nil {
		a.recordWrite(ctx)
	}
	if !running {
		return operationErr
	}
	pid, startErr := a.QEMU.Start(ctx, local)
	if startErr != nil {
		return errors.Join(operationErr, fmt.Errorf("restart QEMU guest after test operation: %w", startErr))
	}
	local.PID, local.State, local.UpdatedAt = pid, StateRunning, a.now()
	if writeErr := writeLocalInstance(path, local); writeErr != nil {
		return errors.Join(operationErr, writeErr)
	}
	a.recordWrite(ctx)
	waitCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	if waitErr := a.QEMU.WaitForSSH(waitCtx, local); waitErr != nil {
		return errors.Join(operationErr, fmt.Errorf("wait for QEMU SSH after test operation: %w", waitErr))
	}
	return operationErr
}

func (a *Adapter) local(ctx context.Context, id string) (Instance, error) {
	root, err := a.dataRoot(ctx)
	if err != nil {
		return Instance{}, err
	}
	path, err := metadataPath(root, id)
	if err != nil {
		return Instance{}, err
	}
	return readOwnedInstance(root, id, path)
}

// readOwnedInstance refuses metadata that points outside the exact manager-
// generated instance directory. Metadata is persisted across API restarts, so
// every effectful path validates it before trusting PID, disk, or workdir.
func readOwnedInstance(root, id, metadata string) (Instance, error) {
	item, err := readLocalInstance(metadata)
	if err != nil {
		return Instance{}, err
	}
	expectedWorkdir := filepath.Join(root, "local-qemu", id)
	if item.ID != id || item.Provider != "local-qemu" || filepath.Clean(item.Workdir) != filepath.Clean(expectedWorkdir) || filepath.Clean(item.Image) != filepath.Join(filepath.Clean(expectedWorkdir), instanceDiskName) {
		return Instance{}, fmt.Errorf("refusing local-qemu metadata that does not match manager-owned instance %q", id)
	}
	info, err := os.Lstat(expectedWorkdir)
	if err != nil {
		return Instance{}, fmt.Errorf("inspect manager-owned local-qemu workdir: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return Instance{}, fmt.Errorf("refusing non-directory or symlink local-qemu workdir for %q", id)
	}
	if item.State == StateRunning && item.PID <= 0 {
		return Instance{}, fmt.Errorf("refusing running local-qemu metadata without a process ID for %q", id)
	}
	return item, nil
}

func metadataPath(root, id string) (string, error) {
	if len(id) != 29 || !strings.HasPrefix(id, "qemu-") {
		return "", fmt.Errorf("invalid local-qemu instance id")
	}
	for _, char := range id[5:] {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return "", fmt.Errorf("invalid local-qemu instance id")
		}
	}
	if strings.TrimSpace(root) == "" {
		return "", fmt.Errorf("local-qemu data directory is unavailable")
	}
	return filepath.Join(root, "local-qemu", id, metadataName), nil
}

func (a *Adapter) dataRoot(ctx context.Context) (string, error) {
	if a.Roots != nil {
		return a.Roots.Pick(ctx, storage.ClassData)
	}
	if root := strings.TrimSpace(a.DataDir); root != "" {
		return root, nil
	}
	return "", fmt.Errorf("local-qemu data directory is unavailable")
}

func (a *Adapter) recordWrite(ctx context.Context) {
	if a.Roots != nil {
		a.Roots.RecordWrite(ctx)
	}
}

func readLocalInstance(path string) (Instance, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Instance{}, err
	}
	var item Instance
	if err := json.Unmarshal(data, &item); err != nil {
		return Instance{}, fmt.Errorf("decode local-qemu metadata: %w", err)
	}
	return item, nil
}

func writeLocalInstance(path string, item Instance) error {
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}
	return storage.WriteFileAtomic(path, data, 0o600)
}

func cleanupOwnedWorkdir(workdir string) error {
	entries, err := os.ReadDir(workdir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return fmt.Errorf("refusing to recursively remove unexpected local-qemu directory %q", entry.Name())
		}
		if err := storage.RemoveFile(filepath.Join(workdir, entry.Name())); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if err := storage.RemoveFile(workdir); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (a *Adapter) now() time.Time {
	if a.Now != nil {
		return a.Now().UTC()
	}
	return time.Now().UTC()
}

func toCloudInstance(item Instance, spec cloudprovider.Spec) cloudprovider.Instance {
	size := spec.Size
	if size == "" {
		size = item.Size
	}
	if size == "" {
		size = "local-qemu"
	}
	user := item.SSHUser
	if user == "" {
		user = spec.SSHUser
	}
	return cloudprovider.Instance{ID: item.ID, Region: "local", Size: size, Image: item.Image, Address: item.Address, SSHUser: user, SSHPort: item.SSHPort, CreatedAt: item.CreatedAt, Tags: spec.Tags}
}

var _ cloudprovider.Provider = (*Adapter)(nil)
var _ cloudprovider.SnapshotResetProvider = (*Adapter)(nil)
