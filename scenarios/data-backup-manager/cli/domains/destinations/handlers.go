package destinations

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

	destinationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/destinations"
	destinationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/destinations/destinations_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client destinationsconnect.DestinationsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: destinationsconnect.NewDestinationsServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) create(ctx cliapp.RunContext) error {
	backend, err := parseBackendKind(ctx.Flag("backend"))
	if err != nil {
		return err
	}
	capBytes, err := parseOptionalInt64(ctx.Flag("cap-bytes"))
	if err != nil {
		return fmt.Errorf("--cap-bytes: %w", err)
	}
	capPolicy, err := parseCapPolicy(ctx.Flag("cap-policy"))
	if err != nil {
		return err
	}
	resp, err := h.client.CreateDestination(context.Background(), connect.NewRequest(&destinationsv1.CreateDestinationRequest{
		Name:        ctx.Flag("name"),
		BackendKind: backend,
		Location:    ctx.Flag("location"),
		CapBytes:    capBytes,
		CapPolicy:   capPolicy,
	}))
	if err != nil {
		return cliapp.WrapAPIError("create destination", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Destination == nil {
		return fmt.Errorf("server returned no destination")
	}
	d := resp.Msg.Destination
	changes := []string{formatDestination(d)}
	if d.BackendKind == destinationsv1.BackendKind_BACKEND_KIND_FILESYSTEM {
		changes = append(changes,
			fmt.Sprintf("bundle root: %s (README.txt, RECOVERY.txt, vrooli-backup-destination.json)", d.Location),
			fmt.Sprintf("kopia repository: %s", d.RepositoryLocation),
			"the repository is encrypted; the passphrase is held in the credential authority and never written to the drive",
		)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Created destination %s.", d.Id)},
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("`destinations get %s` — show this destination", d.Id),
			"`plans create --target-ids <id> --destination-ids " + d.Id + "` — create a backup plan",
			"`runs trigger --plan-id <plan>` — run a backup now",
			"`restores verify --target-id <t> --destination-id " + d.Id + "` — verify a snapshot is restorable",
		},
	})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetDestination(context.Background(), connect.NewRequest(&destinationsv1.GetDestinationRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get destination %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Destination == nil {
		return fmt.Errorf("server returned no destination")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched destination %s.", resp.Msg.Destination.Id)},
		ResultsHeading: "Destination",
		Results:        []string{formatDestination(resp.Msg.Destination)},
	})
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListDestinations(context.Background(), connect.NewRequest(&destinationsv1.ListDestinationsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list destinations", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no destinations response")
	}
	results := make([]string, 0, len(resp.Msg.Destinations))
	for _, d := range resp.Msg.Destinations {
		results = append(results, formatDestination(d))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d destination(s).", len(resp.Msg.Destinations))},
		ResultsHeading: "Destinations",
		Results:        results,
		RetrievalHints: []string{
			"`destinations get <id>` — show a single destination",
			"`destinations create --name <n> --backend <b> --location <l>` — create a destination",
		},
	})
}

func (h *handlers) update(ctx cliapp.RunContext) error {
	capBytes, err := parseOptionalInt64(ctx.Flag("cap-bytes"))
	if err != nil {
		return fmt.Errorf("--cap-bytes: %w", err)
	}
	capPolicy, err := parseCapPolicy(ctx.Flag("cap-policy"))
	if err != nil {
		return err
	}
	resp, err := h.client.UpdateDestination(context.Background(), connect.NewRequest(&destinationsv1.UpdateDestinationRequest{
		Id:        ctx.Flag("id"),
		CapBytes:  capBytes,
		CapPolicy: capPolicy,
	}))
	if err != nil {
		return cliapp.WrapAPIError("update destination", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Destination == nil {
		return fmt.Errorf("server returned no destination")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Updated destination %s.", resp.Msg.Destination.Id)},
		Changes: []string{formatDestination(resp.Msg.Destination)},
	})
}

func (h *handlers) delete(ctx cliapp.RunContext) error {
	deleteRepo := false
	if s := ctx.Flag("delete-repository"); s != "" {
		deleteRepo, _ = strconv.ParseBool(s)
	}
	resp, err := h.client.DeleteDestination(context.Background(), connect.NewRequest(&destinationsv1.DeleteDestinationRequest{
		Id:               ctx.Flag("id"),
		DeleteRepository: deleteRepo,
	}))
	if err != nil {
		return cliapp.WrapAPIError("delete destination", err, nil)
	}
	msg := "No matching destination to delete."
	var changes []string
	if resp != nil && resp.Msg != nil && resp.Msg.Removed {
		msg = "Deleted destination."
		if deleteRepo {
			changes = []string{
				"Removed: catalog row + local resource-kopia metadata/config/cache + credential-authority refs.",
				"NOT removed: the encrypted repository bytes on the backend remain on disk. " +
					"Delete the bundle folder manually if you intend to destroy the backups.",
			}
		} else {
			changes = []string{
				"Removed: catalog row only. Local kopia metadata, credential-authority refs, and the encrypted repository bytes all remain.",
				"Pass --delete-repository to also remove local kopia metadata and credential-authority refs.",
			}
		}
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{msg},
		Changes: changes,
	})
}

func (h *handlers) usage(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetDestinationUsage(context.Background(), connect.NewRequest(&destinationsv1.GetDestinationUsageRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get destination usage %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no usage response")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Usage for destination %s.", id)},
		ResultsHeading: "Usage",
		Results: []string{
			fmt.Sprintf("usage=%d bytes cap=%d bytes state=%s policy=%s",
				resp.Msg.UsageBytes, resp.Msg.CapBytes,
				usageStateLabel(resp.Msg.UsageState),
				capPolicyLabel(resp.Msg.CapPolicy)),
		},
	})
}

func (h *handlers) readiness(ctx cliapp.RunContext) error {
	targetBytes, err := parseOptionalInt64(ctx.Flag("selected-target-bytes"))
	if err != nil {
		return fmt.Errorf("--selected-target-bytes: %w", err)
	}
	retentionCopies, err := parseOptionalInt32(ctx.Flag("retention-copies"))
	if err != nil {
		return fmt.Errorf("--retention-copies: %w", err)
	}
	// Declared "bool": true in the manifest, so it arrives as a boolean flag.
	// Reading it with Flag() returns "" for a set flag, which silently made
	// --cross-platform a no-op.
	crossPlatform := ctx.BoolFlag("cross-platform")
	resp, err := h.client.AnalyzeDestination(context.Background(), connect.NewRequest(&destinationsv1.AnalyzeDestinationRequest{
		Location:              ctx.Flag("location"),
		ProposedSubdir:        ctx.Flag("proposed-subdir"),
		SelectedTargetBytes:   targetBytes,
		RetentionCopies:       retentionCopies,
		CrossPlatformRequired: crossPlatform,
	}))
	if err != nil {
		return cliapp.WrapAPIError("analyze destination readiness", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Report == nil {
		return fmt.Errorf("server returned no readiness report")
	}
	report := resp.Msg.Report
	results := []string{formatReadinessReport(report)}
	for _, c := range report.Checks {
		results = append(results, fmt.Sprintf("%s=%s — %s", c.Code, readinessSeverityLabel(c.Severity), c.Message))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Readiness for %s: %s.", report.Location, readinessSeverityLabel(report.OverallSeverity))},
		ResultsHeading: "Readiness checks",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("`destinations create --name <name> --backend filesystem --location %q` — create a destination at the recommended path after review", report.RecommendedDestinationLocation),
		},
	})
}

func (h *handlers) preparePlan(ctx cliapp.RunContext) error {
	action, err := parsePreparationAction(ctx.Flag("action"))
	if err != nil {
		return err
	}
	planReq := &destinationsv1.PlanDestinationPreparationRequest{
		Location:          ctx.Flag("location"),
		Action:            action,
		DesiredSubdir:     ctx.Flag("subdir"),
		DesiredLabel:      ctx.Flag("label"),
		DesiredFilesystem: ctx.Flag("filesystem"),
	}
	// Once a remediation sequence unmounts the destination, its path is an
	// ordinary directory on the host filesystem and no longer identifies the
	// disk. --device keeps the remaining steps addressed to the right volume.
	if device := strings.TrimSpace(ctx.Flag("device")); device != "" {
		planReq.ExpectedIdentity = &destinationsv1.DestinationDeviceIdentity{DevicePath: device}
	}
	resp, err := h.client.PlanDestinationPreparation(context.Background(), connect.NewRequest(planReq))
	if err != nil {
		return cliapp.WrapAPIError("plan destination preparation", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Plan == nil {
		return fmt.Errorf("server returned no preparation plan")
	}
	plan := resp.Msg.Plan
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Prepared plan %s.", plan.Id)},
		ResultsHeading: "Preparation plan",
		Results: []string{
			formatPreparationPlan(plan),
			fmt.Sprintf("confirmation=%q", plan.ConfirmationPhrase),
		},
		RetrievalHints: []string{
			"`destinations prepare-execute --plan-json '<json>' --confirm '<phrase>' --dry-run true` — validate the plan without executing it",
		},
	})
}

func (h *handlers) prepareExecute(ctx cliapp.RunContext) error {
	plan := &destinationsv1.DestinationPreparationPlan{}
	if err := protojson.Unmarshal([]byte(ctx.Flag("plan-json")), plan); err != nil {
		return fmt.Errorf("--plan-json: %w", err)
	}
	dryRun := true
	if raw := ctx.Flag("dry-run"); raw != "" {
		var err error
		dryRun, err = strconv.ParseBool(raw)
		if err != nil {
			return fmt.Errorf("--dry-run: %w", err)
		}
	}
	// Boolean flag: see the note on --cross-platform above. Read through
	// Flag() this gate could never be satisfied, so no destructive action was
	// reachable even with an operator explicitly acknowledging the risk.
	ack := ctx.BoolFlag("acknowledge-data-loss")
	resp, err := h.client.ExecuteDestinationPreparation(context.Background(), connect.NewRequest(&destinationsv1.ExecuteDestinationPreparationRequest{
		Plan:                plan,
		Confirmation:        ctx.Flag("confirm"),
		DryRun:              &dryRun,
		AcknowledgeDataLoss: ack,
	}))
	if err != nil {
		return cliapp.WrapAPIError("execute destination preparation", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no execution response")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: formatPreparationExecution(resp.Msg),
	})
}

// formatPreparationExecution renders an execution result. A remediation carries
// the control plane's typed outcome — which backend ran, the exact command, and
// the operator command when this host has no automated path — and all of it is
// shown, because a bare "done" gives an operator nothing to verify or resume
// from.
func formatPreparationExecution(msg *destinationsv1.ExecuteDestinationPreparationResponse) []string {
	lines := []string{fmt.Sprintf("Preparation %s for %s (dry-run=%t).", preparationActionLabel(msg.GetAction()), msg.GetLocation(), msg.GetDryRun())}
	if status := msg.GetStatus(); status != "" {
		lines = append(lines, fmt.Sprintf("status=%s changed=%t", status, msg.GetChanged()))
	}
	if backend := msg.GetBackend(); backend != "" {
		lines = append(lines, "backend="+backend)
	}
	if command := msg.GetCommand(); len(command) > 0 {
		lines = append(lines, "command="+strings.Join(command, " "))
	}
	if consistent := msg.GetConsistent(); consistent != "" {
		lines = append(lines, "filesystem-consistent="+consistent)
	}
	if detail := msg.GetDetail(); detail != "" {
		lines = append(lines, "detail="+detail)
	}
	if reason := msg.GetRefusalReason(); reason != "" {
		lines = append(lines, "reason="+reason)
	}
	if operator := msg.GetOperatorCommand(); operator != "" {
		lines = append(lines, "run instead: "+operator)
	}
	return lines
}

// parseBackendKind maps the --backend flag string to the proto BackendKind enum.
func parseBackendKind(s string) (destinationsv1.BackendKind, error) {
	switch s {
	case "filesystem":
		return destinationsv1.BackendKind_BACKEND_KIND_FILESYSTEM, nil
	case "s3":
		return destinationsv1.BackendKind_BACKEND_KIND_S3, nil
	default:
		return destinationsv1.BackendKind_BACKEND_KIND_UNSPECIFIED,
			fmt.Errorf("invalid --backend %q: must be one of filesystem, s3", s)
	}
}

// parseCapPolicy maps the --cap-policy flag string to the proto CapPolicy enum.
// Empty string returns UNSPECIFIED (no cap policy change).
func parseCapPolicy(s string) (destinationsv1.CapPolicy, error) {
	switch s {
	case "":
		return destinationsv1.CapPolicy_CAP_POLICY_UNSPECIFIED, nil
	case "alert-block":
		return destinationsv1.CapPolicy_CAP_POLICY_ALERT_BLOCK, nil
	case "alert-only":
		return destinationsv1.CapPolicy_CAP_POLICY_ALERT_ONLY, nil
	default:
		return destinationsv1.CapPolicy_CAP_POLICY_UNSPECIFIED,
			fmt.Errorf("invalid --cap-policy %q: must be one of alert-block, alert-only", s)
	}
}

// parseOptionalInt64 parses an optional integer flag; returns 0 for empty.
func parseOptionalInt64(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.ParseInt(s, 10, 64)
}

func parseOptionalInt32(s string) (int32, error) {
	if s == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(s, 10, 32)
	return int32(n), err
}

func parsePreparationAction(s string) (destinationsv1.PreparationAction, error) {
	switch s {
	case "create-subdir":
		return destinationsv1.PreparationAction_PREPARATION_ACTION_CREATE_SUBDIR, nil
	case "relabel":
		return destinationsv1.PreparationAction_PREPARATION_ACTION_RELABEL, nil
	case "clear-directory":
		return destinationsv1.PreparationAction_PREPARATION_ACTION_CLEAR_DIRECTORY, nil
	case "format":
		return destinationsv1.PreparationAction_PREPARATION_ACTION_FORMAT, nil
	case "unmount":
		return destinationsv1.PreparationAction_PREPARATION_ACTION_UNMOUNT, nil
	case "check-filesystem":
		return destinationsv1.PreparationAction_PREPARATION_ACTION_CHECK_FILESYSTEM, nil
	case "repair-filesystem":
		return destinationsv1.PreparationAction_PREPARATION_ACTION_REPAIR_FILESYSTEM, nil
	case "mount-read-write":
		return destinationsv1.PreparationAction_PREPARATION_ACTION_MOUNT_READ_WRITE, nil
	default:
		return destinationsv1.PreparationAction_PREPARATION_ACTION_UNSPECIFIED,
			fmt.Errorf("invalid --action %q: must be one of create-subdir, relabel, clear-directory, format, unmount, check-filesystem, repair-filesystem, mount-read-write", s)
	}
}

func backendKindLabel(k destinationsv1.BackendKind) string {
	switch k {
	case destinationsv1.BackendKind_BACKEND_KIND_FILESYSTEM:
		return "filesystem"
	case destinationsv1.BackendKind_BACKEND_KIND_S3:
		return "s3"
	default:
		return "unspecified"
	}
}

func capPolicyLabel(p destinationsv1.CapPolicy) string {
	switch p {
	case destinationsv1.CapPolicy_CAP_POLICY_ALERT_BLOCK:
		return "alert-block"
	case destinationsv1.CapPolicy_CAP_POLICY_ALERT_ONLY:
		return "alert-only"
	default:
		return "unspecified"
	}
}

func usageStateLabel(s destinationsv1.UsageState) string {
	switch s {
	case destinationsv1.UsageState_USAGE_STATE_WITHIN:
		return "within"
	case destinationsv1.UsageState_USAGE_STATE_NEAR:
		return "near"
	case destinationsv1.UsageState_USAGE_STATE_OVER:
		return "over"
	default:
		return "unspecified"
	}
}

func readinessSeverityLabel(s destinationsv1.ReadinessSeverity) string {
	switch s {
	case destinationsv1.ReadinessSeverity_READINESS_SEVERITY_PASS:
		return "pass"
	case destinationsv1.ReadinessSeverity_READINESS_SEVERITY_WARNING:
		return "warning"
	case destinationsv1.ReadinessSeverity_READINESS_SEVERITY_FAIL:
		return "fail"
	case destinationsv1.ReadinessSeverity_READINESS_SEVERITY_UNKNOWN:
		return "unknown"
	default:
		return "unspecified"
	}
}

func preparationActionLabel(a destinationsv1.PreparationAction) string {
	switch a {
	case destinationsv1.PreparationAction_PREPARATION_ACTION_CREATE_SUBDIR:
		return "create-subdir"
	case destinationsv1.PreparationAction_PREPARATION_ACTION_RELABEL:
		return "relabel"
	case destinationsv1.PreparationAction_PREPARATION_ACTION_CLEAR_DIRECTORY:
		return "clear-directory"
	case destinationsv1.PreparationAction_PREPARATION_ACTION_FORMAT:
		return "format"
	case destinationsv1.PreparationAction_PREPARATION_ACTION_UNMOUNT:
		return "unmount"
	case destinationsv1.PreparationAction_PREPARATION_ACTION_CHECK_FILESYSTEM:
		return "check-filesystem"
	case destinationsv1.PreparationAction_PREPARATION_ACTION_REPAIR_FILESYSTEM:
		return "repair-filesystem"
	case destinationsv1.PreparationAction_PREPARATION_ACTION_MOUNT_READ_WRITE:
		return "mount-read-write"
	default:
		return "unspecified"
	}
}

func formatReadinessReport(r *destinationsv1.DestinationReadinessReport) string {
	if r == nil {
		return "(nil)"
	}
	identity := ""
	if r.Identity != nil {
		identity = fmt.Sprintf(" device=%s mount=%s fs=%s size=%d", r.Identity.DevicePath, r.Identity.Mountpoint, r.Identity.Filesystem, r.Identity.TotalBytes)
	}
	return fmt.Sprintf("overall=%s recommended_location=%s action=%s%s",
		readinessSeverityLabel(r.OverallSeverity),
		r.RecommendedDestinationLocation,
		r.RecommendedAction,
		identity)
}

func formatPreparationPlan(p *destinationsv1.DestinationPreparationPlan) string {
	if p == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s action=%s target=%s destructive=%t supported=%t unsupported=%q",
		p.Id,
		preparationActionLabel(p.Action),
		p.TargetPath,
		p.Destructive,
		p.Supported,
		p.UnsupportedReason)
}

func formatDestination(d *destinationsv1.Destination) string {
	if d == nil {
		return "(nil)"
	}
	created := ""
	if d.CreatedAt != nil {
		created = d.CreatedAt.AsTime().Format(time.RFC3339)
	}
	repo := d.RepositoryLocation
	if repo == "" {
		repo = d.Location
	}
	return fmt.Sprintf("%s — %s [backend=%s bundle_root=%s repository=%s encryption=%s cap=%d policy=%s usage=%d state=%s created=%s]",
		d.Id, d.Name,
		backendKindLabel(d.BackendKind), d.Location, repo,
		emptyDash(d.EncryptionAlgorithm),
		d.CapBytes, capPolicyLabel(d.CapPolicy),
		d.UsageBytes, usageStateLabel(d.UsageState),
		created)
}

// emptyDash renders an empty string as a dash for tidy CLI output.
func emptyDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
