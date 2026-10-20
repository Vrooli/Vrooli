// This file implements the operator decision commands (DL-5): raise a
// decision into a goal home and notify the operator once, answer it from the
// desktop, list it, and sync the answer verbatim into FEEDBACK.md.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-manager/internal/effortdecision"
	"agent-manager/internal/goalhome"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const decisionUsage = `usage:
  agent-manager effort decision raise <goal home> --id <id> --question <text> --option key=label --option key=label --recommend <key> [--why <text>] [--default <key> --reversible] [--deadline 24h|<RFC3339>] [--urgency low|normal|high|critical] [--big-picture <text>] [--technical-details <text>] [--note <legacy technical-details fallback>] [--context-url <url>] [--home-ref <repo-relative dir>] [--effort-ref <ref>] [--dry-run] [--json]
  agent-manager effort decision answer <goal home> <id> <option key> [--note <text>] [--home-ref <dir>] [--local-owner] [--json]
  agent-manager effort decision list [<goal home>] [--effort-ref <ref>] [--unrecorded] [--home-ref <dir>] [--json]
  agent-manager effort decision sync <goal home> [--home-ref <dir>] [--json]`

// optionFlags collects repeated --option key=label values.
type optionFlags []goalhome.DecisionOption

func (o *optionFlags) String() string { return fmt.Sprint(*o) }

func (o *optionFlags) Set(value string) error {
	key, label, _ := strings.Cut(value, "=")
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("option %q must be key=label", value)
	}
	*o = append(*o, goalhome.DecisionOption{Key: key, Label: strings.TrimSpace(label)})
	return nil
}

// effortDecision is `agent-manager effort decision <raise|answer|list|sync>`.
func (a *App) effortDecision(args []string) error {
	if len(args) == 0 {
		return errors.New(decisionUsage)
	}
	switch args[0] {
	case "raise":
		return a.decisionRaise(args[1:])
	case "answer":
		return a.decisionAnswer(args[1:])
	case "list":
		return a.decisionList(args[1:])
	case "sync":
		return a.decisionSync(args[1:])
	}
	return errors.New(decisionUsage)
}

// decisionHomeRef is the server's key for a goal home: the directory relative
// to the repository root. --home-ref names it explicitly, for a goal-home copy
// outside the repository.
func decisionHomeRef(dir, override string) (string, error) {
	if strings.TrimSpace(override) != "" {
		return effortdecision.NormalizeGoalHome(override)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	root := cliutil.ResolveRepoRoot()
	if root == "" {
		return "", errors.New("repository root unknown; pass --home-ref <repo-relative goal home>")
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil || !filepath.IsLocal(rel) {
		return "", fmt.Errorf("goal home %s is outside the repository %s; pass --home-ref <repo-relative goal home>", dir, root)
	}
	return effortdecision.NormalizeGoalHome(rel)
}

// parseDecisionDeadline accepts a duration from now or an RFC 3339 time.
// relative is true for a duration.
func parseDecisionDeadline(raw string, now time.Time) (time.Time, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false, nil
	}
	if d, err := time.ParseDuration(raw); err == nil {
		if d < goalhome.MinDecisionDefaultDelay {
			return time.Time{}, true, fmt.Errorf("--deadline must be at least %s away", goalhome.MinDecisionDefaultDelay)
		}
		return now.Add(d).UTC().Truncate(time.Second), true, nil
	}
	at, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("--deadline %q is neither a duration nor an RFC 3339 time", raw)
	}
	if at.Sub(now) < goalhome.MinDecisionDefaultDelay {
		return time.Time{}, false, fmt.Errorf("--deadline must be at least %s away", goalhome.MinDecisionDefaultDelay)
	}
	return at.UTC(), false, nil
}

func (a *App) decisionRaise(args []string) error {
	fs := flag.NewFlagSet("effort decision raise", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	var options optionFlags
	fs.Var(&options, "option", "Answer option key=label (2-8, repeatable, recommended first reads best)")
	id := fs.String("id", "", "Decision ID, unique per goal home, e.g. BAS-D-070")
	question := fs.String("question", "", "One-sentence question, answerable without opening files")
	recommend := fs.String("recommend", "", "Recommended option key")
	why := fs.String("why", "", "One sentence: why the recommendation")
	defaultAnswer := fs.String("default", "", "Option applied at the deadline (reversible decisions only)")
	reversible := fs.Bool("reversible", false, "The decision can be undone later; required for --default")
	deadline := fs.String("deadline", "", "When a default may apply: a duration (at least 12h) or an RFC 3339 time")
	urgency := fs.String("urgency", "high", "low, normal, high or critical")
	bigPicture := fs.String("big-picture", "", "Purpose of the decision and its practical consequences")
	technicalDetails := fs.String("technical-details", "", "Mechanism, evidence, risks, limits and remaining uncertainty")
	note := fs.String("note", "", "Legacy technical-details fallback; ignored when --technical-details is set")
	contextURL := fs.String("context-url", "", "Link for more context")
	homeRef := fs.String("home-ref", "", "Repository-relative goal home (default: the directory relative to the repository root)")
	effortRef := fs.String("effort-ref", "", "Effort reference (default: the enrollment registered for this goal home)")
	dryRun := fs.Bool("dry-run", false, "Print the record, event and QUEUE.md item; contact no server and write nothing")
	dir, rest := goalHomeArg(args)
	if err := cliutil.ParseInterspersed(fs, rest); err != nil {
		return err
	}
	if dir == "" || fs.NArg() != 0 {
		return errors.New(decisionUsage)
	}
	ref, err := decisionHomeRef(dir, *homeRef)
	if err != nil {
		return err
	}
	queuePath := filepath.Join(dir, goalhome.QueueFile)
	queue, err := os.ReadFile(queuePath)
	if err != nil {
		return fmt.Errorf("goal home %s: %w", dir, err)
	}
	now := time.Now().UTC()
	at, relative, err := parseDecisionDeadline(*deadline, now)
	if err != nil {
		return err
	}
	if *defaultAnswer != "" && at.IsZero() {
		at, relative = now.Add(24*time.Hour).Truncate(time.Second), true
	}
	view := goalhome.Decision{
		ID: strings.TrimSpace(*id), Question: strings.TrimSpace(*question), Options: options, Recommended: strings.TrimSpace(*recommend),
		RecommendationReason: strings.TrimSpace(*why), DefaultAnswer: strings.TrimSpace(*defaultAnswer), Reversible: *reversible,
		Urgency: strings.TrimSpace(*urgency), Note: strings.TrimSpace(*note), BigPicture: strings.TrimSpace(*bigPicture),
		TechnicalDetails: strings.TrimSpace(*technicalDetails), ContextURL: strings.TrimSpace(*contextURL), Deadline: at, RaisedAt: now,
	}
	if err := view.Validate(); err != nil {
		return fmt.Errorf("%w\n%s", err, decisionUsage)
	}
	record := &pb.EffortDecision{
		EffortRef: strings.TrimSpace(*effortRef), GoalHome: ref, DecisionId: view.ID, Question: view.Question, Recommended: view.Recommended,
		RecommendationReason: view.RecommendationReason, DefaultAnswer: view.DefaultAnswer, Reversible: view.Reversible, Urgency: view.Urgency,
		Note: view.Note, BigPicture: view.BigPicture, TechnicalDetails: view.TechnicalDetails, ContextUrl: view.ContextURL,
	}
	for _, option := range view.Options {
		record.Options = append(record.Options, &pb.EffortDecisionOption{Key: option.Key, Label: option.Label})
	}
	if !at.IsZero() {
		record.Deadline = timestamppb.New(at)
	}
	result := struct {
		GoalHome  string          `json:"goal_home"`
		DryRun    bool            `json:"dry_run,omitempty"`
		Decision  json.RawMessage `json:"decision"`
		Created   bool            `json:"created"`
		Published bool            `json:"published"`
		EventID   string          `json:"event_id,omitempty"`
		Event     map[string]any  `json:"event,omitempty"`
		Queue     string          `json:"queue"`
		Item      string          `json:"needs_operator_item"`
		Changed   bool            `json:"queue_changed"`
	}{GoalHome: ref, DryRun: *dryRun, Queue: queuePath}
	if *dryRun {
		record.State = effortdecision.Pending
		record.CreatedAt = timestamppb.New(now.Truncate(time.Millisecond))
		event := effortdecision.RequestedEvent(record)
		result.EventID = effortdecision.EventID(event)
		result.Event = map[string]any{"source": event.Source, "event_type": event.EventType, "occurred_at": event.Occurred.Format(time.RFC3339Nano), "payload": event.Payload}
		result.Item = goalhome.NeedsOperatorItem(view)
		_, result.Changed = goalhome.AddNeedsOperatorItem(queue, view)
		result.Decision, _ = protojson.Marshal(record)
		return printDecisionRaise(result, *jsonOut, func() {
			fmt.Printf("DRY RUN: nothing published, %s not written\n", queuePath)
			fmt.Printf("Event %s (%s, id %s):\n", event.EventType, event.Source, result.EventID)
			payload, _ := json.MarshalIndent(event.Payload, "  ", "  ")
			fmt.Printf("  %s\n", payload)
			fmt.Printf("QUEUE.md ## Needs operator item (would %s):\n%s", map[bool]string{true: "add", false: "keep the existing one"}[result.Changed], result.Item)
		})
	}
	if relative {
		// A re-raise with a relative deadline keeps the stored deadline, so
		// repeating the same command stays idempotent.
		if existing := a.storedDecision(ref, view.ID); existing != nil && existing.GetDeadline() != nil {
			record.Deadline = existing.GetDeadline()
		}
	}
	response := &pb.RaiseEffortDecisionResponse{}
	body, err := a.services.Watches.call(apiconnect.AgentManagerServiceRaiseEffortDecisionProcedure, &pb.RaiseEffortDecisionRequest{Decision: record}, response)
	if err != nil {
		return apiError(body, err)
	}
	stored := effortdecision.View(response.GetDecision())
	updated, changed := goalhome.AddNeedsOperatorItem(queue, stored)
	if changed {
		if err := goalhome.WriteIfUnchanged(queuePath, queue, updated); err != nil {
			return fmt.Errorf("decision %s is stored, but %s was not updated (rerun the same raise to retry): %w", view.ID, queuePath, err)
		}
	}
	result.Created, result.Published, result.EventID = response.GetCreated(), response.GetPublished(), response.GetDecision().GetEventId()
	result.Item, result.Changed = goalhome.NeedsOperatorItem(stored), changed
	result.Decision, _ = protojson.Marshal(response.GetDecision())
	return printDecisionRaise(result, *jsonOut, func() {
		state := "already stored; nothing new published"
		if response.GetCreated() {
			state = "stored"
		}
		fmt.Printf("Decision %s for %s %s.\n", view.ID, ref, state)
		switch {
		case response.GetPublished():
			fmt.Printf("Published decision_requested event %s; notification-hub asks the operator once.\n", response.GetDecision().GetEventId())
		case response.GetDecision().GetPublishedAt() != nil:
			fmt.Printf("Event %s was published at %s.\n", response.GetDecision().GetEventId(), effortTimestamp(response.GetDecision().GetPublishedAt()))
		default:
			fmt.Printf("Event not yet published (%s); Agent Manager retries on its scan cycle.\n", response.GetDecision().GetPublishError())
		}
		if changed {
			fmt.Printf("Added the item to ## Needs operator in %s.\n", queuePath)
		} else {
			fmt.Printf("## Needs operator in %s already lists it.\n", queuePath)
		}
	})
}

func printDecisionRaise(result any, jsonOut bool, text func()) error {
	if !jsonOut {
		text()
		return nil
	}
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	cliutil.PrintJSON(data)
	return nil
}

// storedDecision returns the stored record of a decision, or nil when it is
// absent or unreadable.
func (a *App) storedDecision(home, id string) *pb.EffortDecision {
	response := &pb.ListEffortDecisionsResponse{}
	if _, err := a.services.Watches.call(apiconnect.AgentManagerServiceListEffortDecisionsProcedure, &pb.ListEffortDecisionsRequest{GoalHome: home, Limit: 200}, response); err != nil {
		return nil
	}
	for _, d := range response.GetDecisions() {
		if d.GetDecisionId() == id {
			return d
		}
	}
	return nil
}

func (a *App) decisionAnswer(args []string) error {
	fs := flag.NewFlagSet("effort decision answer", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	note := fs.String("note", "", "Operator note, recorded verbatim")
	homeRef := fs.String("home-ref", "", "Repository-relative goal home")
	localOwner := fs.Bool("local-owner", false, "Explicit local operator exchange; unavailable in identified agent runs")
	dir, rest := goalHomeArg(args)
	if err := cliutil.ParseInterspersed(fs, rest); err != nil {
		return err
	}
	if dir == "" || fs.NArg() != 2 {
		return errors.New(decisionUsage)
	}
	if strings.TrimSpace(os.Getenv(cliutil.EnvIdentityToken)) != "" {
		return errors.New("an agent run cannot answer an operator decision; the operator answers on the phone or from an operator shell")
	}
	ref, err := decisionHomeRef(dir, *homeRef)
	if err != nil {
		return err
	}
	service := a.services.Watches
	if *localOwner {
		if err := cliutil.ConfirmLocalOwnerEligible(context.Background()); err != nil {
			return fmt.Errorf("local owner caller authority unavailable: %w", err)
		}
		login, err := authn.ExchangeLocalMachinePrincipal(context.Background())
		if err != nil {
			return fmt.Errorf("local owner exchange unavailable: %w", err)
		}
		service = &WatchService{api: service.api.WithToken(login.Tokens.AccessToken)}
	}
	response := &pb.EffortDecision{}
	body, err := service.call(apiconnect.AgentManagerServiceAnswerEffortDecisionProcedure, &pb.AnswerEffortDecisionRequest{GoalHome: ref, DecisionId: fs.Arg(0), Answer: fs.Arg(1), Note: *note}, response)
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Printf("Decision %s for %s: %s %s (%s). The orchestrator records it with `agent-manager effort decision sync`.\n", response.GetDecisionId(), ref, response.GetState(), response.GetAnswer(), response.GetAnsweredVia())
	return nil
}

func (a *App) decisionList(args []string) error {
	fs := flag.NewFlagSet("effort decision list", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	effortRef := fs.String("effort-ref", "", "Only decisions of this effort")
	unrecorded := fs.Bool("unrecorded", false, "Only decisions not yet recorded in FEEDBACK.md")
	homeRef := fs.String("home-ref", "", "Repository-relative goal home")
	dir, rest := goalHomeArg(args)
	if err := cliutil.ParseInterspersed(fs, rest); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New(decisionUsage)
	}
	request := &pb.ListEffortDecisionsRequest{EffortRef: strings.TrimSpace(*effortRef), UnrecordedOnly: *unrecorded}
	if dir != "" || *homeRef != "" {
		ref, err := decisionHomeRef(dir, *homeRef)
		if err != nil {
			return err
		}
		request.GoalHome = ref
	}
	response := &pb.ListEffortDecisionsResponse{}
	body, err := a.services.Watches.call(apiconnect.AgentManagerServiceListEffortDecisionsProcedure, request, response)
	if err != nil {
		return apiError(body, err)
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	if len(response.GetDecisions()) == 0 {
		fmt.Println("No decisions.")
	}
	for _, d := range response.GetDecisions() {
		fmt.Println(decisionLine(d))
	}
	return nil
}

// decisionLine is one board/list line: id, state, answer, deadline.
func decisionLine(d *pb.EffortDecision) string {
	line := fmt.Sprintf("%s  %s  %s", d.GetDecisionId(), d.GetGoalHome(), d.GetState())
	if d.GetAnswer() != "" {
		line += fmt.Sprintf(" %s (%s)", d.GetAnswer(), d.GetAnsweredVia())
	}
	if d.GetState() == effortdecision.Pending {
		line += " recommended=" + d.GetRecommended()
		if d.GetDefaultAnswer() != "" {
			line += fmt.Sprintf(" default=%s at %s", d.GetDefaultAnswer(), effortTimestamp(d.GetDeadline()))
		}
		if d.GetPublishedAt() == nil {
			line += " unpublished"
		}
	}
	if d.GetRecordedAt() != nil {
		line += " recorded " + effortTimestamp(d.GetRecordedAt())
	}
	return line + "  " + strings.Join(strings.Fields(d.GetQuestion()), " ")
}

// decisionSync writes each answered or defaulted, unrecorded decision of the
// goal home verbatim into FEEDBACK.md, removes its ## Needs operator item, and
// marks it recorded. It is idempotent: an entry already in FEEDBACK.md is not
// written again, and the mark is fenced on the answer time.
func (a *App) decisionSync(args []string) error {
	fs := flag.NewFlagSet("effort decision sync", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	homeRef := fs.String("home-ref", "", "Repository-relative goal home")
	dir, rest := goalHomeArg(args)
	if err := cliutil.ParseInterspersed(fs, rest); err != nil {
		return err
	}
	if dir == "" || fs.NArg() != 0 {
		return errors.New(decisionUsage)
	}
	ref, err := decisionHomeRef(dir, *homeRef)
	if err != nil {
		return err
	}
	response := &pb.ListEffortDecisionsResponse{}
	body, err := a.services.Watches.call(apiconnect.AgentManagerServiceListEffortDecisionsProcedure, &pb.ListEffortDecisionsRequest{GoalHome: ref, UnrecordedOnly: true, Limit: 200}, response)
	if err != nil {
		return apiError(body, err)
	}
	type synced struct {
		ID       string `json:"id"`
		State    string `json:"state"`
		Answer   string `json:"answer,omitempty"`
		Feedback bool   `json:"feedback_written"`
		Queue    bool   `json:"queue_item_removed"`
		Recorded bool   `json:"recorded"`
		Detail   string `json:"detail,omitempty"`
	}
	var results []synced
	feedbackPath, queuePath := filepath.Join(dir, goalhome.FeedbackFile), filepath.Join(dir, goalhome.QueueFile)
	// Oldest first, so FEEDBACK.md lists answers newest on top.
	decisions := response.GetDecisions()
	for i := len(decisions) - 1; i >= 0; i-- {
		d := decisions[i]
		item := synced{ID: d.GetDecisionId(), State: d.GetState(), Answer: d.GetAnswer()}
		if d.GetState() != effortdecision.Answered && d.GetState() != effortdecision.Defaulted {
			if d.GetState() == effortdecision.Expired {
				item.Detail = "expired with no answer and no default; the item stays under ## Needs operator"
			} else {
				item.Detail = "waiting for the operator"
			}
			results = append(results, item)
			continue
		}
		view := effortdecision.View(d)
		feedback, err := os.ReadFile(feedbackPath)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if updated, changed := goalhome.AddFeedbackEntry(feedback, view); changed {
			if err := writeGoalHomeFile(feedbackPath, feedback, updated); err != nil {
				return fmt.Errorf("decision %s: %s: %w", item.ID, feedbackPath, err)
			}
			item.Feedback = true
		}
		queue, err := os.ReadFile(queuePath)
		if err != nil {
			return err
		}
		if updated, changed := goalhome.RemoveNeedsOperatorItem(queue, item.ID); changed {
			if err := goalhome.WriteIfUnchanged(queuePath, queue, updated); err != nil {
				return fmt.Errorf("decision %s: %s: %w", item.ID, queuePath, err)
			}
			item.Queue = true
		}
		marked := &pb.EffortDecision{}
		body, err := a.services.Watches.call(apiconnect.AgentManagerServiceMarkEffortDecisionRecordedProcedure, &pb.MarkEffortDecisionRecordedRequest{GoalHome: ref, DecisionId: item.ID, AnsweredAt: d.GetAnsweredAt()}, marked)
		if err != nil {
			// A newer answer arrived after the read; the next sync records it.
			item.Detail = "not marked recorded: " + apiError(body, err).Error()
		} else {
			item.Recorded = true
		}
		results = append(results, item)
	}
	if *jsonOut {
		data, err := json.Marshal(struct {
			GoalHome  string   `json:"goal_home"`
			Decisions []synced `json:"decisions"`
		}{ref, results})
		if err != nil {
			return err
		}
		cliutil.PrintJSON(data)
		return nil
	}
	if len(results) == 0 {
		fmt.Printf("No unrecorded decisions for %s.\n", ref)
	}
	for _, r := range results {
		switch {
		case r.Feedback || r.Queue || r.Recorded:
			fmt.Printf("%s %s %s: FEEDBACK.md %s, ## Needs operator %s, recorded=%t %s\n", r.ID, r.State, r.Answer, map[bool]string{true: "written", false: "already had it"}[r.Feedback], map[bool]string{true: "item removed", false: "had no item"}[r.Queue], r.Recorded, r.Detail)
		default:
			fmt.Printf("%s %s: %s\n", r.ID, r.State, r.Detail)
		}
	}
	return nil
}

// writeGoalHomeFile replaces an existing file atomically, refusing a
// concurrent change, or creates a missing one.
func writeGoalHomeFile(path string, original, updated []byte) error {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) && len(original) == 0 {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return err
		}
		if _, err = file.Write(updated); err != nil {
			file.Close()
			return err
		}
		return file.Close()
	}
	return goalhome.WriteIfUnchanged(path, original, updated)
}
