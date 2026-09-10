// Package releases provides the deployment-manager release lifecycle CLI.
// Production registration uses the generated ReleasesService client for
// typed release operations; the package keeps the REST compatibility route
// only for health, which has no typed RPC yet.
package releases

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"deployment-manager/cli/cmdutil"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/cli-core/operationstanding"
	releasesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/releases"
	releasesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/releases/releasesv1connect"
	"google.golang.org/protobuf/encoding/protojson"
)

// Commands provides CLI commands for the release lifecycle.
type Commands struct {
	api             *cliutil.APIClient
	operationClient releasesconnect.ReleasesServiceClient
}

// New creates a compatibility command set bound to the REST API. Production
// registration uses NewWithOperationClient so release operations use the
// generated ReleasesService transport.
func New(api *cliutil.APIClient) *Commands {
	return &Commands{api: api}
}

// NewWithOperationClient binds release operations to the generated Releases
// service while retaining the REST client for health and direct compatibility
// construction.
func NewWithOperationClient(api *cliutil.APIClient, client releasesconnect.ReleasesServiceClient) *Commands {
	return &Commands{api: api, operationClient: client}
}

// Run dispatches release subcommands.
func (c *Commands) Run(args []string) error {
	if len(args) == 0 {
		return errors.New("subcommand required: list, get, operation, dossier, health, start, verify, reconcile, recover")
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		return c.list(rest)
	case "get":
		return c.get(rest)
	case "operation":
		return c.operation(rest)
	case "dossier":
		return c.dossier(rest)
	case "health":
		return c.health(rest)
	case "start":
		return c.start(rest)
	case "verify":
		return c.verify(rest)
	case "reconcile":
		return c.reconcile(rest)
	case "recover":
		return c.recover(rest)
	default:
		return fmt.Errorf("unknown releases subcommand: %s", sub)
	}
}

func (c *Commands) dossier(args []string) error {
	fs := flag.NewFlagSet("releases dossier", flag.ContinueOnError)
	format := fs.String("format", "", "Output format (json)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("release ID is required")
	}
	if c.operationClient == nil {
		return errors.New("typed ReleasesService client is required")
	}
	response, err := c.operationClient.Dossier(context.Background(), connect.NewRequest(&releasesv1.GetReleaseDossierRequest{ReleaseId: remaining[0]}))
	if err != nil {
		return err
	}
	if response == nil || response.Msg == nil || response.Msg.GetDossier() == nil {
		return errors.New("typed release dossier response was empty")
	}
	if strings.ToLower(cmdutil.ResolveFormat(*format)) == "json" {
		body, err := protojson.Marshal(response.Msg)
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, string(body))
		return nil
	}
	body, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(response.Msg.GetDossier())
	if err != nil {
		return err
	}
	return renderDossier(body)
}

func renderDossier(body []byte) error {
	var dossier struct {
		Release struct {
			ID        string `json:"id"`
			ReleaseID string `json:"release_id"`
			Status    string `json:"status"`
		} `json:"release"`
		Health struct {
			Status              string   `json:"status"`
			SupportedControls   []string `json:"supported_controls"`
			UnsupportedControls []string `json:"unsupported_controls"`
		} `json:"health"`
		MissingProof []string `json:"missing_proof"`
	}
	if err := json.Unmarshal(body, &dossier); err != nil {
		fmt.Println(string(body))
		return nil
	}
	if dossier.Release.ID == "" {
		dossier.Release.ID = dossier.Release.ReleaseID
	}
	fmt.Printf("Dossier: %s\nRelease status: %s\nHealth: %s\nSupported recovery controls: %s\nUnavailable recovery controls: %s\n", dossier.Release.ID, dossier.Release.Status, dossier.Health.Status, strings.Join(dossier.Health.SupportedControls, ", "), strings.Join(dossier.Health.UnsupportedControls, ", "))
	if len(dossier.MissingProof) > 0 {
		fmt.Printf("Missing proof: %s\n", strings.Join(dossier.MissingProof, ", "))
	}
	return nil
}

func (c *Commands) health(args []string) error {
	fs := flag.NewFlagSet("releases health", flag.ContinueOnError)
	format := fs.String("format", "", "Output format (json)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("release ID is required")
	}
	body, err := c.api.Get("/api/v1/releases/"+url.PathEscape(remaining[0])+"/health", nil)
	if err != nil {
		return err
	}
	if strings.ToLower(cmdutil.ResolveFormat(*format)) == "json" {
		cliutil.PrintJSON(body)
		return nil
	}
	var health struct {
		Status               string   `json:"status"`
		PublicationVerified  bool     `json:"publication_verified"`
		ClientUpdatesHealthy bool     `json:"client_updates_healthy"`
		RecoveryStanding     string   `json:"recovery_standing"`
		SupportedControls    []string `json:"supported_controls"`
		UnsupportedControls  []string `json:"unsupported_controls"`
		Alerts               []struct {
			Code       string `json:"code"`
			Severity   string `json:"severity"`
			Message    string `json:"message"`
			NextAction string `json:"next_action"`
		} `json:"alerts"`
	}
	if err := json.Unmarshal(body, &health); err != nil {
		fmt.Println(string(body))
		return nil
	}
	fmt.Printf("Health: %s\nPublication verified: %t\nClient updates healthy: %t\nRecovery: %s\nSupported recovery controls: %s\nUnavailable recovery controls: %s\n", health.Status, health.PublicationVerified, health.ClientUpdatesHealthy, health.RecoveryStanding, strings.Join(health.SupportedControls, ", "), strings.Join(health.UnsupportedControls, ", "))
	for _, alert := range health.Alerts {
		fmt.Printf("Alert [%s/%s]: %s\nNext action: %s\n", alert.Severity, alert.Code, alert.Message, alert.NextAction)
	}
	return nil
}

func (c *Commands) recover(args []string) error {
	fs := flag.NewFlagSet("releases recover", flag.ContinueOnError)
	review := fs.String("review-key", "", "Exact approved review key (required)")
	candidate := fs.String("candidate-id", "", "Exact candidate identity (required)")
	destination := fs.String("destination-revision-id", "", "Exact destination revision identity (required)")
	action := fs.String("action", "halt", "Owner recovery action")
	predecessor := fs.Int64("expected-predecessor-revision", 0, "Exact predecessor revision required for rollback")
	compatibility := fs.String("data-compatibility", "", "Required migration qualification (use compatible for rollback/forward repair)")
	repairArtifacts := fs.String("repair-artifacts", "", "Forward-repair artifact map, comma-separated platform=id pairs")
	repairBundle := fs.String("repair-bundle-sha256", "", "Exact owner-side repair bundle SHA256 for cloud rollback/forward repair")
	idempotencyKey := fs.String("idempotency-key", "", "Stable recovery request identity for owner retries")
	confirmation := fs.String("confirmation", "", "Explicit execution confirmation")
	dryRun := fs.Bool("dry-run", false, "Preview the owner action without an external effect")
	format := fs.String("format", "", "Output format (json)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: releases recover <release-id> --review-key <key> --candidate-id <id> --destination-revision-id <id> [--action halt|withdraw|rollback|forward_repair] [--dry-run]\n\n")
		fs.PrintDefaults()
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("release ID is required")
	}
	if *review == "" || *candidate == "" || *destination == "" {
		return errors.New("review-key, candidate-id, and destination-revision-id are required")
	}
	if !*dryRun && *confirmation == "" {
		return errors.New("confirmation is required unless --dry-run is set")
	}
	artifacts, err := parseRepairArtifacts(*repairArtifacts)
	if err != nil {
		return err
	}
	if c.operationClient == nil {
		return errors.New("typed ReleasesService client is required")
	}
	response, err := c.operationClient.Recover(context.Background(), connect.NewRequest(&releasesv1.RecoverReleaseRequest{
		ReleaseId: remaining[0], ReviewKey: *review, CandidateId: *candidate, DestinationRevisionId: *destination,
		Action: *action, ExpectedPredecessorRevision: *predecessor, DataCompatibility: *compatibility,
		RepairArtifactIds: artifacts, RepairBundleSha256: strings.TrimSpace(*repairBundle), IdempotencyKey: strings.TrimSpace(*idempotencyKey),
		Confirmation: *confirmation, DryRun: *dryRun,
	}))
	if err != nil {
		return err
	}
	if response == nil || response.Msg == nil || response.Msg.GetReceipt() == nil {
		return errors.New("typed recovery response did not include a receipt")
	}
	if strings.ToLower(cmdutil.ResolveFormat(*format)) == "json" {
		body, err := protojson.Marshal(response.Msg)
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, string(body))
		return nil
	}
	receipt := response.Msg.GetReceipt()
	fmt.Printf("Recovery: %s\nAction: %s\nOutcome: %s\nReceipt: %s\n", remaining[0], receipt.GetAction(), receipt.GetOutcome(), receipt.GetExternalReceipt())
	return nil
}

func parseRepairArtifacts(value string) (map[string]int64, error) {
	result := map[string]int64{}
	for _, entry := range strings.Split(strings.TrimSpace(value), ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return nil, fmt.Errorf("invalid --repair-artifacts entry %q; expected platform=id", entry)
		}
		id, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("invalid repair artifact id %q", parts[1])
		}
		result[strings.TrimSpace(parts[0])] = id
	}
	return result, nil
}

func (c *Commands) list(args []string) error {
	fs := flag.NewFlagSet("releases list", flag.ContinueOnError)
	limit := fs.Int("limit", 0, "Maximum number of releases to return (default 50)")
	format := fs.String("format", "", "Output format (json)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: releases list <profile-id> [--limit <n>]\n\n")
		fs.PrintDefaults()
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("profile ID is required")
	}
	if c.operationClient == nil {
		return errors.New("typed ReleasesService client is required")
	}
	profileID := remaining[0]
	var typedLimit uint32
	if *limit > 0 {
		typedLimit = uint32(*limit)
	}
	response, err := c.operationClient.List(context.Background(), connect.NewRequest(&releasesv1.ListReleasesRequest{ProfileId: profileID, Limit: typedLimit}))
	if err != nil {
		return err
	}
	if response == nil || response.Msg == nil {
		return errors.New("typed release list response was empty")
	}
	if strings.ToLower(cmdutil.ResolveFormat(*format)) == "json" {
		body, err := protojson.Marshal(response.Msg)
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, string(body))
		return nil
	}
	body, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(response.Msg)
	if err != nil {
		return err
	}
	return renderReleaseList(body)
}

func renderReleaseList(body []byte) error {
	var envelope struct {
		Releases []map[string]interface{} `json:"releases"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		fmt.Println(string(body))
		return nil
	}
	if len(envelope.Releases) == 0 {
		fmt.Println("No releases found.")
		return nil
	}
	headers := []string{"ID", "CHANNEL", "VERSION", "STATUS", "COMMIT", "CREATED"}
	rows := make([][]string, 0, len(envelope.Releases))
	for _, r := range envelope.Releases {
		if str(r["id"]) == "" {
			r["id"] = r["release_id"]
		}
		rows = append(rows, []string{
			truncate(str(r["id"]), 12),
			str(r["channel"]),
			str(r["release_version"]),
			str(r["status"]),
			truncate(str(r["git_commit_hash"]), 12),
			str(r["created_at"]),
		})
	}
	cmdutil.PrintTable(headers, rows)
	return nil
}

func (c *Commands) get(args []string) error {
	fs := flag.NewFlagSet("releases get", flag.ContinueOnError)
	format := fs.String("format", "", "Output format (json)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: releases get <release-id>\n\n")
		fs.PrintDefaults()
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("release ID is required")
	}
	if c.operationClient == nil {
		return errors.New("typed ReleasesService client is required")
	}
	response, err := c.operationClient.Get(context.Background(), connect.NewRequest(&releasesv1.GetReleaseRequest{ReleaseId: remaining[0]}))
	if err != nil {
		return err
	}
	if response == nil || response.Msg == nil || response.Msg.GetRelease() == nil {
		return errors.New("typed release response did not include a release")
	}
	if strings.ToLower(cmdutil.ResolveFormat(*format)) == "json" {
		body, err := protojson.Marshal(response.Msg)
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, string(body))
		return nil
	}
	body, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(response.Msg.GetRelease())
	if err != nil {
		return err
	}
	return renderReleaseDetail(body)
}

func (c *Commands) operation(args []string) error {
	fs := flag.NewFlagSet("releases operation", flag.ContinueOnError)
	format := fs.String("format", "", "Output format (json)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: releases operation <operation-id>\n\n")
		fs.PrintDefaults()
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("operation ID is required")
	}
	if c.operationClient == nil {
		return errors.New("typed ReleasesService client is required")
	}
	response, err := c.operationClient.GetOperation(context.Background(), connect.NewRequest(&releasesv1.GetReleaseOperationRequest{OperationId: remaining[0]}))
	if err != nil {
		return err
	}
	if response == nil || response.Msg == nil || response.Msg.GetOperation() == nil {
		return errors.New("typed operation response did not include an operation")
	}
	if strings.ToLower(cmdutil.ResolveFormat(*format)) == "json" {
		body, err := protojson.Marshal(response.Msg)
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, string(body))
		return nil
	}
	return renderProtoOperationResult(response.Msg.GetOperation())
}

func renderProtoOperationResult(operation *releasesv1.ReleaseOperation) error {
	status := strings.ToLower(strings.TrimPrefix(operation.GetStatus().String(), "RELEASE_OPERATION_STATUS_"))
	fallbackID := operation.GetOperationId()
	fmt.Fprintf(os.Stdout, "Release operation %s status: %s\n", fallbackID, status)
	if operation.GetReleaseId() != "" {
		fmt.Fprintf(os.Stdout, "  release: %s\n", operation.GetReleaseId())
	}
	if operation.GetError() != "" {
		fmt.Fprintf(os.Stdout, "  error: %s\n", operation.GetError())
	}
	standing := releaseOperationStanding{
		lifecycle: status,
		phase:     operation.GetActiveStage(),
		reattach:  fmt.Sprintf("deployment-manager releases operation %s", fallbackID),
		directive: releaseOperationDirective(status),
	}
	if operation.GetReleaseId() != "" {
		fmt.Fprintf(os.Stdout, "  inspect release: deployment-manager releases get %s\n", operation.GetReleaseId())
	}
	return operationstanding.WriteText(os.Stdout, standing)
}

func (c *Commands) start(args []string) error {
	fs := flag.NewFlagSet("releases start", flag.ContinueOnError)
	channel := fs.String("channel", "", "Release channel (defaults to profile default; e.g. stable, beta)")
	commit := fs.String("commit", "", "Git commit hash for the release (required)")
	artifactDigest := fs.String("artifact-digest", "", "Immutable artifact digest bound to the approved candidate")
	candidateID := fs.String("candidate-id", "", "Canonical immutable candidate identity")
	destinationRevisionID := fs.String("destination-revision-id", "", "Canonical destination revision identity")
	authorizationEpoch := fs.Uint64("authorization-epoch", 0, "Authorization epoch bound to the approval")
	readinessReviewKey := fs.String("readiness-review-key", "", "Exact approved readiness review key")
	idempotencyKey := fs.String("idempotency-key", "", "Stable lifecycle request idempotency key")
	version := fs.String("version", "", "Release version string (required)")
	notes := fs.String("notes", "", "Release notes")
	platformsCSV := fs.String("platforms", "", "Comma-separated target platforms (default: linux-x64,darwin-arm64,win-x64)")
	format := fs.String("format", "", "Output format (json)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: releases start <profile-id> --commit <hash> --version <version> --artifact-digest <digest> --candidate-id <id> --destination-revision-id <id> --readiness-review-key <key> --authorization-epoch <n> [--channel <name>]\n\n")
		fs.PrintDefaults()
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("profile ID is required")
	}
	if *commit == "" {
		return errors.New("--commit is required")
	}
	if *version == "" {
		return errors.New("--version is required")
	}
	if *artifactDigest == "" {
		return errors.New("--artifact-digest is required")
	}
	if *candidateID == "" {
		return errors.New("--candidate-id is required")
	}
	if *destinationRevisionID == "" {
		return errors.New("--destination-revision-id is required")
	}
	if *readinessReviewKey == "" {
		return errors.New("--readiness-review-key is required")
	}
	if *authorizationEpoch == 0 {
		return errors.New("--authorization-epoch must be greater than zero")
	}

	if c.operationClient == nil {
		return errors.New("typed ReleasesService client is required")
	}
	var platforms []string
	if *platformsCSV != "" {
		platforms = strings.Split(*platformsCSV, ",")
		for i := range platforms {
			platforms[i] = strings.TrimSpace(platforms[i])
		}
	}
	response, err := c.operationClient.Start(context.Background(), connect.NewRequest(&releasesv1.StartReleaseRequest{
		ProfileId:             remaining[0],
		Channel:               *channel,
		GitCommitHash:         *commit,
		ArtifactDigest:        *artifactDigest,
		CandidateId:           *candidateID,
		DestinationRevisionId: *destinationRevisionID,
		AuthorizationEpoch:    *authorizationEpoch,
		IdempotencyKey:        *idempotencyKey,
		ReadinessReviewKey:    *readinessReviewKey,
		ReleaseVersion:        *version,
		ReleaseNotes:          *notes,
		Platforms:             platforms,
	}))
	if err != nil {
		return err
	}
	if response == nil || response.Msg == nil {
		return errors.New("typed release start response was empty")
	}
	if strings.ToLower(cmdutil.ResolveFormat(*format)) == "json" {
		body, err := protojson.Marshal(response.Msg)
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, string(body))
		return nil
	}
	body, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(response.Msg)
	if err != nil {
		return err
	}
	return renderStartResult(body)
}

func (c *Commands) verify(args []string) error {
	fs := flag.NewFlagSet("releases verify", flag.ContinueOnError)
	deep := fs.Bool("deep", false, "Run a deep verification (S3 reachability check)")
	format := fs.String("format", "", "Output format (json)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: releases verify <release-id> [--deep]\n\n")
		fs.PrintDefaults()
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("release ID is required")
	}
	if c.operationClient == nil {
		return errors.New("typed ReleasesService client is required")
	}
	response, err := c.operationClient.Reverify(context.Background(), connect.NewRequest(&releasesv1.ReverifyReleaseRequest{ReleaseId: remaining[0], Deep: *deep}))
	if err != nil {
		return err
	}
	if response == nil || response.Msg == nil || response.Msg.GetRelease() == nil {
		return errors.New("typed release verification response did not include a release")
	}
	if strings.ToLower(cmdutil.ResolveFormat(*format)) == "json" {
		body, err := protojson.Marshal(response.Msg)
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, string(body))
		return nil
	}
	body, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(response.Msg.GetRelease())
	if err != nil {
		return err
	}
	return renderReleaseDetail(body)
}

func (c *Commands) reconcile(args []string) error {
	fs := flag.NewFlagSet("releases reconcile", flag.ContinueOnError)
	deep := fs.Bool("deep", false, "Run a deep owner observation")
	format := fs.String("format", "", "Output format (json)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: releases reconcile <release-id> [--deep]\n\n")
		fs.PrintDefaults()
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("release ID is required")
	}
	if c.operationClient == nil {
		return errors.New("typed ReleasesService client is required")
	}
	response, err := c.operationClient.Reconcile(context.Background(), connect.NewRequest(&releasesv1.ReconcileReleaseRequest{ReleaseId: remaining[0], Deep: *deep}))
	if err != nil {
		return err
	}
	if response == nil || response.Msg == nil || response.Msg.GetRelease() == nil {
		return errors.New("typed reconciliation response did not include a release")
	}
	if strings.ToLower(cmdutil.ResolveFormat(*format)) == "json" {
		body, err := protojson.Marshal(response.Msg)
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, string(body))
		return nil
	}
	body, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(response.Msg.GetRelease())
	if err != nil {
		return err
	}
	return renderReleaseDetail(body)
}

// renderReleaseDetail prints a release record (or {release: ...} envelope)
// as a human-readable report.
func renderReleaseDetail(body []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		fmt.Println(string(body))
		return nil
	}
	rel := raw
	if inner, ok := raw["release"].(map[string]interface{}); ok {
		rel = inner
	}
	if str(rel["id"]) == "" {
		rel["id"] = rel["release_id"]
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Release: %s", str(rel["id"])),
			fmt.Sprintf("Profile: %s", str(rel["profile_id"])),
			fmt.Sprintf("Candidate: %s", str(rel["candidate_id"])),
			fmt.Sprintf("Destination revision: %s", str(rel["destination_revision_id"])),
			fmt.Sprintf("Review: %s", str(rel["readiness_review_key"])),
			fmt.Sprintf("Artifact manifest: %s", str(rel["artifact_digest"])),
			fmt.Sprintf("Channel: %s", str(rel["channel"])),
			fmt.Sprintf("Status: %s", str(rel["status"])),
			fmt.Sprintf("Version: %s", str(rel["release_version"])),
		},
		ResultsHeading: "Per-Platform State",
	}
	if platforms, ok := rel["platforms"].([]interface{}); ok && len(platforms) > 0 {
		for _, p := range platforms {
			pm, ok := p.(map[string]interface{})
			if !ok {
				continue
			}
			line := fmt.Sprintf("%s: %s", str(pm["platform"]), str(pm["status"]))
			if errMsg := str(pm["error"]); errMsg != "" {
				line += " (" + errMsg + ")"
			}
			report.Results = append(report.Results, line)
		}
	} else {
		report.Results = []string{"(no platform rows)"}
	}
	if evidence, ok := rel["verification_evidence"].([]interface{}); ok && len(evidence) > 0 {
		report.RetrievalHints = []string{
			fmt.Sprintf("Verified %d platform(s); use --format json for full evidence", len(evidence)),
		}
	}
	if receipts, ok := rel["publication_receipts"].([]interface{}); ok {
		report.RetrievalHints = append(report.RetrievalHints, fmt.Sprintf("Publication receipts: %d", len(receipts)))
	}
	if receipts, ok := rel["client_update_receipts"].([]interface{}); ok {
		report.RetrievalHints = append(report.RetrievalHints, fmt.Sprintf("Client update receipts: %d", len(receipts)))
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

// renderStartResult prints either the durable operation or the completed
// release envelope returned by start.
func renderStartResult(body []byte) error {
	var env struct {
		OperationID string                   `json:"operation_id"`
		ReleaseID   string                   `json:"release_id"`
		Release     map[string]interface{}   `json:"release"`
		Steps       []map[string]interface{} `json:"steps"`
		Status      string                   `json:"status"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		fmt.Println(string(body))
		return nil
	}
	if env.OperationID != "" {
		return renderOperationResult(body)
	}
	if env.Release == nil {
		fmt.Println(string(body))
		return nil
	}
	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Release %s status: %s", str(env.Release["id"]), str(env.Release["status"])),
		},
		Changes: []string{
			fmt.Sprintf("Channel: %s", str(env.Release["channel"])),
			fmt.Sprintf("Commit: %s", truncate(str(env.Release["git_commit_hash"]), 12)),
			fmt.Sprintf("Version: %s", str(env.Release["release_version"])),
		},
		NextCommand: []string{
			fmt.Sprintf("deployment-manager releases get %s", str(env.Release["id"])),
			fmt.Sprintf("deployment-manager releases verify %s", str(env.Release["id"])),
		},
	}
	if len(env.Steps) > 0 {
		report.Changes = append(report.Changes, fmt.Sprintf("Steps: %d", len(env.Steps)))
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func renderOperationResult(body []byte) error {
	var operation struct {
		OperationID string `json:"operation_id"`
		ReleaseID   string `json:"release_id"`
		Status      string `json:"status"`
		ActiveStage string `json:"active_stage"`
		Error       string `json:"error"`
	}
	if err := json.Unmarshal(body, &operation); err != nil || operation.OperationID == "" {
		fmt.Println(string(body))
		return nil
	}
	fmt.Fprintf(os.Stdout, "Release operation %s status: %s\n", operation.OperationID, operation.Status)
	if operation.ReleaseID != "" {
		fmt.Fprintf(os.Stdout, "  release: %s\n", operation.ReleaseID)
	}
	if operation.Error != "" {
		fmt.Fprintf(os.Stdout, "  error: %s\n", operation.Error)
	}
	standing := releaseOperationStanding{
		lifecycle: operation.Status,
		phase:     operation.ActiveStage,
		reattach:  fmt.Sprintf("deployment-manager releases operation %s", operation.OperationID),
		directive: releaseOperationDirective(operation.Status),
	}
	if operation.ReleaseID != "" {
		fmt.Fprintf(os.Stdout, "  inspect release: deployment-manager releases get %s\n", operation.ReleaseID)
	}
	return operationstanding.WriteText(os.Stdout, standing)
}

type releaseOperationStanding struct {
	lifecycle string
	phase     string
	directive string
	reattach  string
}

func (s releaseOperationStanding) GetLifecycle() string                { return s.lifecycle }
func (s releaseOperationStanding) GetActivePhase() string              { return s.phase }
func (s releaseOperationStanding) GetEtaKnown() bool                   { return false }
func (s releaseOperationStanding) GetEstimatedRemainingSeconds() int32 { return 0 }
func (s releaseOperationStanding) GetDirective() string                { return s.directive }
func (s releaseOperationStanding) GetReattachCommand() string          { return s.reattach }

func releaseOperationDirective(status string) string {
	switch status {
	case "queued", "running":
		return "wait"
	case "ambiguous":
		return "reconcile"
	case "failed", "canceled":
		return "inspect"
	case "complete":
		return "inspect"
	default:
		return "inspect"
	}
}

// str converts an arbitrary value to a string for display.
func str(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// truncate clips a string to max length with "..." appended on overflow.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
