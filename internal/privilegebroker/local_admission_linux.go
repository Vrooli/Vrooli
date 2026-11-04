//go:build linux

package privilegebroker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/operatorstate"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// installLocalAdmissionPolicy runs only inside the existing setup elevation
// child. Re-running setup replaces policy, not unrelated host/firewall state.
func installLocalAdmissionPolicy(ctx context.Context, repoRoot string, uid uint32) error {
	home, err := userHomeForUID(uid)
	if err != nil {
		return err
	}
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", UserHomeDir: func() (string, error) { return home, nil }})
	if err != nil {
		return err
	}
	paths, err := resolver.Resolve(storage.Options{ScenarioID: "vrooli-onboarding"})
	if err != nil {
		return err
	}
	document, err := operatorstate.New(operatorstate.Config{RepoRoot: repoRoot, StateRoot: paths.StateDir}).Load(ctx)
	if err != nil {
		return err
	}
	choice := document.HostSafeguards["local_service_admission"]
	var bindings []LocalAdmissionBinding
	if choice.OptedIn != nil && *choice.OptedIn {
		bindings, err = LocalAdmissionBindings(choice.Config)
		if err != nil {
			return err
		}
	}
	dbPath, err := scenarioruntime.DefaultDBPath(home)
	if err != nil {
		return err
	}
	policy := LocalAdmissionPolicy{Version: 1, UID: uid, RuntimeDB: dbPath, Bindings: bindings}
	for _, path := range []string{filepath.Dir(LocalAdmissionPolicyPath), filepath.Dir(LocalAdmissionStateDir), LocalAdmissionStateDir} {
		if err := rootControlledAdmissionDirectory(filepath.Dir(path)); err != nil {
			return err
		}
		if err := os.MkdirAll(path, 0o700); err != nil {
			return err
		}
		if err := rootControlledAdmissionDirectory(path); err != nil {
			return err
		}
	}
	return writeAdmissionJSON(LocalAdmissionPolicyPath, policy)
}

func installedLocalAdmission(uid uint32, executor Executor) (*LocalAdmissionManager, error) {
	if err := rootControlledAdmissionDirectory(filepath.Dir(LocalAdmissionPolicyPath)); err != nil {
		return nil, err
	}
	if err := rootControlledAdmissionDirectory(LocalAdmissionStateDir); err != nil {
		return nil, err
	}
	info, err := os.Lstat(LocalAdmissionPolicyPath)
	if err != nil {
		return nil, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != 0 || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 || info.Size() > 64*1024 {
		return nil, fmt.Errorf("local admission policy is not root controlled")
	}
	data, err := os.ReadFile(LocalAdmissionPolicyPath)
	if err != nil {
		return nil, err
	}
	var policy LocalAdmissionPolicy
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&policy); err != nil {
		return nil, err
	}
	if policy.Version != 1 || policy.UID != uid || !filepath.IsAbs(policy.RuntimeDB) {
		return nil, fmt.Errorf("invalid installed local admission policy")
	}
	configData, _ := json.Marshal(policy.Bindings)
	var configured any
	if err := json.Unmarshal(configData, &configured); err != nil {
		return nil, err
	}
	if _, err := LocalAdmissionBindings(map[string]any{"bindings": configured}); err != nil {
		return nil, err
	}
	manager := &LocalAdmissionManager{Policy: policy, Executor: executor, StatePath: localAdmissionStateFile}
	manager.Resolve = func(ctx context.Context, binding LocalAdmissionBinding) (int, error) {
		return resolveAdmissionListener(ctx, policy, binding)
	}
	return manager, nil
}

func rootControlledAdmissionDirectory(path string) error {
	for {
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || stat.Uid != 0 || !info.IsDir() || info.Mode().Perm()&0o022 != 0 {
			return fmt.Errorf("local admission directory ancestry is not root controlled")
		}
		parent := filepath.Dir(path)
		if parent == path {
			return nil
		}
		path = parent
	}
}

func resolveAdmissionListener(ctx context.Context, policy LocalAdmissionPolicy, binding LocalAdmissionBinding) (int, error) {
	iface, err := net.InterfaceByName(binding.Interface)
	if err != nil || iface.Flags&net.FlagUp == 0 {
		return 0, fmt.Errorf("home interface is unavailable")
	}
	addresses, err := iface.Addrs()
	if err != nil {
		return 0, err
	}
	onHomeNetwork := false
	for _, address := range addresses {
		ip, subnet, err := net.ParseCIDR(address.String())
		if err == nil && ip.String() == binding.HostIP && subnet.Contains(net.ParseIP(binding.SourceIP)) {
			onHomeNetwork = true
		}
	}
	if !onHomeNetwork {
		return 0, fmt.Errorf("home binding is not on the configured interface")
	}
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: policy.RuntimeDB, ReadOnly: true})
	if err != nil {
		return 0, err
	}
	defer store.Close()
	claims, err := store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{Scenario: binding.Scenario, Variant: scenarioruntime.DefaultVariant, Statuses: []string{scenarioruntime.ClaimStatusBound}})
	if err != nil {
		return 0, err
	}
	var selected *scenarioruntime.PortClaim
	for i := range claims {
		claim := &claims[i]
		if claim.PortName != binding.Listener {
			continue
		}
		if selected != nil {
			return 0, fmt.Errorf("ambiguous live listener")
		}
		selected = claim
	}
	if selected == nil || selected.Port < 1024 || selected.Port > 65535 {
		return 0, fmt.Errorf("live listener missing")
	}
	instance, err := store.GetInstance(ctx, selected.InstanceID)
	if err != nil || instance.Status != scenarioruntime.StatusRunning || instance.Variant != scenarioruntime.DefaultVariant || instance.Scenario != binding.Scenario || instance.HeartbeatDeadlineAt == nil || !instance.HeartbeatDeadlineAt.After(time.Now()) {
		return 0, fmt.Errorf("live lifecycle owner missing")
	}
	listeners := network.CaptureTCPListenerSnapshot().Listening(selected.Port)
	if !listeners.Known || !listeners.Listening || len(listeners.Listeners) == 0 {
		return 0, fmt.Errorf("listener process attribution missing")
	}
	for _, listener := range listeners.Listeners {
		info, err := os.Stat(filepath.Join("/proc", strconv.Itoa(listener.PID)))
		if err != nil {
			return 0, err
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || stat.Uid != policy.UID {
			return 0, fmt.Errorf("listener belongs to another user")
		}
		environment, err := process.ReadEnvironment(listener.PID)
		if err != nil || environment["VROOLI_SCENARIO"] != binding.Scenario || environment[scenarioruntime.EnvVariant] != scenarioruntime.DefaultVariant || environment["UI_PORT"] != strconv.Itoa(selected.Port) {
			return 0, fmt.Errorf("listener does not match installed scenario")
		}
	}
	return selected.Port, nil
}
