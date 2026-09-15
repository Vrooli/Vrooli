// Package cloudtargethandlers binds `vrooli cloud-target` to the target-local
// deployment owner in internal/cloudtarget. Every verb prints JSON; a typed
// refusal prints `{"error":{"code",...}}` and exits with the owner's code.
package cloudtargethandlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	climanifest "github.com/vrooli/vrooli/cli"
	"github.com/vrooli/vrooli/internal/cli/manifestdispatch"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/cloudtarget"
	"github.com/vrooli/vrooli/internal/privilegebroker"
	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/packages/recoverypoint"
)

const groupName = "cloud-target"

var groupCommandNames = map[string][]string{
	"receipt":    {"get"},
	"release":    {"verify", "stage", "activate", "rollback", "list", "prune"},
	"data":       {"inventory", "backup", "restore", "verify"},
	"host":       {"observe", "repair"},
	"credential": {"ingest", "acknowledge", "revoke"},
	"edge":       {"route-apply", "route-rollback", "route-status"},
}

var groupOrder = []string{"receipt", "release", "data", "host", "credential", "edge"}

// RegisteredCommandPaths returns the leaf paths bound by this handler.
func RegisteredCommandPaths() []string {
	paths := make([]string, 0, 8)
	for _, group := range groupOrder {
		for _, name := range groupCommandNames[group] {
			paths = append(paths, groupName+" "+group+" "+name)
		}
	}
	return paths
}

// Deps are the seams the verbs depend on. Zero values select production
// wiring: the operator runtime home, the broker socket, and the OS runner.
type Deps struct {
	Store  func() (*cloudtarget.Store, error)
	Broker func() cloudtarget.BrokerClient
	Runner shell.Runner
	// Data seams the recovery-point verbs run through; zero values select
	// the credential authority for keys and PostgreSQL credentials and the
	// built-in providers.
	Data cloudtarget.DataDeps
	// Credentials seams the credential verbs run through; zero values select
	// the host credential authority, os.Getenv and the command's stdin.
	Credentials cloudtarget.CredentialDeps
}

func (d Deps) dataDeps() cloudtarget.DataDeps { return d.Data }

func (d Deps) store() (*cloudtarget.Store, error) {
	if d.Store != nil {
		return d.Store()
	}
	return cloudtarget.DefaultStore()
}

func (d Deps) broker() cloudtarget.BrokerClient {
	if d.Broker != nil {
		return d.Broker()
	}
	return privilegebroker.NewClient()
}

type service struct {
	run func([]string) error
}

// RootHandler dispatches `vrooli cloud-target`.
func RootHandler[C any](deps rootcli.HandlerDeps[C]) rootcli.Handler[C] {
	return Handler(deps, Deps{})
}

// Handler dispatches `vrooli cloud-target` with explicit seams.
func Handler[C any](deps rootcli.HandlerDeps[C], verbs Deps) rootcli.Handler[C] {
	return rootcli.BindService(deps.Stdout,
		func(C) (cliout.Format, error) { return cliout.FormatHuman, nil },
		func(ctx C, _ cliout.Format) (service, error) {
			commandCtx := &rootcli.CommandContext{Root: deps.Root(ctx), Globals: deps.Globals(ctx), Stdout: deps.Stdout(ctx), Stderr: deps.Stderr(ctx), Context: rootcli.ResolveOperationContext(deps, ctx)}
			return service{run: func(args []string) error {
				if len(args) == 0 || manifestdispatch.WantsHelp(args) {
					return writeHelp(deps.Stdout(ctx))
				}
				return Run(commandCtx, verbs, args)
			}}, nil
		},
		func(_ C, args []string) ([]string, error) { return args, nil },
		func(svc service, args []string) (struct{}, error) { return struct{}{}, svc.run(args) },
		func(io.Writer, cliout.Format, struct{}) error { return nil },
	)
}

// Run parses args through the manifest and dispatches one verb.
func Run(ctx *rootcli.CommandContext, verbs Deps, args []string) error {
	args = collapseEdgeRoute(args)
	subgroups := make([]cliapp.SubcommandGroup, 0, len(groupOrder))
	for _, group := range groupOrder {
		loaded, err := cliapp.LoadFromManifest(climanifest.Bytes(), groupName+"/"+group, bindings(ctx, verbs, group))
		if err != nil {
			return err
		}
		subgroups = append(subgroups, loaded)
	}
	core := cliapp.NewApp(cliapp.AppOptions{Name: "vrooli " + groupName, SubcommandGroups: subgroups})
	return core.RunWithWriters(manifestdispatch.WithJSON(args, ctx.Globals.JSON), ctx.Stdout, ctx.Stderr)
}

// collapseEdgeRoute lets the spelled form `edge route apply` reach the
// manifest command `edge route-apply` (the manifest grammar is one group
// level deep beneath cloud-target).
func collapseEdgeRoute(args []string) []string {
	if len(args) >= 3 && args[0] == "edge" && args[1] == "route" {
		return append([]string{"edge", "route-" + args[2]}, args[3:]...)
	}
	return args
}

func bindings(ctx *rootcli.CommandContext, verbs Deps, group string) map[string]func(cliapp.RunContext) error {
	out := map[string]func(cliapp.RunContext) error{}
	for _, name := range groupCommandNames[group] {
		verb := group + " " + name
		out[name] = func(runCtx cliapp.RunContext) error {
			return dispatch(ctx, verbs, verb, runCtx)
		}
	}
	return out
}

func writeHelp(out io.Writer) error {
	_, err := io.WriteString(out, strings.Join([]string{
		"Usage: vrooli cloud-target <receipt|release|data|host|credential|edge> <verb> [flags]",
		"  vrooli cloud-target receipt get --deployment <id> --operation <id> --step <id>",
		"  vrooli cloud-target release verify --deployment <id> --release-manifest <path> --archive <path>",
		"  vrooli cloud-target release stage --deployment <id> --operation <id> --step <id> --fence <n> --release-manifest <path> --archive <path>",
		"  vrooli cloud-target release activate --deployment <id> --operation <id> --step <id> --fence <n> --release <digest> [--strategy side_by_side|maintenance] [--scenario <id>]... [--port <name>=<n>]... [--data-binding <id>=<scenario>/<path>]... [--legacy-carry <scenario>/<path>]... [--legacy-root <workdir>] [--restart]",
		"  vrooli cloud-target release rollback --deployment <id> --operation <id> --step <id> --fence <n> --to <digest> [--strategy ...]",
		"  vrooli cloud-target release list --deployment <id>",
		"  vrooli cloud-target release prune --deployment <id> --release <digest>...   (refuses active, previous and staging releases)",
		"  vrooli cloud-target data inventory --deployment <id> --workdir <path> --scenario <id> [--bindings <json>]",
		"  vrooli cloud-target data backup --deployment <id> --operation <id> --step <id> --fence <n> --binding <json>... --key-ref <logical_id:field> [--schema-version --configuration-digest --credential-version-ref... --provider --provider-ref --retention-policy --migration-posture]",
		"  vrooli cloud-target data restore --deployment <id> --operation <id> --step <id> --fence <n> --recovery-point <id> --into <binding>=<locator>... [--binding <id>]...",
		"  vrooli cloud-target data verify --deployment <id> --recovery-point <id> [--open] [--expect <json>]",
		"  vrooli cloud-target host observe --kind <typed observation> [--arg <semantic argument>]...",
		"  vrooli cloud-target host repair --action <privilege-broker action> [--subject <json>] [--deployment --operation --step --fence]",
		"  vrooli cloud-target edge route apply --deployment <id> --operation <id> --step <id> --fence <n> (--spec <json> | --spec-file <path>)",
		"  vrooli cloud-target edge route rollback --deployment <id> --operation <id> --step <id> --fence <n>",
		"  vrooli cloud-target edge route status --deployment <id>",
		"  vrooli cloud-target credential ingest --deployment <id> --binding <id> --version <n> --operation <id> --step <id> --fence <n> [--grant <id> --from-env <VAR> --logical-id --field --content-ref]  (sealed payload on stdin)",
		"  vrooli cloud-target credential acknowledge --deployment <id> --binding <id> --version <n> --consumer <ref> --operation <id> --step <id> --fence <n>",
		"  vrooli cloud-target credential revoke --deployment <id> --binding <id> --version <n> --operation <id> --step <id> --fence <n>",
		"",
		"Every effectful verb is fenced and receipted per (operation, step); a re-run replays its receipt.",
		"JSON flags (--subject, --bindings, --binding, --spec) accept raw JSON or b64:<base64url JSON> for argv-only transports.",
		"Exit codes: 0 ok, 1 failed, 2 refused (stale fence, mismatch, not eligible).",
		"",
	}, "\n"))
	return err
}

func dispatch(ctx *rootcli.CommandContext, verbs Deps, verb string, runCtx cliapp.RunContext) error {
	out := runCtx.Stdout()
	result, err := execute(ctx, verbs, verb, runCtx)
	if result != nil {
		if writeErr := cliout.WriteJSONValue(out, result); writeErr != nil {
			return writeErr
		}
	}
	if err == nil {
		return nil
	}
	typed := cloudtarget.AsError(err)
	if result == nil {
		if writeErr := cliout.WriteJSONValue(out, map[string]any{"error": typed}); writeErr != nil {
			return writeErr
		}
	}
	return rootcli.ExitCodeError{Code: typed.ExitCode(), Message: typed.Error(), Silent_: true}
}

// execute runs one verb and returns the JSON value to print. When a verb
// produced a receipt before failing, both the receipt and the error are
// returned so the caller prints the receipt and exits with the typed code.
func execute(ctx *rootcli.CommandContext, verbs Deps, verb string, runCtx cliapp.RunContext) (any, error) {
	store, err := verbs.store()
	if err != nil {
		return nil, err
	}
	operationCtx := ctx.OperationContext()
	switch verb {
	case "receipt get":
		receipt, err := store.ReadReceipt(runCtx.Flag("deployment"), runCtx.Flag("operation"), runCtx.Flag("step"))
		if err != nil {
			return nil, err
		}
		return receipt, nil
	case "release verify":
		if _, err := store.DeploymentDir(runCtx.Flag("deployment")); err != nil {
			return nil, err
		}
		report, _, err := cloudtarget.Verify(operationCtx, cloudtarget.VerifyRequest{ManifestPath: runCtx.Flag("release-manifest"), ArchivePath: runCtx.Flag("archive")})
		if err != nil {
			return map[string]any{"report": report, "error": cloudtarget.AsError(err)}, err
		}
		return map[string]any{"report": report}, nil
	case "release stage":
		effect, err := effectFromFlags(runCtx)
		if err != nil {
			return nil, err
		}
		return withReceipt(store.Stage(operationCtx, cloudtarget.StageRequest{Effect: effect, Verify: cloudtarget.VerifyRequest{ManifestPath: runCtx.Flag("release-manifest"), ArchivePath: runCtx.Flag("archive")}}))
	case "release activate":
		effect, err := effectFromFlags(runCtx)
		if err != nil {
			return nil, err
		}
		ports, err := parsePortFlags(runCtx.FlagValues("port"))
		if err != nil {
			return nil, err
		}
		dataBindings, err := cloudtarget.ParseDataBindingFlags(runCtx.FlagValues("data-binding"))
		if err != nil {
			return nil, err
		}
		carry, err := cloudtarget.ParseLegacyCarryFlags(runCtx.FlagValues("legacy-carry"))
		if err != nil {
			return nil, err
		}
		return withReceipt(store.Activate(operationCtx, cloudtarget.ActivateRequest{
			Effect: effect, Release: runCtx.Flag("release"), Strategy: runCtx.Flag("strategy"), Scenarios: runCtx.FlagValues("scenario"),
			Activator: cloudtarget.ScenarioRestartActivator{Runner: verbs.Runner}, Ports: ports, DataBindings: dataBindings, LegacyCarry: carry,
			LegacyRoot: strings.TrimSpace(runCtx.Flag("legacy-root")), RestartIfActive: runCtx.BoolFlag("restart"),
		}))
	case "release rollback":
		effect, err := effectFromFlags(runCtx)
		if err != nil {
			return nil, err
		}
		ports, err := parsePortFlags(runCtx.FlagValues("port"))
		if err != nil {
			return nil, err
		}
		dataBindings, err := cloudtarget.ParseDataBindingFlags(runCtx.FlagValues("data-binding"))
		if err != nil {
			return nil, err
		}
		return withReceipt(store.Rollback(operationCtx, cloudtarget.RollbackRequest{Effect: effect, To: runCtx.Flag("to"), Strategy: runCtx.Flag("strategy"), Scenarios: runCtx.FlagValues("scenario"), Activator: cloudtarget.ScenarioRestartActivator{Runner: verbs.Runner}, Ports: ports, DataBindings: dataBindings}))
	case "release list":
		listing, err := store.ListReleases(runCtx.Flag("deployment"))
		if err != nil {
			return nil, err
		}
		return listing, nil
	case "release prune":
		report, err := store.PruneReleases(cloudtarget.PruneRequest{DeploymentID: runCtx.Flag("deployment"), Releases: runCtx.FlagValues("release")})
		if err != nil {
			return map[string]any{"report": report, "error": cloudtarget.AsError(err)}, err
		}
		return map[string]any{"report": report}, nil
	case "data inventory":
		rawBindings, err := jsonFlag(runCtx.Flag("bindings"))
		if err != nil {
			return nil, err
		}
		bindingsList, err := cloudtarget.ParseBindings(rawBindings)
		if err != nil {
			return nil, err
		}
		report, err := cloudtarget.Inventory(cloudtarget.InventoryRequest{DeploymentID: runCtx.Flag("deployment"), Workdir: runCtx.Flag("workdir"), Scenario: runCtx.Flag("scenario"), Bindings: bindingsList})
		if err != nil {
			return nil, err
		}
		return report, nil
	case "data backup":
		effect, err := effectFromFlags(runCtx)
		if err != nil {
			return nil, err
		}
		rawBindings, err := jsonFlags(runCtx.FlagValues("binding"))
		if err != nil {
			return nil, err
		}
		bindingsList, err := cloudtarget.ParseDataBindings(rawBindings)
		if err != nil {
			return nil, err
		}
		manifest, result, err := store.DataBackup(operationCtx, cloudtarget.DataBackupRequest{
			Effect: effect, RecoveryPointID: runCtx.Flag("recovery-point"), RecoveryPointDir: runCtx.Flag("recovery-point-dir"), Bindings: bindingsList,
			Refs:   recoverypoint.Refs{SchemaVersion: runCtx.Flag("schema-version"), ConfigurationDigest: runCtx.Flag("configuration-digest"), CredentialVersionRefs: runCtx.FlagValues("credential-version-ref")},
			KeyRef: runCtx.Flag("key-ref"), Provider: runCtx.Flag("provider"), ProviderRef: runCtx.Flag("provider-ref"), RetentionPolicy: runCtx.Flag("retention-policy"), MigrationPosture: runCtx.Flag("migration-posture"),
			Deps: verbs.dataDeps(),
		})
		value, err := withReceipt(result, err)
		if value != nil && manifest.ID != "" {
			value.(map[string]any)["recovery_point"] = manifest
		}
		return value, err
	case "data restore":
		effect, err := effectFromFlags(runCtx)
		if err != nil {
			return nil, err
		}
		into, err := cloudtarget.ParseRestoreTargets(runCtx.FlagValues("into"))
		if err != nil {
			return nil, err
		}
		report, result, err := store.DataRestore(operationCtx, cloudtarget.DataRestoreRequest{Effect: effect, RecoveryPointID: runCtx.Flag("recovery-point"), RecoveryPointDir: runCtx.Flag("recovery-point-dir"), Into: into, Bindings: runCtx.FlagValues("binding"), Deps: verbs.dataDeps()})
		value, err := withReceipt(result, err)
		if value != nil {
			value.(map[string]any)["report"] = report
		}
		return value, err
	case "data verify":
		expect, err := cloudtarget.ParseExpectedInventory(runCtx.Flag("expect"))
		if err != nil {
			return nil, err
		}
		report, err := store.DataVerify(operationCtx, cloudtarget.DataVerifyRequest{DeploymentID: runCtx.Flag("deployment"), RecoveryPointID: runCtx.Flag("recovery-point"), RecoveryPointDir: runCtx.Flag("recovery-point-dir"), OpenArtifacts: runCtx.BoolFlag("open"), Expect: expect, Deps: verbs.dataDeps()})
		if err != nil {
			return map[string]any{"report": report, "error": cloudtarget.AsError(err)}, err
		}
		return map[string]any{"report": report}, nil
	case "host repair":
		subject, err := jsonFlag(runCtx.Flag("subject"))
		if err != nil {
			return nil, err
		}
		req := cloudtarget.HostRepairRequest{Action: runCtx.Flag("action"), Subject: []byte(subject)}
		if strings.TrimSpace(runCtx.Flag("operation")) != "" {
			effect, err := effectFromFlags(runCtx)
			if err != nil {
				return nil, err
			}
			req.Effect = effect
		}
		result, effect, err := store.HostRepair(operationCtx, req, verbs.broker(), verbs.Runner)
		value := map[string]any{"result": result}
		if effect != nil {
			value["receipt"] = effect.Receipt
			value["replayed"] = effect.Replayed
		}
		if err != nil {
			value["error"] = cloudtarget.AsError(err)
		}
		return value, err
	case "host observe":
		result, err := cloudtarget.Observe(operationCtx, cloudtarget.ObservationRequest{Kind: runCtx.Flag("kind"), Args: runCtx.FlagValues("arg")}, verbs.Runner)
		value := map[string]any{"result": result}
		if err != nil {
			value["error"] = cloudtarget.AsError(err)
		}
		return value, err
	case "credential ingest":
		effect, err := effectFromFlags(runCtx)
		if err != nil {
			return nil, err
		}
		version, err := optionalInt64Flag(runCtx, "version")
		if err != nil {
			return nil, err
		}
		creds := verbs.Credentials
		if creds.Stdin == nil && ctx.Stdin != nil {
			creds.Stdin = ctx.Stdin
		}
		return withReceipt(store.CredentialIngest(operationCtx, cloudtarget.CredentialIngestRequest{Effect: effect, BindingID: runCtx.Flag("binding"), LogicalID: runCtx.Flag("logical-id"), Field: runCtx.Flag("field"), Version: version, ContentRef: runCtx.Flag("content-ref"), GrantRef: runCtx.Flag("grant"), FromEnv: runCtx.Flag("from-env")}, creds))
	case "credential acknowledge":
		effect, err := effectFromFlags(runCtx)
		if err != nil {
			return nil, err
		}
		version, err := optionalInt64Flag(runCtx, "version")
		if err != nil {
			return nil, err
		}
		return withReceipt(store.CredentialAcknowledge(operationCtx, cloudtarget.CredentialAckRequest{Effect: effect, BindingID: runCtx.Flag("binding"), Version: version, Consumer: runCtx.Flag("consumer")}, verbs.Credentials))
	case "credential revoke":
		effect, err := effectFromFlags(runCtx)
		if err != nil {
			return nil, err
		}
		version, err := optionalInt64Flag(runCtx, "version")
		if err != nil {
			return nil, err
		}
		return withReceipt(store.CredentialRevoke(operationCtx, cloudtarget.CredentialRevokeRequest{Effect: effect, BindingID: runCtx.Flag("binding"), Version: version}, verbs.Credentials))
	case "edge route-apply":
		effect, err := effectFromFlags(runCtx)
		if err != nil {
			return nil, err
		}
		raw, err := specBytes(runCtx)
		if err != nil {
			return nil, err
		}
		spec, err := cloudtarget.ParseEdgeRouteSpec(raw)
		if err != nil {
			return nil, err
		}
		report, result, err := store.EdgeRouteApply(operationCtx, cloudtarget.EdgeRouteApplyRequest{Effect: effect, Spec: spec, Paths: caddyPathsFromFlags(runCtx), Controller: cloudtarget.SelectCaddyController(verbs.broker(), verbs.Runner, effect.DeploymentID)})
		return withEdgeReceipt(report, result, err)
	case "edge route-rollback":
		effect, err := effectFromFlags(runCtx)
		if err != nil {
			return nil, err
		}
		report, result, err := store.EdgeRouteRollback(operationCtx, cloudtarget.EdgeRouteRollbackRequest{Effect: effect, Paths: caddyPathsFromFlags(runCtx), Controller: cloudtarget.SelectCaddyController(verbs.broker(), verbs.Runner, effect.DeploymentID)})
		return withEdgeReceipt(report, result, err)
	case "edge route-status":
		status, err := store.EdgeRouteStatus(runCtx.Flag("deployment"), caddyPathsFromFlags(runCtx))
		if err != nil {
			return nil, err
		}
		return status, nil
	default:
		return nil, fmt.Errorf("cloud-target verb %q has no handler", verb)
	}
}

// JSONArgPrefix marks a base64url-encoded JSON flag value. Argv-only
// transports refuse quotes and braces in arguments, so the cloud side
// encodes JSON payloads this way; the verbs accept both spellings.
const JSONArgPrefix = "b64:"

// jsonFlag returns the JSON text of a flag that is either raw JSON or
// b64:<base64url JSON>.
func jsonFlag(value string) (string, error) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, JSONArgPrefix) {
		return value, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(strings.TrimPrefix(value, JSONArgPrefix), "="))
	if err != nil {
		return "", &cloudtarget.Error{Code: cloudtarget.CodeInvalidArgument, Message: "b64 flag value is not base64url: " + err.Error(), Exit: cloudtarget.ExitRefused}
	}
	if !json.Valid(raw) {
		return "", &cloudtarget.Error{Code: cloudtarget.CodeInvalidArgument, Message: "b64 flag value does not decode to JSON", Exit: cloudtarget.ExitRefused}
	}
	return string(raw), nil
}

// EncodeJSONArg renders JSON for an argv-only transport (the inverse of
// jsonFlag); exported so the cloud side and tests share the exact spelling.
func EncodeJSONArg(raw []byte) string {
	return JSONArgPrefix + base64.RawURLEncoding.EncodeToString(raw)
}

func jsonFlags(values []string) ([]string, error) {
	out := make([]string, 0, len(values))
	for _, value := range values {
		decoded, err := jsonFlag(value)
		if err != nil {
			return nil, err
		}
		out = append(out, decoded)
	}
	return out, nil
}

// parsePortFlags decodes repeated `--port <name>=<number>` values.
func parsePortFlags(values []string) (map[string]int, error) {
	if len(values) == 0 {
		return nil, nil
	}
	ports := make(map[string]int, len(values))
	for _, value := range values {
		name, number, ok := strings.Cut(strings.TrimSpace(value), "=")
		port, err := strconv.Atoi(strings.TrimSpace(number))
		if !ok || strings.TrimSpace(name) == "" || err != nil || port <= 0 || port > 65535 {
			return nil, &cloudtarget.Error{Code: cloudtarget.CodeInvalidArgument, Message: fmt.Sprintf("--port %q must be <name>=<1-65535>", value), Exit: cloudtarget.ExitRefused}
		}
		ports[strings.TrimSpace(name)] = port
	}
	return ports, nil
}

// optionalInt64Flag parses a numeric flag; empty means zero.
func optionalInt64Flag(runCtx cliapp.RunContext, name string) (int64, error) {
	text := strings.TrimSpace(runCtx.Flag(name))
	if text == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return 0, &cloudtarget.Error{Code: cloudtarget.CodeInvalidArgument, Message: fmt.Sprintf("--%s %q is not an integer", name, text), Exit: cloudtarget.ExitRefused}
	}
	return value, nil
}

func withEdgeReceipt(report cloudtarget.EdgeRouteReport, result cloudtarget.EffectResult, err error) (any, error) {
	value, err := withReceipt(result, err)
	if value == nil {
		return nil, err
	}
	value.(map[string]any)["report"] = report
	return value, err
}

func specBytes(runCtx cliapp.RunContext) ([]byte, error) {
	inline := strings.TrimSpace(runCtx.Flag("spec"))
	path := strings.TrimSpace(runCtx.Flag("spec-file"))
	switch {
	case inline != "" && path != "":
		return nil, &cloudtarget.Error{Code: cloudtarget.CodeInvalidArgument, Message: "--spec and --spec-file are mutually exclusive", Exit: cloudtarget.ExitRefused}
	case inline != "":
		decoded, err := jsonFlag(inline)
		if err != nil {
			return nil, err
		}
		return []byte(decoded), nil
	case path != "":
		raw, err := os.ReadFile(path) //nolint:gosec // operator-supplied spec path
		if err != nil {
			return nil, &cloudtarget.Error{Code: cloudtarget.CodeInvalidArgument, Message: fmt.Sprintf("read --spec-file: %v", err), Exit: cloudtarget.ExitRefused}
		}
		return raw, nil
	default:
		return nil, &cloudtarget.Error{Code: cloudtarget.CodeInvalidArgument, Message: "--spec or --spec-file is required", Exit: cloudtarget.ExitRefused}
	}
}

func caddyPathsFromFlags(runCtx cliapp.RunContext) cloudtarget.CaddyPaths {
	return cloudtarget.CaddyPaths{MainConfig: strings.TrimSpace(runCtx.Flag("caddy-main")), ConfDir: strings.TrimSpace(runCtx.Flag("caddy-conf-dir")), DataDir: strings.TrimSpace(runCtx.Flag("caddy-data-dir"))}
}

func withReceipt(result cloudtarget.EffectResult, err error) (any, error) {
	value := map[string]any{"receipt": result.Receipt, "replayed": result.Replayed}
	if err != nil {
		value["error"] = cloudtarget.AsError(err)
	}
	if result.Receipt.SchemaVersion == 0 {
		// The verb refused before a receipt existed (stale fence, bad
		// argument); print only the error.
		return nil, err
	}
	return value, err
}

func effectFromFlags(runCtx cliapp.RunContext) (cloudtarget.EffectRequest, error) {
	fenceText := strings.TrimSpace(runCtx.Flag("fence"))
	fence, err := strconv.ParseUint(fenceText, 10, 64)
	if err != nil {
		return cloudtarget.EffectRequest{}, &cloudtarget.Error{Code: cloudtarget.CodeInvalidArgument, Message: fmt.Sprintf("--fence %q is not an unsigned integer", fenceText), Exit: cloudtarget.ExitRefused}
	}
	return cloudtarget.EffectRequest{DeploymentID: runCtx.Flag("deployment"), OperationID: runCtx.Flag("operation"), Step: runCtx.Flag("step"), Fence: fence}, nil
}

func versionFlag(runCtx cliapp.RunContext) (int64, error) {
	text := strings.TrimSpace(runCtx.Flag("version"))
	version, err := strconv.ParseInt(text, 10, 64)
	if err != nil || version <= 0 {
		return 0, &cloudtarget.Error{Code: cloudtarget.CodeInvalidArgument, Message: fmt.Sprintf("--version %q is not a positive integer", text), Exit: cloudtarget.ExitRefused}
	}
	return version, nil
}

// ErrorJSON renders a typed error the way dispatch prints it; exported for
// tests and for callers that compose output.
func ErrorJSON(err error) []byte {
	raw, _ := json.Marshal(map[string]any{"error": cloudtarget.AsError(err)})
	return raw
}
