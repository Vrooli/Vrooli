package commands

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/packages/capacity"
	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/policy"
	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/upstream"
)

func ModelCommands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default()
	}
	return cliapp.SubcommandGroup{
		Name:        "models",
		Description: "Pull and inspect digest-pinned decision models in the managed daemon",
		Subcommands: []cliapp.Command{
			{Name: "list", Description: "List pinned models and whether the daemon holds the pinned digest", Usage: "resource-ollaya models list [--json]", Run: h.ListModels},
			{Name: "pull", Description: "Pull pinned models by digest (default: every default_pull model)", Usage: "resource-ollaya models pull [--model <name>] [--timeout 30m] [--json]", Run: h.PullModels},
			{Name: "retention", Description: "Report each installed model's size, unique bytes, residency and whether disk eviction must keep it", Usage: "resource-ollaya models retention [--json]", Run: h.RetentionInventory},
			{Name: "evict", Description: "Fenced disk eviction of one idle, non-required, digest-pinned model; records the reacquire command first", Usage: "resource-ollaya models evict <name> --digest <digest> --size <bytes> [--reason <text>] [--requested-by <who>] [--json]", Run: h.EvictModel},
		},
	}
}

type modelRow struct {
	Name           string `json:"name"`
	PinnedDigest   string `json:"pinned_digest"`
	LocalDigest    string `json:"local_digest,omitempty"`
	Installed      bool   `json:"installed"`
	MatchesPinned  bool   `json:"matches_pinned"`
	DefaultPull    bool   `json:"default_pull"`
	DiskBytes      int64  `json:"disk_bytes"`
	License        string `json:"license"`
	InstalledError string `json:"error,omitempty"`
}

func (h *Handlers) ListModels(args []string) error {
	fs := flag.NewFlagSet("models list", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	jsonOutput := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	p, err := h.policy()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	installed, localErr := h.client().LocalModels(ctx)
	rows := h.modelRows(p, installed, localErr)
	if *jsonOutput {
		return writeJSON(h.Stdout, map[string]any{"models": rows})
	}
	for _, row := range rows {
		fmt.Fprintf(h.Stdout, "%-20s installed=%t pinned=%t default_pull=%t disk_bytes=%d\n", row.Name, row.Installed, row.MatchesPinned, row.DefaultPull, row.DiskBytes)
	}
	return localErr
}

func (h *Handlers) PullModels(args []string) error {
	fs := flag.NewFlagSet("models pull", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	model := fs.String("model", "", "Pinned model name; omitted means every default_pull model")
	timeout := fs.Duration("timeout", 30*time.Minute, "Overall pull deadline")
	jsonOutput := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	p, err := h.policy()
	if err != nil {
		return err
	}
	names := p.DefaultPulls()
	if name := strings.TrimSpace(*model); name != "" {
		pinned, ok := p.Models[name]
		if !ok {
			return fmt.Errorf("model %q is not pinned in model-policy.json", name)
		}
		names = append([]string{name}, pinned.Routes...)
	}
	// Resolve the complete model set before obtaining admission. The initial
	// inventory is read-only; no pull endpoint is reachable until the shared
	// disk-job lease has been reserved.
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	var root string
	var client *upstream.Client
	var verifyDaemon func(context.Context) error
	var releaseLifecycle func()
	if h.PrepareModelPull != nil {
		root, client, verifyDaemon, releaseLifecycle, err = h.PrepareModelPull(ctx)
	} else {
		root, client, verifyDaemon, releaseLifecycle, err = prepareManagedModelPull(ctx, h.GetEnv)
	}
	if err != nil {
		return err
	}
	if releaseLifecycle != nil {
		defer releaseLifecycle()
	}
	if root == "" || client == nil || verifyDaemon == nil {
		return errors.New("managed Ollaya model-pull owner is incomplete; refusing daemon access")
	}
	if err := verifyDaemon(ctx); err != nil {
		return fmt.Errorf("verify managed Ollaya owner before model inventory: %w", err)
	}
	installed, err := client.LocalModels(ctx)
	if err != nil {
		return h.fail(err)
	}
	missing := make([]string, 0, len(names))
	for _, name := range names {
		got, present := installed[upstream.InstalledName(name)]
		if present && got != p.Models[name].Digest {
			return fmt.Errorf("model %s is already installed with manifest %s; refusing to replace it with pinned manifest %s", name, got, p.Models[name].Digest)
		}
		if !present {
			missing = append(missing, name)
		}
	}
	if len(missing) == 0 {
		return h.writePullResult(*jsonOutput, names, h.modelRows(p, installed, nil))
	}
	var admission *modelDiskAdmission
	var lease *modelDiskLease
	var closeAdmission func() error
	if h.ModelDiskAdmit != nil {
		lease, closeAdmission, err = h.ModelDiskAdmit(ctx, missing, p)
	} else {
		admission, closeAdmission, err = newModelDiskAdmission(ctx, root, verifyDaemon)
		if err == nil {
			lease, err = admission.Admit(ctx, missing, p)
		}
	}
	if closeAdmission != nil {
		defer func() { _ = closeAdmission() }()
	}
	if err != nil {
		return err
	}
	if lease == nil {
		return errors.New("shared Ollaya disk admission returned no lease; refusing model pull")
	}
	// A failed or interrupted daemon pull can leave partial files, and the
	// daemon API cannot prove that remote work or cleanup stopped. In that
	// case the durable lease intentionally remains active for owner recovery.
	lease.verifyDaemon = verifyDaemon
	workCtx, stopMonitor := lease.StartMonitor(ctx)
	defer stopMonitor()
	for _, name := range missing {
		if verifyDaemon != nil {
			if err := verifyDaemon(workCtx); err != nil {
				return fmt.Errorf("verify managed Ollaya owner before pull; disk-job lease %s remains active: %w", lease.ID(), err)
			}
		}
		reference := upstream.InstalledName(name) + "@sha256:" + p.Models[name].Digest
		fmt.Fprintf(h.Stderr, "pulling %s\n", reference)
		if err := client.Pull(workCtx, reference); err != nil {
			return h.fail(fmt.Errorf("pull %s; disk-job lease %s remains active because cleanup is unverified: %w", name, lease.ID(), err))
		}
		if verifyDaemon != nil {
			if err := verifyDaemon(workCtx); err != nil {
				return fmt.Errorf("managed Ollaya owner changed after pull; disk-job lease %s remains active: %w", lease.ID(), err)
			}
		}
		installed, err = client.LocalModels(workCtx)
		if err != nil {
			return h.fail(fmt.Errorf("verify pulled model %s; disk-job lease %s remains active: %w", name, lease.ID(), err))
		}
		if got := installed[upstream.InstalledName(name)]; got != p.Models[name].Digest {
			return fmt.Errorf("pulled model %s manifest %q does not match pinned digest %q; disk-job lease %s remains active", name, got, p.Models[name].Digest, lease.ID())
		}
	}
	if verifyDaemon != nil {
		if err := verifyDaemon(workCtx); err != nil {
			return fmt.Errorf("verify managed Ollaya owner before completion; disk-job lease %s remains active: %w", lease.ID(), err)
		}
	}
	installed, err = client.LocalModels(workCtx)
	if err != nil {
		return h.fail(fmt.Errorf("verify installed Ollaya model set; disk-job lease %s remains active: %w", lease.ID(), err))
	}
	for _, name := range names {
		if got := installed[upstream.InstalledName(name)]; got != p.Models[name].Digest {
			return fmt.Errorf("installed model %s manifest %q does not match pinned digest %q; disk-job lease %s remains active", name, got, p.Models[name].Digest, lease.ID())
		}
	}
	receipt, err := lease.Finish(ctx, p, names)
	if err != nil {
		return err
	}
	stopMonitor()
	if *jsonOutput {
		verified := make(map[string]string, len(names))
		for _, name := range names {
			verified[name] = installed[upstream.InstalledName(name)]
		}
		return writeJSON(h.Stdout, map[string]any{"pulled": names, "models": h.modelRows(p, installed, nil), "disk_job_receipt": receipt, "verified_model_digests": verified})
	}
	fmt.Fprintf(h.Stdout, "pulled: %s\ndisk job: %s (%s)\n", strings.Join(names, ", "), receipt.ID, receipt.Status)
	return nil
}

func (h *Handlers) writePullResult(jsonOutput bool, names []string, rows []modelRow) error {
	if jsonOutput {
		return writeJSON(h.Stdout, map[string]any{"pulled": names, "models": rows})
	}
	fmt.Fprintf(h.Stdout, "already pinned: %s\n", strings.Join(names, ", "))
	return nil
}

const ollayaDiskJobOwner = "resource-ollaya:model-pull"
const ollayaDiskHeartbeatInterval = 20 * time.Second

type modelDiskAdmitter interface {
	ReserveDiskJob(context.Context, capacity.DiskJobRequest) (capacity.DiskJobLease, error)
}

type modelDiskAdmission struct {
	broker modelDiskAdmitter
	root   string
	verify func(context.Context) error
}

type modelDiskLease struct {
	lease        capacity.DiskJobLease
	root         string
	baseline     int64
	peak         int64
	last         int64
	mu           sync.Mutex
	verifyDaemon func(context.Context) error
}

func newModelDiskAdmission(ctx context.Context, root string, verify func(context.Context) error) (*modelDiskAdmission, func() error, error) {
	if verify == nil {
		return nil, nil, errors.New("managed Ollaya owner verification is required for disk admission")
	}
	if err := verify(ctx); err != nil {
		return nil, nil, fmt.Errorf("verify managed Ollaya process before disk admission: %w", err)
	}
	broker, err := capacity.NewBroker(ctx, "")
	if err != nil {
		return nil, nil, fmt.Errorf("open shared capacity broker: %w", err)
	}
	return &modelDiskAdmission{broker: broker, root: root, verify: verify}, broker.Close, nil
}

func (a *modelDiskAdmission) Admit(ctx context.Context, names []string, p policy.Policy) (*modelDiskLease, error) {
	if a == nil || a.broker == nil {
		return nil, errors.New("shared Ollaya disk admission is unavailable")
	}
	if a.verify == nil {
		return nil, errors.New("managed Ollaya owner verification is required for disk admission")
	}
	if err := a.verify(ctx); err != nil {
		return nil, fmt.Errorf("verify managed Ollaya process before reservation: %w", err)
	}
	var peak int64
	for _, name := range names {
		bytes := p.Models[name].DiskBytes
		if bytes <= 0 || peak > math.MaxInt64-bytes {
			return nil, fmt.Errorf("model %s has no bounded disk estimate; refusing admission", name)
		}
		peak += bytes
	}
	// Allow 25% for manifest/index/temporary duplication plus a fixed 64 MiB
	// metadata margin. Unknown/overflowing estimates fail closed.
	if peak > (math.MaxInt64-(64<<20))*4/5 {
		return nil, errors.New("Ollaya pull estimate overflows disk admission")
	}
	peak = peak + peak/4 + (64 << 20)
	baseline, err := modelDirectoryBytes(a.root)
	if err != nil {
		return nil, fmt.Errorf("sample Ollaya model store before admission: %w", err)
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, fmt.Errorf("create Ollaya disk-job identity: %w", err)
	}
	lease, err := a.broker.ReserveDiskJob(ctx, capacity.DiskJobRequest{
		OwnerID: ollayaDiskJobOwner, JobID: hex.EncodeToString(nonce[:]),
		Paths:      []capacity.DiskJobPath{{Path: a.root, Role: "ollaya-model-store-and-pull-temporaries", PeakBytes: peak}},
		FloorBytes: 0,
	})
	if err != nil {
		return nil, fmt.Errorf("reserve shared Ollaya model-store capacity: %w", err)
	}
	if _, err := lease.SamplePath(ctx, a.root, baseline, 0); err != nil {
		return nil, fmt.Errorf("record Ollaya disk-job baseline; lease %s remains active for recovery: %w", lease.Reservation().ID, err)
	}
	if err := a.verify(ctx); err != nil {
		return nil, fmt.Errorf("managed Ollaya process changed after disk reservation; lease %s remains active: %w", lease.Reservation().ID, err)
	}
	return &modelDiskLease{lease: lease, root: a.root, baseline: baseline, last: baseline, peak: peak}, nil
}

func (l *modelDiskLease) ID() string { return l.lease.Reservation().ID }

func (l *modelDiskLease) StartMonitor(parent context.Context) (context.Context, func()) {
	ctx, cancel := context.WithCancel(parent)
	go func() {
		ticker := time.NewTicker(ollayaDiskHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if (l.verifyDaemon != nil && l.verifyDaemon(ctx) != nil) || l.sample(ctx) != nil || l.lease.Heartbeat(ctx) != nil {
					cancel()
					return
				}
			}
		}
	}()
	return ctx, cancel
}

func (l *modelDiskLease) sample(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	bytes, err := modelDirectoryBytes(l.root)
	if err != nil {
		return err
	}
	if bytes > l.last {
		l.last = bytes
	}
	delta := l.last - l.baseline
	extra := int64(0)
	if delta > l.peak {
		extra = delta
	}
	_, err = l.lease.SamplePath(ctx, l.root, l.last, extra)
	return err
}

func (l *modelDiskLease) Finish(ctx context.Context, p policy.Policy, names []string) (capacity.DiskJobReceipt, error) {
	if err := l.sample(ctx); err != nil {
		return capacity.DiskJobReceipt{}, fmt.Errorf("sample Ollaya model-store before lease completion; lease %s remains active: %w", l.ID(), err)
	}
	for _, name := range names {
		// The caller has verified the daemon manifest. This final metadata check
		// ensures the required model set has no missing digest before receipt.
		if !isPinnedDigest(p.Models[name].Digest) {
			return capacity.DiskJobReceipt{}, fmt.Errorf("model %s has invalid pinned digest; lease %s remains active", name, l.ID())
		}
	}
	receipt, err := l.lease.Complete(ctx)
	if err != nil {
		return capacity.DiskJobReceipt{}, fmt.Errorf("complete Ollaya disk job %s: %w", l.ID(), err)
	}
	return receipt, nil
}

func isPinnedDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

// modelDirectoryBytes sums the model store while a pull may be writing into it.
// The daemon downloads into temporary files and renames them into place, so an
// entry that was listed can be gone by the time it is inspected; that entry has
// not grown the store and must not abort the sample, because an aborted sample
// cancels the pull it was only meant to account for.
func modelDirectoryBytes(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			if errors.Is(walkErr, fs.ErrNotExist) && path != root {
				return nil
			}
			return walkErr
		}
		size, err := storedEntryBytes(path, entry)
		if err != nil {
			return err
		}
		if size < 0 || total > math.MaxInt64-size {
			return errors.New("model-store byte count overflow")
		}
		total += size
		return nil
	})
	return total, err
}

// storedEntryBytes is the bytes one store entry holds: zero for a directory or
// an entry that vanished mid-walk, an error for anything that is not a regular
// file (a symlink could point outside the admitted store).
func storedEntryBytes(path string, entry os.DirEntry) (int64, error) {
	info, err := entry.Info()
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return 0, fmt.Errorf("model store contains symlink %s", path)
	}
	if info.IsDir() {
		return 0, nil
	}
	if !info.Mode().IsRegular() {
		return 0, fmt.Errorf("unsupported model-store entry %s", path)
	}
	return info.Size(), nil
}

func (h *Handlers) modelRows(p policy.Policy, installed map[string]string, localErr error) []modelRow {
	rows := make([]modelRow, 0, len(p.Models))
	for _, name := range sortedModelNames(p) {
		model := p.Models[name]
		row := modelRow{Name: name, PinnedDigest: model.Digest, DefaultPull: model.DefaultPull, DiskBytes: model.DiskBytes, License: model.License}
		if localErr != nil {
			row.InstalledError = localErr.Error()
		} else if digest, ok := installed[upstream.InstalledName(name)]; ok {
			row.Installed, row.LocalDigest, row.MatchesPinned = true, digest, digest == model.Digest
		}
		rows = append(rows, row)
	}
	return rows
}

func sortedModelNames(p policy.Policy) []string {
	names := make([]string, 0, len(p.Models))
	for name := range p.Models {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
