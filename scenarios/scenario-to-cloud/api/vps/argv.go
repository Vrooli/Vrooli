package vps

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/edge"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/reach"
)

// This file derives the typed target invocations of every plan action from
// the action's inputs. An invocation is {verb, argv}; the same builders feed
// the executor and the shell preview, so what the operator reviews is what
// runs. Nothing here composes a shell string: JSON payloads travel as
// b64:<base64url> arguments (the target verbs decode both spellings) and a
// credential value travels only on standard input.

// Identity is the durable operation identity every fenced target verb
// carries. The target refuses a fence below the highest it accepted and
// replays the receipt of an (operation, step) it already completed.
type Identity struct {
	OperationID string
	Fence       uint64
}

// CommandContext is what the argv builders read beside the action inputs.
type CommandContext struct {
	DeploymentID string
	ScenarioID   string
	Identity     Identity
	Manifest     domain.CloudManifest
}

// TargetCommand is one typed invocation attributed to a receipt step.
type TargetCommand struct {
	Step    string
	Command reach.Command
}

// JSONArgPrefix marks a base64url-encoded JSON argument; it mirrors
// internal/cli/cloudtargethandlers.JSONArgPrefix.
const JSONArgPrefix = "b64:"

// EncodeJSONArg renders a JSON document as an argv-safe argument.
func EncodeJSONArg(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return JSONArgPrefix + base64.RawURLEncoding.EncodeToString(raw), nil
}

// DecodeJSONArg is the inverse of EncodeJSONArg (used by tests and fakes).
func DecodeJSONArg(value string) ([]byte, error) {
	if !strings.HasPrefix(value, JSONArgPrefix) {
		return []byte(value), nil
	}
	return base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, JSONArgPrefix))
}

// Default verb timeouts per action, mirroring DefaultStepConfigs.
func verbTimeout(actionID string) time.Duration {
	if cfg, ok := DefaultStepConfigs[actionID]; ok {
		return cfg.CommandTimeout
	}
	return 0
}

func (cc CommandContext) fenceArgs(step string) []string {
	return []string{"--deployment", cc.DeploymentID, "--operation", cc.Identity.OperationID, "--step", step, "--fence", strconv.FormatUint(cc.Identity.Fence, 10)}
}

func csv(value string) []string {
	var out []string
	for _, part := range strings.Split(value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

// portPairs turns "api=3001,ui=3000" into sorted --port arguments.
func portArgs(ports string) []string {
	var args []string
	for _, pair := range csv(ports) {
		args = append(args, "--port", pair)
	}
	return args
}

func effectful(step string, verb string, args []string, timeout time.Duration) TargetCommand {
	return TargetCommand{Step: step, Command: reach.Command{Verb: verb, Args: append(args, "--json"), RequiredScope: "vrooli:write", Effectful: true, Timeout: timeout}}
}

func readOnly(step string, verb string, args []string, timeout time.Duration) TargetCommand {
	return TargetCommand{Step: step, Command: reach.Command{Verb: verb, Args: append(args, "--json"), RequiredScope: "vrooli:read", Timeout: timeout}}
}

// ActionCommands derives the target invocations of one action. Actions whose
// effect is not a target verb (release.deliver, credentials.provision, the
// cloud-side readiness probes) return the verbs they still need (for example
// the release listing that proves the active pointer) and nothing else.
func ActionCommands(action execplan.Action, cc CommandContext) ([]TargetCommand, error) {
	in := action.Inputs
	timeout := verbTimeout(action.ID)
	switch action.OwnerOperation {
	case execplan.OpHostPrepare:
		subject, err := EncodeJSONArg(map[string]any{"apt": map[string]any{"packages": csv(in["packages"])}})
		if err != nil {
			return nil, err
		}
		args := append([]string{"--action", "apt.packages.ensure", "--subject", subject}, cc.fenceArgs(action.ID)...)
		return []TargetCommand{effectful(action.ID, "cloud-target host repair", args, timeout)}, nil
	case execplan.OpEdgeFirewallAllow:
		var out []TargetCommand
		for _, port := range csv(in["ports"]) {
			number, err := strconv.Atoi(port)
			if err != nil {
				return nil, apierrors.Newf(apierrors.CodeInvalidRequest, "edge.firewall.allow port %q is not a number", port)
			}
			subject, err := EncodeJSONArg(map[string]any{"edge": map[string]any{"port": number}})
			if err != nil {
				return nil, err
			}
			step := action.ID + "." + port
			args := append([]string{"--action", "edge.ufw.allow", "--subject", subject}, cc.fenceArgs(step)...)
			out = append(out, effectful(step, "cloud-target host repair", args, timeout))
		}
		return out, nil
	case execplan.OpDataInventory:
		bindings := inventoryBindings(in["data_bindings"])
		var out []TargetCommand
		for _, scenario := range csv(in["scenarios"]) {
			encoded, err := EncodeJSONArg(bindings[scenario])
			if err != nil {
				return nil, err
			}
			args := []string{"--deployment", cc.DeploymentID, "--workdir", in["workdir"], "--scenario", scenario, "--bindings", encoded}
			out = append(out, readOnly(action.ID+"."+scenario, "cloud-target data inventory", args, timeout))
		}
		return out, nil
	case execplan.OpReleaseVerify:
		if err := requireReleaseID(action); err != nil {
			return nil, err
		}
		args := []string{"--deployment", cc.DeploymentID, "--release-manifest", in["release_manifest"], "--archive", in["archive"]}
		return []TargetCommand{readOnly(action.ID, "cloud-target release verify", args, timeout)}, nil
	case execplan.OpReleaseStage:
		if err := requireReleaseID(action); err != nil {
			return nil, err
		}
		args := append(cc.fenceArgs(action.ID), "--release-manifest", in["release_manifest"], "--archive", in["archive"])
		return []TargetCommand{effectful(action.ID, "cloud-target release stage", args, timeout)}, nil
	case execplan.OpDataBackup:
		args := append([]string{}, cc.fenceArgs(action.ID)...)
		specs, err := backupBindingArgs(in, cc)
		if err != nil {
			return nil, err
		}
		args = append(args, specs...)
		args = append(args, "--key-ref", in["recovery_key_ref"], "--migration-posture", in["migration_posture"], "--retention-policy", in["retention_policy"])
		if in["schema_version"] != "" {
			args = append(args, "--schema-version", in["schema_version"])
		}
		if in["configuration_digest"] != "" {
			args = append(args, "--configuration-digest", in["configuration_digest"])
		}
		return []TargetCommand{effectful(action.ID, "cloud-target data backup", args, timeout)}, nil
	case execplan.OpWorkloadStop:
		subject, err := EncodeJSONArg(map[string]any{"process": map[string]any{"scenario": in["scenario"], "workdir": in["workdir"]}})
		if err != nil {
			return nil, err
		}
		args := append([]string{"--action", "process.stop.scoped", "--subject", subject}, cc.fenceArgs(action.ID)...)
		return []TargetCommand{effectful(action.ID, "cloud-target host repair", args, timeout)}, nil
	case execplan.OpReleaseActivate:
		if err := requireReleaseID(action); err != nil {
			return nil, err
		}
		return []TargetCommand{effectful(action.ID, "cloud-target release activate", activateArgs(action.ID, in["release_id"], in, cc, false), timeout)}, nil
	case execplan.OpWorkloadStart:
		out := []TargetCommand{readOnly(action.ID+".observe", "cloud-target release list", []string{"--deployment", cc.DeploymentID}, timeout)}
		if in["release_id"] != "" {
			out = append(out, effectful(action.ID, "cloud-target release activate", activateArgs(action.ID, in["release_id"], in, cc, true), timeout))
		}
		return out, nil
	case execplan.OpConfigApply:
		args := []string{"--yes", "yes", "--environment", in["environment"]}
		if resources := strings.TrimSpace(in["resources"]); resources != "" {
			args = append(args, "--resources", resources)
		}
		if scenarios := strings.TrimSpace(in["scenarios"]); scenarios != "" {
			args = append(args, "--scenarios", scenarios)
		}
		if selection := strings.TrimSpace(in["selection_json_b64"]); selection != "" {
			args = append(args, "--selection-b64", selection)
		}
		return []TargetCommand{effectful(action.ID, "setup", args, timeout)}, nil
	case execplan.OpRuntimeStartDeps:
		var out []TargetCommand
		for _, res := range csv(in["resources"]) {
			out = append(out, effectful(action.ID+".resource."+res, "resource start", []string{res}, timeout))
		}
		for _, scen := range csv(in["scenarios"]) {
			out = append(out, effectful(action.ID+".scenario."+scen, "scenario start", []string{scen}, timeout))
		}
		return out, nil
	case execplan.OpEdgeRouteApply:
		spec, err := EdgeSpecFor(cc, in)
		if err != nil {
			return nil, err
		}
		raw, err := edge.TargetSpecJSON(spec)
		if err != nil {
			return nil, err
		}
		encoded := JSONArgPrefix + base64.RawURLEncoding.EncodeToString(raw)
		args := append(cc.fenceArgs(action.ID), "--spec", encoded)
		return []TargetCommand{effectful(action.ID, "cloud-target edge route-apply", args, timeout)}, nil
	case execplan.OpEdgeRouteRetire:
		return []TargetCommand{effectful(action.ID, "cloud-target edge route-rollback", cc.fenceArgs(action.ID), timeout)}, nil
	case execplan.OpVerifyReadiness, execplan.OpReleaseRetainPredecesor, execplan.OpArtifactsRetire:
		return []TargetCommand{readOnly(action.ID+".observe", "cloud-target release list", []string{"--deployment", cc.DeploymentID}, timeout)}, nil
	case execplan.OpReleaseDeliver, execplan.OpCredentialsProvision, execplan.OpGrantsRevoke, execplan.OpDataRetire, execplan.OpInputResumeHandoff:
		return nil, nil
	}
	return nil, apierrors.Newf(apierrors.CodeUnsupportedCapability, "No target invocation for owner operation %q", action.OwnerOperation)
}

func requireReleaseID(action execplan.Action) error {
	if strings.TrimSpace(action.Inputs["release_id"]) == "" {
		return apierrors.New(apierrors.CodeReleaseVerificationFailed, "the bundle is not part of a built release; build the release artifact set (release-manifest.json beside the bundle) before staging").
			WithDetail("reason", "release_manifest_missing").WithDetail("action", action.ID)
	}
	return nil
}

func activateArgs(step, releaseID string, in map[string]string, cc CommandContext, restart bool) []string {
	args := append(cc.fenceArgs(step), "--release", releaseID)
	strategy := in["strategy"]
	if strategy == "" {
		strategy = execplan.StrategyMaintenance
	}
	args = append(args, "--strategy", strategy)
	for _, scenario := range csv(in["scenarios"]) {
		args = append(args, "--scenario", scenario)
	}
	args = append(args, portArgs(in["ports"])...)
	for _, spec := range csv(in["data_bindings"]) {
		args = append(args, "--data-binding", spec)
	}
	for _, carry := range csv(in["legacy_carry"]) {
		args = append(args, "--legacy-carry", carry)
	}
	if in["legacy_root"] != "" {
		args = append(args, "--legacy-root", in["legacy_root"])
	}
	if restart {
		args = append(args, "--restart")
	}
	return args
}

// inventoryBindings groups <id>=<scenario>/<path> specs per scenario in the
// shape `data inventory --bindings` accepts.
func inventoryBindings(specs string) map[string][]map[string]string {
	out := map[string][]map[string]string{}
	for _, spec := range csv(specs) {
		id, location, ok := strings.Cut(spec, "=")
		if !ok {
			continue
		}
		scenario, rel, ok := strings.Cut(location, "/")
		if !ok {
			continue
		}
		out[scenario] = append(out[scenario], map[string]string{"id": id, "path": rel})
	}
	for scenario := range out {
		sort.Slice(out[scenario], func(i, j int) bool { return out[scenario][i]["id"] < out[scenario][j]["id"] })
	}
	return out
}

// backupBindingArgs renders the declared persistent data as the typed
// `--binding` arguments of `cloud-target data backup`.
func backupBindingArgs(in map[string]string, cc CommandContext) ([]string, error) {
	workdir := ""
	if cc.Manifest.Target.VPS != nil {
		workdir = strings.TrimSpace(cc.Manifest.Target.VPS.Workdir)
	}
	var args []string
	for _, spec := range strings.Split(in["binding_specs"], ";") {
		if spec = strings.TrimSpace(spec); spec == "" {
			continue
		}
		fields := strings.SplitN(spec, ":", 5)
		if len(fields) < 5 {
			return nil, apierrors.Newf(apierrors.CodeInvalidRequest, "data.backup binding spec %q is not <id>:<kind>:<locator>:<owner>:<migration_owner>", spec)
		}
		binding, err := engineBinding(fields[0], fields[1], fields[2], fields[3], fields[4], workdir)
		if err != nil {
			return nil, err
		}
		encoded, err := EncodeJSONArg(binding)
		if err != nil {
			return nil, err
		}
		args = append(args, "--binding", encoded)
	}
	if len(args) == 0 {
		return nil, apierrors.New(apierrors.CodeInvalidRequest, "data.backup declares no bindings")
	}
	return args, nil
}

// engineBinding mirrors backup.ResolveBindings for the binding syntaxes the
// closure declares.
func engineBinding(id, kind, locator, owner, migrationOwner, workdir string) (map[string]any, error) {
	binding := map[string]any{"id": id, "owner": owner, "migration_owner": migrationOwner}
	switch kind {
	case "database", "schema":
		binding["kind"], binding["provider"], binding["locator"] = domain.DataBindingKindSQL, domain.BackupProviderPostgres, locator
	case "sqlite":
		binding["kind"], binding["provider"], binding["locator"] = domain.DataBindingKindSQL, domain.BackupProviderSQLite, locator
	case "dir":
		if workdir == "" || owner == "" {
			return nil, apierrors.Newf(apierrors.CodeInvalidRequest, "persistent data %s: dir binding needs a workdir and an owner", id)
		}
		binding["kind"], binding["provider"], binding["locator"] = domain.DataBindingKindFiles, domain.BackupProviderObjectStore, strings.TrimRight(workdir, "/")+"/scenarios/"+owner+"/"+strings.Trim(locator, "/")
	case "bucket":
		binding["kind"], binding["provider"], binding["locator"] = domain.DataBindingKindFiles, domain.BackupProviderObjectStore, locator
	case "application":
		binding["kind"], binding["provider"], binding["locator"] = domain.DataBindingKindApplication, domain.BackupProviderApplicationHooks, locator
	default:
		return nil, apierrors.Newf(apierrors.CodeInvalidRequest, "persistent data %s: binding kind %q is not supported", id, kind)
	}
	return binding, nil
}

// EdgeSpecFor derives the owner-scoped edge spec of the deployment from the
// action inputs and the manifest: one route per public listener the
// manifest exposes through the domain, ACME options from the manifest.
func EdgeSpecFor(cc CommandContext, in map[string]string) (domain.EdgeSpec, error) {
	domainName := strings.TrimSpace(in["domain"])
	if domainName == "" {
		return domain.EdgeSpec{}, apierrors.New(apierrors.CodeInvalidRequest, "edge.route.apply requires a domain")
	}
	port, err := strconv.Atoi(strings.TrimSpace(in["upstream_port"]))
	if err != nil || port <= 0 {
		return domain.EdgeSpec{}, apierrors.Newf(apierrors.CodeInvalidRequest, "edge.route.apply upstream_port %q is not a port", in["upstream_port"])
	}
	environment := strings.TrimSpace(cc.Manifest.Edge.ACMEEnvironment)
	if environment == "" {
		environment = "production"
	}
	routeHost := strings.TrimSpace(in["route_host"])
	if routeHost == "" {
		routeHost = domainName
	}
	listenerID := strings.TrimSpace(in["listener_id"])
	if listenerID == "" {
		listenerID = cc.ScenarioID + "/ui"
	}
	spec := domain.EdgeSpec{
		SchemaVersion:   "1",
		DeploymentID:    cc.DeploymentID,
		ScenarioID:      cc.ScenarioID,
		Domain:          domainName,
		Routes:          []domain.EdgeRoute{{Host: routeHost, UpstreamPort: port, ListenerID: listenerID}},
		FirewallAllow:   []int{80, 443},
		ACMEEnvironment: environment,
		ACMEEmail:       strings.TrimSpace(in["tls_email"]),
	}
	digest, err := edge.Digest(spec)
	if err != nil {
		return domain.EdgeSpec{}, err
	}
	spec.Digest = digest
	return spec, nil
}

// ShellPreviewFor renders the display command of one action from its typed
// invocations. It is derived from the argv and never executed.
func ShellPreviewFor(action execplan.Action, cc CommandContext, remote func(argv []string) string, scp func(local, remotePath string) string) string {
	in := action.Inputs
	switch action.OwnerOperation {
	case execplan.OpReleaseDeliver:
		lines := []string{scp(in["artifact_path"], in["destination"])}
		if in["release_id"] != "" {
			lines = append(lines, scp("<release>/release-manifest.json", in["release_manifest"]), scp("<release>/vrooli-<goos>-<goarch>", in["native_cli"]))
		}
		return strings.Join(lines, " && ")
	case execplan.OpCredentialsProvision:
		return "(credential descriptors " + in["descriptors"] + " are materialised through the credential authority; each value travels on the ingest verb's standard input, never in argv)"
	case execplan.OpGrantsRevoke:
		return "(credential grants for " + in["descriptors"] + " are revoked through the credential authority)"
	case execplan.OpDataRetire:
		return "(persistent data " + in["bindings"] + ": " + in["retention_policy"] + "; no target verb deletes data)"
	case execplan.OpInputResumeHandoff:
		return "(resume onboarding: " + in["reference"] + ")"
	}
	commands, err := ActionCommands(action, cc)
	if err != nil {
		return "(" + err.Error() + ")"
	}
	lines := make([]string, 0, len(commands))
	for _, command := range commands {
		lines = append(lines, remote(command.Command.Argv()[1:]))
	}
	if action.OwnerOperation == execplan.OpVerifyReadiness {
		for _, check := range csv(in["checks"]) {
			switch check {
			case "https", "public":
				lines = append(lines, fmt.Sprintf("curl -fsS https://%s/health", in["domain"]))
			case "origin":
				lines = append(lines, fmt.Sprintf("curl -fsS --resolve %s:443:%s https://%s/health", in["domain"], in["host"], in["domain"]))
			}
		}
	}
	return strings.Join(lines, " && ")
}
