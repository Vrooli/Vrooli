package models

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/packages/capacity/modelretention"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/ensure"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/policy"
)

const retentionResource = "ollama"

// RetentionDeps are the owner seams for `models retention|evict|reacquire`.
// Production wiring is defaultRetentionDeps; tests replace each seam.
type RetentionDeps struct {
	// PublicStore reads tags and residency through the public edge.
	PublicStore func() (*StoreClient, error)
	// ModelRoot is the managed model store (for manifest accounting).
	ModelRoot func() (string, error)
	// PrepareMutation holds the lifecycle lock and binds the private client.
	PrepareMutation func(context.Context) (ensure.ManagedModelMutation, error)
	Paths           func() (modelretention.Paths, error)
	LoadPolicy      func() (policy.Policy, error)
	// Ensure runs `resource-ollama ensure` with the given arguments.
	Ensure func([]string) error
	Now    func() time.Time
}

func (h *Handlers) retentionDeps() RetentionDeps {
	d := h.Retention
	if d.PublicStore == nil {
		d.PublicStore = func() (*StoreClient, error) {
			client, err := ensure.NewLoopbackClient()
			if err != nil {
				return nil, err
			}
			return &StoreClient{BaseURL: client.BaseURL, HTTP: client.HTTP}, nil
		}
	}
	if d.ModelRoot == nil {
		d.ModelRoot = ensure.EffectiveModelRoot
	}
	if d.PrepareMutation == nil {
		d.PrepareMutation = ensure.PrepareManagedModelMutation
	}
	if d.Paths == nil {
		d.Paths = func() (modelretention.Paths, error) { return modelretention.StorePaths(retentionResource) }
	}
	if d.LoadPolicy == nil {
		d.LoadPolicy = func() (policy.Policy, error) {
			p, _, err := h.loadPolicy()
			return p, err
		}
	}
	if d.Ensure == nil {
		d.Ensure = ensure.CommandGroup(nil).Commands[0].Run
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	return d
}

// StoreClient is the small tags/ps/delete surface retention needs, over the
// endpoint and transport an ensure client already verified.
type StoreClient struct {
	BaseURL string
	HTTP    *http.Client
}

type storeTag struct {
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	Digest     string    `json:"digest"`
	ModifiedAt time.Time `json:"modified_at"`
}

func (c *StoreClient) tags(ctx context.Context) ([]storeTag, error) {
	var out struct {
		Models []storeTag `json:"models"`
	}
	if err := c.do(ctx, http.MethodGet, "/api/tags", nil, &out); err != nil {
		return nil, err
	}
	return out.Models, nil
}

func (c *StoreClient) loaded(ctx context.Context) (map[string]bool, error) {
	var out struct {
		Models []struct {
			Name  string `json:"name"`
			Model string `json:"model"`
		} `json:"models"`
	}
	if err := c.do(ctx, http.MethodGet, "/api/ps", nil, &out); err != nil {
		return nil, err
	}
	loaded := map[string]bool{}
	for _, m := range out.Models {
		for _, name := range []string{m.Name, m.Model} {
			if key := modelretention.CanonicalName(name); key != "" {
				loaded[key] = true
			}
		}
	}
	return loaded, nil
}

// remove calls Ollama's delete, which accepts only the DELETE method.
func (c *StoreClient) remove(ctx context.Context, name string) error {
	body, _ := json.Marshal(map[string]string{"model": name})
	return c.do(ctx, http.MethodDelete, "/api/delete", body, nil)
}

func (c *StoreClient) do(ctx context.Context, method, path string, body []byte, out any) error {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.BaseURL, "/")+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := c.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ollama %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("ollama %s: HTTP %d: %s", path, resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func reacquireArgv(name string) []string {
	return []string{"resource-ollama", "models", "reacquire", name}
}

// Retention prints the store's retention view for storage-manager.
func (h *Handlers) RetentionInventory(args []string) error {
	fs := flag.NewFlagSet("models retention", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	asJSON := fs.Bool("json", false, "Emit the vrooli.model-store-retention/v1 document")
	if err := fs.Parse(args); err != nil {
		return err
	}
	deps := h.retentionDeps()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	inv, err := buildRetentionInventory(ctx, deps)
	if err != nil {
		return err
	}
	if *asJSON {
		return writeJSON(h.Stdout, inv)
	}
	for _, m := range inv.Models {
		state := "eligible"
		if m.Required {
			state = "required (" + m.RequiredReason + ")"
		}
		fmt.Fprintf(h.Stdout, "%s\t%d\tunique=%d\tloaded=%t\t%s\n", m.Name, m.Bytes, m.UniqueBytes, m.Loaded, state)
	}
	return nil
}

func buildRetentionInventory(ctx context.Context, deps RetentionDeps) (modelretention.Inventory, error) {
	p, err := deps.LoadPolicy()
	if err != nil {
		return modelretention.Inventory{}, err
	}
	store, err := deps.PublicStore()
	if err != nil {
		return modelretention.Inventory{}, fmt.Errorf("verify local Ollama endpoint: %w", err)
	}
	tags, err := store.tags(ctx)
	if err != nil {
		return modelretention.Inventory{}, err
	}
	loaded, err := store.loaded(ctx)
	if err != nil {
		return modelretention.Inventory{}, err
	}
	root, err := deps.ModelRoot()
	if err != nil {
		return modelretention.Inventory{}, err
	}
	footprint, err := modelretention.ReadFootprint(root, ensure.RegistryHost)
	if err != nil {
		return modelretention.Inventory{}, err
	}
	required := canonicalRequired(p)
	inv := modelretention.Inventory{Schema: modelretention.InventorySchema, Resource: retentionResource, ModelRoot: root, StoreBytes: footprint.StoreBytes, ObservedAt: deps.Now().UTC()}
	for _, tag := range tags {
		key := modelretention.CanonicalName(tag.Name)
		fp, ok := footprint.Models[key]
		unique := fp.UniqueBytes
		if !ok {
			unique = 0
		}
		reason, isRequired := required[key]
		inv.Models = append(inv.Models, modelretention.InventoryModel{
			Name: tag.Name, Digest: tag.Digest, Bytes: tag.Size, UniqueBytes: unique, ModifiedAt: tag.ModifiedAt,
			Loaded: loaded[key], Required: isRequired, RequiredReason: reason,
			Reacquire: reacquireArgv(tag.Name), ReacquireExact: false,
		})
	}
	sort.Slice(inv.Models, func(i, j int) bool { return inv.Models[i].Name < inv.Models[j].Name })
	return inv, nil
}

func canonicalRequired(p policy.Policy) map[string]string {
	out := map[string]string{}
	for model, reason := range p.RequiredModels() {
		out[modelretention.CanonicalName(model)] = reason
	}
	return out
}

// errRefused exits non-zero after the JSON result already named the refusal.
var errRefused = errors.New("eviction refused")

// Evict is the one fenced eviction operation for the Ollama store.
func (h *Handlers) Evict(args []string) error {
	fs := flag.NewFlagSet("models evict", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	digest := fs.String("digest", "", "Installed manifest digest the decision saw (required)")
	size := fs.Int64("size", -1, "Installed size in bytes the decision saw (required)")
	reason := fs.String("reason", "", "Why the model is evicted (recorded in the receipt)")
	requestedBy := fs.String("requested-by", "", "Who asked (recorded in the receipt)")
	asJSON := fs.Bool("json", false, "Emit JSON")
	name, rest := splitPositional(args)
	if err := fs.Parse(rest); err != nil {
		return err
	}
	if name == "" && fs.NArg() > 0 {
		name = fs.Arg(0)
	}
	if strings.TrimSpace(name) == "" || strings.TrimSpace(*digest) == "" || *size < 0 {
		return errors.New("usage: resource-ollama models evict <name> --digest <digest> --size <bytes>")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	result, err := evictModel(ctx, h.retentionDeps(), evictRequest{Name: name, Digest: *digest, Size: *size, Reason: *reason, RequestedBy: *requestedBy})
	if *asJSON {
		if writeErr := writeJSON(h.Stdout, result); writeErr != nil {
			return writeErr
		}
	} else if result.Refused != "" {
		fmt.Fprintf(h.Stdout, "refused: %s\n", result.Refused)
	} else if result.Evicted {
		fmt.Fprintf(h.Stdout, "evicted %s (%d bytes freed); reacquire with: %s\n", result.Model, result.FreedBytes, strings.Join(result.Reacquire, " "))
	}
	if err != nil {
		return err
	}
	if result.Refused != "" {
		return errRefused
	}
	return nil
}

type evictRequest struct {
	Name, Digest string
	Size         int64
	Reason       string
	RequestedBy  string
}

func evictModel(ctx context.Context, deps RetentionDeps, req evictRequest) (modelretention.EvictResult, error) {
	key := modelretention.CanonicalName(req.Name)
	result := modelretention.EvictResult{Resource: retentionResource, Model: req.Name, Digest: req.Digest, Reacquire: reacquireArgv(req.Name)}
	refuse := func(format string, args ...any) (modelretention.EvictResult, error) {
		result.Refused = fmt.Sprintf(format, args...)
		return result, nil
	}
	paths, err := deps.Paths()
	if err != nil {
		return result, err
	}
	result.ReceiptsFile = paths.ReceiptsFile
	// The fence first: from here no request for the model can start, and one
	// already in flight refuses the eviction.
	releaseFence, err := modelretention.Fence{Dir: paths.FenceDir}.AcquireEviction(key)
	if errors.Is(err, modelretention.ErrModelInUse) {
		return refuse("%v", err)
	}
	if err != nil {
		return result, err
	}
	defer releaseFence()
	// An early pass through the public edge refuses a doomed eviction without
	// taking the lifecycle lock; the pass under the lock below is authoritative.
	public, err := deps.PublicStore()
	if err != nil {
		return result, err
	}
	if refusal, _, err := checkEvictable(ctx, public, deps, key, req); err != nil || refusal != "" {
		if err != nil {
			return result, err
		}
		return refuse("%s", refusal)
	}
	mutation, err := deps.PrepareMutation(ctx)
	if err != nil {
		return result, err
	}
	defer mutation.Release()
	if err := mutation.Verify(ctx); err != nil {
		return result, err
	}
	store := &StoreClient{BaseURL: mutation.Client.BaseURL, HTTP: mutation.Client.HTTP}
	refusal, installed, err := checkEvictable(ctx, store, deps, key, req)
	if err != nil {
		return result, err
	}
	if installed != nil {
		result.Model = installed.Name
		result.Reacquire = reacquireArgv(installed.Name)
	}
	if refusal != "" {
		return refuse("%s", refusal)
	}
	var unique int64
	if fp, err := modelretention.ReadFootprint(mutation.ModelRoot, ensure.RegistryHost); err == nil {
		unique = fp.Models[key].UniqueBytes
	}
	usage, _ := modelretention.ReadUsage(paths.UsageFile)
	receipt := modelretention.Receipt{
		Resource: retentionResource, Model: installed.Name, Digest: installed.Digest, Bytes: installed.Size, UniqueBytes: unique,
		Status: modelretention.ReceiptEvicting, At: deps.Now().UTC(), LastRequestAt: usage.Models[key].LastRequestAt,
		Reacquire: result.Reacquire, ReacquireExact: false, Reason: req.Reason, RequestedBy: req.RequestedBy,
	}
	if err := modelretention.AppendReceipt(ctx, paths.ReceiptsFile, receipt); err != nil {
		return result, fmt.Errorf("record reacquisition receipt before deleting: %w", err)
	}
	if err := mutation.Verify(ctx); err != nil {
		return result, err
	}
	deleteErr := store.remove(ctx, installed.Name)
	after, listErr := store.tags(ctx)
	gone := listErr == nil
	for _, tag := range after {
		if modelretention.CanonicalName(tag.Name) == key {
			gone = false
		}
	}
	receipt.At = deps.Now().UTC()
	if deleteErr != nil || !gone {
		receipt.Status = modelretention.ReceiptFailed
		receipt.Error = errors.Join(deleteErr, listErr).Error()
		if receipt.Error == "" {
			receipt.Error = "model still listed after delete"
		}
		_ = modelretention.AppendReceipt(ctx, paths.ReceiptsFile, receipt)
		return result, fmt.Errorf("delete %s: %s", installed.Name, receipt.Error)
	}
	receipt.Status = modelretention.ReceiptEvicted
	if err := modelretention.AppendReceipt(ctx, paths.ReceiptsFile, receipt); err != nil {
		return result, fmt.Errorf("model deleted but completion receipt failed: %w", err)
	}
	result.Evicted = true
	result.FreedBytes = unique
	return result, nil
}

// checkEvictable applies every refusal in a fixed order: residency (re-read
// now), the policy (re-read now, so a role reassignment wins), then identity.
func checkEvictable(ctx context.Context, store *StoreClient, deps RetentionDeps, key string, req evictRequest) (string, *storeTag, error) {
	loaded, err := store.loaded(ctx)
	if err != nil {
		return "", nil, err
	}
	if loaded[key] {
		return "model is loaded in memory", nil, nil
	}
	p, err := deps.LoadPolicy()
	if err != nil {
		return "", nil, err
	}
	if reason, ok := canonicalRequired(p)[key]; ok {
		return "model is required: " + reason, nil, nil
	}
	tags, err := store.tags(ctx)
	if err != nil {
		return "", nil, err
	}
	for i := range tags {
		if modelretention.CanonicalName(tags[i].Name) != key {
			continue
		}
		installed := tags[i]
		if installed.Digest != req.Digest || installed.Size != req.Size {
			return fmt.Sprintf("model identity changed since the decision: installed digest %s size %d, decided digest %s size %d", installed.Digest, installed.Size, req.Digest, req.Size), &installed, nil
		}
		return "", &installed, nil
	}
	return "model is not installed", nil, nil
}

// Reacquire pulls an evicted model back through the admitted ensure path.
func (h *Handlers) Reacquire(args []string) error {
	fs := flag.NewFlagSet("models reacquire", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	timeout := fs.Int("timeout-seconds", 0, "Overall pull deadline in seconds (0 means ensure's default)")
	name, rest := splitPositional(args)
	if err := fs.Parse(rest); err != nil {
		return err
	}
	if name == "" && fs.NArg() > 0 {
		name = fs.Arg(0)
	}
	if strings.TrimSpace(name) == "" {
		return errors.New("usage: resource-ollama models reacquire <name>")
	}
	deps := h.retentionDeps()
	ensureArgs, err := reacquireEnsureArgs(name, deps.Now(), *timeout)
	if err != nil {
		return err
	}
	if paths, err := deps.Paths(); err == nil {
		if receipt, ok, _ := modelretention.LatestReceipt(paths.ReceiptsFile, name); ok {
			fmt.Fprintf(h.Stderr, "reacquiring %s (evicted %s at %s, %d bytes; the registry tag may now resolve to a newer digest than %s)\n", name, receipt.Status, receipt.At.Format(time.RFC3339), receipt.Bytes, receipt.Digest)
		}
	}
	return deps.Ensure(ensureArgs)
}

func reacquireEnsureArgs(name string, now time.Time, timeoutSeconds int) ([]string, error) {
	config := map[string]any{"models": []map[string]any{{
		"name":         strings.TrimSpace(name),
		"reason":       "reacquire a model evicted from disk by storage-manager",
		"owner":        "storage-manager",
		"review_after": now.AddDate(0, 3, 0).Format("2006-01-02"),
	}}}
	raw, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	args := []string{"--config-base64", base64.StdEncoding.EncodeToString(raw)}
	if timeoutSeconds > 0 {
		args = append(args, "--timeout-seconds", strconv.Itoa(timeoutSeconds))
	}
	return args, nil
}

// splitPositional lets the model name come before the flags.
func splitPositional(args []string) (string, []string) {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		return args[0], args[1:]
	}
	return "", args
}
