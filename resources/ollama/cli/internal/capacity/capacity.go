// Package capacity implements `resource-ollama capacity plan`.
package capacity

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/packages/capacity/companion"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/ensure"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/policy"
)

const (
	defaultRuntimeMemoryLimitGB = 12
	bytesPerGB                  = 1024 * 1024 * 1024
)

type HostCollector interface {
	Collect(ctx context.Context) (hostinventory.Snapshot, error)
}

type OllamaClient interface {
	ListTags(ctx context.Context) (map[string]bool, error)
	ListRunning(ctx context.Context) ([]ensure.RunningModel, error)
	Unload(ctx context.Context, model string) error
}

type systemHostCollector struct{}

func (systemHostCollector) Collect(ctx context.Context) (hostinventory.Snapshot, error) {
	return hostinventory.Collect(ctx)
}

type Handlers struct {
	NewClient   func() OllamaClient
	Host        HostCollector
	GetEnv      func(string) string
	Stdout      io.Writer
	Stderr      io.Writer
	SourceRoot  string
	PolicyPath  string
	RuntimePath string
}

func Default() *Handlers {
	h := &Handlers{
		Host:   systemHostCollector{},
		GetEnv: os.Getenv,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	h.NewClient = func() OllamaClient { return ensure.NewClient() }
	return h
}

func Commands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default()
	}
	return cliapp.SubcommandGroup{
		Name:        "capacity",
		Description: "Plan Ollama model demand against host and runtime capacity",
		Subcommands: append([]cliapp.Command{
			{
				Name:        "plan",
				Description: "Estimate scenario Ollama model demand and residency pressure",
				Usage:       "resource-ollama capacity plan --scenario <name> [--all-scenarios] [--json]",
				Run:         h.Plan,
			},
		}, companion.CapacitySubcommands("ollama", []string{"qwen3.5:9b", "qwen3.5:4b", "qwen3:4b", "qwen3:1.7b"},
			func(_ context.Context, label string) error { return h.Degrade([]string{"--to", label}) },
			func(_ context.Context, label string) error { return h.Degrade([]string{"--to", label}) }, h.Stdout, h.Stderr)...),
	}
}

// degradeResult is the JSON envelope of a degrade actuation.
type degradeResult struct {
	Unloaded       string   `json:"unloaded,omitempty"`
	UnloadedModels []string `json:"unloaded_models,omitempty"`
	FreedBytes     int64    `json:"freed_bytes"`
	Remaining      int      `json:"remaining_loaded"`
	Message        string   `json:"message"`
}

// Degrade unloads the Nth-largest loaded model (default the single largest) to
// free VRAM for a higher-priority workload — the ollama side of the broker's
// degrade rung (§8.9). It is idempotent: with no models loaded it reports a
// no-op rather than erroring, so a repeated degrade never fails.
func (h *Handlers) Degrade(args []string) error {
	fs := flag.NewFlagSet("degrade", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	nth := fs.Int("nth", 1, "unload the Nth-largest loaded model (1 = largest)")
	jsonOut := fs.Bool("json", false, "emit JSON")
	target := fs.String("to", "", "degrade to a model-policy capacity rung")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *nth < 1 {
		return fmt.Errorf("--nth must be >= 1, got %d", *nth)
	}

	ctx := context.Background()
	newClient := h.NewClient
	if newClient == nil {
		newClient = func() OllamaClient { return ensure.NewClient() }
	}
	client := newClient()
	running, err := client.ListRunning(ctx)
	if err != nil {
		return fmt.Errorf("list running models: %w", err)
	}

	result := degradeResult{Remaining: len(running)}
	if len(running) == 0 {
		result.Message = "no models loaded; degrade is a no-op"
		return h.writeDegrade(*jsonOut, result)
	}
	// Largest VRAM footprint first, so the default unloads what frees the most.
	sort.SliceStable(running, func(i, j int) bool { return running[i].SizeVRAM > running[j].SizeVRAM })
	if strings.TrimSpace(*target) != "" {
		targetBytes, err := h.targetBytes(strings.TrimSpace(*target))
		if err != nil {
			return err
		}
		var total int64
		for _, model := range running {
			total += model.SizeVRAM
		}
		for total > targetBytes && len(running) > 0 {
			model := running[0]
			if err := client.Unload(ctx, model.Name); err != nil {
				return fmt.Errorf("unload %s: %w", model.Name, err)
			}
			result.UnloadedModels = append(result.UnloadedModels, model.Name)
			result.FreedBytes += model.SizeVRAM
			total -= model.SizeVRAM
			running = running[1:]
		}
		result.Remaining = len(running)
		result.Message = fmt.Sprintf("degraded to %s (%d model(s) unloaded)", *target, len(result.UnloadedModels))
		return h.writeDegrade(*jsonOut, result)
	}
	if *nth > len(running) {
		result.Message = fmt.Sprintf("only %d model(s) loaded; nothing at position %d to unload", len(running), *nth)
		return h.writeDegrade(*jsonOut, result)
	}
	modelTarget := running[*nth-1]
	if err := client.Unload(ctx, modelTarget.Name); err != nil {
		return fmt.Errorf("unload %s: %w", modelTarget.Name, err)
	}
	result.Unloaded = modelTarget.Name
	result.UnloadedModels = []string{modelTarget.Name}
	result.FreedBytes = modelTarget.SizeVRAM
	result.Remaining = len(running) - 1
	result.Message = fmt.Sprintf("unloaded %s (freed %d bytes VRAM)", modelTarget.Name, modelTarget.SizeVRAM)
	return h.writeDegrade(*jsonOut, result)
}

func (h *Handlers) targetBytes(label string) (int64, error) {
	getenv := h.GetEnv
	if getenv == nil {
		getenv = os.Getenv
	}
	var p policy.Policy
	var err error
	if strings.TrimSpace(h.PolicyPath) != "" {
		p, err = policy.LoadFile(h.PolicyPath)
	} else {
		p, _, err = policy.LoadDefaultFile(getenv)
	}
	if err != nil {
		return 0, fmt.Errorf("load model policy for capacity rung %q: %w", label, err)
	}
	model, ok := p.Models[label]
	if !ok || model.VRAMGBEstimate <= 0 {
		return 0, fmt.Errorf("capacity rung %q is not a model with a positive VRAM estimate", label)
	}
	return int64(model.VRAMGBEstimate * bytesPerGB), nil
}

func (h *Handlers) writeDegrade(jsonOut bool, result degradeResult) error {
	if jsonOut {
		return cliout.NewEncoder(h.Stdout).Encode(result)
	}
	_, err := fmt.Fprintln(h.Stdout, result.Message)
	return err
}

func (h *Handlers) Plan(args []string) error {
	fs := flag.NewFlagSet("capacity plan", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	scenario := fs.String("scenario", "", "Scenario name to inspect under scenarios/<name>")
	allScenarios := fs.Bool("all-scenarios", false, "Inspect every scenarios/*/.vrooli/service.json manifest")
	asJSON := fs.Bool("json", false, "Emit a machine-readable JSON report")
	root := fs.String("root", "", "Vrooli source root; defaults to VROOLI_SOURCE_ROOT or auto-detection")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*scenario) == "" && !*allScenarios {
		return errors.New("--scenario or --all-scenarios is required")
	}
	if strings.TrimSpace(*scenario) != "" && *allScenarios {
		return errors.New("--scenario and --all-scenarios are mutually exclusive")
	}

	req := PlanRequest{
		Scenario:     strings.TrimSpace(*scenario),
		AllScenarios: *allScenarios,
		SourceRoot:   strings.TrimSpace(*root),
	}
	report, err := h.BuildReport(context.Background(), req)
	if err != nil {
		return err
	}
	if *asJSON {
		if err := cliout.NewEncoder(h.Stdout).Encode(report); err != nil {
			return err
		}
	} else {
		renderText(h.Stdout, report)
	}
	if len(report.Failures) > 0 {
		return fmt.Errorf("capacity plan found %d failure(s)", len(report.Failures))
	}
	return nil
}

type PlanRequest struct {
	Scenario     string
	AllScenarios bool
	SourceRoot   string
}

type Report struct {
	Scenarios       []ScenarioDemand `json:"scenarios"`
	Models          []ModelDemand    `json:"models"`
	Totals          Totals           `json:"totals"`
	Runtime         RuntimeSettings  `json:"runtime"`
	Host            HostSummary      `json:"host"`
	Ollama          OllamaState      `json:"ollama"`
	Warnings        []string         `json:"warnings,omitempty"`
	Failures        []string         `json:"failures,omitempty"`
	PolicyPath      string           `json:"policy_path"`
	SourceRoot      string           `json:"source_root"`
	ManifestPattern string           `json:"manifest_pattern"`
}

type ScenarioDemand struct {
	Name       string   `json:"name"`
	Manifest   string   `json:"manifest"`
	Roles      []string `json:"roles,omitempty"`
	DirectRefs []string `json:"direct_refs,omitempty"`
	Models     []string `json:"models"`
	Warnings   []string `json:"warnings,omitempty"`
}

type ModelDemand struct {
	Ref                 string   `json:"ref"`
	Scenarios           []string `json:"scenarios"`
	Roles               []string `json:"roles,omitempty"`
	Direct              bool     `json:"direct"`
	Capabilities        []string `json:"capabilities,omitempty"`
	DiskGBEstimate      float64  `json:"disk_gb_estimate,omitempty"`
	RAMGBEstimate       float64  `json:"ram_gb_estimate,omitempty"`
	VRAMGBEstimate      float64  `json:"vram_gb_estimate,omitempty"`
	CapacityConfidence  string   `json:"capacity_confidence,omitempty"`
	CapacitySourceKind  string   `json:"capacity_source_kind,omitempty"`
	CapacitySource      string   `json:"capacity_source,omitempty"`
	Installed           bool     `json:"installed"`
	Loaded              bool     `json:"loaded"`
	LoadedSizeVRAMBytes int64    `json:"loaded_size_vram_bytes,omitempty"`
	Warnings            []string `json:"warnings,omitempty"`
}

type Totals struct {
	DistinctModels      int     `json:"distinct_models"`
	EstimatedDiskGB     float64 `json:"estimated_disk_gb"`
	EstimatedRAMGB      float64 `json:"estimated_ram_gb"`
	EstimatedVRAMGB     float64 `json:"estimated_vram_gb"`
	ResidentRAMGB       float64 `json:"resident_ram_gb"`
	ResidentVRAMGB      float64 `json:"resident_vram_gb"`
	RuntimeMemoryBudget float64 `json:"runtime_memory_budget_gb"`
	HostMemoryBudgetGB  float64 `json:"host_memory_budget_gb,omitempty"`
	GPUBudgetGB         float64 `json:"gpu_budget_gb,omitempty"`
}

type RuntimeSettings struct {
	MemoryLimitGB   float64 `json:"memory_limit_gb"`
	NumParallel     int     `json:"num_parallel"`
	MaxLoadedModels int     `json:"max_loaded_models"`
}

type HostSummary struct {
	OS              string   `json:"os"`
	Arch            string   `json:"arch"`
	CPUCores        int      `json:"cpu_cores"`
	MemoryTotalGB   float64  `json:"memory_total_gb,omitempty"`
	MemoryAvailGB   float64  `json:"memory_available_gb,omitempty"`
	GPUCount        int      `json:"gpu_count"`
	MaxGPUVRAMGB    float64  `json:"max_gpu_vram_gb,omitempty"`
	DockerNvidia    bool     `json:"docker_nvidia_runtime"`
	ProbeWarnings   []string `json:"probe_warnings,omitempty"`
	ProbeStatusNote string   `json:"probe_status_note,omitempty"`
}

type OllamaState struct {
	TagsAvailable bool                  `json:"tags_available"`
	PSAvailable   bool                  `json:"ps_available"`
	Processor     ensure.ProcessorState `json:"processor"`
	Installed     []string              `json:"installed,omitempty"`
	Loaded        []string              `json:"loaded,omitempty"`
	Warnings      []string              `json:"warnings,omitempty"`
}

type manifestFile struct {
	Dependencies struct {
		Resources map[string]resourceDependency `json:"resources"`
	} `json:"dependencies"`
}

type resourceDependency struct {
	Type    string          `json:"type"`
	Enabled *bool           `json:"enabled,omitempty"`
	Raw     json.RawMessage `json:"-"`
}

func (r *resourceDependency) UnmarshalJSON(data []byte) error {
	type alias resourceDependency
	var aux alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*r = resourceDependency(aux)
	r.Raw = append(r.Raw[:0], data...)
	return nil
}

func (h *Handlers) BuildReport(ctx context.Context, req PlanRequest) (Report, error) {
	root, err := h.sourceRoot(req.SourceRoot)
	if err != nil {
		return Report{}, err
	}
	p, policyPath, err := h.loadPolicy()
	if err != nil {
		return Report{}, err
	}
	manifests, pattern, err := manifestPaths(root, req)
	if err != nil {
		return Report{}, err
	}
	if len(manifests) == 0 {
		return Report{}, fmt.Errorf("no scenario manifests matched %s", pattern)
	}

	runtimeSettings, runtimeWarnings := h.runtimeSettings(root)
	snap, hostWarnings := h.hostSnapshot(ctx)
	ollamaState := h.ollamaState(ctx)

	report := Report{
		Runtime:         runtimeSettings,
		Host:            summarizeHost(snap),
		Ollama:          ollamaState,
		Warnings:        append([]string{}, runtimeWarnings...),
		PolicyPath:      policyPath,
		SourceRoot:      root,
		ManifestPattern: pattern,
	}
	report.Warnings = append(report.Warnings, hostWarnings...)
	report.Host.ProbeWarnings = append(report.Host.ProbeWarnings, snap.Warnings...)

	acc := map[string]*ModelDemand{}
	for _, manifest := range manifests {
		demand, err := readScenarioDemand(manifest, p)
		if err != nil {
			report.Warnings = append(report.Warnings, err.Error())
			continue
		}
		if len(demand.Models) == 0 && len(demand.Warnings) == 0 {
			continue
		}
		report.Scenarios = append(report.Scenarios, demand)
		report.Warnings = append(report.Warnings, demand.Warnings...)
		for _, ref := range demand.Models {
			entry := acc[ref]
			if entry == nil {
				entry = &ModelDemand{Ref: ref}
				acc[ref] = entry
			}
			entry.Scenarios = addUnique(entry.Scenarios, demand.Name)
			if contains(demand.DirectRefs, ref) {
				entry.Direct = true
			}
			for _, role := range demand.Roles {
				if roleModel, ok := p.Roles[role]; ok && roleModel.Model == ref {
					entry.Roles = addUnique(entry.Roles, role)
				}
			}
		}
	}
	report.Models = flattenModels(acc, p, ollamaState)
	report.Totals = calculateTotals(report.Models, runtimeSettings, p, snap)
	report.Warnings = append(report.Warnings, modelWarnings(report.Models, runtimeSettings)...)
	report.Failures = append(report.Failures, budgetFailures(report.Totals, runtimeSettings, p, snap)...)
	sort.Slice(report.Scenarios, func(i, j int) bool { return report.Scenarios[i].Name < report.Scenarios[j].Name })
	return report, nil
}

func (h *Handlers) sourceRoot(explicit string) (string, error) {
	if explicit != "" {
		return filepath.Abs(explicit)
	}
	if h.SourceRoot != "" {
		return filepath.Abs(h.SourceRoot)
	}
	for _, key := range []string{"VROOLI_SOURCE_ROOT", "VROOLI_CLI_SOURCE_ROOT"} {
		if v := strings.TrimSpace(h.getenv(key)); v != "" {
			return filepath.Abs(v)
		}
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := wd; ; dir = filepath.Dir(dir) {
		if fileExists(filepath.Join(dir, "resources", "ollama", "resource.json")) && fileExists(filepath.Join(dir, "scenarios")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return "", errors.New("locate Vrooli source root: pass --root or set VROOLI_SOURCE_ROOT")
}

func (h *Handlers) loadPolicy() (policy.Policy, string, error) {
	if h.PolicyPath != "" {
		p, err := policy.LoadFile(h.PolicyPath)
		return p, h.PolicyPath, err
	}
	return policy.LoadDefaultFile(h.getenv)
}

func (h *Handlers) runtimeSettings(root string) (RuntimeSettings, []string) {
	path := h.RuntimePath
	if path == "" {
		path = filepath.Join(root, "resources", "ollama", "resource.json")
	}
	settings, warnings := readRuntimeSettings(path)
	if v := strings.TrimSpace(h.getenv("OLLAMA_NUM_PARALLEL")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			settings.NumParallel = n
		} else {
			warnings = append(warnings, fmt.Sprintf("ignore invalid OLLAMA_NUM_PARALLEL=%q", v))
		}
	}
	if v := strings.TrimSpace(h.getenv("OLLAMA_MAX_LOADED_MODELS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			settings.MaxLoadedModels = n
		} else {
			warnings = append(warnings, fmt.Sprintf("ignore invalid OLLAMA_MAX_LOADED_MODELS=%q", v))
		}
	}
	return settings, warnings
}

func (h *Handlers) hostSnapshot(ctx context.Context) (hostinventory.Snapshot, []string) {
	if h.Host == nil {
		h.Host = systemHostCollector{}
	}
	snap, err := h.Host.Collect(ctx)
	if err != nil {
		return hostinventory.Snapshot{}, []string{fmt.Sprintf("host inventory unavailable: %v", err)}
	}
	return snap, nil
}

func (h *Handlers) ollamaState(ctx context.Context) OllamaState {
	client := h.NewClient
	if client == nil {
		client = func() OllamaClient { return ensure.NewClient() }
	}
	c := client()
	state := OllamaState{}
	tags, err := c.ListTags(ctx)
	if err != nil {
		state.Warnings = append(state.Warnings, fmt.Sprintf("Ollama /api/tags unavailable: %v", err))
	} else {
		state.TagsAvailable = true
		for ref := range tags {
			state.Installed = append(state.Installed, ref)
		}
		sort.Strings(state.Installed)
	}
	running, err := c.ListRunning(ctx)
	if err != nil {
		state.Warnings = append(state.Warnings, fmt.Sprintf("Ollama /api/ps unavailable: %v", err))
	} else {
		state.PSAvailable = true
		state.Processor = ensure.SummarizeProcessors(running).Processor
		for _, m := range running {
			state.Loaded = append(state.Loaded, m.Name)
		}
		sort.Strings(state.Loaded)
	}
	return state
}

func (h *Handlers) getenv(key string) string {
	if h.GetEnv != nil {
		return h.GetEnv(key)
	}
	return os.Getenv(key)
}

func manifestPaths(root string, req PlanRequest) ([]string, string, error) {
	if req.AllScenarios {
		pattern := filepath.Join(root, "scenarios", "*", ".vrooli", "service.json")
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, pattern, err
		}
		sort.Strings(matches)
		return matches, pattern, nil
	}
	path := filepath.Join(root, "scenarios", req.Scenario, ".vrooli", "service.json")
	return []string{path}, path, nil
}

func readScenarioDemand(path string, p policy.Policy) (ScenarioDemand, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ScenarioDemand{}, fmt.Errorf("read scenario manifest %s: %w", path, err)
	}
	var mf manifestFile
	if err := json.Unmarshal(data, &mf); err != nil {
		return ScenarioDemand{}, fmt.Errorf("parse scenario manifest %s: %w", path, err)
	}
	name := filepath.Base(filepath.Dir(filepath.Dir(path)))
	out := ScenarioDemand{Name: name, Manifest: path}
	dep, ok := mf.Dependencies.Resources["ollama"]
	if !ok {
		return out, nil
	}
	if dep.Enabled != nil && !*dep.Enabled {
		return out, nil
	}
	var cfg ensure.Config
	if err := json.Unmarshal(dep.Raw, &cfg); err != nil {
		return out, fmt.Errorf("parse ollama dependency in %s: %w", path, err)
	}
	resolution, err := p.Resolve(cfg.ResolveRequest())
	out.Warnings = append(out.Warnings, resolution.Warnings...)
	if err != nil {
		out.Warnings = append(out.Warnings, fmt.Sprintf("%s: %v", name, err))
	}
	for _, roleReq := range cfg.ModelRoles {
		if role := strings.TrimSpace(roleReq.Role); role != "" {
			out.Roles = addUnique(out.Roles, role)
		}
	}
	for _, m := range resolution.Models {
		out.Models = addUnique(out.Models, m.Ref)
		if m.Source == "direct" {
			out.DirectRefs = addUnique(out.DirectRefs, m.Ref)
		}
	}
	sort.Strings(out.Roles)
	sort.Strings(out.DirectRefs)
	sort.Strings(out.Models)
	return out, nil
}

func readRuntimeSettings(path string) (RuntimeSettings, []string) {
	settings := RuntimeSettings{
		MemoryLimitGB:   defaultRuntimeMemoryLimitGB,
		NumParallel:     4,
		MaxLoadedModels: 3,
	}
	var warnings []string
	data, err := os.ReadFile(path)
	if err != nil {
		return settings, []string{fmt.Sprintf("read Ollama resource runtime settings: %v", err)}
	}
	var raw struct {
		ManagedService struct {
			Environment map[string]string `json:"environment"`
		} `json:"managed_service"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return settings, []string{fmt.Sprintf("parse Ollama resource runtime settings: %v", err)}
	}
	if v := strings.TrimSpace(raw.ManagedService.Environment["OLLAMA_NUM_PARALLEL"]); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			settings.NumParallel = n
		}
	}
	if v := strings.TrimSpace(raw.ManagedService.Environment["OLLAMA_MAX_LOADED_MODELS"]); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			settings.MaxLoadedModels = n
		}
	}
	return settings, warnings
}

func summarizeHost(snap hostinventory.Snapshot) HostSummary {
	out := HostSummary{
		OS:            snap.OS,
		Arch:          snap.Arch,
		CPUCores:      snap.CPU.Cores,
		MemoryTotalGB: bytesToGB(snap.Memory.TotalBytes),
		MemoryAvailGB: bytesToGB(snap.Memory.AvailableBytes),
		GPUCount:      len(snap.GPUs),
		DockerNvidia:  snap.DockerGPU.NvidiaRuntime,
	}
	for _, gpu := range snap.GPUs {
		out.MaxGPUVRAMGB = math.Max(out.MaxGPUVRAMGB, bytesToGB(gpu.VRAMBytes))
	}
	if len(snap.ProbeStatuses) > 0 {
		keys := make([]string, 0, len(snap.ProbeStatuses))
		for k := range snap.ProbeStatuses {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, k+"="+snap.ProbeStatuses[k])
		}
		out.ProbeStatusNote = strings.Join(parts, ", ")
	}
	return out
}

func flattenModels(acc map[string]*ModelDemand, p policy.Policy, state OllamaState) []ModelDemand {
	installed := set(state.Installed)
	loaded := set(state.Loaded)
	refs := make([]string, 0, len(acc))
	for ref := range acc {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	out := make([]ModelDemand, 0, len(refs))
	for _, ref := range refs {
		model := *acc[ref]
		sort.Strings(model.Scenarios)
		sort.Strings(model.Roles)
		model.Installed = installed[ref] || installed[withLatestTag(ref)]
		model.Loaded = loaded[ref] || loaded[withLatestTag(ref)]
		if m, ok := p.Models[ref]; ok {
			model.Capabilities = append([]string{}, m.Capabilities...)
			sort.Strings(model.Capabilities)
			model.DiskGBEstimate = m.DiskSizeGBEstimate
			model.RAMGBEstimate = m.RAMGBEstimate
			model.VRAMGBEstimate = m.VRAMGBEstimate
			if prov, ok := m.Provenance["capacity_estimates"]; ok {
				model.CapacityConfidence = prov.Confidence
				model.CapacitySourceKind = prov.SourceKind
				model.CapacitySource = prov.Source
				if prov.Confidence == "low" {
					model.Warnings = append(model.Warnings, "capacity estimate confidence is low")
				}
			}
		} else {
			model.Warnings = append(model.Warnings, "model is not in model-policy catalog; capacity estimates unavailable")
		}
		out = append(out, model)
	}
	return out
}

func calculateTotals(models []ModelDemand, runtime RuntimeSettings, p policy.Policy, snap hostinventory.Snapshot) Totals {
	totals := Totals{
		DistinctModels:      len(models),
		RuntimeMemoryBudget: runtime.MemoryLimitGB,
		HostMemoryBudgetGB:  bytesToGB(snap.Memory.TotalBytes) * float64(p.Constraints.ResidentModelBudgetPercent) / 100,
	}
	for _, m := range models {
		totals.EstimatedDiskGB += m.DiskGBEstimate
		totals.EstimatedRAMGB += m.RAMGBEstimate
		totals.EstimatedVRAMGB += m.VRAMGBEstimate
	}
	totals.ResidentRAMGB = largestN(models, runtime.MaxLoadedModels, func(m ModelDemand) float64 { return m.RAMGBEstimate })
	totals.ResidentVRAMGB = largestN(models, runtime.MaxLoadedModels, func(m ModelDemand) float64 { return m.VRAMGBEstimate })
	for _, gpu := range snap.GPUs {
		totals.GPUBudgetGB = math.Max(totals.GPUBudgetGB, bytesToGB(gpu.VRAMBytes)*float64(p.Constraints.ResidentModelBudgetPercent)/100)
	}
	return totals
}

func largestN(models []ModelDemand, n int, value func(ModelDemand) float64) float64 {
	if n <= 0 || len(models) == 0 {
		return 0
	}
	values := make([]float64, 0, len(models))
	for _, m := range models {
		values = append(values, value(m))
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(values)))
	if n > len(values) {
		n = len(values)
	}
	var sum float64
	for i := 0; i < n; i++ {
		sum += values[i]
	}
	return sum
}

func modelWarnings(models []ModelDemand, runtime RuntimeSettings) []string {
	var warnings []string
	directCount := 0
	for _, m := range models {
		if m.Direct {
			directCount++
		}
		for _, warning := range m.Warnings {
			warnings = append(warnings, fmt.Sprintf("%s: %s", m.Ref, warning))
		}
	}
	if directCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d direct model exception(s) requested; prefer model_roles where possible", directCount))
	}
	if len(models) > runtime.MaxLoadedModels {
		warnings = append(warnings, fmt.Sprintf("%d distinct models exceed OLLAMA_MAX_LOADED_MODELS=%d; likely unload/reload churn", len(models), runtime.MaxLoadedModels))
	}
	return warnings
}

func budgetFailures(t Totals, runtime RuntimeSettings, p policy.Policy, snap hostinventory.Snapshot) []string {
	var failures []string
	if t.ResidentRAMGB > runtime.MemoryLimitGB {
		failures = append(failures, fmt.Sprintf("resident RAM estimate %.1f GB exceeds Ollama runtime memory limit %.1f GB", t.ResidentRAMGB, runtime.MemoryLimitGB))
	}
	if t.HostMemoryBudgetGB > 0 && t.ResidentRAMGB > t.HostMemoryBudgetGB {
		failures = append(failures, fmt.Sprintf("resident RAM estimate %.1f GB exceeds host policy budget %.1f GB (%d%% of host RAM)", t.ResidentRAMGB, t.HostMemoryBudgetGB, p.Constraints.ResidentModelBudgetPercent))
	}
	if len(snap.GPUs) > 0 && t.GPUBudgetGB > 0 && t.ResidentVRAMGB > t.GPUBudgetGB {
		failures = append(failures, fmt.Sprintf("resident VRAM estimate %.1f GB exceeds largest GPU policy budget %.1f GB", t.ResidentVRAMGB, t.GPUBudgetGB))
	}
	return failures
}

func renderText(w io.Writer, report Report) {
	fmt.Fprintf(w, "Ollama capacity plan\n")
	fmt.Fprintf(w, "Source root: %s\n", report.SourceRoot)
	fmt.Fprintf(w, "Policy: %s\n", report.PolicyPath)
	fmt.Fprintf(w, "Scenarios: %d, models: %d\n", len(report.Scenarios), len(report.Models))
	fmt.Fprintf(w, "Runtime: memory_limit=%.1f GB, parallel=%d, max_loaded_models=%d\n", report.Runtime.MemoryLimitGB, report.Runtime.NumParallel, report.Runtime.MaxLoadedModels)
	fmt.Fprintf(w, "Host: %s/%s, cpu=%d, memory=%.1f GB, gpu_count=%d, max_gpu_vram=%.1f GB\n",
		report.Host.OS, report.Host.Arch, report.Host.CPUCores, report.Host.MemoryTotalGB, report.Host.GPUCount, report.Host.MaxGPUVRAMGB)
	fmt.Fprintf(w, "Ollama: processor=%s, loaded=%d, installed=%d\n", emptyDefault(string(report.Ollama.Processor), "unknown"), len(report.Ollama.Loaded), len(report.Ollama.Installed))
	fmt.Fprintf(w, "Totals: disk=%.1f GB, all_ram=%.1f GB, resident_ram=%.1f GB, resident_vram=%.1f GB\n",
		report.Totals.EstimatedDiskGB, report.Totals.EstimatedRAMGB, report.Totals.ResidentRAMGB, report.Totals.ResidentVRAMGB)
	if len(report.Scenarios) > 0 {
		fmt.Fprintln(w, "\nDemand:")
		for _, s := range report.Scenarios {
			fmt.Fprintf(w, "- %s: %s", s.Name, strings.Join(s.Models, ", "))
			if len(s.Roles) > 0 {
				fmt.Fprintf(w, " (roles: %s)", strings.Join(s.Roles, ", "))
			}
			fmt.Fprintln(w)
		}
	}
	if len(report.Models) > 0 {
		fmt.Fprintln(w, "\nModels:")
		for _, m := range report.Models {
			labels := []string{}
			if m.Installed {
				labels = append(labels, "installed")
			}
			if m.Loaded {
				labels = append(labels, "loaded")
			}
			if m.Direct {
				labels = append(labels, "direct")
			}
			labelText := ""
			if len(labels) > 0 {
				labelText = " [" + strings.Join(labels, ", ") + "]"
			}
			fmt.Fprintf(w, "- %s%s: disk=%.1f GB ram=%.1f GB vram=%.1f GB confidence=%s scenarios=%s\n",
				m.Ref, labelText, m.DiskGBEstimate, m.RAMGBEstimate, m.VRAMGBEstimate, emptyDefault(m.CapacityConfidence, "unknown"), strings.Join(m.Scenarios, ", "))
		}
	}
	if len(report.Ollama.Warnings) > 0 || len(report.Warnings) > 0 {
		fmt.Fprintln(w, "\nWarnings:")
		for _, warning := range append(report.Warnings, report.Ollama.Warnings...) {
			fmt.Fprintf(w, "- %s\n", warning)
		}
	}
	if len(report.Failures) > 0 {
		fmt.Fprintln(w, "\nFailures:")
		for _, failure := range report.Failures {
			fmt.Fprintf(w, "- %s\n", failure)
		}
	}
}

func withLatestTag(ref string) string {
	if strings.Contains(ref, ":") {
		return ref
	}
	return ref + ":latest"
}

func bytesToGB(v uint64) float64 {
	if v == 0 {
		return 0
	}
	return float64(v) / bytesPerGB
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func addUnique(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" || contains(values, value) {
		return values
	}
	return append(values, value)
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func set(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}

func emptyDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
