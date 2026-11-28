package commands

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/packages/capacity/modelretention"
	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/policy"
	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/upstream"
)

const (
	retentionResource = "ollaya"
	ollayaRegistry    = "ollaya.dev"
)

// Every Ollaya model in model-policy.json is pinned by manifest digest and
// `models pull --model <name>` reproduces exactly that digest, so a pinned
// model whose installed digest matches is regenerable. Default pulls (and the
// targets a default router needs) are required; opt-in and bake-off models
// are eviction-eligible when idle.

func (h *Handlers) retentionPaths() (modelretention.Paths, error) {
	if h.RetentionPaths != nil {
		return h.RetentionPaths()
	}
	return modelretention.StorePaths(retentionResource)
}

func (h *Handlers) retentionModelRoot() (string, error) {
	if h.ModelRoot != nil {
		return h.ModelRoot()
	}
	return ollayaModelRoot(h.GetEnv)
}

// requiredModels maps installed names disk eviction must keep to why.
func requiredModels(p policy.Policy) map[string]string {
	out := map[string]string{}
	for _, name := range p.DefaultPulls() {
		out[modelretention.CanonicalName(upstream.InstalledName(name))] = "default pull"
		for _, route := range p.Models[name].Routes {
			key := modelretention.CanonicalName(upstream.InstalledName(route))
			if _, ok := out[key]; !ok {
				out[key] = "route target of default pull " + name
			}
		}
	}
	return out
}

// pinnedName finds the policy name whose installed name is installed.
func pinnedName(p policy.Policy, installed string) (string, policy.Model, bool) {
	key := modelretention.CanonicalName(installed)
	for name, model := range p.Models {
		if modelretention.CanonicalName(upstream.InstalledName(name)) == key {
			return name, model, true
		}
	}
	return "", policy.Model{}, false
}

func reacquireArgv(policyName string) []string {
	return []string{"resource-ollaya", "models", "pull", "--model", policyName}
}

// evictionBlock is why the owner refuses a non-required model outright.
func evictionBlock(p policy.Policy, installed upstream.InstalledModel) (string, []string) {
	name, model, ok := pinnedName(p, installed.Name)
	if !ok {
		return "not pinned in model-policy.json, so no command reproduces it", nil
	}
	if model.Digest != installed.Digest {
		return fmt.Sprintf("installed digest %s differs from the pinned digest %s, so a pull would not reproduce it", installed.Digest, model.Digest), reacquireArgv(name)
	}
	return "", reacquireArgv(name)
}

// RetentionInventory prints the store's retention view for storage-manager.
func (h *Handlers) RetentionInventory(args []string) error {
	fs := flag.NewFlagSet("models retention", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	jsonOutput := fs.Bool("json", false, "Emit the vrooli.model-store-retention/v1 document")
	if err := fs.Parse(args); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	inv, err := h.retentionInventory(ctx)
	if err != nil {
		return h.fail(err)
	}
	if *jsonOutput {
		return writeJSON(h.Stdout, inv)
	}
	for _, m := range inv.Models {
		state := "eligible"
		if m.Required {
			state = "required (" + m.RequiredReason + ")"
		} else if m.Blocked != "" {
			state = "blocked (" + m.Blocked + ")"
		}
		fmt.Fprintf(h.Stdout, "%s\t%d\tunique=%d\tloaded=%t\t%s\n", m.Name, m.Bytes, m.UniqueBytes, m.Loaded, state)
	}
	return nil
}

func (h *Handlers) retentionInventory(ctx context.Context) (modelretention.Inventory, error) {
	p, err := h.policy()
	if err != nil {
		return modelretention.Inventory{}, err
	}
	client := h.client()
	installed, err := client.Installed(ctx)
	if err != nil {
		return modelretention.Inventory{}, err
	}
	residents, err := client.Residents(ctx)
	if err != nil {
		return modelretention.Inventory{}, err
	}
	loaded := map[string]bool{}
	for _, r := range residents {
		loaded[modelretention.CanonicalName(r.Name)] = true
	}
	root, err := h.retentionModelRoot()
	if err != nil {
		return modelretention.Inventory{}, err
	}
	footprint, err := modelretention.ReadFootprint(root, ollayaRegistry)
	if err != nil {
		return modelretention.Inventory{}, err
	}
	required := requiredModels(p)
	now := time.Now
	if h.Now != nil {
		now = h.Now
	}
	inv := modelretention.Inventory{Schema: modelretention.InventorySchema, Resource: retentionResource, ModelRoot: root, StoreBytes: footprint.StoreBytes, ObservedAt: now().UTC()}
	for _, m := range installed {
		key := modelretention.CanonicalName(m.Name)
		reason, isRequired := required[key]
		blocked, reacquire := evictionBlock(p, m)
		inv.Models = append(inv.Models, modelretention.InventoryModel{
			Name: m.Name, Digest: m.Digest, Bytes: m.Size, UniqueBytes: footprint.Models[key].UniqueBytes, ModifiedAt: m.ModifiedAt,
			Loaded: loaded[key], Required: isRequired, RequiredReason: reason,
			Reacquire: reacquire, ReacquireExact: blocked == "", Blocked: blocked,
		})
	}
	sort.Slice(inv.Models, func(i, j int) bool { return inv.Models[i].Name < inv.Models[j].Name })
	return inv, nil
}

var errEvictionRefused = errors.New("eviction refused")

// EvictModel is the one fenced eviction operation for the Ollaya store.
func (h *Handlers) EvictModel(args []string) error {
	fs := flag.NewFlagSet("models evict", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	digest := fs.String("digest", "", "Installed manifest digest the decision saw (required)")
	size := fs.Int64("size", -1, "Installed size in bytes the decision saw (required)")
	reason := fs.String("reason", "", "Why the model is evicted (recorded in the receipt)")
	requestedBy := fs.String("requested-by", "", "Who asked (recorded in the receipt)")
	jsonOutput := fs.Bool("json", false, "Emit JSON")
	var name string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		name, args = args[0], args[1:]
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if name == "" && fs.NArg() > 0 {
		name = fs.Arg(0)
	}
	if strings.TrimSpace(name) == "" || strings.TrimSpace(*digest) == "" || *size < 0 {
		return errors.New("usage: resource-ollaya models evict <name> --digest <digest> --size <bytes>")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	result, err := h.evict(ctx, name, *digest, *size, *reason, *requestedBy)
	if *jsonOutput {
		if writeErr := writeJSON(h.Stdout, result); writeErr != nil {
			return writeErr
		}
	} else if result.Refused != "" {
		fmt.Fprintf(h.Stdout, "refused: %s\n", result.Refused)
	} else if result.Evicted {
		fmt.Fprintf(h.Stdout, "evicted %s (%d bytes freed); reacquire with: %s\n", result.Model, result.FreedBytes, strings.Join(result.Reacquire, " "))
	}
	if err != nil {
		return h.fail(err)
	}
	if result.Refused != "" {
		return errEvictionRefused
	}
	return nil
}

// checkEvictable applies every refusal in a fixed order: residency (re-read
// now), the policy (re-read now, so a route or default-pull change wins),
// identity, then exact reacquisition. It fills result's model and reacquire.
func (h *Handlers) checkEvictable(ctx context.Context, client *upstream.Client, key, digest string, size int64, result *modelretention.EvictResult) (string, *upstream.InstalledModel, error) {
	residents, err := client.Residents(ctx)
	if err != nil {
		return "", nil, err
	}
	for _, r := range residents {
		if modelretention.CanonicalName(r.Name) == key {
			return "model is loaded in memory", nil, nil
		}
	}
	p, err := h.policy()
	if err != nil {
		return "", nil, err
	}
	if why, ok := requiredModels(p)[key]; ok {
		return "model is required: " + why, nil, nil
	}
	installed, err := client.Installed(ctx)
	if err != nil {
		return "", nil, err
	}
	for i := range installed {
		if modelretention.CanonicalName(installed[i].Name) != key {
			continue
		}
		target := installed[i]
		result.Model = target.Name
		if target.Digest != digest || target.Size != size {
			return fmt.Sprintf("model identity changed since the decision: installed digest %s size %d, decided digest %s size %d", target.Digest, target.Size, digest, size), &target, nil
		}
		blocked, reacquire := evictionBlock(p, target)
		result.Reacquire = reacquire
		result.ReacquireExact = blocked == ""
		if blocked != "" {
			return "model cannot be reacquired exactly: " + blocked, &target, nil
		}
		return "", &target, nil
	}
	return "model is not installed", nil, nil
}

func (h *Handlers) evict(ctx context.Context, name, digest string, size int64, reason, requestedBy string) (modelretention.EvictResult, error) {
	key := modelretention.CanonicalName(name)
	result := modelretention.EvictResult{Resource: retentionResource, Model: name, Digest: digest}
	refuse := func(format string, args ...any) (modelretention.EvictResult, error) {
		result.Refused = fmt.Sprintf(format, args...)
		return result, nil
	}
	paths, err := h.retentionPaths()
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
	// An early pass through the public endpoint refuses a doomed eviction
	// without taking the lifecycle lock; the pass under the lock is authoritative.
	public := h.client()
	if refusal, _, err := h.checkEvictable(ctx, &public, key, digest, size, &result); err != nil || refusal != "" {
		if err != nil {
			return result, err
		}
		return refuse("%s", refusal)
	}
	var root string
	var client *upstream.Client
	var verify func(context.Context) error
	var release func()
	if h.PrepareModelPull != nil {
		root, client, verify, release, err = h.PrepareModelPull(ctx)
	} else {
		root, client, verify, release, err = prepareManagedModelPull(ctx, h.GetEnv)
	}
	if err != nil {
		return result, err
	}
	if release != nil {
		defer release()
	}
	if client == nil || verify == nil {
		return result, errors.New("managed Ollaya owner is incomplete; refusing daemon access")
	}
	if err := verify(ctx); err != nil {
		return result, err
	}
	refusal, target, err := h.checkEvictable(ctx, client, key, digest, size, &result)
	if err != nil {
		return result, err
	}
	if refusal != "" {
		return refuse("%s", refusal)
	}
	var unique int64
	if fp, err := modelretention.ReadFootprint(root, ollayaRegistry); err == nil {
		unique = fp.Models[key].UniqueBytes
	}
	usage, _ := modelretention.ReadUsage(paths.UsageFile)
	now := time.Now
	if h.Now != nil {
		now = h.Now
	}
	receipt := modelretention.Receipt{
		Resource: retentionResource, Model: target.Name, Digest: target.Digest, Bytes: target.Size, UniqueBytes: unique,
		Status: modelretention.ReceiptEvicting, At: now().UTC(), LastRequestAt: usage.Models[key].LastRequestAt,
		Reacquire: result.Reacquire, ReacquireExact: true, Reason: reason, RequestedBy: requestedBy,
	}
	if err := modelretention.AppendReceipt(ctx, paths.ReceiptsFile, receipt); err != nil {
		return result, fmt.Errorf("record reacquisition receipt before deleting: %w", err)
	}
	if err := verify(ctx); err != nil {
		return result, err
	}
	deleteErr := client.Delete(ctx, target.Name)
	after, listErr := client.Installed(ctx)
	gone := listErr == nil
	for _, m := range after {
		if modelretention.CanonicalName(m.Name) == key {
			gone = false
		}
	}
	receipt.At = now().UTC()
	if deleteErr != nil || !gone {
		receipt.Status = modelretention.ReceiptFailed
		if joined := errors.Join(deleteErr, listErr); joined != nil {
			receipt.Error = joined.Error()
		} else {
			receipt.Error = "model still listed after delete"
		}
		_ = modelretention.AppendReceipt(ctx, paths.ReceiptsFile, receipt)
		return result, fmt.Errorf("delete %s: %s", target.Name, receipt.Error)
	}
	receipt.Status = modelretention.ReceiptEvicted
	if err := modelretention.AppendReceipt(ctx, paths.ReceiptsFile, receipt); err != nil {
		return result, fmt.Errorf("model deleted but completion receipt failed: %w", err)
	}
	result.Evicted = true
	result.FreedBytes = unique
	return result, nil
}
