package privilegebroker

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

const (
	LocalAdmissionPolicyPath = "/etc/vrooli/local-service-admission.json"
	LocalAdmissionStateDir   = "/var/lib/vrooli/local-service-admission"
	localAdmissionStateFile  = LocalAdmissionStateDir + "/state.json"
)

// LocalAdmissionBinding is operator configuration, installed by project setup.
// An address scopes packets; it does not authenticate a device or application.
type LocalAdmissionBinding struct {
	ID        string `json:"id"`
	Scenario  string `json:"scenario"`
	Listener  string `json:"listener"`
	SourceIP  string `json:"source_ip"`
	HostIP    string `json:"host_ip"`
	Interface string `json:"interface"`
}

type LocalAdmissionPolicy struct {
	Version   int                     `json:"version"`
	UID       uint32                  `json:"uid"`
	RuntimeDB string                  `json:"runtime_db"`
	Bindings  []LocalAdmissionBinding `json:"bindings"`
}

type LocalAdmissionSubject struct {
	Binding string `json:"binding"`
}

type LocalAdmissionReport struct {
	Binding       string `json:"binding"`
	BindingDigest string `json:"binding_digest"`
	Enabled       bool   `json:"enabled"`
	Port          int    `json:"port,omitempty"`
	ManagedRules  int    `json:"managed_rules"`
	Reason        string `json:"reason,omitempty"`
}

var admissionID = regexp.MustCompile(`^[a-z][a-z0-9-]{0,47}$`)
var admissionInterface = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{0,14}$`)

func (b LocalAdmissionBinding) Validate() error {
	if !admissionID.MatchString(b.ID) || !admissionID.MatchString(b.Scenario) || b.Listener != "ui" || !admissionInterface.MatchString(b.Interface) {
		return fmt.Errorf("invalid local-service binding identity, UI listener or interface")
	}
	for _, address := range []string{b.SourceIP, b.HostIP} {
		ip := net.ParseIP(address)
		if ip == nil || ip.To4() == nil || !ip.IsPrivate() || ip.String() != address {
			return fmt.Errorf("local-service binding requires canonical private IPv4 addresses")
		}
	}
	if b.SourceIP == b.HostIP {
		return fmt.Errorf("local-service source must be a different device")
	}
	return nil
}

func (b LocalAdmissionBinding) Digest() string {
	data, _ := json.Marshal(b)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// LocalAdmissionBindings validates exactly the same config at setup and at the
// safeguard boundary. No port, command, range or filesystem path is accepted.
func LocalAdmissionBindings(config map[string]any) ([]LocalAdmissionBinding, error) {
	for key := range config {
		if key != "bindings" {
			return nil, fmt.Errorf("unknown local-service parameter %q", key)
		}
	}
	var document struct {
		Bindings []LocalAdmissionBinding `json:"bindings"`
	}
	data, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return nil, err
	}
	if len(document.Bindings) > 16 {
		return nil, fmt.Errorf("at most sixteen local-service bindings are allowed")
	}
	seen := map[string]bool{}
	for _, binding := range document.Bindings {
		if err := binding.Validate(); err != nil {
			return nil, err
		}
		if seen[binding.ID] {
			return nil, fmt.Errorf("duplicate local-service binding")
		}
		seen[binding.ID] = true
	}
	return document.Bindings, nil
}

type localAdmissionState struct {
	Disabled map[string]bool `json:"disabled"`
}
type admissionRule struct {
	Interface, Source, Host, Tag string
	Port                         int
}

// LocalAdmissionManager owns convergence for setup-installed bindings. Calls
// serialize with background convergence so revoke cannot race an automatic add.
type LocalAdmissionManager struct {
	mu        sync.Mutex
	Policy    LocalAdmissionPolicy
	Executor  Executor
	Resolve   func(context.Context, LocalAdmissionBinding) (int, error)
	StatePath string
}

func (m *LocalAdmissionManager) tag(id string) string {
	return "vrooli-local-v1-" + strconv.FormatUint(uint64(m.Policy.UID), 10) + "-" + id
}

func (m *LocalAdmissionManager) loadState() (localAdmissionState, error) {
	state := localAdmissionState{Disabled: map[string]bool{}}
	data, err := os.ReadFile(m.StatePath)
	if os.IsNotExist(err) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return state, err
	}
	if state.Disabled == nil {
		state.Disabled = map[string]bool{}
	}
	return state, nil
}

func writeAdmissionJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".local-admission-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err := f.Chmod(0o600); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(append(data, '\n')); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	directory, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func (m *LocalAdmissionManager) status(ctx context.Context) (bool, []admissionRule, error) {
	out, err := m.Executor.Run(ctx, "ufw", "status", "numbered")
	if err != nil {
		return false, nil, err
	}
	return parseAdmissionRules(string(out), "vrooli-local-v1-"+strconv.FormatUint(uint64(m.Policy.UID), 10)+"-")
}

// Parse only exact, tagged IPv4 allow rules. Substring matches must not turn a
// different address/port or a broad operator rule into verified admission.
var admissionRuleLine = regexp.MustCompile(`^\[\s*[0-9]+\]\s+([0-9.]+)\s+([0-9]+)/tcp on ([A-Za-z0-9_.:-]+)\s+ALLOW IN\s+([0-9.]+)\s+# (vrooli-local-v1-[a-z0-9-]+)\s*$`)

func parseAdmissionRules(text, prefix string) (bool, []admissionRule, error) {
	if strings.Contains(text, "Status: inactive") {
		return false, nil, nil
	}
	if !strings.Contains(text, "Status: active") {
		return false, nil, fmt.Errorf("firewall status is unknown")
	}
	var rules []admissionRule
	for _, line := range strings.Split(text, "\n") {
		if !strings.Contains(line, "# "+prefix) {
			continue
		}
		match := admissionRuleLine.FindStringSubmatch(strings.TrimSpace(line))
		if match == nil {
			return true, nil, fmt.Errorf("managed firewall rule has an unexpected shape")
		}
		port, _ := strconv.Atoi(match[2])
		if port < 1 || port > 65535 {
			return true, nil, fmt.Errorf("invalid managed port")
		}
		for _, address := range []string{match[1], match[4]} {
			ip := net.ParseIP(address)
			if ip == nil || ip.To4() == nil || !ip.IsPrivate() || ip.String() != address {
				return true, nil, fmt.Errorf("invalid managed address")
			}
		}
		rules = append(rules, admissionRule{Host: match[1], Port: port, Interface: match[3], Source: match[4], Tag: match[5]})
	}
	return true, rules, nil
}

func admissionArgs(rule admissionRule, remove bool) []string {
	args := []string{"allow", "in", "on", rule.Interface, "from", rule.Source, "to", rule.Host, "port", strconv.Itoa(rule.Port), "proto", "tcp", "comment", rule.Tag}
	if remove {
		args = append([]string{"--force", "delete"}, args...)
	}
	return args
}

func rulesWithTag(rules []admissionRule, tag string) []admissionRule {
	var result []admissionRule
	for _, rule := range rules {
		if rule.Tag == tag {
			result = append(result, rule)
		}
	}
	return result
}

func (m *LocalAdmissionManager) Execute(ctx context.Context, action, id string) Result {
	m.mu.Lock()
	defer m.mu.Unlock()
	if action != ActionLocalAdmissionInspect && action != ActionLocalAdmissionEnsure && action != ActionLocalAdmissionRevoke {
		return NewFailure("", action, "action_not_allowed")
	}
	return m.execute(ctx, action, id)
}

func (m *LocalAdmissionManager) execute(ctx context.Context, action, id string) Result {
	var binding *LocalAdmissionBinding
	for i := range m.Policy.Bindings {
		if m.Policy.Bindings[i].ID == id {
			binding = &m.Policy.Bindings[i]
			break
		}
	}
	if binding == nil {
		return NewFailure("", action, "local_binding_not_installed")
	}
	report := &LocalAdmissionReport{Binding: id, BindingDigest: binding.Digest()}
	result := Result{Status: "pending", LocalAdmission: report}
	state, err := m.loadState()
	if err != nil {
		return NewFailure("", action, "local_state_unavailable")
	}
	report.Enabled = !state.Disabled[id]
	active, rules, err := m.status(ctx)
	if err != nil {
		return NewFailure("", action, "local_firewall_unavailable")
	}
	result.Evidence = Evidence{Available: true, Active: active}
	managed := rulesWithTag(rules, m.tag(id))
	report.ManagedRules = len(managed)
	if action == ActionLocalAdmissionRevoke {
		if !state.Disabled[id] {
			state.Disabled[id] = true
			if err := writeAdmissionJSON(m.StatePath, state); err != nil {
				return NewFailure("", action, "local_state_write_failed")
			}
		}
		report.Enabled = false
		return m.remove(ctx, result, managed)
	}
	if action == ActionLocalAdmissionEnsure && !report.Enabled {
		delete(state.Disabled, id)
		if err := writeAdmissionJSON(m.StatePath, state); err != nil {
			return NewFailure("", action, "local_state_write_failed")
		}
		report.Enabled = true
	}
	if !report.Enabled {
		report.Reason = "revoked"
		return result
	}
	port, err := m.Resolve(ctx, *binding)
	if err != nil {
		report.Reason = "live_listener_unverified"
		if action == ActionLocalAdmissionEnsure {
			return m.remove(ctx, result, managed)
		}
		return result
	}
	report.Port = port
	want := admissionRule{Interface: binding.Interface, Source: binding.SourceIP, Host: binding.HostIP, Port: port, Tag: m.tag(id)}
	if !active {
		result.Status = "verified"
		report.Reason = "firewall_inactive"
		return result
	}
	found := false
	for _, rule := range managed {
		if rule == want {
			found = true
		}
	}
	result.Evidence.RuleFound, result.Evidence.Managed = found, found
	if action == ActionLocalAdmissionInspect {
		if found && len(managed) == 1 {
			result.Status = "verified"
		} else {
			report.Reason = "managed_rule_missing_or_stale"
		}
		return result
	}
	if !found {
		output, err := m.Executor.Run(ctx, "ufw", admissionArgs(want, false)...)
		if err != nil {
			failure := NewFailure("", action, "local_rule_apply_failed")
			failure.Evidence.Detail = boundedDetail(output)
			return failure
		}
		_, after, err := m.status(ctx)
		if err != nil || len(rulesWithTag(after, m.tag(id))) == 0 {
			failure := NewFailure("", action, "local_rule_not_verified")
			failure.Evidence.Detail = boundedDetail(append([]byte("Apply command completed; managed-rule readback did not verify. "), output...))
			return failure
		}
		verified := false
		for _, rule := range after {
			if rule == want {
				verified = true
			}
		}
		if !verified {
			return NewFailure("", action, "local_rule_not_verified")
		}
		result.Changed = true
	}
	for _, rule := range managed {
		if rule != want {
			if _, err := m.Executor.Run(ctx, "ufw", admissionArgs(rule, true)...); err != nil {
				return NewFailure("", action, "local_stale_rule_remove_failed")
			}
			result.Changed = true
		}
	}
	_, after, err := m.status(ctx)
	remaining := rulesWithTag(after, m.tag(id))
	if err != nil || len(remaining) != 1 || remaining[0] != want {
		return NewFailure("", action, "local_convergence_not_verified")
	}
	report.ManagedRules = 1
	result.Evidence.RuleFound = true
	result.Evidence.Managed = true
	result.Status = "verified"
	if result.Changed {
		result.Status = "changed"
	}
	return result
}

func (m *LocalAdmissionManager) remove(ctx context.Context, result Result, rules []admissionRule) Result {
	for _, rule := range rules {
		if _, err := m.Executor.Run(ctx, "ufw", admissionArgs(rule, true)...); err != nil {
			return NewFailure("", "", "local_rule_remove_failed")
		}
		result.Changed = true
	}
	_, after, err := m.status(ctx)
	if err != nil || len(rulesWithTag(after, m.tag(result.LocalAdmission.Binding))) != 0 {
		return NewFailure("", "", "local_revocation_not_verified")
	}
	result.LocalAdmission.ManagedRules = 0
	if result.LocalAdmission.Reason == "" {
		result.Status = "verified"
	}
	return result
}

// Reconcile is owned by the installed broker, not a Portal watchdog. It also
// removes tagged orphan rules after setup removes a previously installed binding.
func (m *LocalAdmissionManager) Reconcile(ctx context.Context) []Result {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, err := m.loadState()
	if err != nil {
		return []Result{NewFailure("", ActionLocalAdmissionEnsure, "local_state_unavailable")}
	}
	var results []Result
	for _, binding := range m.Policy.Bindings {
		action := ActionLocalAdmissionEnsure
		if state.Disabled[binding.ID] {
			action = ActionLocalAdmissionRevoke
		}
		result := m.execute(ctx, action, binding.ID)
		result.Action = action
		results = append(results, result)
	}
	_, rules, err := m.status(ctx)
	if err != nil {
		return append(results, NewFailure("", ActionLocalAdmissionEnsure, "local_firewall_unavailable"))
	}
	for _, rule := range rules {
		installed := false
		for _, binding := range m.Policy.Bindings {
			if rule.Tag == m.tag(binding.ID) {
				installed = true
			}
		}
		if !installed {
			result := Result{Action: ActionLocalAdmissionRevoke, Status: "changed", Changed: true, LocalAdmission: &LocalAdmissionReport{Binding: strings.TrimPrefix(rule.Tag, m.tag(""))}}
			if _, err := m.Executor.Run(ctx, "ufw", admissionArgs(rule, true)...); err != nil {
				result.Status = "failed"
				result.Code = "local_orphan_remove_failed"
			}
			_, after, err := m.status(ctx)
			if err != nil || len(rulesWithTag(after, rule.Tag)) != 0 {
				result.Status = "failed"
				result.Code = "local_orphan_remove_unverified"
			}
			results = append(results, result)
		}
	}
	return results
}
