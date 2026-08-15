package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"sync"

	"github.com/vrooli/vrooli/internal/hostreq"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/safeguards"
	autohealrecoveryprivileges "github.com/vrooli/vrooli/internal/safeguards/autoheal-recovery-privileges"
	"github.com/vrooli/vrooli/internal/safeguards/clock"
	cloudflaredrecoveryprivileges "github.com/vrooli/vrooli/internal/safeguards/cloudflared-recovery-privileges"
	crashkernelreserve "github.com/vrooli/vrooli/internal/safeguards/crashkernel-reserve"
	dnsresolution "github.com/vrooli/vrooli/internal/safeguards/dns-resolution"
	dockerhostfirewall "github.com/vrooli/vrooli/internal/safeguards/docker-host-firewall"
	edacmodules "github.com/vrooli/vrooli/internal/safeguards/edac-modules"
	hosthardening "github.com/vrooli/vrooli/internal/safeguards/host-hardening"
	kernelconfig "github.com/vrooli/vrooli/internal/safeguards/kernel-config"
	loginkeyringunlock "github.com/vrooli/vrooli/internal/safeguards/login-keyring-unlock"
	modelpolicydrift "github.com/vrooli/vrooli/internal/safeguards/model-policy-drift"
	natprotection "github.com/vrooli/vrooli/internal/safeguards/nat-protection"
	"github.com/vrooli/vrooli/internal/safeguards/netconsole"
	nvidiadriver "github.com/vrooli/vrooli/internal/safeguards/nvidia-driver"
	ollamaresourcecontrols "github.com/vrooli/vrooli/internal/safeguards/ollama-resource-controls"
	onboardingapplyprivileges "github.com/vrooli/vrooli/internal/safeguards/onboarding-apply-privileges"
	pstorenative "github.com/vrooli/vrooli/internal/safeguards/pstore-native"
	pstoreobservability "github.com/vrooli/vrooli/internal/safeguards/pstore-observability"
	pstoreramoops "github.com/vrooli/vrooli/internal/safeguards/pstore-ramoops"
	remotedesktopaccess "github.com/vrooli/vrooli/internal/safeguards/remote-desktop-access"
	remotesessionprotection "github.com/vrooli/vrooli/internal/safeguards/remote-session-protection"
	tcptuning "github.com/vrooli/vrooli/internal/safeguards/tcp-tuning"
	tpmcredentialaccess "github.com/vrooli/vrooli/internal/safeguards/tpm-credential-access"
	vroolilauncher "github.com/vrooli/vrooli/internal/safeguards/vrooli-launcher"
	workspacesandboxuserns "github.com/vrooli/vrooli/internal/safeguards/workspace-sandbox-userns"
	"github.com/vrooli/vrooli/internal/tools"
	"github.com/vrooli/vrooli/internal/tools/cloudflared"
	"github.com/vrooli/vrooli/internal/tools/docker"
	kdumptools "github.com/vrooli/vrooli/internal/tools/kdump-tools"
	"github.com/vrooli/vrooli/internal/tools/mcelog"
	"github.com/vrooli/vrooli/internal/tools/protoc"
	protocgenconnectgo "github.com/vrooli/vrooli/internal/tools/protoc-gen-connect-go"
	protocgenes "github.com/vrooli/vrooli/internal/tools/protoc-gen-es"
	protocgengo "github.com/vrooli/vrooli/internal/tools/protoc-gen-go"
	"github.com/vrooli/vrooli/internal/tools/quint"
	"github.com/vrooli/vrooli/internal/tools/rasdaemon"
	"github.com/vrooli/vrooli/internal/tools/stripe"
	"github.com/vrooli/vrooli/internal/tools/vault"
)

// customToolHandlers must stay in sync with every tool.json "handler" field
// under internal/tools/. The invariant is enforced by
// TestToolManifestsReferenceRegisteredHandlers.
var customToolHandlers = map[string]func(hostreqkit.ToolManifest) hostreqkit.Handler{
	"cloudflared":           cloudflared.NewHandler,
	"docker":                docker.NewHandler,
	"kdump_tools":           kdumptools.NewHandler,
	"mcelog":                mcelog.NewHandler,
	"protoc":                protoc.NewHandler,
	"protoc_gen_connect_go": protocgenconnectgo.NewHandler,
	"protoc_gen_es":         protocgenes.NewHandler,
	"protoc_gen_go":         protocgengo.NewHandler,
	"quint":                 quint.NewHandler,
	"rasdaemon":             rasdaemon.NewHandler,
	"stripe":                stripe.NewHandler,
	"vault":                 vault.NewHandler,
}

// customSafeguardHandlers must stay in sync with every safeguard.json
// "handler" field under internal/safeguards/. The invariant is enforced by
// TestSafeguardManifestsReferenceRegisteredHandlers.
var customSafeguardHandlers = map[string]func(hostreqkit.SafeguardManifest) hostreqkit.Handler{
	"clock":                           clock.NewHandler,
	"autoheal_recovery_privileges":    autohealrecoveryprivileges.NewHandler,
	"onboarding_apply_privileges":     onboardingapplyprivileges.NewHandler,
	"model_policy_drift":              modelpolicydrift.NewHandler,
	"cloudflared_recovery_privileges": cloudflaredrecoveryprivileges.NewHandler,
	"crashkernel_reserve":             crashkernelreserve.NewHandler,
	"dns_resolution":                  dnsresolution.NewHandler,
	"docker_host_firewall":            dockerhostfirewall.NewHandler,
	"tpm_credential_access":           tpmcredentialaccess.NewHandler,
	"edac_modules":                    edacmodules.NewHandler,
	"host_hardening":                  hosthardening.NewHandler,
	"kernel_config":                   kernelconfig.NewHandler,
	"login_keyring_unlock":            loginkeyringunlock.NewHandler,
	"nat_protection":                  natprotection.NewHandler,
	"netconsole":                      netconsole.NewHandler,
	"nvidia_driver":                   nvidiadriver.NewHandler,
	"remote_desktop_access":           remotedesktopaccess.NewHandler,
	"ollama_resource_controls":        ollamaresourcecontrols.NewHandler,
	"pstore_observability":            pstoreobservability.NewHandler,
	"pstore_native":                   pstorenative.NewHandler,
	"pstore_ramoops":                  pstoreramoops.NewHandler,
	"remote_session_protection":       remotesessionprotection.NewHandler,
	"tcp_tuning":                      tcptuning.NewHandler,
	"vrooli_launcher":                 vroolilauncher.NewHandler,
	"workspace_sandbox_userns":        workspacesandboxuserns.NewHandler,
}

type registry struct {
	tools      map[string]hostreqkit.Handler
	safeguards map[string]hostreqkit.Handler
}

// newRegistry builds a registry from a static list of handlers. It panics on
// invalid input because the caller is in-process Go code whose inputs are
// validated at compile time; use loadRegistry for data-driven construction.
func newRegistry(items ...hostreqkit.Handler) registry {
	r := registry{
		tools:      map[string]hostreqkit.Handler{},
		safeguards: map[string]hostreqkit.Handler{},
	}
	for _, item := range items {
		if err := r.register(item); err != nil {
			panic(err)
		}
	}
	return r
}

func (r *registry) register(item hostreqkit.Handler) error {
	if item == nil {
		return errors.New("runtime registry: nil handler")
	}
	name := strings.TrimSpace(item.Name())
	if name == "" {
		return errors.New("runtime registry: handler name is required")
	}
	target := r.handlersForKind(item.Kind())
	if _, exists := target[name]; exists {
		return fmt.Errorf("runtime registry: duplicate %s handler %q", item.Kind(), name)
	}
	target[name] = item
	return nil
}

func (r registry) lookup(kind hostreq.Kind, name string) hostreqkit.Handler {
	return r.handlersForKind(kind)[strings.TrimSpace(name)]
}

func (r registry) names(kind hostreq.Kind) []string {
	target := r.handlersForKind(kind)
	result := make([]string, 0, len(target))
	for name := range target {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func (r registry) handlersForKind(kind hostreq.Kind) map[string]hostreqkit.Handler {
	switch kind {
	case hostreq.KindSafeguard:
		return r.safeguards
	default:
		return r.tools
	}
}

var (
	runtimeRegistryOnce sync.Once
	runtimeRegistryVal  registry
	runtimeRegistryErr  error
)

// ensureRegistry lazily loads the process-wide registry and memoizes the
// result (including errors) for the life of the process. Loading is deferred
// so that manifest/handler inconsistencies surface as returnable errors
// rather than init-time panics that kill the binary before main runs.
func ensureRegistry() (registry, error) {
	runtimeRegistryOnce.Do(func() {
		runtimeRegistryVal, runtimeRegistryErr = loadRegistry()
	})
	return runtimeRegistryVal, runtimeRegistryErr
}

func loadRegistry() (registry, error) {
	r := registry{
		tools:      make(map[string]hostreqkit.Handler),
		safeguards: make(map[string]hostreqkit.Handler),
	}
	if err := loadTools(&r, tools.Manifests); err != nil {
		return registry{}, err
	}
	if err := loadSafeguards(&r, safeguards.Manifests); err != nil {
		return registry{}, err
	}
	return r, nil
}

func loadTools(r *registry, fsys fs.FS) error {
	var loadErr error
	_ = fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			loadErr = fmt.Errorf("walk tool manifests at %s: %w", path, walkErr)
			return fs.SkipAll
		}
		if d.IsDir() || d.Name() != "tool.json" {
			return nil
		}
		data, readErr := fs.ReadFile(fsys, path)
		if readErr != nil {
			loadErr = fmt.Errorf("read tool manifest %s: %w", path, readErr)
			return fs.SkipAll
		}
		var manifest hostreqkit.ToolManifest
		if jsonErr := json.Unmarshal(data, &manifest); jsonErr != nil {
			loadErr = fmt.Errorf("parse tool manifest %s: %w", path, jsonErr)
			return fs.SkipAll
		}
		if strings.TrimSpace(manifest.Name) == "" {
			loadErr = fmt.Errorf("tool manifest %s has no name", path)
			return fs.SkipAll
		}
		var h hostreqkit.Handler
		if manifest.Handler != "" {
			ctor, ok := customToolHandlers[manifest.Handler]
			if !ok {
				loadErr = fmt.Errorf(
					"tool %q references unknown handler %q (register it in internal/runtime/registry.go or drop the handler field in %s)",
					manifest.Name, manifest.Handler, path,
				)
				return fs.SkipAll
			}
			h = ctor(manifest)
		} else {
			h = newGenericToolHandler(manifest)
		}
		if regErr := r.register(h); regErr != nil {
			loadErr = fmt.Errorf("register tool from %s: %w", path, regErr)
			return fs.SkipAll
		}
		return nil
	})
	return loadErr
}

func loadSafeguards(r *registry, fsys fs.FS) error {
	var loadErr error
	_ = fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			loadErr = fmt.Errorf("walk safeguard manifests at %s: %w", path, walkErr)
			return fs.SkipAll
		}
		if d.IsDir() || d.Name() != "safeguard.json" {
			return nil
		}
		data, readErr := fs.ReadFile(fsys, path)
		if readErr != nil {
			loadErr = fmt.Errorf("read safeguard manifest %s: %w", path, readErr)
			return fs.SkipAll
		}
		var manifest hostreqkit.SafeguardManifest
		if jsonErr := json.Unmarshal(data, &manifest); jsonErr != nil {
			loadErr = fmt.Errorf("parse safeguard manifest %s: %w", path, jsonErr)
			return fs.SkipAll
		}
		if strings.TrimSpace(manifest.Name) == "" {
			loadErr = fmt.Errorf("safeguard manifest %s has no name", path)
			return fs.SkipAll
		}
		ctor, ok := customSafeguardHandlers[manifest.Handler]
		if !ok {
			loadErr = fmt.Errorf(
				"safeguard %q references unknown handler %q (register it in internal/runtime/registry.go or drop the handler field in %s)",
				manifest.Name, manifest.Handler, path,
			)
			return fs.SkipAll
		}
		if regErr := r.register(ctor(manifest)); regErr != nil {
			loadErr = fmt.Errorf("register safeguard from %s: %w", path, regErr)
			return fs.SkipAll
		}
		return nil
	})
	return loadErr
}

func lookupHandler(kind hostreq.Kind, name string) (hostreqkit.Handler, error) {
	reg, err := ensureRegistry()
	if err != nil {
		return nil, err
	}
	return reg.lookup(kind, name), nil
}

// HasHandler reports whether a handler is registered for (kind, name). The
// error is non-nil when the embedded manifests failed to load (e.g. a
// tool.json references a handler missing from customToolHandlers); callers
// should surface that error rather than treating the boolean as authoritative.
func HasHandler(kind hostreq.Kind, name string) (bool, error) {
	reg, err := ensureRegistry()
	if err != nil {
		return false, err
	}
	return reg.lookup(kind, name) != nil, nil
}
