// Package proposal implements `git-control-tower proposal`: commit proposals
// bound to exact repository content.
//
// anchor, create, list, show, edit, withdraw and refresh write only Git
// Control Tower's database, so orchestrators may call them. approve is
// human-only: it shows the exact files, flags and message, asks for
// "confirm", then prepares and confirms a single-use intent for
// repo.apply_proposal and applies the proposal once.
package proposal

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"git-control-tower/cli/internal/callerheader"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	humancontrolv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/human_control"
	humancontrolconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/human_control/human_control_v1connect"
	proposalsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/proposals"
	proposalsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/proposals/proposals_v1connect"
)

// applyOperation is the human-control operation for applying a proposal.
const applyOperation = "repo.apply_proposal"

// applyClientTimeout covers the server's commit budget, which includes the
// configured pre-commit check (up to two runs of at most 30 minutes total).
const applyClientTimeout = 31 * time.Minute

// Client factories. Overridable in tests.
var (
	proposalClientFactory = func(core *cliapp.ScenarioApp) proposalsconnect.ProposalServiceClient {
		httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
		return proposalsconnect.NewProposalServiceClient(httpClient, baseURL, connect.WithInterceptors(callerheader.New()))
	}
	applyClientFactory = func(core *cliapp.ScenarioApp) proposalsconnect.ProposalServiceClient {
		httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, applyClientTimeout)
		return proposalsconnect.NewProposalServiceClient(httpClient, baseURL, connect.WithInterceptors(callerheader.New()))
	}
	humanControlClientFactory = func(core *cliapp.ScenarioApp) humancontrolconnect.HumanControlServiceClient {
		httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
		return humancontrolconnect.NewHumanControlServiceClient(httpClient, baseURL, connect.WithInterceptors(callerheader.New()))
	}
	confirmInput  io.Reader = os.Stdin
	output        io.Writer = os.Stdout
	creatorRunEnv           = "VROOLI_RUN_ID"
	// detectCaller classifies the local process. Agents never approve, even
	// in a session the server would verify as the OS user.
	detectCaller = cliutil.DetectCallerKind
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "proposal",
		Description: "Commit proposals: exact files and a linked message; the operator approves",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "anchor", NeedsAPI: true, Description: "Fingerprint files already dirty in a scope at epoch admission (--effort REF --epoch E<n> --scope GLOB...)", Run: func(a []string) error { return runAnchor(core, a) }},
			{Name: "create", NeedsAPI: true, Description: "Create a proposal from listed paths (--request-file F | --effort --epoch --subject [--body-file] --path P... [--run ID...]); --validate-only stores nothing", Run: func(a []string) error { return runCreate(core, a) }},
			{Name: "list", NeedsAPI: true, Description: "List proposals with freshness ([--state open|committed|withdrawn|superseded|all] [--effort REF])", Run: func(a []string) error { return runList(core, a) }},
			{Name: "show", NeedsAPI: true, Description: "Show one proposal: files, flags, exclusions, message and history (ID [--message])", Run: func(a []string) error { return runShow(core, a) }},
			{Name: "edit", NeedsAPI: true, Description: "Edit subject, body or trailers, or drop files (ID --revision N [--subject S] [--body-file F] [--trailer K=V...] [--remove-path P...])", Run: func(a []string) error { return runEdit(core, a) }},
			{Name: "approve", NeedsAPI: true, Description: "Human only: review and commit exactly the proposal (ID [--yes] [--skip-precommit])", Run: func(a []string) error { return runApprove(core, a) }},
			{Name: "withdraw", NeedsAPI: true, Description: "Withdraw an open proposal (ID [--reason R])", Run: func(a []string) error { return runWithdraw(core, a) }},
			{Name: "refresh", NeedsAPI: true, Description: "Re-hash current content into a new revision (ID [--revision N])", Run: func(a []string) error { return runRefresh(core, a) }},
		},
	}
}

type repeated []string

func (r *repeated) String() string     { return strings.Join(*r, ",") }
func (r *repeated) Set(v string) error { *r = append(*r, v); return nil }

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	return fs
}

func printJSON(msg proto.Message) error {
	body, err := protojson.MarshalOptions{Multiline: true, UseProtoNames: true}.Marshal(msg)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(body)
	return nil
}

func singleID(fs *flag.FlagSet) (string, error) {
	if fs.NArg() != 1 || strings.TrimSpace(fs.Arg(0)) == "" {
		return "", fmt.Errorf("usage: git-control-tower proposal %s <proposal-id>", fs.Name())
	}
	return strings.TrimSpace(fs.Arg(0)), nil
}

func runAnchor(core *cliapp.ScenarioApp, args []string) error {
	fs := newFlagSet("anchor")
	var scopes repeated
	effort := fs.String("effort", "", "Agent Manager effort reference, e.g. effort:browser-automation-studio-rehabilitation")
	epoch := fs.String("epoch", "", "Epoch label, e.g. E27")
	repoID := fs.String("repo-id", "", "Repository ID (default: active repository)")
	jsonOut := fs.Bool("json", false, "Emit JSON")
	fs.Var(&scopes, "scope", "Path or glob in scope (repeatable); include declared shared paths")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	scopes = append(scopes, fs.Args()...)
	if strings.TrimSpace(*effort) == "" || strings.TrimSpace(*epoch) == "" || len(scopes) == 0 {
		return errors.New("usage: git-control-tower proposal anchor --effort REF --epoch E<n> --scope GLOB [--scope GLOB...]")
	}
	resp, err := proposalClientFactory(core).AnchorScope(context.Background(), connect.NewRequest(&proposalsv1.AnchorScopeRequest{
		RepositoryId: *repoID, EffortRef: *effort, Epoch: *epoch, Scopes: scopes, CreatorRunId: os.Getenv(creatorRunEnv),
	}))
	if err != nil {
		return err
	}
	anchor := resp.Msg.GetAnchor()
	if *jsonOut {
		return printJSON(anchor)
	}
	fmt.Fprintf(output, "Anchor %s: %d dirty files in scope at %s (%s %s)\n", anchor.GetId(), len(anchor.GetFiles()), short(anchor.GetHead()), anchor.GetEffortRef(), anchor.GetEpoch())
	return nil
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := newFlagSet("create")
	var paths, runs, plans, trailerFlags, gates repeated
	requestFile := fs.String("request-file", "", "CreateProposalRequest as proto JSON; flags override its fields")
	effort := fs.String("effort", "", "Agent Manager effort reference")
	effortRevision := fs.String("effort-revision", "", "Effort revision")
	epoch := fs.String("epoch", "", "Epoch label, e.g. E27")
	anchorID := fs.String("anchor", "", "Anchor ID (default: newest anchor for the effort and epoch)")
	subject := fs.String("subject", "", "Commit subject, e.g. \"bas: one ReplaySpec renderer (E27)\"")
	body := fs.String("body", "", "Commit body")
	bodyFile := fs.String("body-file", "", "Read the commit body from a file")
	epochFile := fs.String("epoch-file", "", "Epoch file that records acceptance")
	acceptedLine := fs.String("accepted-line-sha256", "", "SHA-256 of the epoch file's ACCEPTED line")
	metricBefore := fs.String("metric-before", "", "Metric before the epoch")
	metricAfter := fs.String("metric-after", "", "Metric after the epoch")
	repoID := fs.String("repo-id", "", "Repository ID (default: active repository)")
	validateOnly := fs.Bool("validate-only", false, "Build and validate without storing")
	jsonOut := fs.Bool("json", false, "Emit JSON")
	fs.Var(&paths, "path", "File to commit (repeatable; positional paths also accepted)")
	fs.Var(&runs, "run", "Agent Manager run ID (repeatable)")
	fs.Var(&plans, "plan", "Plan Manager plan reference (repeatable)")
	fs.Var(&trailerFlags, "trailer", "Extra trailer KEY=VALUE (repeatable)")
	fs.Var(&gates, "gate", "Gate receipt digest or ID (repeatable)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	req := &proposalsv1.CreateProposalRequest{}
	if *requestFile != "" {
		data, err := os.ReadFile(*requestFile)
		if err != nil {
			return err
		}
		if err := (protojson.UnmarshalOptions{}).Unmarshal(data, req); err != nil {
			return fmt.Errorf("parse %s: %w", *requestFile, err)
		}
	}
	if req.Work == nil {
		req.Work = &proposalsv1.ProposalWork{}
	}
	if req.Evidence == nil {
		req.Evidence = &proposalsv1.ProposalEvidence{}
	}
	setIf(&req.RepositoryId, *repoID)
	setIf(&req.AnchorId, *anchorID)
	setIf(&req.Work.EffortRef, *effort)
	setIf(&req.Work.EffortRevision, *effortRevision)
	setIf(&req.Work.Epoch, *epoch)
	setIf(&req.Subject, *subject)
	setIf(&req.Body, *body)
	if *bodyFile != "" {
		data, err := os.ReadFile(*bodyFile)
		if err != nil {
			return err
		}
		req.Body = string(data)
	}
	setIf(&req.Evidence.EpochFile, *epochFile)
	setIf(&req.Evidence.AcceptedLineSha256, *acceptedLine)
	setIf(&req.Evidence.MetricBefore, *metricBefore)
	setIf(&req.Evidence.MetricAfter, *metricAfter)
	req.Evidence.GateReceipts = append(req.Evidence.GateReceipts, gates...)
	req.Work.RunIds = append(req.Work.RunIds, runs...)
	req.Work.Plans = append(req.Work.Plans, plans...)
	req.Paths = append(append(req.Paths, paths...), fs.Args()...)
	for _, raw := range trailerFlags {
		key, value, ok := strings.Cut(raw, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return fmt.Errorf("--trailer must be KEY=VALUE, got %q", raw)
		}
		req.Trailers = append(req.Trailers, &proposalsv1.ProposalTrailer{Key: strings.TrimSpace(key), Value: strings.TrimSpace(value)})
	}
	if req.CreatorRunId == "" {
		req.CreatorRunId = os.Getenv(creatorRunEnv)
	}
	if *validateOnly {
		req.ValidateOnly = true
	}
	if len(req.Paths) == 0 || strings.TrimSpace(req.Subject) == "" {
		return errors.New("usage: git-control-tower proposal create --subject S --path P [--path P...] [--effort REF --epoch E<n>] (or --request-file F)")
	}
	resp, err := proposalClientFactory(core).CreateProposal(context.Background(), connect.NewRequest(req))
	if err != nil {
		return err
	}
	if *jsonOut {
		return printJSON(resp.Msg)
	}
	p := resp.Msg.GetProposal()
	verb := "Created"
	if !resp.Msg.GetStored() {
		verb = "Validated (not stored)"
	}
	fmt.Fprintf(output, "%s proposal %s r%d: %d files, %d excluded\n", verb, p.GetId(), p.GetRevision(), len(p.GetFiles()), len(p.GetExcluded()))
	if id := resp.Msg.GetSupersededId(); id != "" {
		fmt.Fprintf(output, "Superseded %s\n", id)
	}
	printIssues(resp.Msg.GetIssues())
	printFiles(p)
	return nil
}

func setIf(target *string, value string) {
	if strings.TrimSpace(value) != "" {
		*target = value
	}
}

func parseStates(raw string) ([]proposalsv1.ProposalState, error) {
	var states []proposalsv1.ProposalState
	for _, part := range strings.Split(raw, ",") {
		switch strings.TrimSpace(strings.ToLower(part)) {
		case "", "open":
			states = append(states, proposalsv1.ProposalState_PROPOSAL_STATE_OPEN)
		case "committed":
			states = append(states, proposalsv1.ProposalState_PROPOSAL_STATE_COMMITTED)
		case "withdrawn":
			states = append(states, proposalsv1.ProposalState_PROPOSAL_STATE_WITHDRAWN)
		case "superseded":
			states = append(states, proposalsv1.ProposalState_PROPOSAL_STATE_SUPERSEDED)
		case "all":
			return []proposalsv1.ProposalState{proposalsv1.ProposalState_PROPOSAL_STATE_OPEN, proposalsv1.ProposalState_PROPOSAL_STATE_COMMITTED, proposalsv1.ProposalState_PROPOSAL_STATE_WITHDRAWN, proposalsv1.ProposalState_PROPOSAL_STATE_SUPERSEDED}, nil
		default:
			return nil, fmt.Errorf("unknown state %q", part)
		}
	}
	return states, nil
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	fs := newFlagSet("list")
	state := fs.String("state", "open", "open, committed, withdrawn, superseded, all, or a comma list")
	effort := fs.String("effort", "", "Only proposals for this effort reference")
	limit := fs.Int("limit", 50, "Maximum proposals")
	repoID := fs.String("repo-id", "", "Repository ID (default: active repository)")
	jsonOut := fs.Bool("json", false, "Emit JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	states, err := parseStates(*state)
	if err != nil {
		return err
	}
	resp, err := proposalClientFactory(core).ListProposals(context.Background(), connect.NewRequest(&proposalsv1.ListProposalsRequest{RepositoryId: *repoID, States: states, EffortRef: *effort, Limit: int32(*limit)}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return printJSON(resp.Msg)
	}
	fmt.Fprintf(output, "Proposals (%d open)\n", resp.Msg.GetOpenCount())
	for _, p := range resp.Msg.GetProposals() {
		fmt.Fprintf(output, "  %s r%d %-10s %-11s %3d files  %s %s  %s\n", p.GetId(), p.GetRevision(), stateLabel(p.GetState()), freshnessLabel(p.GetFreshness().GetState()),
			len(p.GetFiles()), p.GetWork().GetEpoch(), flagSummary(p), p.GetMessage().GetSubject())
	}
	return nil
}

func runShow(core *cliapp.ScenarioApp, args []string) error {
	fs := newFlagSet("show")
	messageOnly := fs.Bool("message", false, "Print only the rendered commit message")
	jsonOut := fs.Bool("json", false, "Emit JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	id, err := singleID(fs)
	if err != nil {
		return err
	}
	resp, err := proposalClientFactory(core).GetProposal(context.Background(), connect.NewRequest(&proposalsv1.GetProposalRequest{Id: id}))
	if err != nil {
		return err
	}
	p := resp.Msg.GetProposal()
	switch {
	case *jsonOut:
		return printJSON(p)
	case *messageOnly:
		fmt.Fprintln(output, p.GetMessage().GetRendered())
		return nil
	}
	printProposal(p)
	return nil
}

func runEdit(core *cliapp.ScenarioApp, args []string) error {
	fs := newFlagSet("edit")
	var trailerFlags, removePaths repeated
	revision := fs.Int("revision", 0, "Revision you reviewed (required)")
	subject := fs.String("subject", "", "New subject")
	body := fs.String("body", "", "New body")
	bodyFile := fs.String("body-file", "", "Read the new body from a file")
	replaceTrailers := fs.Bool("replace-trailers", false, "Replace the trailer list with --trailer values")
	jsonOut := fs.Bool("json", false, "Emit JSON")
	fs.Var(&trailerFlags, "trailer", "Trailer KEY=VALUE (repeatable; requires --replace-trailers)")
	fs.Var(&removePaths, "remove-path", "Drop a file from the proposal (repeatable)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	id, err := singleID(fs)
	if err != nil {
		return err
	}
	if *revision <= 0 {
		return errors.New("--revision is required; read it with `git-control-tower proposal show <id>`")
	}
	req := &proposalsv1.EditProposalRequest{Id: id, ExpectedRevision: int32(*revision), RemovePaths: removePaths, ReplaceTrailers: *replaceTrailers}
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "subject":
			req.Subject = proto.String(*subject)
		case "body":
			req.Body = proto.String(*body)
		}
	})
	if *bodyFile != "" {
		data, err := os.ReadFile(*bodyFile)
		if err != nil {
			return err
		}
		req.Body = proto.String(string(data))
	}
	for _, raw := range trailerFlags {
		key, value, ok := strings.Cut(raw, "=")
		if !ok {
			return fmt.Errorf("--trailer must be KEY=VALUE, got %q", raw)
		}
		req.Trailers = append(req.Trailers, &proposalsv1.ProposalTrailer{Key: strings.TrimSpace(key), Value: strings.TrimSpace(value)})
	}
	if len(req.Trailers) > 0 && !req.ReplaceTrailers {
		return errors.New("--trailer requires --replace-trailers; pass the full list you want")
	}
	resp, err := proposalClientFactory(core).EditProposal(context.Background(), connect.NewRequest(req))
	if err != nil {
		return err
	}
	if *jsonOut {
		return printJSON(resp.Msg)
	}
	fmt.Fprintf(output, "Edited %s: now r%d\n", id, resp.Msg.GetProposal().GetRevision())
	printIssues(resp.Msg.GetIssues())
	return nil
}

func runWithdraw(core *cliapp.ScenarioApp, args []string) error {
	fs := newFlagSet("withdraw")
	reason := fs.String("reason", "", "Reason recorded in history")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	id, err := singleID(fs)
	if err != nil {
		return err
	}
	if _, err := proposalClientFactory(core).WithdrawProposal(context.Background(), connect.NewRequest(&proposalsv1.WithdrawProposalRequest{Id: id, Reason: *reason})); err != nil {
		return err
	}
	fmt.Fprintf(output, "Withdrew %s\n", id)
	return nil
}

func runRefresh(core *cliapp.ScenarioApp, args []string) error {
	fs := newFlagSet("refresh")
	revision := fs.Int("revision", 0, "Revision you reviewed (default: current)")
	jsonOut := fs.Bool("json", false, "Emit JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	id, err := singleID(fs)
	if err != nil {
		return err
	}
	resp, err := proposalClientFactory(core).RefreshProposal(context.Background(), connect.NewRequest(&proposalsv1.RefreshProposalRequest{Id: id, ExpectedRevision: int32(*revision)}))
	if err != nil {
		return err
	}
	if *jsonOut {
		return printJSON(resp.Msg)
	}
	p := resp.Msg.GetProposal()
	fmt.Fprintf(output, "Refreshed %s: r%d %s, %d files\n", id, p.GetRevision(), stateLabel(p.GetState()), len(p.GetFiles()))
	if dropped := resp.Msg.GetDroppedPaths(); len(dropped) > 0 {
		fmt.Fprintf(output, "Dropped (clean now): %s\n", strings.Join(dropped, ", "))
	}
	return nil
}

// runApprove is the human approval path. It never commits without the
// operator typing "confirm" (or passing --yes), and it applies exactly once.
func runApprove(core *cliapp.ScenarioApp, args []string) error {
	fs := newFlagSet("approve")
	yes := fs.Bool("yes", false, "Skip the confirmation prompt")
	skipPrecommit := fs.Bool("skip-precommit", false, "Skip the configured pre-commit check for this commit")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	id, err := singleID(fs)
	if err != nil {
		return err
	}
	if kind := detectCaller(); kind != cliutil.CallerKindHuman {
		return fmt.Errorf("approval refused: this process runs as %s; only the operator approves proposals (run it from your own terminal or the GCT UI)", kind)
	}
	ctx := context.Background()
	human := humanControlClientFactory(core)
	authority, err := human.GetAuthorityStatus(ctx, connect.NewRequest(&humancontrolv1.GetAuthorityStatusRequest{}))
	if err != nil {
		return err
	}
	if !authority.Msg.GetCanMutate() {
		reason := authority.Msg.GetReason()
		if reason == "" {
			reason = "authenticate through the configured provider"
		}
		return fmt.Errorf("approval unavailable: %s; only the operator can approve proposals", reason)
	}
	got, err := proposalClientFactory(core).GetProposal(ctx, connect.NewRequest(&proposalsv1.GetProposalRequest{Id: id}))
	if err != nil {
		return err
	}
	p := got.Msg.GetProposal()
	if p.GetState() != proposalsv1.ProposalState_PROPOSAL_STATE_OPEN {
		return fmt.Errorf("proposal %s is %s; only open proposals can be approved", id, stateLabel(p.GetState()))
	}
	printProposal(p)
	if fresh := p.GetFreshness(); fresh.GetState() != proposalsv1.FreshnessState_FRESHNESS_STATE_FRESH {
		return fmt.Errorf("proposal is %s; run `git-control-tower proposal refresh %s` and review it again", freshnessLabel(fresh.GetState()), id)
	}
	if staged := p.GetFreshness().GetForeignStagedPaths(); len(staged) > 0 {
		return fmt.Errorf("other paths are staged (%s); unstage them first so the commit holds only the proposal", strings.Join(staged, ", "))
	}
	subjectContext := "proposal:" + id + "@" + strconv.Itoa(int(p.GetRevision()))
	preview, err := human.PrepareMutation(ctx, connect.NewRequest(&humancontrolv1.PrepareMutationRequest{RepositoryId: p.GetRepositoryId(), Operation: applyOperation, SubjectContext: subjectContext}))
	if err != nil {
		return err
	}
	fmt.Fprintf(output, "\nRepository: %s\nBranch: %s\nRevision: %s\nSubject digest: %s\n", preview.Msg.GetRepositoryPath(), preview.Msg.GetBranch(), preview.Msg.GetExpectedRevision(), preview.Msg.GetSubjectDigest())
	if !*yes {
		fmt.Fprint(output, "Type 'confirm' to commit exactly this proposal: ")
		line, readErr := bufio.NewReader(confirmInput).ReadString('\n')
		if readErr != nil && readErr != io.EOF {
			return readErr
		}
		if strings.TrimSpace(line) != "confirm" {
			return errors.New("approval not confirmed; nothing was staged or committed")
		}
	}
	intent, err := human.ConfirmMutation(ctx, connect.NewRequest(&humancontrolv1.ConfirmMutationRequest{
		RepositoryId: preview.Msg.GetRepositoryId(), Operation: applyOperation, SubjectContext: subjectContext,
		ExpectedRevision: preview.Msg.GetExpectedRevision(), SubjectDigest: preview.Msg.GetSubjectDigest(),
	}))
	if err != nil {
		return err
	}
	if intent.Msg.GetIntentId() == "" {
		return errors.New("mutation intent response did not contain an intent id")
	}
	resp, err := applyClientFactory(core).ApplyProposal(ctx, connect.NewRequest(&proposalsv1.ApplyProposalRequest{
		RepositoryId: preview.Msg.GetRepositoryId(), IntentId: intent.Msg.GetIntentId(), Id: id, Revision: p.GetRevision(), SkipPrecommitOnce: *skipPrecommit,
	}))
	if err != nil {
		return err
	}
	return reportApply(resp.Msg)
}

func reportApply(result *proposalsv1.ApplyProposalResponse) error {
	if result.GetSuccess() {
		fmt.Fprintf(output, "Committed %s\n", result.GetCommitOid())
		if !result.GetCommitVerified() {
			for _, note := range result.GetVerificationNotes() {
				fmt.Fprintf(output, "  ! %s\n", note)
			}
		}
		return nil
	}
	if precommit := result.GetPrecommit(); precommit != nil && precommit.GetStatus() != "passed" {
		fmt.Fprintf(output, "Pre-commit %s (exit %d): %s\n", precommit.GetStatus(), precommit.GetExitCode(), precommit.GetSummary())
		if tail := strings.TrimSpace(precommit.GetStderr() + "\n" + precommit.GetStdout()); tail != "" {
			fmt.Fprintln(output, tail)
		}
	}
	if refusal := result.GetRefusal(); refusal != nil {
		message := fmt.Sprintf("apply refused (%s): %s", refusal.GetCode(), refusal.GetDetail())
		if len(refusal.GetPaths()) > 0 {
			message += "\n  " + strings.Join(refusal.GetPaths(), "\n  ")
		}
		return errors.New(message)
	}
	return fmt.Errorf("apply failed: %s", result.GetError())
}

// --- rendering ---

func short(oid string) string {
	if len(oid) > 12 {
		return oid[:12]
	}
	if oid == "" {
		return "(unborn)"
	}
	return oid
}

func stateLabel(state proposalsv1.ProposalState) string {
	return strings.ToLower(strings.TrimPrefix(state.String(), "PROPOSAL_STATE_"))
}

func freshnessLabel(state proposalsv1.FreshnessState) string {
	return strings.ToLower(strings.TrimPrefix(state.String(), "FRESHNESS_STATE_"))
}

func flagSummary(p *proposalsv1.Proposal) string {
	counts := map[string]int{}
	var order []string
	for _, file := range p.GetFiles() {
		for _, flag := range file.GetFlags() {
			if counts[flag.GetCode()] == 0 {
				order = append(order, flag.GetCode())
			}
			counts[flag.GetCode()]++
		}
	}
	if len(order) == 0 {
		return "[no flags]"
	}
	parts := make([]string, 0, len(order))
	for _, code := range order {
		parts = append(parts, fmt.Sprintf("%s:%d", code, counts[code]))
	}
	return "[" + strings.Join(parts, " ") + "]"
}

func printIssues(issues []*proposalsv1.TrailerIssue) {
	for _, issue := range issues {
		level := "warning"
		if issue.GetSeverity() == proposalsv1.IssueSeverity_ISSUE_SEVERITY_ERROR {
			level = "error"
		}
		fmt.Fprintf(output, "  %s: %s: %s\n", level, issue.GetKey(), issue.GetMessage())
	}
}

func printFiles(p *proposalsv1.Proposal) {
	for _, file := range p.GetFiles() {
		marker := map[proposalsv1.FileChangeKind]string{proposalsv1.FileChangeKind_FILE_CHANGE_KIND_ADDED: "A", proposalsv1.FileChangeKind_FILE_CHANGE_KIND_MODIFIED: "M", proposalsv1.FileChangeKind_FILE_CHANGE_KIND_DELETED: "D"}[file.GetKind()]
		var flags []string
		for _, flag := range file.GetFlags() {
			text := flag.GetCode()
			if flag.GetDetail() != "" && flag.GetCode() != "mixed_prior_uncommitted" {
				text += "=" + flag.GetDetail()
			}
			flags = append(flags, text)
		}
		drift := ""
		if file.GetDrifted() {
			drift = " DRIFTED"
		}
		fmt.Fprintf(output, "  %s %s%s %s\n", marker, file.GetPath(), drift, strings.Join(flags, " "))
	}
	for _, exclusion := range p.GetExcluded() {
		fmt.Fprintf(output, "  - %s (excluded: %s)\n", exclusion.GetPath(), exclusion.GetReason())
	}
}

func printProposal(p *proposalsv1.Proposal) {
	fresh := p.GetFreshness()
	fmt.Fprintf(output, "Proposal %s r%d (%s, %s)\n", p.GetId(), p.GetRevision(), stateLabel(p.GetState()), freshnessLabel(fresh.GetState()))
	fmt.Fprintf(output, "Work: %s %s  base %s on %s  by %s (%s)\n", p.GetWork().GetEffortRef(), p.GetWork().GetEpoch(), short(p.GetBaseHead()), p.GetBranch(), p.GetCreatedBy().GetSubject(), p.GetCreatedBy().GetKind())
	if p.GetCommitOid() != "" {
		fmt.Fprintf(output, "Commit: %s\n", p.GetCommitOid())
	}
	if len(fresh.GetDriftedPaths()) > 0 {
		fmt.Fprintf(output, "Drifted: %s\n", strings.Join(fresh.GetDriftedPaths(), ", "))
	}
	if len(fresh.GetBaseChangedPaths()) > 0 {
		fmt.Fprintf(output, "Changed since base: %s\n", strings.Join(fresh.GetBaseChangedPaths(), ", "))
	}
	if len(fresh.GetForeignStagedPaths()) > 0 {
		fmt.Fprintf(output, "Other staged paths: %s\n", strings.Join(fresh.GetForeignStagedPaths(), ", "))
	}
	fmt.Fprintf(output, "Files (%d):\n", len(p.GetFiles()))
	printFiles(p)
	fmt.Fprintln(output, "Message:")
	for _, line := range strings.Split(p.GetMessage().GetRendered(), "\n") {
		fmt.Fprintf(output, "  | %s\n", line)
	}
	for _, trailer := range p.GetMessage().GetTrailers() {
		fmt.Fprintf(output, "  %s: %s [%s]\n", trailer.GetKey(), trailer.GetValue(), strings.ToLower(strings.TrimPrefix(trailer.GetResolution().String(), "TRAILER_RESOLUTION_")))
	}
	printIssues(p.GetIssues())
}
