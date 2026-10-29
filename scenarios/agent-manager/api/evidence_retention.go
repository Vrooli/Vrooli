package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-manager/internal/evidenceretention"
	"agent-manager/internal/orchestration/obs"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/operatordecision"
	corestorage "github.com/vrooli/api-core/storage"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// evidenceRetentionInterval spaces automatic prune cycles. Each cycle walks
// the evidence home and the work-record roots (about a minute on this host).
const evidenceRetentionInterval = 6 * time.Hour

// buildEvidenceRetention wires the owner of ~/.vrooli/evidence. Agent Manager
// owns it because effort enrollments are the only lifecycle signal for the
// delivery and research work that writes there. A non-live variant gets no
// owner: it must never classify or prune the live evidence home.
//
// The approvals let an agent ask for a prune: they raise an operator decision
// through publisher and apply the approved prune when the signed resolution
// arrives. A nil publisher keeps asks durable but un-notified.
func (s *Server) buildEvidenceRetention(repoRoot string, publisher operatordecision.Publisher) (*evidenceretention.Service, *evidenceretention.Approvals, error) {
	ns, err := corestorage.ResolveNamespace(corestorage.NamespaceConfig{FallbackScenario: "agent-manager"})
	if err != nil {
		return nil, nil, err
	}
	if !ns.IsLive() {
		return nil, nil, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, err
	}
	root := strings.TrimSpace(os.Getenv("AGENT_MANAGER_EVIDENCE_ROOT"))
	if root == "" {
		root = filepath.Join(home, ".vrooli", "evidence")
	}
	resolver, err := corestorage.NewResolver(corestorage.ResolverConfig{})
	if err != nil {
		return nil, nil, err
	}
	receipts, err := resolver.Path(corestorage.Options{ScenarioID: "agent-manager"}, corestorage.ClassState, "evidence-retention")
	if err != nil {
		return nil, nil, err
	}
	service, err := evidenceretention.New(evidenceretention.Options{
		Root:           root,
		ReferenceRoots: []string{repoRoot, filepath.Join(home, ".vrooli", "plan-artifacts"), filepath.Join(home, ".vrooli", "research")},
		ReceiptDir:     receipts,
		Efforts:        s.evidenceEfforts,
	})
	if err != nil {
		return nil, nil, err
	}
	asks := evidenceretention.NewFileAskStore(filepath.Join(receipts, "approvals"))
	return service, evidenceretention.NewApprovals(service, asks, evidenceretention.NewApprovalRaiser(asks, publisher)), nil
}

// decisionEffects lists the notification-hub resolution owners that live
// outside the orchestrator.
func (s *Server) decisionEffects() []operatordecision.DomainEffect {
	if s.evidenceApprovals == nil {
		return nil
	}
	return []operatordecision.DomainEffect{s.evidenceApprovals}
}

// evidenceEfforts adapts effort enrollments to the retention lifecycle view.
func (s *Server) evidenceEfforts(ctx context.Context) ([]evidenceretention.Effort, error) {
	if s.supervisionService == nil || s.supervisionService.Efforts == nil {
		return nil, fmt.Errorf("effort service unavailable")
	}
	var out []evidenceretention.Effort
	token := ""
	for page := 0; page < 50; page++ {
		resp, err := s.supervisionService.Efforts.List(ctx, &pb.ListEffortsRequest{PageSize: 100, PageToken: token})
		if err != nil {
			return nil, err
		}
		for _, e := range resp.GetEfforts() {
			out = append(out, effortLifecycle(e))
		}
		if token = resp.GetNextPageToken(); token == "" {
			return out, nil
		}
	}
	return out, fmt.Errorf("effort enrollment listing exceeded 5000 rows")
}

// effortLifecycle maps one enrollment. Terminal means withdrawn, or a
// delivery commission that is stopped and finalized.
func effortLifecycle(e *pb.EffortEnrollment) evidenceretention.Effort {
	out := evidenceretention.Effort{
		Ref:  e.GetEffortRef(),
		Keys: []string{e.GetDisplayName(), e.GetWorkspace(), destinationHome(e.GetDestinationRef())},
	}
	d := e.GetDelivery()
	if d != nil {
		out.Keys = append(out.Keys, d.GetCommissionId(), d.GetOutcome().GetPath())
	}
	switch {
	case e.GetWithdrawn():
		out.Terminal = true
		if e.GetUpdatedAt() != nil {
			out.TerminalAt = e.GetUpdatedAt().AsTime()
		}
	case d != nil && d.GetState() == "stopped" && d.GetFinalizedAt() != nil:
		out.Terminal = true
		out.TerminalAt = d.GetFinalizedAt().AsTime()
	}
	return out
}

// registerEvidenceRetentionRoutes exposes the dry-run report and the
// owner-authorized prune beside Agent Manager's other storage contracts.
func (s *Server) registerEvidenceRetentionRoutes(router *mux.Router, service *evidenceretention.Service, approvals *evidenceretention.Approvals) {
	if service == nil {
		return
	}
	var authorize evidenceretention.Authorizer
	if s.maintenance != nil {
		authorize = s.maintenance.AuthorizeOwner
	}
	handler := evidenceretention.NewHandler(service, authorize, approvals)
	router.HandleFunc("/api/v1/storage/evidence", handler.Report).Methods("GET")
	router.HandleFunc("/api/v1/storage/evidence/prune", handler.Prune).Methods("POST")
	router.HandleFunc("/api/v1/storage/evidence/prune/requests/{id}", handler.RequestStatus).Methods("GET")
}

// startEvidenceRetention runs the automatic cycle: only efforts whose every
// matching enrollment is terminal past the grace window lose their
// unreferenced regenerable intermediates. Each cycle refreshes latest.json.
func startEvidenceRetention(ctx context.Context, service *evidenceretention.Service) {
	if service == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(evidenceRetentionInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			cycle, cancel := context.WithTimeout(ctx, 30*time.Minute)
			receipt, err := service.Prune(cycle, evidenceretention.PruneRequest{Reason: "automatic: terminal effort past grace"})
			cancel()
			obs.Component("evidence-retention").Info("evidence retention cycle", "removed", len(receipt.Removed), "removed_bytes", receipt.RemovedBytes, "receipt", receipt.ReceiptPath, "error", err)
		}
	}()
}

// destinationHome turns a destination reference such as
// "path:.vrooli/work/delivery/desktop-packaging-delivery/CHARTER.md#goal"
// into the home directory it names, so a delivery home's evidence matches.
func destinationHome(ref string) string {
	ref = strings.TrimSpace(ref)
	if i := strings.Index(ref, ":"); i > 0 && !strings.Contains(ref[:i], "/") {
		ref = ref[i+1:]
	}
	if i := strings.Index(ref, "#"); i >= 0 {
		ref = ref[:i]
	}
	ref = strings.TrimRight(ref, "/")
	if filepath.Ext(ref) != "" {
		ref = filepath.Dir(ref)
	}
	if ref == "." || ref == "/" {
		return ""
	}
	return ref
}
