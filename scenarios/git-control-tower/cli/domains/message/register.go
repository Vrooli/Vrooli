// Package message owns explicit staged-message collection and drafting jobs.
// Commands never stage, commit, create proposals, or automatically start a new job.
package message

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"git-control-tower/cli/internal/callerheader"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
	"github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory/advisory_v1connect"
)

var clientFactory = func(core *cliapp.ScenarioApp) advisory_v1connect.AdvisoryServiceClient {
	client, base := cliapp.NewConnectHTTPClientWithTimeout(core, 70*time.Second)
	return advisory_v1connect.NewAdvisoryServiceClient(client, base, connect.WithInterceptors(callerheader.New()))
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	bindings := map[string]cliapp.PrimitiveHandler{
		"AdvisoryService.CollectStagedMessage": cliapp.ProtoOperational(collect, collectionReport),
		"AdvisoryService.CheckStagedMessage": cliapp.ProtoOperational(check, func(_ cliapp.OperationContext, r *v1.CheckStagedMessageResponse) cliapp.OperationalReport {
			return cliapp.OperationalReport{Status: []string{fmt.Sprintf("Applicable: %t", r.GetApplicable()), "Current snapshot: " + r.GetCurrentSnapshotDigest(), r.GetReason()}}
		}),
		"AdvisoryService.StartStagedMessage":        cliapp.ProtoOperational(start, jobReport),
		"AdvisoryService.GetStagedMessageJob":       cliapp.ProtoOperational(job("status"), jobReport),
		"AdvisoryService.WaitStagedMessageJob":      cliapp.ProtoOperational(job("wait"), jobReport),
		"AdvisoryService.CancelStagedMessageJob":    cliapp.ProtoOperational(job("cancel"), jobReport),
		"AdvisoryService.ReadStagedMessageEvidence": cliapp.ProtoOperational(evidence, evidenceReport),
	}
	return cliapp.LoadFromManifestPrimitives(manifest, "message", bindings)
}

func collect(ctx cliapp.OperationContext) (*v1.StagedMessageCollection, error) {
	r, err := clientFactory(ctx.Core()).CollectStagedMessage(context.Background(), connect.NewRequest(&v1.CollectStagedMessageRequest{RepositoryId: ctx.Flag("repo-id")}))
	if err != nil {
		return nil, err
	}
	return r.Msg, nil
}

func check(ctx cliapp.OperationContext) (*v1.CheckStagedMessageResponse, error) {
	digest := strings.TrimSpace(ctx.Flag("snapshot"))
	if digest == "" {
		return nil, fmt.Errorf("--snapshot is required")
	}
	r, err := clientFactory(ctx.Core()).CheckStagedMessage(context.Background(), connect.NewRequest(&v1.CheckStagedMessageRequest{RepositoryId: ctx.Flag("repo-id"), SnapshotDigest: digest}))
	if err != nil {
		return nil, err
	}
	return r.Msg, nil
}

func start(ctx cliapp.OperationContext) (*v1.StagedMessageJob, error) {
	key, collection, digest := strings.TrimSpace(ctx.Flag("request-key")), strings.TrimSpace(ctx.Flag("collection")), strings.TrimSpace(ctx.Flag("snapshot"))
	parent := strings.TrimSpace(ctx.Flag("continue-from"))
	if key == "" || digest == "" || (collection == "" && parent == "") {
		return nil, fmt.Errorf("--request-key, --snapshot and either --collection or --continue-from are required; collection never starts an agent")
	}
	if parent != "" && collection != "" {
		return nil, fmt.Errorf("choose either --collection for a new draft or --continue-from for deeper investigation")
	}
	contexts := ctx.FlagValues("context-id")
	if len(contexts) > 20 {
		return nil, fmt.Errorf("select at most 20 context IDs")
	}
	seen := map[string]bool{}
	for _, id := range contexts {
		if strings.TrimSpace(id) == "" || seen[id] {
			return nil, fmt.Errorf("context IDs must be nonempty and unique")
		}
		seen[id] = true
	}
	note := ctx.Flag("note")
	if len(note) > 4000 {
		return nil, fmt.Errorf("--note exceeds 4000 UTF-8 bytes")
	}
	if parent != "" && (len(contexts) > 0 || note != "") {
		return nil, fmt.Errorf("continuation preserves the original context and note; omit --context-id and --note")
	}
	r, err := clientFactory(ctx.Core()).StartStagedMessage(context.Background(), connect.NewRequest(&v1.StartStagedMessageRequest{RepositoryId: ctx.Flag("repo-id"), CollectionId: collection, SnapshotDigest: digest, SelectedContextIds: contexts, RequestKey: key, OperatorNote: note, ContinuationOfJobId: parent}))
	if err != nil {
		return nil, err
	}
	return r.Msg, nil
}

func evidence(ctx cliapp.OperationContext) (*v1.ReadStagedMessageEvidenceResponse, error) {
	id, digest := strings.TrimSpace(ctx.Positional("job")), strings.TrimSpace(ctx.Flag("snapshot"))
	if id == "" || digest == "" {
		return nil, fmt.Errorf("a job ID and --snapshot are required")
	}
	cursor, err := strconv.ParseUint(ctx.Flag("cursor"), 10, 32)
	if err != nil || cursor > 4096 {
		return nil, fmt.Errorf("--cursor must be 0 through 4096")
	}
	limit, err := strconv.ParseUint(ctx.Flag("max-bytes"), 10, 32)
	if err != nil || limit < 1024 || limit > 49152 {
		return nil, fmt.Errorf("--max-bytes must be 1024 through 49152")
	}
	inventory := ctx.BoolFlag("inventory")
	var ids []uint32
	seen := map[uint64]bool{}
	for _, value := range ctx.FlagValues("file-id") {
		id, err := strconv.ParseUint(value, 10, 32)
		if err != nil || id < 1 || id > 4096 || seen[id] {
			return nil, fmt.Errorf("--file-id must be a unique stable ID from 1 through 4096")
		}
		seen[id] = true
		ids = append(ids, uint32(id))
	}
	if len(ids) > 64 || (len(ids) > 0 && (inventory || cursor != 0)) {
		return nil, fmt.Errorf("select at most 64 file IDs without --inventory or a nonzero --cursor")
	}
	r, err := clientFactory(ctx.Core()).ReadStagedMessageEvidence(context.Background(), connect.NewRequest(&v1.ReadStagedMessageEvidenceRequest{RepositoryId: ctx.Flag("repo-id"), JobId: id, SnapshotDigest: digest, Cursor: uint32(cursor), MaxBytes: uint32(limit), InventoryOnly: inventory, FileIds: ids}))
	if err != nil {
		return nil, err
	}
	return r.Msg, nil
}

func evidenceReport(_ cliapp.OperationContext, r *v1.ReadStagedMessageEvidenceResponse) cliapp.OperationalReport {
	report := cliapp.OperationalReport{Status: []string{"Snapshot: " + r.GetSnapshotDigest(), fmt.Sprintf("Files in page: %d; selection: %d; end reached: %t", len(r.GetFiles()), r.GetTotalFiles(), r.GetComplete())}}
	for _, file := range r.GetFiles() {
		report.Triage = append(report.Triage, cliapp.TriageGroup{Heading: fmt.Sprintf("%d: %s", file.GetFileId(), file.GetPath()), Items: []string{file.GetPatch(), file.GetOmissionReason()}})
	}
	if !r.GetComplete() && r.GetNextCursor() > 0 {
		report.NextSteps = []string{fmt.Sprintf("Read the next immutable evidence page with --cursor %d using the same job and snapshot; use --json for structured coverage details.", r.GetNextCursor())}
	}
	if !r.GetComplete() && r.GetNextCursor() == 0 {
		report.NextSteps = []string{"Retry only the requested file IDs absent from this page; targeted reads do not use a sequential cursor."}
	}
	return report
}

func job(action string) func(cliapp.OperationContext) (*v1.StagedMessageJob, error) {
	return func(ctx cliapp.OperationContext) (*v1.StagedMessageJob, error) {
		id, key := strings.TrimSpace(ctx.Positional("job")), strings.TrimSpace(ctx.Flag("request-key"))
		if id == "" && key == "" {
			return nil, fmt.Errorf("a job ID or --request-key is required")
		}
		req := &v1.StagedMessageJobRequest{RepositoryId: ctx.Flag("repo-id"), JobId: id, RequestKey: key}
		if action == "wait" {
			seconds, err := strconv.Atoi(ctx.Flag("timeout"))
			if err != nil || seconds < 1 || seconds > 55 {
				return nil, fmt.Errorf("--timeout must be 1 through 55 seconds")
			}
			req.TimeoutSeconds = int32(seconds)
		}
		client := clientFactory(ctx.Core())
		var r *connect.Response[v1.StagedMessageJob]
		var err error
		switch action {
		case "status":
			r, err = client.GetStagedMessageJob(context.Background(), connect.NewRequest(req))
		case "wait":
			r, err = client.WaitStagedMessageJob(context.Background(), connect.NewRequest(req))
		case "cancel":
			r, err = client.CancelStagedMessageJob(context.Background(), connect.NewRequest(req))
		}
		if err != nil {
			return nil, err
		}
		return r.Msg, nil
	}
}

func collectionReport(_ cliapp.OperationContext, r *v1.StagedMessageCollection) cliapp.OperationalReport {
	report := cliapp.OperationalReport{Status: []string{"Collection: " + r.GetId(), "Status: " + r.GetStatus(), "Snapshot: " + r.GetSnapshot().GetDigest(), fmt.Sprintf("Suggestions: %d; possible related contexts: %d", len(r.GetSuggestions()), len(r.GetContexts()))}, NextSteps: []string{"Review the evidence; start drafting only with message start and an explicit request key. No agent was started."}}
	for _, s := range r.GetSuggestions() {
		report.Triage = append(report.Triage, cliapp.TriageGroup{Heading: s.GetId(), Items: []string{s.GetTitle(), s.GetBody()}})
	}
	for _, c := range r.GetContexts() {
		report.Triage = append(report.Triage, cliapp.TriageGroup{Heading: c.GetId(), Items: []string{c.GetTitle(), c.GetDetail(), "Possible paths: " + strings.Join(c.GetMatchedPaths(), ", ")}})
	}
	for _, o := range r.GetOmissions() {
		report.Triage = append(report.Triage, cliapp.TriageGroup{Heading: "Evidence limitation", Items: []string{o.GetPath(), o.GetReason()}})
	}
	return report
}

func jobReport(_ cliapp.OperationContext, r *v1.StagedMessageJob) cliapp.OperationalReport {
	report := cliapp.OperationalReport{Status: []string{"Job: " + r.GetId(), "Status: " + r.GetStatus(), "Snapshot: " + r.GetSnapshotDigest(), "Phase: " + r.GetPhase(), r.GetError()}}
	if result := r.GetResult(); result != nil {
		report.Triage = append(report.Triage, cliapp.TriageGroup{Heading: "Suggested message", Items: []string{result.GetTitle(), result.GetBody()}})
	}
	if !r.GetApplicable() {
		report.Triage = append(report.Triage, cliapp.TriageGroup{Heading: "Applicability", Items: []string{r.GetApplicabilityReason()}})
	}
	if r.GetStatus() == "queued" || r.GetStatus() == "running" || r.GetStatus() == "cancelling" {
		report.NextSteps = []string{"git-control-tower message wait " + r.GetId() + " --repo-id " + r.GetRepositoryId() + " --json", "Cancellation remains pending until Agent Manager confirms its disposition."}
	} else {
		report.NextSteps = []string{"Review the result before filling the commit editor. Repository commit remains a separate action."}
	}
	return report
}
