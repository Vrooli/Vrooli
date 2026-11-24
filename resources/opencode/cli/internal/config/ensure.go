package config

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/platform-go"
)

// Defaults carries the static config defaults formerly in config/defaults.sh.
type Defaults struct {
	Provider           string // active provider (empty = auto-detect)
	CloudRole          string // openrouter policy role backing the cloud default (code.default)
	ChatModel          string // cloud default chat model (resolved from CloudRole when empty)
	CompletionModel    string // cloud default completion model (resolved from CloudRole when empty)
	GoDefaultModel     string // opencode-go default model when Go subscription is active
	NumCtx             int    // per-model num_ctx for the local coder
	LocalRole          string // ollama policy role backing local coding (code.local)
	OrcaRouterRole     string // policy role used when OrcaRouter is selected explicitly
	AbliterationRole   string // policy role used when Abliteration is selected explicitly
	ProviderPreference string // explicit provider id (OPENCODE_PROVIDER), empty for the default tier order
	LegacyTargets      []string
}

// DefaultDefaults returns the built-in defaults, honoring the same env
// overrides as the legacy shell configuration.
func DefaultDefaults(getenv func(string) string) Defaults {
	if getenv == nil {
		getenv = os.Getenv
	}
	numCtx := 16384
	if v := strings.TrimSpace(getenv("OPENCODE_OLLAMA_NUM_CTX")); v != "" {
		if n, err := parseInt(v); err == nil && n > 0 {
			numCtx = n
		}
	}
	cloudRole := strings.TrimSpace(getenv("OPENCODE_DEFAULT_CHAT_ROLE"))
	if cloudRole == "" {
		cloudRole = "code.default"
	}
	goModel := strings.TrimSpace(getenv("OPENCODE_GO_DEFAULT_MODEL"))
	if goModel == "" {
		goModel = "deepseek-v4.1-flash"
	}
	return Defaults{
		Provider:           "",
		CloudRole:          cloudRole,
		GoDefaultModel:     goModel,
		NumCtx:             numCtx,
		LocalRole:          "code.local",
		OrcaRouterRole:     "code.default",
		AbliterationRole:   "research.unrestricted.default",
		ProviderPreference: strings.TrimSpace(getenv("OPENCODE_PROVIDER")),
		LegacyTargets:      []string{"openrouter/qwen3-coder", "openrouter/qwen/qwen3-coder"},
	}
}

// RoleResolution is the local-role view the config writer needs from the SSOT.
type RoleResolution struct {
	Model    string
	Sampling Sampling
}

// Resolver is the injectable seam over the Ollama probe SSOT
// (`resource-ollama`). The config writer NEVER probes the daemon directly.
type Resolver interface {
	InstalledModels(ctx context.Context) ([]string, error)
	LocalRole(ctx context.Context, role string) (RoleResolution, error)
}

// LocalModelLister is an optional Resolver extension. When implemented, every
// installed model that serves a chat/generate policy role is projected into
// OpenCode's provider block, so the picker follows the Ollama policy instead
// of a hand-maintained list. Fakes and older resolvers simply omit it.
type LocalModelLister interface {
	LocalModels(ctx context.Context, installed []string) ([]LocalModel, error)
}

// LocalPolicyModelLister exposes identities owned by the current policy
// catalog so Ensure can remove entries whose roles or rights were retired.
type LocalPolicyModelLister interface {
	LocalPolicyModels(ctx context.Context) ([]string, error)
}

// LocalInferenceRightChecker evaluates one catalog model against current
// trusted context without requiring the Ollama daemon to be running.
type LocalInferenceRightChecker interface {
	LocalInferenceAllowed(ctx context.Context, model string) (bool, error)
}

// LocalChatRoleChecker confirms that a model remains assigned to at least one
// current chat/generate role, independent of runtime installation state.
type LocalChatRoleChecker interface {
	LocalChatRoleEligible(ctx context.Context, model string) (bool, error)
}

// Ollama's public activity edge owns port 11434. OpenCode must always use this
// loopback endpoint so inference cannot bypass live artifact-rights checks by
// inheriting an upstream OLLAMA_HOST value.
const ollamaActivityEdgeURL = "http://127.0.0.1:11434"

// ProviderModelCatalog lists the concrete models a hosted provider resource
// declares. A custom OpenAI-compatible provider must name its models in
// opencode.json; OpenCode only discovers catalogs for its built-in providers.
type ProviderModelCatalog func(ctx context.Context, providerID string) ([]ProviderModel, error)

// EnsureOptions configures Ensure.
type EnsureOptions struct {
	ConfigPath       string
	Defaults         Defaults
	HaveOpenCodeGo   bool
	HaveOpenRouter   bool
	HaveOrcaRouter   bool
	HaveAbliteration bool
	Resolver         Resolver
	ProviderModels   ProviderModelCatalog
	// SelectedOllamaModel commits an explicit local selection into the same
	// config write that projects its resource-owned model catalog.
	SelectedOllamaModel string
	Logf                func(format string, args ...any)
}

// Ensure decides the provider (OpenCode Go → OpenRouter → local Ollama),
// renders opencode.json preserving the permission map and unknown keys,
// and writes only on a real change. Returns whether the file changed.
func Ensure(ctx context.Context, opts EnsureOptions) (bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	logf := opts.Logf
	if logf == nil {
		logf = func(string, ...any) {}
	}
	d := opts.Defaults

	installed, listErr := opts.Resolver.InstalledModels(ctx)
	ollamaReachable := listErr == nil && len(installed) > 0
	localModel := ""
	var sampling Sampling
	var localRoleErr error
	if ollamaReachable {
		rr, err := opts.Resolver.LocalRole(ctx, d.LocalRole)
		if err != nil {
			localRoleErr = err
		} else if rr.Model == "" {
			localRoleErr = fmt.Errorf("policy returned an empty model")
		} else if !containsModel(installed, rr.Model) {
			localRoleErr = fmt.Errorf("policy model %q is not installed", rr.Model)
		} else {
			localModel = rr.Model
			sampling = rr.Sampling
		}
	}

	var extras []LocalModel
	if ollamaReachable {
		if lister, ok := opts.Resolver.(LocalModelLister); ok {
			models, err := lister.LocalModels(ctx, installed)
			if err != nil {
				logf("WARNING: could not project Ollama policy models into OpenCode config: %v", err)
			} else {
				extras = models
			}
		}
	}
	var policyModels []string
	if lister, ok := opts.Resolver.(LocalPolicyModelLister); ok {
		models, err := lister.LocalPolicyModels(ctx)
		if err != nil {
			if ollamaReachable {
				return false, fmt.Errorf("resolve Ollama policy model identities for picker reconciliation: %w", err)
			}
			logf("WARNING: could not reconcile Ollama policy models while its daemon is unavailable: %v", err)
		} else {
			policyModels = models
		}
	}

	var (
		provider        string
		chatModel       string
		completionModel string
		useOllama       bool
		useOpenCodeGo   bool
		useOpenRouter   bool
		useOrcaRouter   bool
		useAbliteration bool
	)

	// Explicit operator preference wins over the default tier order. The
	// provider must still have a usable key; otherwise the default order runs
	// and the warning explains why the preference was ignored.
	switch strings.ToLower(strings.TrimSpace(d.ProviderPreference)) {
	case orcaRouterProviderID:
		if opts.HaveOrcaRouter {
			model, err := resolveProviderRole(ctx, "resource-orcarouter", d.OrcaRouterRole)
			if err != nil {
				return false, fmt.Errorf("resolve OrcaRouter role %q: %w", d.OrcaRouterRole, err)
			}
			useOrcaRouter = true
			provider = orcaRouterProviderID
			chatModel = model
			completionModel = model
			logf("Selected OrcaRouter by OPENCODE_PROVIDER (provider=%s, model=%s)", provider, chatModel)
		} else {
			logf("WARNING: OPENCODE_PROVIDER=%s requested but ORCAROUTER_API_KEY is not configured", orcaRouterProviderID)
		}
	case abliterationProviderID:
		if opts.HaveAbliteration {
			model, err := resolveProviderRole(ctx, "resource-abliteration", d.AbliterationRole)
			if err != nil {
				return false, fmt.Errorf("resolve Abliteration role %q: %w", d.AbliterationRole, err)
			}
			useAbliteration = true
			provider = abliterationProviderID
			chatModel = model
			completionModel = model
			logf("Selected Abliteration by OPENCODE_PROVIDER (provider=%s, model=%s)", provider, chatModel)
		} else {
			logf("WARNING: OPENCODE_PROVIDER=%s requested but ABLIT_KEY is not configured", abliterationProviderID)
		}
	}

	// Tier 1: OpenCode Go subscription (cheapest, subscription-billed)
	if provider == "" && opts.HaveOpenCodeGo {
		useOpenCodeGo = true
		provider = goProviderID
		chatModel = d.GoDefaultModel
		completionModel = d.GoDefaultModel
		logf("Selected OpenCode Go subscription (provider=%s, model=%s)", provider, chatModel)
	}

	// Tier 2: OpenRouter metered API (fallback when no Go key)
	if provider == "" && !useOpenCodeGo && opts.HaveOpenRouter {
		useOpenRouter = true
		if d.ChatModel == "" || d.CompletionModel == "" {
			role := strings.TrimSpace(d.CloudRole)
			if role == "" {
				role = "code.default"
			}
			model, err := resolveCloudModel(ctx, role)
			if err != nil {
				return false, fmt.Errorf("resolve OpenRouter cloud default model (role %q): %w", role, err)
			}
			if d.ChatModel == "" {
				d.ChatModel = model
			}
			if d.CompletionModel == "" {
				d.CompletionModel = model
			}
		}
		provider = "openrouter"
		chatModel = d.ChatModel
		completionModel = d.CompletionModel
		logf("Selected OpenRouter API (provider=%s, model=%s)", provider, chatModel)
	}

	// Tier 3: Local Ollama (fallback when no cloud credentials)
	if provider == "" && !useOpenCodeGo && !useOpenRouter && ollamaReachable {
		if localModel != "" {
			useOllama = true
			provider = ollamaProviderID
			chatModel = localModel
			completionModel = localModel
			logf("Self-healed to local Ollama (provider=%s, model=%s)", provider, chatModel)
		} else if localRoleErr != nil {
			if len(extras) == 0 {
				return false, fmt.Errorf("resolve local Ollama role %q: %w", d.LocalRole, localRoleErr)
			}
			logf("WARNING: local Ollama default role %q is unavailable: %v; eligible installed models remain selectable", d.LocalRole, localRoleErr)
		}
	}

	canonicalPath, err := CanonicalConfigPath(opts.ConfigPath)
	if err != nil {
		return false, err
	}
	unlock, err := LockConfig(ctx, canonicalPath)
	if err != nil {
		return false, err
	}
	defer unlock()
	opts.ConfigPath = canonicalPath
	currentPolicyModels := append([]string(nil), policyModels...)
	if _, ok := opts.Resolver.(LocalPolicyModelLister); ok {
		previousModels, readErr := readPolicyModelState(canonicalPath)
		if readErr != nil {
			return false, readErr
		}
		policyModels = mergeModelIDs(previousModels, currentPolicyModels)
	}
	existing, readErr := os.ReadFile(opts.ConfigPath)
	if readErr != nil && !os.IsNotExist(readErr) {
		return false, fmt.Errorf("read %s: %w", opts.ConfigPath, readErr)
	}
	freshFile := os.IsNotExist(readErr) || len(bytes.TrimSpace(existing)) == 0

	in := Inputs{
		Provider:        provider,
		ChatModel:       chatModel,
		CompletionModel: completionModel,
	}

	currentModel := currentModelOf(existing)
	currentProvider := ""
	if i := strings.Index(currentModel, "/"); i >= 0 {
		currentProvider = currentModel[:i]
	}
	localSelectionRejected := false
	if currentProvider == ollamaProviderID {
		selectedLocalModel := strings.TrimSpace(strings.TrimPrefix(currentModel, ollamaProviderID+"/"))
		available := selectedLocalModel != "" && (listErr != nil || containsModel(installed, selectedLocalModel))
		if checker, ok := opts.Resolver.(LocalInferenceRightChecker); ok && selectedLocalModel != "" {
			var rightsAllowed bool
			rightsAllowed, err = checker.LocalInferenceAllowed(ctx, selectedLocalModel)
			if err != nil {
				return false, fmt.Errorf("revalidate explicitly selected Ollama model %q: %w", selectedLocalModel, err)
			}
			available = available && rightsAllowed
			if roleChecker, ok := opts.Resolver.(LocalChatRoleChecker); ok {
				roleAllowed, roleErr := roleChecker.LocalChatRoleEligible(ctx, selectedLocalModel)
				if roleErr != nil {
					return false, fmt.Errorf("revalidate chat-role eligibility for explicitly selected Ollama model %q: %w", selectedLocalModel, roleErr)
				}
				available = available && roleAllowed
			} else {
				available = available && (selectedLocalModel == localModel || containsLocalModel(extras, selectedLocalModel))
			}
		} else {
			available = available && (selectedLocalModel == localModel || containsLocalModel(extras, selectedLocalModel) || listErr != nil)
		}
		if available && !explicitNonOllamaPreference(d.ProviderPreference) {
			// Preserve an explicit local choice when cloud credentials appear
			// later. The activity edge still authorizes every actual inference.
			provider = ollamaProviderID
			chatModel = selectedLocalModel
			completionModel = selectedLocalModel
			useOllama = true
			useOpenCodeGo = false
			useOpenRouter = false
			useOrcaRouter = false
			useAbliteration = false
			logf("Preserving explicitly selected local Ollama model %s", selectedLocalModel)
		} else if !available {
			localSelectionRejected = true
		}
	}
	in.Provider = provider
	in.ChatModel = chatModel
	in.CompletionModel = completionModel

	if localSelectionRejected {
		if provider == "" {
			return false, fmt.Errorf("active Ollama model %q is no longer installed or locally authorized, and no eligible fallback is available", strings.TrimPrefix(currentModel, ollamaProviderID+"/"))
		}
		in.Repoint = true
	} else if freshFile {
		in.Repoint = true
	} else if useOllama && (currentProvider == "" || currentProvider == "openrouter" || currentProvider == goProviderID) {
		in.Repoint = true
	} else if useOpenCodeGo && currentProvider != goProviderID {
		in.Repoint = true
	} else if useOpenRouter && currentProvider != "openrouter" {
		in.Repoint = true
	} else if useOrcaRouter && currentProvider != orcaRouterProviderID {
		in.Repoint = true
	} else if useAbliteration && currentProvider != abliterationProviderID {
		in.Repoint = true
	}

	if useOpenCodeGo {
		in.Go = &GoProvider{
			BaseURL:    goBaseURL,
			ChatModel:  chatModel,
			SmallModel: completionModel,
		}
	}
	addOpenAIProvider := func(provider OpenAIProvider) {
		if opts.ProviderModels != nil {
			models, err := opts.ProviderModels(ctx, provider.ID)
			if err != nil {
				logf("WARNING: could not list %s models for OpenCode: %v", provider.ID, err)
			} else {
				provider.Models = models
			}
		}
		in.OpenAIProviders = append(in.OpenAIProviders, provider)
	}
	if opts.HaveOrcaRouter {
		addOpenAIProvider(OpenAIProvider{
			ID:      orcaRouterProviderID,
			Name:    orcaRouterName,
			NPM:     orcaRouterNPM,
			BaseURL: orcaRouterBaseURL,
		})
	}
	if opts.HaveAbliteration {
		addOpenAIProvider(OpenAIProvider{
			ID:      abliterationProviderID,
			Name:    abliterationName,
			NPM:     abliterationNPM,
			BaseURL: abliterationBaseURL,
		})
	}
	if (ollamaReachable && (localModel != "" || len(extras) > 0)) || len(policyModels) > 0 {
		in.Ollama = &OllamaProvider{
			BaseURL:      ollamaActivityEdgeURL + "/api",
			ChatModel:    localModel,
			SmallModel:   localModel,
			NumCtx:       d.NumCtx,
			Sampling:     sampling,
			PolicyModels: policyModels,
			ExtraModels:  extras,
		}
	}

	rendered, err := Render(existing, in)
	if err != nil {
		return false, err
	}
	if selected := strings.TrimSpace(opts.SelectedOllamaModel); selected != "" {
		rendered, err = SelectOllamaModel(rendered, selected)
		if err != nil {
			return false, fmt.Errorf("select prepared Ollama model: %w", err)
		}
	}

	if !freshFile && bytes.Equal(normalize(existing), normalize(rendered)) {
		if _, ok := opts.Resolver.(LocalPolicyModelLister); ok {
			if err := writePolicyModelState(ctx, canonicalPath, currentPolicyModels); err != nil {
				return false, err
			}
		}
		return false, nil
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if _, ok := opts.Resolver.(LocalPolicyModelLister); ok {
		// Persist ownership before config projection. If the config write fails,
		// the next ensure still knows which entries it owns and must reconcile.
		if err := writePolicyModelState(ctx, canonicalPath, policyModels); err != nil {
			return false, err
		}
	}
	if err := os.MkdirAll(filepath.Dir(opts.ConfigPath), 0o755); err != nil {
		return false, fmt.Errorf("mkdir config dir: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if err := WriteConfigAtomically(ctx, opts.ConfigPath, rendered); err != nil {
		return false, fmt.Errorf("write %s: %w", opts.ConfigPath, err)
	}
	if _, ok := opts.Resolver.(LocalPolicyModelLister); ok {
		if err := writePolicyModelState(ctx, canonicalPath, currentPolicyModels); err != nil {
			return false, err
		}
	}

	switch {
	case freshFile:
		logf("Created OpenCode config at %s (provider=%s)", opts.ConfigPath, provider)
	case useOpenCodeGo:
		logf("Switched to OpenCode Go subscription (provider=%s, model=%s)", provider, chatModel)
	case useOrcaRouter:
		logf("Using OrcaRouter provider (provider=%s, model=%s)", provider, chatModel)
	case useAbliteration:
		logf("Using Abliteration provider (provider=%s, model=%s)", provider, chatModel)
	case useOllama:
		logf("Self-healed OpenCode model -> %s/%s (no cloud key; local Ollama reachable)", provider, chatModel)
	default:
		logf("Updated OpenCode config at %s", opts.ConfigPath)
	}

	if provider == "" {
		logf("WARNING: No usable provider credentials found. Set OPENCODE_GO_KEY, OPENROUTER_API_KEY, ORCAROUTER_API_KEY, or ABLIT_KEY, or ensure Ollama is running.")
	}
	return true, nil
}

func explicitNonOllamaPreference(preference string) bool {
	preference = strings.TrimSpace(preference)
	return preference != "" && !strings.EqualFold(preference, ollamaProviderID)
}

func containsLocalModel(models []LocalModel, target string) bool {
	for _, model := range models {
		if strings.TrimSpace(model.Model) == target {
			return true
		}
	}
	return false
}

// LockConfig serializes every OpenCode config read-modify-write operation
// across processes. Atomic rename protects readers from partial files; this
// lock prevents independent ensure and selection commands from losing updates.
func LockConfig(ctx context.Context, path string) (func(), error) {
	absPath, err := CanonicalConfigPath(path)
	if err != nil {
		return nil, err
	}
	unlock, err := platform.AcquireFileLockContext(ctx, absPath+".lock")
	if err != nil {
		return nil, fmt.Errorf("lock OpenCode config: %w", err)
	}
	return unlock, nil
}

// CanonicalConfigPath resolves parent-directory aliases and a leaf config
// symlink so every read-modify-write command derives the same lock and state
// paths for the same underlying file.
func CanonicalConfigPath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("OpenCode config path is required")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve OpenCode config path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return "", fmt.Errorf("create OpenCode config directory: %w", err)
	}
	canonicalDir, err := filepath.EvalSymlinks(filepath.Dir(absPath))
	if err != nil {
		return "", fmt.Errorf("resolve OpenCode config directory: %w", err)
	}
	canonicalPath := filepath.Join(canonicalDir, filepath.Base(absPath))
	info, err := os.Lstat(canonicalPath)
	if err == nil && info.Mode()&os.ModeSymlink != 0 {
		canonicalPath, err = filepath.EvalSymlinks(canonicalPath)
	}
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("resolve OpenCode config symlink: %w", err)
	}
	return canonicalPath, nil
}

// WriteConfigAtomically replaces the OpenCode config without dropping its file
// mode, symlink, or unrelated content. It checks cancellation immediately
// before the rename commit point.
func WriteConfigAtomically(ctx context.Context, path string, data []byte) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	mode := os.FileMode(0o644)
	if linkInfo, err := os.Lstat(path); err == nil && linkInfo.Mode()&os.ModeSymlink != 0 {
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			return fmt.Errorf("resolve OpenCode config symlink: %w", err)
		}
		path = resolved
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("inspect OpenCode config path: %w", err)
	}
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect OpenCode config permissions: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create OpenCode config directory: %w", err)
	}
	temp, err := os.CreateTemp(dir, ".opencode.json-*")
	if err != nil {
		return fmt.Errorf("create temporary OpenCode config: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(mode); err != nil {
		temp.Close()
		return fmt.Errorf("set OpenCode config permissions: %w", err)
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return fmt.Errorf("write temporary OpenCode config: %w", err)
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return fmt.Errorf("sync temporary OpenCode config: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temporary OpenCode config: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace OpenCode config: %w", err)
	}
	if dirHandle, err := os.Open(dir); err == nil {
		defer dirHandle.Close()
		if err := dirHandle.Sync(); err != nil {
			return fmt.Errorf("sync OpenCode config directory: %w", err)
		}
	}
	return nil
}

// --- helpers ------------------------------------------------------------------

func currentModelOf(existing []byte) string {
	if len(bytes.TrimSpace(existing)) == 0 {
		return ""
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(existing, &top); err != nil {
		return ""
	}
	return rawString(top[keyModel])
}

func containsModel(installed []string, model string) bool {
	for _, m := range installed {
		if m == model || m == model+":latest" {
			return true
		}
	}
	return false
}

func normalize(data []byte) []byte {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return data
	}
	out, err := json.Marshal(v)
	if err != nil {
		return data
	}
	return out
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// --- cloud model role resolver (execs the resource-openrouter SSOT) -----------

var resolveCloudModel = execResolveCloudModel

func execResolveCloudModel(ctx context.Context, role string) (string, error) {
	return resolveProviderRole(ctx, "resource-openrouter", role)
}

// resolveProviderRole asks any provider resource for the concrete model behind
// a role. This is the generic seam new providers use; no provider id is baked
// into the resolution path.
func resolveProviderRole(ctx context.Context, bin, role string) (string, error) {
	role = strings.TrimSpace(role)
	if role == "" {
		return "", fmt.Errorf("%s role is required", bin)
	}
	out, err := exec.CommandContext(ctx, bin, "policy", "resolve", "--role", role, "--field", "model").Output()
	if err != nil {
		return "", fmt.Errorf("exec %s policy resolve --role %s --field model: %w", bin, role, err)
	}
	model := strings.TrimSpace(string(out))
	if model == "" {
		return "", fmt.Errorf("%s resolved an empty model for role %q", bin, role)
	}
	return model, nil
}

// --- default Resolver (execs the resource-ollama SSOT) ------------------------

type ExecResolver struct {
	Command string
}

func (r ExecResolver) bin() string {
	if strings.TrimSpace(r.Command) != "" {
		return r.Command
	}
	return "resource-ollama"
}

func (r ExecResolver) InstalledModels(ctx context.Context) ([]string, error) {
	out, err := exec.CommandContext(ctx, r.bin(), "models", "list", "--json").Output()
	if err != nil {
		return nil, err
	}
	var payload struct {
		Models []string `json:"models"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(out), &payload); err != nil {
		return nil, err
	}
	return payload.Models, nil
}

func (r ExecResolver) LocalRole(ctx context.Context, role string) (RoleResolution, error) {
	out, err := exec.CommandContext(ctx, r.bin(), "policy", "roles", "--require-local-inference", "--json").Output()
	if err != nil {
		return RoleResolution{}, err
	}
	var payload policyRolesPayload
	if err := json.Unmarshal(bytes.TrimSpace(out), &payload); err != nil {
		return RoleResolution{}, err
	}
	for _, candidate := range payload.Roles {
		if candidate.Role != role {
			continue
		}
		if candidate.RightsDecision == nil || !strings.EqualFold(strings.TrimSpace(candidate.RightsDecision.Status), "allowed") || !strings.EqualFold(strings.TrimSpace(candidate.RightsDecision.Use), "local_inference") {
			status := "missing"
			if candidate.RightsDecision != nil {
				status = candidate.RightsDecision.Status
			}
			return RoleResolution{}, fmt.Errorf("Ollama role %q local-inference rights are %s", role, status)
		}
		rr := RoleResolution{Model: candidate.Model}
		if candidate.Sampling != nil {
			if candidate.Sampling.HasTemperature {
				t := candidate.Sampling.Temperature
				rr.Sampling.Temperature = &t
			}
			if candidate.Sampling.HasTopP {
				p := candidate.Sampling.TopP
				rr.Sampling.TopP = &p
			}
			if candidate.Sampling.HasTopK {
				k := candidate.Sampling.TopK
				rr.Sampling.TopK = &k
			}
		}
		return rr, nil
	}
	return RoleResolution{}, fmt.Errorf("Ollama policy did not return role %q", role)
}

// LocalModels projects the installed, policy-declared chat/generate models so
// OpenCode's picker follows the Ollama policy rather than a manual list.
func (r ExecResolver) LocalModels(ctx context.Context, installed []string) ([]LocalModel, error) {
	out, err := exec.CommandContext(ctx, r.bin(), "policy", "roles", "--require-local-inference", "--json").Output()
	if err != nil {
		return nil, err
	}
	return parseLocalModelsWithRights(out, installed, true)
}

func (r ExecResolver) LocalPolicyModels(ctx context.Context) ([]string, error) {
	out, err := exec.CommandContext(ctx, r.bin(), "policy", "models", "--json").Output()
	if err != nil {
		return nil, err
	}
	var payload struct {
		Models []struct {
			Model string `json:"model"`
		} `json:"models"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(out), &payload); err != nil {
		return nil, fmt.Errorf("decode Ollama policy model identities: %w", err)
	}
	models := make([]string, 0, len(payload.Models))
	for _, entry := range payload.Models {
		model := strings.TrimSpace(entry.Model)
		if model == "" {
			continue
		}
		models = append(models, model)
	}
	sort.Strings(models)
	return models, nil
}

func (r ExecResolver) LocalInferenceAllowed(ctx context.Context, model string) (bool, error) {
	out, err := exec.CommandContext(ctx, r.bin(), "policy", "models", "--require-local-inference", "--json").Output()
	if err != nil {
		return false, err
	}
	var payload struct {
		Models []struct {
			Model          string `json:"model"`
			RightsDecision *struct {
				Status string `json:"status"`
				Use    string `json:"use"`
			} `json:"rights_decision"`
		} `json:"models"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(out), &payload); err != nil {
		return false, fmt.Errorf("decode Ollama local-inference model decisions: %w", err)
	}
	for _, entry := range payload.Models {
		if strings.TrimSpace(entry.Model) != strings.TrimSpace(model) {
			continue
		}
		if entry.RightsDecision == nil {
			return false, fmt.Errorf("Ollama model %q has no local-inference rights decision", model)
		}
		return strings.EqualFold(strings.TrimSpace(entry.RightsDecision.Status), "allowed") && strings.EqualFold(strings.TrimSpace(entry.RightsDecision.Use), "local_inference"), nil
	}
	return false, fmt.Errorf("Ollama policy does not contain model %q", model)
}

func (r ExecResolver) LocalChatRoleEligible(ctx context.Context, model string) (bool, error) {
	rolesOut, err := exec.CommandContext(ctx, r.bin(), "policy", "roles", "--json").Output()
	if err != nil {
		return false, err
	}
	var payload policyRolesPayload
	if err := json.Unmarshal(bytes.TrimSpace(rolesOut), &payload); err != nil {
		return false, fmt.Errorf("decode Ollama chat-role catalog: %w", err)
	}
	modelsOut, err := exec.CommandContext(ctx, r.bin(), "policy", "models", "--json").Output()
	if err != nil {
		return false, err
	}
	var models struct {
		Models []struct {
			Model        string   `json:"model"`
			Capabilities []string `json:"capabilities"`
			PolicySHA256 string   `json:"policy_sha256"`
		} `json:"models"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(modelsOut), &models); err != nil {
		return false, fmt.Errorf("decode Ollama policy model catalog: %w", err)
	}
	modelCapabilities := make(map[string][]string, len(models.Models))
	modelPolicySHA := ""
	for _, entry := range models.Models {
		entrySHA := strings.TrimSpace(entry.PolicySHA256)
		if entrySHA == "" {
			return false, fmt.Errorf("Ollama policy model catalog has a model without a policy digest")
		}
		if modelPolicySHA == "" {
			modelPolicySHA = entrySHA
		} else if entrySHA != modelPolicySHA {
			return false, fmt.Errorf("Ollama policy model catalog has inconsistent policy digests")
		}
		if identity := strings.TrimSpace(entry.Model); identity != "" {
			modelCapabilities[identity] = entry.Capabilities
		}
	}
	if len(models.Models) == 0 {
		return false, fmt.Errorf("Ollama policy model catalog is empty")
	}
	for _, role := range payload.Roles {
		if strings.TrimSpace(role.PolicySHA256) == "" || strings.TrimSpace(role.PolicySHA256) != modelPolicySHA {
			return false, fmt.Errorf("Ollama policy role and model catalogs describe different policy revisions")
		}
	}
	capabilities := modelCapabilities[strings.TrimSpace(model)]
	if !servesChat(capabilities) {
		return false, nil
	}
	for _, role := range payload.Roles {
		if strings.TrimSpace(role.Model) == strings.TrimSpace(model) && servesChat(role.RequiredCapabilities) && satisfiesCapabilities(capabilities, role.RequiredCapabilities) {
			return true, nil
		}
		if servesChat(role.RequiredCapabilities) {
			for _, fallback := range role.Fallbacks {
				if strings.TrimSpace(fallback) == strings.TrimSpace(model) && satisfiesCapabilities(capabilities, role.RequiredCapabilities) {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func satisfiesCapabilities(modelCapabilities, requiredCapabilities []string) bool {
	available := make(map[string]struct{}, len(modelCapabilities))
	for _, capability := range modelCapabilities {
		available[strings.TrimSpace(capability)] = struct{}{}
	}
	for _, capability := range requiredCapabilities {
		required := strings.TrimSpace(capability)
		if required == "" {
			continue
		}
		if _, ok := available[required]; !ok {
			return false
		}
	}
	return true
}

type policyRolesPayload struct {
	Roles []policyRole `json:"roles"`
}

type policyRole struct {
	Role                 string   `json:"role"`
	Model                string   `json:"model"`
	Fallbacks            []string `json:"fallbacks"`
	RequiredCapabilities []string `json:"required_capabilities"`
	Preference           int      `json:"preference"`
	Capabilities         []string `json:"capabilities"`
	ContextWindowTokens  int      `json:"context_window_tokens"`
	MaxTokens            *int     `json:"max_tokens"`
	PolicySHA256         string   `json:"policy_sha256"`
	RightsDecision       *struct {
		Status string `json:"status"`
		Use    string `json:"use"`
	} `json:"rights_decision,omitempty"`
	Sampling *struct {
		Temperature    float64 `json:"temperature"`
		TopP           float64 `json:"top_p"`
		TopK           int     `json:"top_k"`
		HasTemperature bool    `json:"has_temperature"`
		HasTopP        bool    `json:"has_top_p"`
		HasTopK        bool    `json:"has_top_k"`
	} `json:"sampling"`
}

func parseLocalModels(raw []byte, installed []string) ([]LocalModel, error) {
	return parseLocalModelsWithRights(raw, installed, false)
}

func parseLocalModelsWithRights(raw []byte, installed []string, requireRights bool) ([]LocalModel, error) {
	var payload policyRolesPayload
	if err := json.Unmarshal(bytes.TrimSpace(raw), &payload); err != nil {
		return nil, fmt.Errorf("decode ollama policy roles: %w", err)
	}
	byModel := map[string]LocalModel{}
	rank := map[string]int{}
	for _, role := range payload.Roles {
		model := strings.TrimSpace(role.Model)
		if model == "" || !containsModel(installed, model) || !servesChat(role.Capabilities) {
			continue
		}
		if requireRights && role.RightsDecision == nil {
			return nil, fmt.Errorf("Ollama policy role %q has no local-inference rights decision", role.Role)
		}
		if requireRights && (!strings.EqualFold(strings.TrimSpace(role.RightsDecision.Status), "allowed") || !strings.EqualFold(strings.TrimSpace(role.RightsDecision.Use), "local_inference")) {
			continue
		}
		// One entry per model. The most preferred role (lowest policy
		// preference) owns its sampling stance.
		if previous, ok := rank[model]; ok && previous > 0 && (role.Preference == 0 || previous <= role.Preference) {
			continue
		}
		entry := LocalModel{Model: model, NumCtx: role.ContextWindowTokens}
		if role.MaxTokens != nil {
			entry.MaxOutput = *role.MaxTokens
		}
		if role.Sampling != nil {
			if role.Sampling.HasTemperature {
				value := role.Sampling.Temperature
				entry.Sampling.Temperature = &value
			}
			if role.Sampling.HasTopP {
				value := role.Sampling.TopP
				entry.Sampling.TopP = &value
			}
			if role.Sampling.HasTopK {
				value := role.Sampling.TopK
				entry.Sampling.TopK = &value
			}
		}
		byModel[model] = entry
		rank[model] = role.Preference
	}
	models := make([]LocalModel, 0, len(byModel))
	for _, model := range byModel {
		models = append(models, model)
	}
	sort.Slice(models, func(i, j int) bool { return models[i].Model < models[j].Model })
	return models, nil
}

func servesChat(capabilities []string) bool {
	for _, capability := range capabilities {
		switch strings.ToLower(strings.TrimSpace(capability)) {
		case "chat", "generate":
			return true
		}
	}
	return false
}
