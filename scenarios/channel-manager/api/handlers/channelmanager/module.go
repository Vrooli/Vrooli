// Package channelmanager exposes the manual-operations API.  It intentionally
// never accepts a platform credential: identity creation takes a Vault path
// reference and every platform interaction remains an operator action.
package channelmanager

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	assetstudio "channel-manager/integrations/assetstudio"
	bas "channel-manager/integrations/bas"
	contentdesk "channel-manager/integrations/contentdesk"
	core "channel-manager/internal/channelmanager"
	"channel-manager/internal/module"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	channelmanagerv1 "github.com/vrooli/vrooli/packages/proto/gen/go/channel-manager/v1/channelmanager"
	channelmanagerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/channel-manager/v1/channelmanager/channelmanager_v1connect"
)

type api struct {
	service   *core.Service
	store     core.Store
	deliverer contentdesk.Deliverer
	browser   core.BrowserDispatch
	assets    assetstudio.Resolver
	mu        sync.Mutex
}

func Module(service *core.Service, store core.Store) module.Module {
	return moduleWithAssetResolver(service, store, contentdesk.NewClient(), bas.NewClient(), assetstudio.NewClient())
}

func moduleWithDeliverer(service *core.Service, store core.Store, deliverer contentdesk.Deliverer) module.Module {
	return moduleWithDependencies(service, store, deliverer, nil)
}

func moduleWithDependencies(service *core.Service, store core.Store, deliverer contentdesk.Deliverer, browser core.BrowserDispatch) module.Module {
	return moduleWithAssetResolver(service, store, deliverer, browser, nil)
}

func moduleWithAssetResolver(service *core.Service, store core.Store, deliverer contentdesk.Deliverer, browser core.BrowserDispatch, assets assetstudio.Resolver) module.Module {
	h := &api{service: service, store: store, deliverer: deliverer, browser: browser, assets: assets}
	return module.Module{Name: "channel-manager", Endpoints: Endpoints, Mount: func(r *mux.Router) {
		connectPath, connectHandler := channelmanagerconnect.NewChannelManagerServiceHandler(h)
		connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		r.HandleFunc("/api/v1/channel-manager/overview", h.overview).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/channel-manager/identities", h.createIdentity).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/channel-manager/identities/{id}", h.updateIdentity).Methods(http.MethodPut)
		r.HandleFunc("/api/v1/channel-manager/identities/{id}/retire", h.retireIdentity).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/channel-manager/identities/{id}/timeline", h.timeline).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/channel-manager/identities/{id}/start", h.start).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/channel-manager/actions", h.enqueue).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/channel-manager/actions/{id}/complete", h.complete).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/channel-manager/actions/{id}/complete-release", h.completeRelease).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/channel-manager/actions/{id}/dispatch-browser", h.dispatchBrowser).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/channel-manager/identities/{id}/automation", h.assignAutomation).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/channel-manager/portfolio", h.configurePortfolio).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/channel-manager/identities/{id}/observations", h.observe).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/channel-manager/identities/{id}/eligibility", h.eligibility).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/channel-manager/releases", h.release).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/channel-manager/releases/preview", h.previewRelease).Methods(http.MethodPost)
	}}
}

func (h *api) GetBrowserExecutionReview(ctx context.Context, request *connect.Request[channelmanagerv1.GetBrowserExecutionReviewRequest]) (*connect.Response[channelmanagerv1.GetBrowserExecutionReviewResponse], error) {
	inspector, ok := h.browser.(core.BrowserInspector)
	if !ok || inspector == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("browser execution review is not configured"))
	}
	h.mu.Lock()
	action := h.service.Actions[request.Msg.ActionId]
	if action == nil || action.ExecutionID == "" {
		h.mu.Unlock()
		return nil, connect.NewError(connect.CodeNotFound, errors.New("browser execution not found for action"))
	}
	executionID := action.ExecutionID
	h.mu.Unlock()
	review, err := inspector.Inspect(ctx, executionID)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewResponse(&channelmanagerv1.GetBrowserExecutionReviewResponse{ExecutionId: review.ExecutionID, Status: review.Status, Failure: review.Failure, ArtifactRefs: review.ArtifactRefs}), nil
}

func (h *api) dispatchBrowser(w http.ResponseWriter, r *http.Request) {
	if h.browser == nil {
		writeError(w, errors.New("browser automation is not configured"))
		return
	}
	h.mu.Lock()
	executionID, err := h.service.DispatchBrowser(r.Context(), mux.Vars(r)["id"], h.browser)
	if err == nil {
		err = h.store.Save(r.Context(), h.service)
	}
	h.mu.Unlock()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"execution_id": executionID})
}

func (h *api) assignAutomation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConsumerProfileKey string   `json:"consumer_profile_key"`
		SessionProfileRef  string   `json:"session_profile_ref"`
		WorkflowRef        string   `json:"workflow_ref"`
		EnabledActionKinds []string `json:"enabled_action_kinds"`
		OperatorNote       string   `json:"operator_note"`
	}
	if !decode(w, r, &req) {
		return
	}
	h.mu.Lock()
	err := h.service.AssignAutomation(mux.Vars(r)["id"], req.ConsumerProfileKey, req.SessionProfileRef, req.WorkflowRef, req.EnabledActionKinds, req.OperatorNote)
	if err == nil {
		err = h.store.Save(r.Context(), h.service)
	}
	h.mu.Unlock()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"identity_id": mux.Vars(r)["id"]})
}

func (h *api) configurePortfolio(w http.ResponseWriter, r *http.Request) {
	var policy core.PortfolioPolicy
	if !decode(w, r, &policy) {
		return
	}
	h.mu.Lock()
	err := h.service.ConfigurePortfolioPolicy(policy)
	if err == nil {
		err = h.store.Save(r.Context(), h.service)
	}
	h.mu.Unlock()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, policy)
}

func (h *api) previewRelease(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlatformID                string `json:"platform_id"`
		Caption                   string `json:"caption"`
		Title                     string `json:"title"`
		PostType                  string `json:"post_type"`
		FormatKind                string `json:"format_kind"`
		MediaWidth                int    `json:"media_width"`
		MediaHeight               int    `json:"media_height"`
		DisclosureVisible         bool   `json:"disclosure_visible"`
		DisclosureInVisibleRegion bool   `json:"disclosure_in_visible_region"`
		FirstComment              string `json:"first_comment"`
	}
	if !decode(w, r, &req) {
		return
	}
	h.mu.Lock()
	preview, err := h.service.PreviewRelease(core.PreviewInput{PlatformID: req.PlatformID, Caption: req.Caption, Title: req.Title, PostType: req.PostType, FormatKind: req.FormatKind, MediaWidth: req.MediaWidth, MediaHeight: req.MediaHeight, DisclosureVisible: req.DisclosureVisible, DisclosureInVisibleRegion: req.DisclosureInVisibleRegion, FirstComment: req.FirstComment})
	h.mu.Unlock()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (h *api) GetOverview(_ context.Context, _ *connect.Request[channelmanagerv1.GetOverviewRequest]) (*connect.Response[channelmanagerv1.GetOverviewResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	response := &channelmanagerv1.GetOverviewResponse{}
	for _, identity := range h.service.Identities {
		response.Identities = append(response.Identities, &channelmanagerv1.Identity{Id: identity.ID, PlatformId: identity.PlatformID, Purpose: identity.Purpose, EnvironmentRef: identity.EnvironmentRef, Status: identity.Status, LaneGrants: identity.LaneGrants, Handle: identity.Handle, DisplayLabel: identity.DisplayLabel, Lifecycle: identity.Lifecycle, AutomationMode: identity.AutomationMode})
	}
	for _, action := range h.service.Actions {
		response.Actions = append(response.Actions, &channelmanagerv1.Action{Id: action.ID, IdentityId: action.IdentityID, Kind: action.Kind, Window: action.Window.Format(time.RFC3339), Status: string(action.Status), RolledCount: int32(action.RolledCount)})
	}
	return connect.NewResponse(response), nil
}

func (h *api) GetEligibility(_ context.Context, request *connect.Request[channelmanagerv1.GetEligibilityRequest]) (*connect.Response[channelmanagerv1.GetEligibilityResponse], error) {
	if request.Msg.IdentityId == "" || request.Msg.Lane == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("identity_id and lane are required"))
	}
	h.mu.Lock()
	eligibility := h.service.Eligibility(request.Msg.IdentityId, request.Msg.Lane)
	h.mu.Unlock()
	return connect.NewResponse(&channelmanagerv1.GetEligibilityResponse{Eligibility: eligibility}), nil
}

func (h *api) SubmitRelease(ctx context.Context, request *connect.Request[channelmanagerv1.SubmitReleaseRequest]) (*connect.Response[channelmanagerv1.SubmitReleaseResponse], error) {
	if len(request.Msg.AssetIds) > 0 {
		if h.assets == nil {
			return nil, connect.NewError(connect.CodeUnavailable, errors.New("asset studio verification is not configured"))
		}
		for _, assetID := range request.Msg.AssetIds {
			reference, err := h.assets.ResolveReleasedAsset(ctx, assetID)
			if err != nil || reference.ID != assetID {
				if err == nil {
					err = errors.New("asset studio returned a mismatched asset reference")
				}
				return nil, connect.NewError(connect.CodeFailedPrecondition, err)
			}
		}
	}
	h.mu.Lock()
	receipt, err := h.service.ReleaseWithOptions(request.Msg.IdentityId, request.Msg.Lane, request.Msg.DraftId, request.Msg.IdempotencyKey, core.ReleaseOptions{AssetIDs: request.Msg.AssetIds, DisclosureVisible: request.Msg.DisclosureVisible}, time.Now().UTC())
	if err == nil {
		err = h.store.Save(ctx, h.service)
	}
	h.mu.Unlock()
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&channelmanagerv1.SubmitReleaseResponse{Receipt: receiptMessage(receipt)}), nil
}

func (h *api) DeliverReleaseOutcome(ctx context.Context, request *connect.Request[channelmanagerv1.DeliverReleaseOutcomeRequest]) (*connect.Response[channelmanagerv1.DeliverReleaseOutcomeResponse], error) {
	if h.deliverer == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("content desk delivery is not configured"))
	}
	h.mu.Lock()
	var receipt *core.ReleaseReceipt
	for _, candidate := range h.service.Releases {
		if candidate.ID == request.Msg.ReleaseId {
			receipt = candidate
			break
		}
	}
	if receipt == nil || !receipt.Complete() {
		h.mu.Unlock()
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("completed release receipt not found"))
	}
	copy := *receipt
	h.mu.Unlock()
	err := h.deliverer.DeliverRelease(ctx, contentdesk.ReleaseOutcome{ReceiptID: copy.ID, DraftID: copy.DraftID, Status: copy.Status, PlatformPostID: copy.PlatformPostID, PublishedURL: copy.PublishedURL, PublishedAt: copy.CompletedAt})
	h.mu.Lock()
	markErr := h.service.MarkReleaseDelivery(copy.ID, err == nil, errorText(err))
	if markErr == nil {
		markErr = h.store.Save(ctx, h.service)
	}
	h.mu.Unlock()
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	if markErr != nil {
		return nil, connect.NewError(connect.CodeInternal, markErr)
	}
	return connect.NewResponse(&channelmanagerv1.DeliverReleaseOutcomeResponse{DeliveryStatus: "delivered"}), nil
}

func (h *api) DeliverMetricSample(ctx context.Context, request *connect.Request[channelmanagerv1.DeliverMetricSampleRequest]) (*connect.Response[channelmanagerv1.DeliverMetricSampleResponse], error) {
	if h.deliverer == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("content desk delivery is not configured"))
	}
	h.mu.Lock()
	sample := h.service.MetricSamples[request.Msg.SampleId]
	if sample == nil {
		h.mu.Unlock()
		return nil, connect.NewError(connect.CodeNotFound, errors.New("metric sample not found"))
	}
	copy := *sample
	h.mu.Unlock()
	err := h.deliverer.DeliverMetric(ctx, contentdesk.MetricSample{ID: copy.ID, ReleaseID: copy.ReleaseID, DraftID: copy.DraftID, Metric: copy.Metric, Value: copy.Value, ObservedAt: copy.ObservedAt})
	h.mu.Lock()
	if err == nil {
		err = h.service.AcknowledgeMetric(copy.ID)
	}
	if err == nil {
		err = h.store.Save(ctx, h.service)
	}
	h.mu.Unlock()
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewResponse(&channelmanagerv1.DeliverMetricSampleResponse{DeliveryStatus: "acknowledged"}), nil
}

func (h *api) AssignAutomation(ctx context.Context, request *connect.Request[channelmanagerv1.AssignAutomationRequest]) (*connect.Response[channelmanagerv1.AssignAutomationResponse], error) {
	h.mu.Lock()
	err := h.service.AssignAutomation(request.Msg.IdentityId, request.Msg.ConsumerProfileKey, request.Msg.SessionProfileRef, request.Msg.WorkflowRef, request.Msg.EnabledActionKinds, request.Msg.OperatorNote)
	if err == nil {
		err = h.store.Save(ctx, h.service)
	}
	h.mu.Unlock()
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&channelmanagerv1.AssignAutomationResponse{IdentityId: request.Msg.IdentityId}), nil
}

func (h *api) DispatchBrowserAction(ctx context.Context, request *connect.Request[channelmanagerv1.DispatchBrowserActionRequest]) (*connect.Response[channelmanagerv1.DispatchBrowserActionResponse], error) {
	if h.browser == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("browser automation is not configured"))
	}
	h.mu.Lock()
	executionID, err := h.service.DispatchBrowser(ctx, request.Msg.ActionId, h.browser)
	if err == nil {
		err = h.store.Save(ctx, h.service)
	}
	h.mu.Unlock()
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&channelmanagerv1.DispatchBrowserActionResponse{ExecutionId: executionID}), nil
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func receiptMessage(receipt *core.ReleaseReceipt) *channelmanagerv1.ReleaseReceipt {
	return &channelmanagerv1.ReleaseReceipt{Id: receipt.ID, DraftId: receipt.DraftID, ActionId: receipt.ActionID, Status: receipt.Status, PlatformPostId: receipt.PlatformPostID, PublishedUrl: receipt.PublishedURL, FirstCommentStatus: receipt.FirstCommentStatus}
}

func Schema() string { return core.Schema() }

var Endpoints = []module.EndpointDescriptor{{
	ID: "channel_manager_overview", Path: channelmanagerconnect.ChannelManagerServiceGetOverviewProcedure, Method: http.MethodPost,
	Summary: "Get channel manager overview", Description: "Returns identity and queued-action references without credential values.", Category: "channel-manager",
	Response: &module.Schema{Type: "object", Properties: map[string]string{"identities": "array<Identity>", "actions": "array<Action>"}},
},
	{ID: "channel_manager_eligibility", Path: channelmanagerconnect.ChannelManagerServiceGetEligibilityProcedure, Method: http.MethodPost, Summary: "Get identity lane eligibility", Category: "channel-manager"},
	{ID: "channel_manager_submit_release", Path: channelmanagerconnect.ChannelManagerServiceSubmitReleaseProcedure, Method: http.MethodPost, Summary: "Submit idempotent release", Category: "channel-manager"},
	{ID: "channel_manager_deliver_release", Path: channelmanagerconnect.ChannelManagerServiceDeliverReleaseOutcomeProcedure, Method: http.MethodPost, Summary: "Deliver completed release outcome to Content Desk", Category: "channel-manager"},
	{ID: "channel_manager_deliver_metric", Path: channelmanagerconnect.ChannelManagerServiceDeliverMetricSampleProcedure, Method: http.MethodPost, Summary: "Deliver metric sample to Content Desk", Category: "channel-manager"},
	{ID: "channel_manager_assign_automation", Path: channelmanagerconnect.ChannelManagerServiceAssignAutomationProcedure, Method: http.MethodPost, Summary: "Assign an operator-approved BAS profile reference", Category: "channel-manager"},
	{ID: "channel_manager_dispatch_browser", Path: channelmanagerconnect.ChannelManagerServiceDispatchBrowserActionProcedure, Method: http.MethodPost, Summary: "Dispatch a durable queued action to BAS", Category: "channel-manager"},
	{ID: "channel_manager_browser_execution_review", Path: channelmanagerconnect.ChannelManagerServiceGetBrowserExecutionReviewProcedure, Method: http.MethodPost, Summary: "Review bounded BAS execution status and artifact references", Category: "channel-manager"},
}

func (h *api) overview(w http.ResponseWriter, _ *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()
	support := map[string]int{}
	for _, outcome := range h.service.ProgramOutcomes {
		support[outcome.ProgramID]++
	}
	identities := make(map[string]core.Identity, len(h.service.Identities))
	for id, identity := range h.service.Identities {
		identities[id] = publicIdentity(*identity)
	}
	automation := make(map[string]publicAutomationAssignment, len(h.service.Automation))
	for identityID, assignment := range h.service.Automation {
		automation[identityID] = publicAutomation(assignment)
	}
	writeJSON(w, http.StatusOK, map[string]any{"identities": identities, "actions": h.service.Actions, "platforms": h.service.Platforms, "programs": h.service.Programs, "program_support": support, "flags": h.service.Flags, "releases": h.service.Releases, "metric_samples": h.service.MetricSamples, "automation": automation, "portfolio": h.service.Portfolio, "program_outcomes": h.service.ProgramOutcomes, "environment_checks": h.service.EnvironmentChecks, "activity_events": h.service.Timeline("", "", "")})
}

func publicIdentity(identity core.Identity) core.Identity {
	identity.VaultRef = ""
	return identity
}

// publicAutomation preserves operator readiness without disclosing BAS's
// opaque profile/workflow references through the unauthenticated overview.
type publicAutomationAssignment struct {
	ProfileKey   string   `json:"profile_key"`
	EnabledKinds []string `json:"enabled_kinds"`
	OperatorNote string   `json:"operator_note"`
}

func publicAutomation(assignment core.AutomationAssignment) publicAutomationAssignment {
	return publicAutomationAssignment{ProfileKey: assignment.ProfileKey, EnabledKinds: append([]string(nil), assignment.EnabledKinds...), OperatorNote: assignment.OperatorNote}
}

func (h *api) createIdentity(w http.ResponseWriter, r *http.Request) {
	var identity core.Identity
	if !decode(w, r, &identity) {
		return
	}
	h.mu.Lock()
	err := h.service.CreateIdentity(identity)
	if err == nil {
		err = h.store.Save(r.Context(), h.service)
	}
	h.mu.Unlock()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, publicIdentity(identity))
}

func (h *api) updateIdentity(w http.ResponseWriter, r *http.Request) {
	var identity core.Identity
	if !decode(w, r, &identity) {
		return
	}
	h.mu.Lock()
	err := h.service.UpdateIdentity(mux.Vars(r)["id"], identity)
	if err == nil {
		err = h.store.Save(r.Context(), h.service)
	}
	h.mu.Unlock()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, publicIdentity(identity))
}

func (h *api) retireIdentity(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	err := h.service.RetireIdentity(mux.Vars(r)["id"])
	if err == nil {
		err = h.store.Save(r.Context(), h.service)
	}
	h.mu.Unlock()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "retired"})
}

func (h *api) timeline(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	events := h.service.Timeline(mux.Vars(r)["id"], r.URL.Query().Get("action_id"), r.URL.Query().Get("event_type"))
	h.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

func (h *api) start(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProgramID string `json:"program_id"`
	}
	if !decode(w, r, &req) {
		return
	}
	h.mu.Lock()
	err := h.service.StartProgram(mux.Vars(r)["id"], req.ProgramID)
	if err == nil {
		err = h.store.Save(r.Context(), h.service)
	}
	h.mu.Unlock()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "warming"})
}

func (h *api) enqueue(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IdentityID     string    `json:"identity_id"`
		Kind           string    `json:"kind"`
		Window         time.Time `json:"window"`
		Seed           uint64    `json:"seed"`
		IdempotencyKey string    `json:"idempotency_key"`
	}
	if !decode(w, r, &req) {
		return
	}
	if req.Window.IsZero() {
		req.Window = time.Now().UTC()
	}
	h.mu.Lock()
	a, err := h.service.Enqueue(req.IdentityID, req.Kind, req.Window, req.Seed, req.IdempotencyKey)
	if err == nil {
		err = h.store.Save(r.Context(), h.service)
	}
	h.mu.Unlock()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

func (h *api) complete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Evidence string `json:"evidence"`
	}
	if !decode(w, r, &req) {
		return
	}
	h.mu.Lock()
	err := h.service.Complete(mux.Vars(r)["id"], req.Evidence, time.Now().UTC())
	if err == nil {
		err = h.store.Save(r.Context(), h.service)
	}
	h.mu.Unlock()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "succeeded"})
}

func (h *api) completeRelease(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlatformPostID       string `json:"platform_post_id"`
		PublishedURL         string `json:"published_url"`
		FirstCommentStatus   string `json:"first_comment_status"`
		FirstCommentEvidence string `json:"first_comment_evidence"`
	}
	if !decode(w, r, &req) {
		return
	}
	h.mu.Lock()
	receipt, err := h.service.CompleteRelease(mux.Vars(r)["id"], req.PlatformPostID, req.PublishedURL, req.FirstCommentStatus, req.FirstCommentEvidence, time.Now().UTC())
	if err == nil {
		err = h.store.Save(r.Context(), h.service)
	}
	h.mu.Unlock()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, receipt)
}

func (h *api) observe(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Metric string  `json:"metric"`
		Value  float64 `json:"value"`
	}
	if !decode(w, r, &req) {
		return
	}
	h.mu.Lock()
	flag, err := h.service.RecordObservation(mux.Vars(r)["id"], req.Metric, req.Value, time.Now().UTC(), 3, .5)
	if err == nil {
		err = h.store.Save(r.Context(), h.service)
	}
	h.mu.Unlock()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"flag": flag})
}
func (h *api) eligibility(w http.ResponseWriter, r *http.Request) {
	lane := r.URL.Query().Get("lane")
	if lane == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "lane is required"})
		return
	}
	h.mu.Lock()
	result := h.service.Eligibility(mux.Vars(r)["id"], lane)
	h.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]string{"eligibility": result})
}
func (h *api) release(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IdentityID     string `json:"identity_id"`
		Lane           string `json:"lane"`
		DraftID        string `json:"draft_id"`
		IdempotencyKey string `json:"idempotency_key"`
	}
	if !decode(w, r, &req) {
		return
	}
	if req.IdempotencyKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "idempotency_key is required"})
		return
	}
	h.mu.Lock()
	receipt, err := h.service.Release(req.IdentityID, req.Lane, req.DraftID, req.IdempotencyKey, time.Now().UTC())
	if err == nil {
		err = h.store.Save(r.Context(), h.service)
	}
	h.mu.Unlock()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, receipt)
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON request"})
		return false
	}
	return true
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, core.ErrPaused) || errors.Is(err, core.ErrCadence) || errors.Is(err, core.ErrForbiddenAction) || errors.Is(err, core.ErrPreconditions) {
		status = http.StatusConflict
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
