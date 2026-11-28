package ensure

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	runtimestorage "github.com/vrooli/vrooli/internal/resources/runtime/storage"
	"github.com/vrooli/vrooli/packages/capacity"
)

const (
	ollamaDiskOwner          = "resource-ollama:model-pull"
	manifestWriteAllowance   = int64(3 * maxManifestBytes)
	metadataPerBlobAllowance = int64(1 << 20)
	maxPinnedDescriptors     = 64
)

var diskHeartbeatInterval = 20 * time.Second

type diskJobAdmitter interface {
	ReserveDiskJob(context.Context, capacity.DiskJobRequest) (capacity.DiskJobLease, error)
}

// DiskAdmission owns the Ollama-specific adapter while all accounting and
// admission decisions remain in the shared capacity package.
type DiskAdmission struct {
	Broker       diskJobAdmitter
	Root         string
	Client       *Client
	Fetch        PreflightClient
	VerifyDaemon func(context.Context) error
}

func NewDiskAdmission(ctx context.Context, client *Client) (*DiskAdmission, func() error, error) {
	root, err := effectiveModelRoot()
	if err != nil {
		return nil, nil, err
	}
	managedClient, identity, err := resolveManagedOllamaClient(ctx, root)
	if err != nil {
		return nil, nil, err
	}
	if client == nil {
		client = managedClient
	} else {
		*client = *managedClient
	}
	broker, err := capacity.NewBroker(ctx, "")
	if err != nil {
		return nil, nil, fmt.Errorf("open shared capacity broker: %w", err)
	}
	verify := func(ctx context.Context) error { return verifyManagedOllama(ctx, identity) }
	return &DiskAdmission{Broker: broker, Root: root, Client: client, Fetch: registryPreflight{}, VerifyDaemon: verify}, broker.Close, nil
}

func effectiveModelRoot() (string, error) {
	home, err := hostreqkit.InvokingUserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve invoking user's home for Ollama storage: %w", err)
	}
	_, _, sudoed := hostreqkit.InvokingUserIDs()
	root, err := configuredModelRoot(os.Getenv, home, runtime.GOOS, sudoed)
	if err != nil {
		return "", err
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve Ollama model root: %w", err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolve Ollama model-root symlinks: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("Ollama model root is unavailable or not a directory: %s", root)
	}
	return root, nil
}

// configuredModelRoot resolves the managed Ollama resource's data directory
// through the same shared storage authority as resource lifecycle setup.
// RESOURCE_DATA_DIR is intentionally ignored here because callers often run
// `resource-ollama ensure` from a scenario whose own data directory is in that
// variable. An explicit OLLAMA_MODELS override is accepted only when it agrees
// with the managed resource's declared `${RESOURCE_DATA_DIR}/models` path.
func configuredModelRoot(getenv func(string) string, home, runtimeOS string, sudoed bool) (string, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	cfg := runtimestorage.ResolverConfig{
		AppID:     "vrooli",
		RuntimeOS: runtimeOS,
		EnvGet:    getenv,
		UserHomeDir: func() (string, error) {
			if strings.TrimSpace(home) == "" {
				return "", errors.New("invoking user's home directory is unavailable")
			}
			return home, nil
		},
	}
	if sudoed && runtimeOS == "linux" {
		cfg.EnvGet = func(key string) string {
			switch key {
			case "XDG_DATA_HOME":
				return filepath.Join(home, ".local", "share")
			case "XDG_STATE_HOME":
				return filepath.Join(home, ".local", "state")
			default:
				return getenv(key)
			}
		}
		cfg.UserConfigDir = func() (string, error) { return filepath.Join(home, ".config"), nil }
		cfg.UserCacheDir = func() (string, error) { return filepath.Join(home, ".cache"), nil }
	}
	resolver, err := runtimestorage.NewResolver(cfg)
	if err != nil {
		return "", fmt.Errorf("resolve managed Ollama resource storage: %w", err)
	}
	paths, err := resolver.Resolve(runtimestorage.Options{ResourceID: "ollama"})
	if err != nil {
		return "", fmt.Errorf("resolve managed Ollama resource paths: %w", err)
	}
	root := filepath.Join(paths.DataDir, "models")
	if !filepath.IsAbs(root) {
		return "", errors.New("managed Ollama model-store path is not absolute; refusing pull")
	}
	if override := strings.TrimSpace(getenv("OLLAMA_MODELS")); override != "" {
		override, err = filepath.Abs(override)
		if err != nil {
			return "", fmt.Errorf("resolve OLLAMA_MODELS override: %w", err)
		}
		if filepath.Clean(override) != filepath.Clean(root) {
			return "", errors.New("OLLAMA_MODELS does not match the managed Ollama resource model-store path; refusing an unverified write location")
		}
	}
	return root, nil
}

func (a *DiskAdmission) Admit(ctx context.Context, modelRef string) (PullLease, error) {
	if a == nil || a.Broker == nil || a.Client == nil || a.Fetch == nil {
		return nil, errors.New("shared Ollama disk admission is not configured")
	}
	if a.VerifyDaemon != nil {
		if err := a.VerifyDaemon(ctx); err != nil {
			return nil, fmt.Errorf("verify managed Ollama process and model store before admission: %w", err)
		}
	}
	root, err := filepath.EvalSymlinks(a.Root)
	if err != nil {
		return nil, fmt.Errorf("resolve Ollama model root before admission: %w", err)
	}
	pinned, err := PreparePinnedPull(ctx, modelRef, a.Fetch)
	if err != nil {
		return nil, err
	}
	closePin := func() { _ = pinned.Close() }
	if len(pinned.LayerDigests()) > maxPinnedDescriptors {
		closePin()
		return nil, fmt.Errorf("pinned manifest has more than %d bounded descriptors", maxPinnedDescriptors)
	}
	peak := pinned.PeakBytes()
	metadataAllowance := int64(len(pinned.LayerDigests())) * metadataPerBlobAllowance
	if peak > math.MaxInt64-manifestWriteAllowance || peak+manifestWriteAllowance > math.MaxInt64-metadataAllowance {
		closePin()
		return nil, errors.New("pinned pull peak overflows capacity accounting")
	}
	peak += manifestWriteAllowance + metadataAllowance
	if peak <= 0 {
		closePin()
		return nil, errors.New("pinned pull has unknown peak; refusing admission")
	}
	baseline, err := directoryBytes(root)
	if err != nil {
		closePin()
		return nil, fmt.Errorf("sample Ollama model root before reservation: %w", err)
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		closePin()
		return nil, fmt.Errorf("create disk-job identity: %w", err)
	}
	lease, err := a.Broker.ReserveDiskJob(ctx, capacity.DiskJobRequest{
		OwnerID:    ollamaDiskOwner,
		JobID:      hex.EncodeToString(nonce[:]),
		Paths:      []capacity.DiskJobPath{{Path: root, Role: "ollama-model-store-and-pull-temporaries", PeakBytes: peak}},
		FloorBytes: 0, // The owner applies the configured floor; Ollama never lowers it.
	})
	if err != nil {
		closePin()
		return nil, fmt.Errorf("reserve shared Ollama model-store capacity: %w", err)
	}
	workCtx, cancel := context.WithCancel(ctx)
	pullLease := &ollamaPullLease{root: root, baseline: baseline, lastObserved: baseline, peak: peak, pinned: pinned, disk: lease, client: a.Client, modelRef: modelRef, expectedDigest: pinned.ManifestDigest(), workCtx: workCtx, cancel: cancel, verifyDaemon: a.VerifyDaemon}
	if a.VerifyDaemon != nil {
		if err := a.VerifyDaemon(ctx); err != nil {
			closePin()
			return pullLease, fmt.Errorf("managed Ollama process changed after disk reservation; lease %s remains active: %w", lease.Reservation().ID, err)
		}
	}
	baseline, err = directoryBytes(root)
	if err != nil {
		closePin()
		return pullLease, fmt.Errorf("sample Ollama model root after reservation; lease %s remains active for owner recovery: %w", lease.Reservation().ID, err)
	}
	pullLease.baseline, pullLease.lastObserved = baseline, baseline
	if _, err := lease.SamplePath(ctx, root, baseline, 0); err != nil {
		closePin()
		return pullLease, fmt.Errorf("record Ollama model-store baseline; lease %s remains active for owner recovery: %w", lease.Reservation().ID, err)
	}
	go pullLease.heartbeat()
	return pullLease, nil
}

type ollamaPullLease struct {
	root, modelRef               string
	baseline, lastObserved, peak int64
	pinned                       *PinnedPull
	disk                         capacity.DiskJobLease
	client                       *Client
	expectedDigest               string
	workCtx                      context.Context
	cancel                       context.CancelFunc
	verifyDaemon                 func(context.Context) error
	sampleMu                     sync.Mutex
	stopOnce                     sync.Once
}

func (l *ollamaPullLease) WorkContext() context.Context { return l.workCtx }
func (l *ollamaPullLease) LeaseID() string              { return l.disk.Reservation().ID }
func (l *ollamaPullLease) PullReference() string        { return l.pinned.PullReference() }

func (l *ollamaPullLease) heartbeat() {
	ticker := time.NewTicker(diskHeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-l.workCtx.Done():
			return
		case <-ticker.C:
			if err := l.VerifyRemote(l.workCtx); err != nil {
				l.cancel()
				return
			}
			if err := l.samplePath(l.workCtx); err != nil {
				l.cancel()
				return
			}
			if err := l.disk.Heartbeat(l.workCtx); err != nil {
				l.cancel()
				return
			}
		}
	}
}

func (l *ollamaPullLease) Abandon() {
	l.stopOnce.Do(func() {
		l.cancel()
		_ = l.pinned.Close()
	})
}

func (l *ollamaPullLease) Install(ctx context.Context, target string) error {
	// Recheck just before mutation. Ollama's copy API has no no-clobber option;
	// never overwrite a target that became installed since the initial listing.
	tags, err := l.client.ListTags(ctx)
	if err != nil {
		return fmt.Errorf("check target before pinned install: %w", err)
	}
	if modelReferenceInstalled(tags, target) {
		return fmt.Errorf("target %s became installed during acquisition; refusing to overwrite it", target)
	}
	if err := l.pinned.InstallTarget(ctx, l.root, target); err != nil {
		return err
	}
	if err := l.client.DeleteModel(ctx, l.pinned.PullReference()); err != nil {
		return fmt.Errorf("remove temporary pinned model alias: %w", err)
	}
	return nil
}

func (l *ollamaPullLease) ObserveProgress(ctx context.Context, _ PullProgress) error {
	if err := l.VerifyRemote(ctx); err != nil {
		l.cancel()
		return fmt.Errorf("verify managed Ollama process during pull: %w", err)
	}
	return l.samplePath(ctx)
}

func (l *ollamaPullLease) VerifyRemote(ctx context.Context) error {
	if l.verifyDaemon == nil {
		return nil
	}
	return l.verifyDaemon(ctx)
}

func (l *ollamaPullLease) samplePath(ctx context.Context) error {
	l.sampleMu.Lock()
	defer l.sampleMu.Unlock()
	currentBytes, err := directoryBytes(l.root)
	if err != nil {
		return fmt.Errorf("scan model-store growth: %w", err)
	}
	// Capacity receipts require monotonic samples. Preserve the maximum observed
	// logical path size when Ollama removes resumable metadata/partials.
	if currentBytes > l.lastObserved {
		l.lastObserved = currentBytes
	}
	delta := l.lastObserved - l.baseline
	largerPeak := int64(0)
	if delta > l.peak {
		largerPeak = delta
	}
	if _, err := l.disk.SamplePath(ctx, l.root, l.lastObserved, largerPeak); err != nil {
		return fmt.Errorf("record or extend shared model-store peak: %w", err)
	}
	return nil
}

func (l *ollamaPullLease) VerifyInstalled(ctx context.Context, model ModelInfo) error {
	if l.verifyDaemon != nil {
		if err := l.verifyDaemon(ctx); err != nil {
			return fmt.Errorf("verify managed Ollama process before installation completion: %w", err)
		}
	}
	if model.Digest != l.expectedDigest {
		return fmt.Errorf("installed model digest %q does not match pinned manifest %q", model.Digest, l.expectedDigest)
	}
	if model.Size <= 0 || model.Size > l.peak {
		return fmt.Errorf("installed model size %d is unknown or exceeds reserved peak %d", model.Size, l.peak)
	}
	return nil
}

func (l *ollamaPullLease) RecordInstallProvenance(ctx context.Context, model ModelInfo) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("record Ollama install provenance: %w", err)
	}
	if l.verifyDaemon != nil {
		if err := l.verifyDaemon(ctx); err != nil {
			return fmt.Errorf("verify managed Ollama process before provenance receipt: %w", err)
		}
	}
	if model.Digest != l.expectedDigest {
		return fmt.Errorf("installed model digest %q does not match pinned manifest %q", model.Digest, l.expectedDigest)
	}
	if err := recordModelInstallProvenance(l.root, l.pinned, model, time.Now()); err != nil {
		return fmt.Errorf("record exact Ollama install provenance: %w", err)
	}
	return nil
}

func (l *ollamaPullLease) VerifyCleanup(ctx context.Context) error {
	if l.verifyDaemon != nil {
		if err := l.verifyDaemon(ctx); err != nil {
			return fmt.Errorf("verify managed Ollama process during cleanup: %w", err)
		}
	}
	tags, err := l.client.ListTags(ctx)
	if err != nil {
		return fmt.Errorf("verify temporary alias removal: %w", err)
	}
	if tags[l.pinned.PullReference()] || tags[withLatestTag(l.pinned.PullReference())] {
		return errors.New("temporary pinned model alias remains installed")
	}
	for _, digest := range l.pinned.LayerDigests() {
		name := "sha256-" + strings.TrimPrefix(digest, "sha256:")
		if err := filepath.WalkDir(filepath.Join(l.root, "blobs"), func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			if strings.Contains(d.Name(), name) && (strings.Contains(d.Name(), "partial") || strings.Contains(d.Name(), ".tmp")) {
				return fmt.Errorf("incomplete Ollama blob artifact remains: %s", d.Name())
			}
			return nil
		}); err != nil {
			return err
		}
	}
	if err := l.samplePath(ctx); err != nil {
		return fmt.Errorf("record final model-store growth: %w", err)
	}
	return nil
}

func (l *ollamaPullLease) Complete(ctx context.Context) error {
	if err := l.pinned.Close(); err != nil {
		return fmt.Errorf("close temporary Ollama registry pin: %w", err)
	}
	if _, err := l.disk.Complete(ctx); err != nil {
		return fmt.Errorf("complete shared Ollama disk job: %w", err)
	}
	l.stopOnce.Do(func() { l.cancel() })
	return nil
}

func directoryBytes(root string) (int64, error) {
	rootInfo, err := os.Stat(root)
	if err != nil {
		return 0, err
	}
	rootDevice, err := fileDevice(rootInfo)
	if err != nil {
		return 0, fmt.Errorf("identify Ollama model-root filesystem: %w", err)
	}
	var total int64
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		device, err := fileDevice(info)
		if err != nil {
			return fmt.Errorf("identify Ollama model-store filesystem for %s: %w", path, err)
		}
		if device != rootDevice {
			return fmt.Errorf("Ollama model-store subtree crosses filesystems at %s", path)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("Ollama model-store contains a symlink: %s", path)
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported non-regular model-store entry: %s", path)
		}
		if info.Size() < 0 || info.Size() > math.MaxInt64-total {
			return errors.New("Ollama model-store byte count overflows")
		}
		total += info.Size()
		return nil
	})
	return total, err
}

func fileDevice(info os.FileInfo) (uint64, error) {
	sys := reflect.ValueOf(info.Sys())
	if sys.Kind() == reflect.Pointer {
		sys = sys.Elem()
	}
	if !sys.IsValid() || sys.Kind() != reflect.Struct {
		return 0, errors.New("filesystem identity is unavailable")
	}
	field := sys.FieldByName("Dev")
	if !field.IsValid() {
		return 0, errors.New("platform does not expose filesystem identity; refusing split-store ambiguity")
	}
	switch field.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return field.Uint(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Int() < 0 {
			return 0, errors.New("filesystem identity is negative")
		}
		return uint64(field.Int()), nil
	default:
		return 0, errors.New("platform filesystem identity has an unsupported representation")
	}
}

type registryPreflight struct{}

func (registryPreflight) FetchManifest(ctx context.Context, repository, tag string) ([]byte, error) {
	segments := strings.Split(repository, "/")
	for i := range segments {
		segments[i] = url.PathEscape(segments[i])
	}
	endpoint := "https://" + RegistryHost + "/v2/" + strings.Join(segments, "/") + "/manifests/" + url.PathEscape(tag)
	client := registryHTTPClient(false)
	defer client.CloseIdleConnections()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json, application/vnd.oci.image.manifest.v1+json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry manifest request returned HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > maxManifestBytes {
		return nil, errors.New("registry manifest exceeds the bounded response size")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxManifestBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) == 0 || len(body) > maxManifestBytes {
		return nil, errors.New("registry manifest has empty or oversized body")
	}
	return body, nil
}

func (registryPreflight) ValidateBlobSize(ctx context.Context, repository, digest string, expectedSize int64) error {
	if !validDigest(digest) || expectedSize <= 0 {
		return errors.New("blob digest or expected size is invalid")
	}
	segments := strings.Split(repository, "/")
	for i := range segments {
		segments[i] = url.PathEscape(segments[i])
	}
	endpoint := "https://" + RegistryHost + "/v2/" + strings.Join(segments, "/") + "/blobs/" + url.PathEscape(digest)
	client := registryHTTPClient(true)
	defer client.CloseIdleConnections()
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("blob HEAD returned HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength != expectedSize {
		return fmt.Errorf("registry reports %d bytes; manifest declares %d", resp.ContentLength, expectedSize)
	}
	return nil
}

func registryHTTPClient(allowBlobStorageRedirect bool) *http.Client {
	client := &http.Client{
		Transport: &http.Transport{Proxy: nil, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 || req.URL.Scheme != "https" || req.URL.User != nil || req.URL.Fragment != "" {
				return http.ErrUseLastResponse
			}
			host := strings.ToLower(req.URL.Hostname())
			port := req.URL.Port()
			if port != "" && port != "443" {
				return http.ErrUseLastResponse
			}
			if host == RegistryHost {
				return nil
			}
			if allowBlobStorageRedirect && strings.HasSuffix(host, ".r2.cloudflarestorage.com") && strings.Count(host, ".") >= 3 {
				return nil
			}
			return http.ErrUseLastResponse
		},
	}
	return client
}
