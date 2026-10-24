package supervision

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/goalhome"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const deliveryCheckpointGrace = 2 * time.Minute

var deliveryIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,99}$`)

// DeliveryPressure uses the persisted owner clock. Charter and scoreboard text
// never advance, pause or accept a commission.
func DeliveryPressure(d *pb.EffortDelivery, now time.Time) *pb.EffortDeliveryPressure {
	if d == nil {
		return nil
	}
	p := &pb.EffortDeliveryPressure{CommissionId: d.CommissionId, BudgetSeconds: d.BudgetSeconds, PausedSeconds: d.PausedSeconds}
	if d.StartedAt == nil || d.BudgetSeconds <= 0 {
		p.Phase = "UNCOMMISSIONED"
		return p
	}
	end := now
	if d.PausedAt != nil {
		end = d.PausedAt.AsTime()
	}
	if d.StoppedAt != nil && d.StoppedAt.AsTime().Before(end) {
		end = d.StoppedAt.AsTime()
	}
	p.ElapsedSeconds = max(0, int64(end.Sub(d.StartedAt.AsTime())/time.Second)-d.PausedSeconds)
	hard := d.StartedAt.AsTime().Add(time.Duration(d.PausedSeconds+2*d.BudgetSeconds) * time.Second)
	if d.PausedAt != nil {
		hard = hard.Add(now.Sub(d.PausedAt.AsTime()))
	}
	p.HardStopAt = timestamppb.New(hard)
	p.CheckpointDeadline = timestamppb.New(hard.Add(deliveryCheckpointGrace))
	p.Phase, p.Instruction = "ON TRACK", "Deliver the full product acceptance intent. Keep independent surface work moving."
	switch {
	case p.ElapsedSeconds >= 2*d.BudgetSeconds || d.State == "stopped":
		p.Phase, p.Instruction = "HARD STOP", "Stop new product work. Checkpoint accepted, partial, remaining and unverified outcomes. Hand back to the operator; time never grants acceptance."
		if d.StoppedAt != nil {
			p.CheckpointDeadline = timestamppb.New(d.StoppedAt.AsTime().Add(deliveryCheckpointGrace))
		}
		p.Disposition = "stopped; product acceptance remains independently evidenced"
	case p.ElapsedSeconds*2 >= 3*d.BudgetSeconds:
		p.Phase, p.Instruction = "CONVERGE", "Secure a coherent partial result. Admit no new scope. Preserve evidence, recovery and truthful remaining work."
	case p.ElapsedSeconds >= d.BudgetSeconds:
		p.Phase, p.Instruction = "OVER", "Prioritize acceptance intent. Log deferred optional work explicitly; do not relabel unmet requirements as accepted."
	case p.ElapsedSeconds*4 >= 3*d.BudgetSeconds:
		p.Phase, p.Instruction = "APPROACHING", "Stop optional polish and triage remaining work by product value. Preserve all required behavior."
	}
	if d.State == "paused" {
		p.Phase, p.Instruction = "PAUSED", "The operator paused this commission: "+d.Reason+". Preserve retained ownership; resume through the owner clock."
	}
	return p
}

func deliveryContext(e *pb.EffortEnrollment, now time.Time) string {
	p := DeliveryPressure(e.Delivery, now)
	if p == nil {
		return ""
	}
	return fmt.Sprintf("\n\nDelivery commission %s (%s): %s; elapsed %ds / budget %ds, paused %ds. %s Read the current effort board and product charter; preserve the honest surface scoreboard.", p.CommissionId, e.EffortRef, p.Phase, p.ElapsedSeconds, p.BudgetSeconds, p.PausedSeconds, p.Instruction)
}

// transitionDelivery is reached only through operator enrollment. Reconciliation
// treats Delivery as immutable. All clock timestamps are assigned here.
func (s *EffortService) transitionDelivery(ctx context.Context, old, next *pb.EffortEnrollment) error {
	d := next.GetDelivery()
	prior := old.GetDelivery()
	if d == nil {
		if prior != nil {
			return errors.New("a delivery commission cannot be erased; stop it explicitly")
		}
		return nil
	}
	if !deliveryIDPattern.MatchString(d.CommissionId) || d.BudgetSeconds < 1 || d.BudgetSeconds > 30*86400 || d.MaxWorkers < 1 || d.MaxWorkers > 32 {
		return errors.New("delivery needs a commission id, budget between 1 second and 30 days, and 1–32 workers")
	}
	if d.State != "running" && d.State != "paused" && d.State != "stopped" {
		return errors.New("delivery state must be running, paused or stopped")
	}
	now := s.now().UTC()
	if prior == nil || prior.CommissionId != d.CommissionId {
		if prior != nil && (prior.State != "stopped" || prior.FinalizedAt == nil) {
			return errors.New("finish the prior commission before recommissioning")
		}
		if d.State != "running" {
			return errors.New("a new delivery commission starts in running state")
		}
		if old != nil {
			runs, err := s.deliveryRuns(ctx, old)
			if err != nil {
				return err
			}
			for _, run := range runs {
				if !run.Status.IsTerminal() {
					return fmt.Errorf("retained owner %s is %s; reconcile it before commissioning", run.ID, run.Status)
				}
			}
		}
		next.Delivery = &pb.EffortDelivery{CommissionId: d.CommissionId, BudgetSeconds: d.BudgetSeconds, MaxWorkers: d.MaxWorkers, State: "running", StartedAt: timestamppb.New(now)}
		return nil
	}
	if d.BudgetSeconds != prior.BudgetSeconds || d.MaxWorkers != prior.MaxWorkers {
		return errors.New("commission policy is fixed; use a new commission after honest closure")
	}
	state, reason := d.State, strings.TrimSpace(d.Reason)
	next.Delivery = proto.Clone(prior).(*pb.EffortDelivery)
	d = next.Delivery
	if prior.State == "stopped" {
		if state != "stopped" {
			return errors.New("a stopped commission cannot resume; recommission explicitly")
		}
		return nil
	}
	if state == prior.State {
		return nil
	}
	switch state {
	case "paused":
		if reason == "" {
			return errors.New("pause requires a reason, including any infrastructure hold")
		}
		runs, err := s.deliveryRuns(ctx, old)
		if err != nil {
			return err
		}
		for _, run := range runs {
			if !run.Status.IsTerminal() && run.Status != domain.RunStatusParked {
				return fmt.Errorf("checkpoint and park run %s before pausing its clock", run.ID)
			}
		}
		d.PausedAt = timestamppb.New(now)
	case "running":
		if prior.PausedAt == nil {
			return errors.New("paused commission has no owner pause timestamp")
		}
		d.PausedSeconds += max(0, int64(now.Sub(prior.PausedAt.AsTime())/time.Second))
		d.PausedAt = nil
	case "stopped":
		if reason == "" {
			return errors.New("stopping requires a disposition reason")
		}
		if d.PausedAt != nil {
			d.PausedSeconds += max(0, int64(now.Sub(d.PausedAt.AsTime())/time.Second))
			d.PausedAt = nil
		}
		d.StoppedAt = timestamppb.New(now)
	}
	d.State, d.Reason = state, reason
	return nil
}

// deliveryRuns joins the native run owner and the exact retained declarations.
// Unknown owners refuse lifecycle changes; absent scans never imply closure.
func (s *EffortService) deliveryRuns(ctx context.Context, e *pb.EffortEnrollment) ([]*domain.Run, error) {
	byID := map[uuid.UUID]*domain.Run{}
	if s.runRegistry != nil {
		for offset := 0; ; offset += 100 {
			if offset >= 10000 {
				return nil, errors.New("delivery run inventory exceeded its bound; reconcile before changing lifecycle")
			}
			page, err := s.runRegistry.List(ctx, repository.RunListFilter{ListFilter: repository.ListFilter{Limit: 100, Offset: offset}, NonTerminalOnly: true})
			if err != nil {
				return nil, err
			}
			for _, run := range page {
				for _, ref := range run.WorkReferences {
					if domain.IsPublicActiveEffortReference(ref) && ref.Id == e.EffortRef && (ref.Relationship == "worker" || ref.Relationship == "orchestrator") {
						byID[run.ID] = run
					}
				}
			}
			if len(page) < 100 {
				break
			}
		}
	}
	for _, subject := range e.Subjects {
		if subject.GetRunId() == "" || (subject.Role != "orchestrator" && subject.Role != "worker") {
			continue
		}
		id, err := uuid.Parse(subject.RunId)
		if err != nil {
			return nil, err
		}
		if byID[id] != nil {
			continue
		}
		if s.controller == nil {
			return nil, errors.New("delivery run owner unavailable")
		}
		run, err := s.controller.GetRun(ctx, id)
		if err != nil {
			return nil, err
		}
		if run == nil || run.ID != id {
			return nil, fmt.Errorf("retained delivery run %s unavailable", id)
		}
		byID[id] = run
	}
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id.String())
	}
	sort.Strings(ids)
	runs := make([]*domain.Run, 0, len(ids))
	for _, id := range ids {
		runs = append(runs, byID[uuid.MustParse(id)])
	}
	return runs, nil
}

// AcquireDeliveryAdmission serializes commissioned admission until the run row
// is persisted. Global resource and authority admission remain independent.
func (s *EffortService) AcquireDeliveryAdmission(ctx context.Context, refs []*eventpb.WorkReference, parent *domain.Run) (map[string]string, string, func(), error) {
	s.deliveryMu.Lock()
	release := func() { s.deliveryMu.Unlock() }
	bindings := map[string]string{}
	text := ""
	for _, ref := range refs {
		if !domain.IsPublicActiveEffortReference(ref) || (ref.Relationship != "orchestrator" && ref.Relationship != "worker") {
			continue
		}
		e, _, err := s.repo.GetEffort(ctx, ref.Id)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			release()
			return nil, "", nil, err
		}
		if e.Delivery == nil {
			continue
		}
		p := DeliveryPressure(e.Delivery, s.now())
		if e.Withdrawn || e.Delivery.State != "running" || p.Phase == "HARD STOP" {
			release()
			return nil, "", nil, fmt.Errorf("delivery %s does not admit work: %s", ref.Id, p.Phase)
		}
		if ref.Revision != e.TargetRevision {
			release()
			return nil, "", nil, errors.New("delivery target revision changed; reconcile the commission")
		}
		if ref.Relationship == "worker" && (parent == nil || parent.ResolvedConfig == nil || parent.ResolvedConfig.DeliveryCommissions[ref.Id] != e.Delivery.CommissionId) {
			release()
			return nil, "", nil, errors.New("worker parent does not own this delivery commission")
		}
		runs, err := s.deliveryRuns(ctx, e)
		if err != nil {
			release()
			return nil, "", nil, err
		}
		count := 0
		for _, run := range runs {
			if run.Status.IsTerminal() {
				continue
			}
			for _, r := range run.WorkReferences {
				if domain.IsPublicActiveEffortReference(r) && r.Id == ref.Id && r.Relationship == ref.Relationship {
					count++
					break
				}
			}
		}
		limit := int(e.Delivery.MaxWorkers)
		if ref.Relationship == "orchestrator" {
			limit = 1
		}
		if count >= limit {
			release()
			return nil, "", nil, fmt.Errorf("delivery %s %s limit reached (%d/%d)", ref.Id, ref.Relationship, count, limit)
		}
		bindings[ref.Id] = e.Delivery.CommissionId
		text += deliveryContext(e, s.now())
	}
	return bindings, text, release, nil
}

// DeliveryRunContext fences continuations against the immutable run binding.
// At the hard deadline only the two-minute checkpoint window remains.
func (s *EffortService) DeliveryRunContext(ctx context.Context, run *domain.Run) (string, error) {
	return s.deliveryRunContext(ctx, run, false)
}

// DeliveryInitialContext refuses queued product execution at the hard deadline;
// checkpoint grace belongs only to owners that have already started.
func (s *EffortService) DeliveryInitialContext(ctx context.Context, run *domain.Run) (string, error) {
	return s.deliveryRunContext(ctx, run, true)
}

func (s *EffortService) deliveryRunContext(ctx context.Context, run *domain.Run, initial bool) (string, error) {
	if run == nil {
		return "", nil
	}
	text := ""
	seen := map[string]bool{}
	for _, ref := range run.WorkReferences {
		if !domain.IsPublicActiveEffortReference(ref) || (ref.Relationship != "orchestrator" && ref.Relationship != "worker") || seen[ref.Id] {
			continue
		}
		seen[ref.Id] = true
		e, _, err := s.repo.GetEffort(ctx, ref.Id)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return "", err
		}
		if e.Delivery == nil {
			continue
		}
		if e.Withdrawn || run.ResolvedConfig == nil || e.Delivery.CommissionId != run.ResolvedConfig.DeliveryCommissions[ref.Id] {
			return "", errors.New("retained run no longer owns this delivery commission")
		}
		p := DeliveryPressure(e.Delivery, s.now())
		if e.Delivery.State == "paused" || (p.Phase == "HARD STOP" && (initial || !s.now().Before(p.CheckpointDeadline.AsTime()))) {
			return "", fmt.Errorf("delivery continuation refused: %s", p.Phase)
		}
		text += deliveryContext(e, s.now())
	}
	return text, nil
}

func (s *EffortService) tickDelivery(ctx context.Context) error {
	page, err := s.repo.ListEfforts(ctx, s.deliveryScanAfter, 100)
	if err != nil {
		return err
	}
	if len(page) < 100 {
		s.deliveryScanAfter = ""
	} else {
		s.deliveryScanAfter = page[len(page)-1].EffortRef
	}
	var errs []error
	for _, e := range page {
		if e.Delivery == nil || e.Delivery.FinalizedAt != nil {
			continue
		}
		if err := s.advanceDelivery(ctx, e); err != nil {
			errs = append(errs, fmt.Errorf("delivery %s: %w", e.EffortRef, err))
		}
	}
	return errors.Join(errs...)
}

func (s *EffortService) advanceDelivery(ctx context.Context, e *pb.EffortEnrollment) error {
	p := DeliveryPressure(e.Delivery, s.now())
	if e.Delivery.State == "paused" && !e.Withdrawn {
		return nil
	}
	if (p.Phase == "HARD STOP" || e.Withdrawn) && e.Delivery.StoppedAt == nil {
		next := proto.Clone(e).(*pb.EffortEnrollment)
		next.Delivery.State = "stopped"
		next.Delivery.Reason = "delivery budget reached twice its commissioned duration"
		// Record when checkpointing actually became possible, including after an
		// owner restart. HardStopAt still reports the original admission deadline.
		next.Delivery.StoppedAt = timestamppb.New(s.now().UTC())
		if e.Withdrawn {
			next.Delivery.Reason = "withdrawn enrollment: " + e.WithdrawalReason
		}
		if err := s.saveDelivery(ctx, e, next, "hard-stop"); err != nil {
			return err
		}
		e = next
		p = DeliveryPressure(e.Delivery, s.now())
	}
	runs, err := s.deliveryRuns(ctx, e)
	if err != nil {
		return err
	}
	_, _, err = s.repo.AppendEffortSignal(ctx, e.EffortRef, "delivery_pressure", "", p.Instruction, "delivery:"+e.Delivery.CommissionId+":"+p.Phase)
	if err != nil {
		return err
	}
	for _, run := range runs {
		if run.Status.IsTerminal() || run.ResolvedConfig == nil || run.ResolvedConfig.DeliveryCommissions[e.EffortRef] != e.Delivery.CommissionId {
			continue
		}
		if p.Phase == "HARD STOP" && !s.now().Before(p.CheckpointDeadline.AsTime()) {
			if s.controller == nil {
				return errors.New("delivery stop controller unavailable")
			}
			if err := s.controller.StopRun(ctx, run.ID); err != nil {
				return err
			}
			continue
		}
		if p.Phase == "ON TRACK" {
			continue
		}
		// Parked orchestrators use the existing neutral signal-checkin dispatcher.
		// Waiting workers retain their producer; their orchestrator reconciles it.
		if run.Status != domain.RunStatusRunning {
			continue
		}
		// The lifecycle continuation receipt deduplicates a threshold notice per owner.
		if s.controller != nil {
			notice := fmt.Sprintf("Delivery commission %s: %s. %s", e.Delivery.CommissionId, p.Phase, p.Instruction)
			if err := s.controller.ContinueRun(ctx, run.ID, notice, "delivery:"+e.Delivery.CommissionId+":"+p.Phase+":"+run.ID.String()); err != nil {
				return err
			}
		}
	}
	if p.Phase != "HARD STOP" || s.now().Before(p.CheckpointDeadline.AsTime()) {
		return nil
	}
	// Verify terminal readback before reporting finalization, including async stops.
	for _, run := range runs {
		if run.ResolvedConfig == nil || run.ResolvedConfig.DeliveryCommissions[e.EffortRef] != e.Delivery.CommissionId {
			continue
		}
		current, err := s.controller.GetRun(ctx, run.ID)
		if err != nil {
			return err
		}
		if current == nil || !current.Status.IsTerminal() {
			return errors.New("waiting for delivery owner terminal readback")
		}
	}
	home, ok := goalHomeDestination(e.DestinationRef)
	if !ok {
		return errors.New("delivery handback requires a registered product home")
	}
	d := &pb.EffortDecision{EffortRef: e.EffortRef, GoalHome: home, DecisionId: "delivery-" + e.Delivery.CommissionId, Question: "Delivery stopped. Accept the evidenced partial result, recommission, or change scope?", Options: []*pb.EffortDecisionOption{{Key: "accept-partial", Label: "Accept the evidenced partial result"}, {Key: "recommission", Label: "Recommission with a new budget"}, {Key: "rescope", Label: "Change scope before another commission"}}, Recommended: "rescope", RecommendationReason: "Review the remaining outcomes before choosing another commission.", Note: "Time did not establish acceptance. Read CHARTER.md, STATUS.md and LOG.md; unverified outcomes remain unverified."}
	outcome := s.projectGoalHome(ctx, e, &pb.EffortBoardRow{})
	if err := s.writeDeliveryOutcome(home, e.Delivery.CommissionId, d.DecisionId, outcome); err != nil {
		return err
	}
	if _, err := s.RaiseDecision(ctx, &pb.RaiseEffortDecisionRequest{Decision: d}, ""); err != nil {
		return err
	}
	next := proto.Clone(e).(*pb.EffortEnrollment)
	next.Delivery.FinalizedAt = timestamppb.New(s.now().UTC())
	next.Delivery.Outcome = outcome
	next.Delivery.DecisionId = d.DecisionId
	return s.saveDelivery(ctx, e, next, "finalized")
}

// Append a terminal observation without rewriting the product scoreboard or
// its active obligations. The native frozen snapshot remains the durable result.
func (s *EffortService) writeDeliveryOutcome(home, commission, decision string, outcome *pb.EffortGoalHome) error {
	root, err := openSafeRoot(filepath.Join(s.config.ProjectRoot, filepath.FromSlash(home)))
	if err != nil {
		return err
	}
	defer root.Close()
	before, err := safeRead(root, goalhome.StatusFile, s.config.FileBytes)
	if err != nil {
		return err
	}
	marker := "<!-- delivery-outcome:" + commission + " -->"
	if bytes.Contains(before, []byte(marker)) {
		return nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "\n\n## Delivery outcome\n\n%s\nCommission `%s` stopped. Time did not establish product acceptance.\nOperator decision: `%s`; publication and answer remain native owner facts.\n", marker, commission, decision)
	if outcome == nil || len(outcome.SurfaceScoreboard) == 0 {
		b.WriteString("Product surface assessment is unavailable; remaining outcomes are unverified.\n")
	} else {
		for _, row := range outcome.SurfaceScoreboard {
			fmt.Fprintf(&b, "- %s: %s (reported product state; retain independent evidence).\n", row.Surface, row.State)
		}
	}
	b.WriteString("Retain the scoreboard, LOG evidence, open feedback and unfinished work. Accept partial, recommission or rescope explicitly; the heartbeat cannot restart this commission.\n")
	after := append(append([]byte(nil), before...), []byte(b.String())...)
	if int64(len(after)) > s.config.FileBytes {
		return errors.New("terminal STATUS exceeds the bounded file limit; compact resolved history and retry")
	}
	name := ".delivery-outcome-" + uuid.NewString()
	f, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer root.Remove(name)
	if _, err := f.Write(after); err != nil {
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
	current, err := safeRead(root, goalhome.StatusFile, s.config.FileBytes)
	if err != nil {
		return err
	}
	if !bytes.Equal(current, before) {
		return goalhome.ErrConcurrentChange
	}
	return root.Rename(name, goalhome.StatusFile)
}

func (s *EffortService) saveDelivery(ctx context.Context, old, next *pb.EffortEnrollment, transition string) error {
	_, ob, err := s.repo.GetEffort(ctx, old.EffortRef)
	if err != nil {
		return err
	}
	next.Revision = old.Revision + 1
	next.UpdatedAt = timestamppb.New(s.now().UTC())
	key := "delivery:" + old.EffortRef + ":" + next.Delivery.CommissionId + ":" + transition
	return s.repo.SaveEffort(ctx, next, ob, old.Revision, key, effortDigest(next))
}
