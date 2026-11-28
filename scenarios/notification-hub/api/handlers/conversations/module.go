package conversations

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"maps"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/filekind"
	apilifecycle "github.com/vrooli/api-core/lifecycle"
	"github.com/vrooli/api-core/operatordecision"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/shared"
	"google.golang.org/protobuf/types/known/structpb"

	"notification-hub/internal/hub"
	identity "notification-hub/internal/identity"
	"notification-hub/internal/linkedartifact"
	"notification-hub/internal/module"
)

// ErrorReasonHeader is the Connect error metadata key that names why a
// FailedPrecondition happened, when a client needs to tell causes apart.
const ErrorReasonHeader = "Hub-Error-Reason"

// ReasonAskChanged is the ErrorReasonHeader value of Answer on an ask the
// requester revised while the answer was being made.
const ReasonAskChanged = "ask_changed"

// warnAskReadRestricted is the packet-check code of an ask that already exists
// but that the caller cannot read back: it belongs to another conversation and
// another owner account.
const warnAskReadRestricted = "ask_read_restricted"

// ReasonContextNotAccepted is the ErrorReasonHeader value of RequestContext on
// an ask whose requester did not declare it answers questions.
const ReasonContextNotAccepted = "context_not_accepted"

// ReasonAskRefused marks a FailedPrecondition that refused an agent's ask for a
// blocking decision-quality finding (hub.RefusedError): nothing was stored, and
// the message names each finding and its fix.
const ReasonAskRefused = "ask_refused"

type handler struct {
	service                  *hub.Service
	verifier                 identity.Verifier
	stepUp                   rootPasswordAuthenticator
	beforeRootReadDisclosure func() // test seam, immediately before final association check
	linkedArtifacts          *linkedartifact.Resolver
	// services verifies the service-principal roles: ask readers and senders. A
	// nil verifier turns its role off.
	services identity.ServiceVerifiers
	// uploads holds one token per file-sized upload being received; uploadWait is
	// how long an upload waits for one (see gateUploads).
	uploads    chan struct{}
	uploadWait time.Duration
}

func Module(service *hub.Service) module.Module {
	return ModuleWithVerifier(service, nil)
}

func ModuleWithVerifier(service *hub.Service, verifier identity.Verifier) module.Module {
	return ModuleWithIdentities(service, verifier, identity.ServiceVerifiers{})
}

// ModuleWithIdentities is ModuleWithVerifier plus the verifiers of the service
// roles: the tokens that read a service's own asks (askreader.NewVerifier) and
// the tokens that send as a service (hubsender.NewVerifier).
func ModuleWithIdentities(service *hub.Service, verifier identity.Verifier, services identity.ServiceVerifiers) module.Module {
	return ModuleWithLinkedArtifacts(service, verifier, services, nil)
}

func ModuleWithLinkedArtifacts(service *hub.Service, verifier identity.Verifier, services identity.ServiceVerifiers, artifacts *linkedartifact.Resolver) module.Module {
	return newHandler(service, verifier, services, artifacts).module()
}

func newHandler(service *hub.Service, verifier identity.Verifier, services identity.ServiceVerifiers, artifacts *linkedartifact.Resolver) *handler {
	return &handler{service: service, verifier: verifier, services: services, linkedArtifacts: artifacts, uploads: make(chan struct{}, maxConcurrentUploads), uploadWait: uploadSlotWait}
}

// ModuleWithStepUp wires the authenticator used for a fresh root decision.
func ModuleWithStepUp(service *hub.Service, verifier identity.Verifier, services identity.ServiceVerifiers, artifacts *linkedartifact.Resolver, forwarder identity.Forwarder) module.Module {
	h := newHandler(service, verifier, services, artifacts)
	h.stepUp = forwarder
	return h.module()
}

func (h *handler) module() module.Module {
	return module.Module{Name: "conversations", Mount: func(r *mux.Router) {
		path, general, upload := h.connectHandlers()
		r.Handle(connectv1.ConversationsServicePutAttachmentBlobProcedure, upload).Methods(http.MethodPost)
		r.Handle(connectv1.ConversationsServiceWaitProcedure, budgetWait(general)).Methods(http.MethodPost)
		r.PathPrefix(path).Handler(general)
		r.HandleFunc("/api/v1/identity/root-decision-read", h.rootDecisionRead).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/identity/root-enrollment", h.rootEnrollment).Methods(http.MethodPost, http.MethodDelete)
		r.HandleFunc(attachmentContentPath, h.serveAttachment).Methods(http.MethodGet, http.MethodHead)
		r.HandleFunc(answerImageContentPath, h.serveAnswerImage).Methods(http.MethodGet, http.MethodHead)
		r.HandleFunc(answerFileContentPath, h.serveAnswerFile).Methods(http.MethodGet, http.MethodHead)
		r.HandleFunc(linkedArtifactMetadataPath, h.linkedArtifactMetadata).Methods(http.MethodGet)
		r.HandleFunc(linkedArtifactContentPath, h.serveLinkedArtifact).Methods(http.MethodGet, http.MethodHead)
		r.HandleFunc("/api/v1/conversations/asks/{ask_id}/portfolio-continuity", h.attestPortfolioContinuity).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/conversations/asks/{ask_id}/followthrough", h.reportFollowthrough).Methods(http.MethodPost)
	}, Endpoints: Endpoints}
}

type linkedArtifactMetadata struct {
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	MIMEType   string `json:"mimeType"`
	SizeBytes  int64  `json:"sizeBytes"`
	ContentURL string `json:"contentUrl"`
}

func (h *handler) authorizeLinkedArtifact(w http.ResponseWriter, r *http.Request) bool {
	owner, err := identity.Subject(r.Context(), r.Header, h.verifier)
	if err != nil || owner == "" {
		http.Error(w, "verified owner identity is required", http.StatusUnauthorized)
		return false
	}
	if _, err := h.service.GetAskFor(r.Context(), mux.Vars(r)["ask_id"], owner, hub.OwnerAudience()); err != nil {
		notFound(w)
		return false
	}
	return true
}

func (h *handler) linkedArtifactMetadata(w http.ResponseWriter, r *http.Request) {
	if !h.authorizeLinkedArtifact(w, r) {
		return
	}
	file, info, err := h.linkedArtifacts.Open(r.URL.Query().Get("path"))
	if err != nil {
		linkedArtifactError(w, err)
		return
	}
	defer file.Close()
	name := filepath.Base(info.Name())
	result := filekind.Classify(name, nil)
	contentURL := linkedArtifactContentPath + "?path=" + url.QueryEscape(r.URL.Query().Get("path"))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "private, no-store")
	_ = json.NewEncoder(w).Encode(linkedArtifactMetadata{Name: name, Kind: string(result.Kind), MIMEType: result.MIME, SizeBytes: info.Size(), ContentURL: contentURL})
}

func (h *handler) serveLinkedArtifact(w http.ResponseWriter, r *http.Request) {
	if !h.authorizeLinkedArtifact(w, r) {
		return
	}
	file, info, err := h.linkedArtifacts.Open(r.URL.Query().Get("path"))
	if err != nil {
		linkedArtifactError(w, err)
		return
	}
	defer file.Close()
	name := filepath.Base(info.Name())
	result := filekind.Classify(name, nil)
	header := w.Header()
	header.Set("Content-Type", result.MIME)
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Cache-Control", "private, no-store")
	header.Set("Content-Security-Policy", attachmentContentCSP)
	if disposition := mime.FormatMediaType("inline", map[string]string{"filename": name}); disposition != "" {
		header.Set("Content-Disposition", disposition)
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(transferTimeout))
	http.ServeContent(w, r, name, info.ModTime(), file)
}

func linkedArtifactError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, linkedartifact.ErrUnavailable):
		http.Error(w, "linked artifact source is unavailable", http.StatusServiceUnavailable)
	case errors.Is(err, linkedartifact.ErrTooLarge):
		http.Error(w, "linked artifact exceeds the preview limit", http.StatusRequestEntityTooLarge)
	default:
		notFound(w)
	}
}

// producer admits verified senders to read or withdraw their own asks. Ask and
// packet writes remain available to identityless callers; answers and
// dismissals still use owner.
func (h *handler) producer(ctx context.Context, headers http.Header) (identity.Producer, error) {
	producer, err := h.directProducer(ctx, headers)
	if err != nil {
		return identity.Producer{}, err
	}
	if producer.Anonymous() {
		return identity.Producer{}, identity.DirectProducerError(errors.New("no identity presented"))
	}
	return producer, nil
}

func (h *handler) directProducer(ctx context.Context, headers http.Header) (identity.Producer, error) {
	return identity.ResolveDirectProducer(ctx, headers, h.verifier, h.services, h.service.ResolveRecipient(ctx))
}

// reader resolves which asks the caller may read. A caller with no verified
// identity is refused: asks hold what the operator and a requester said to each
// other, and loopback reachability proves nothing about who is asking.
func (h *handler) reader(ctx context.Context, headers http.Header) (hub.AskAudience, error) {
	audience, err := identity.ReaderFor(ctx, headers, h.verifier, h.services)
	if err == nil && audience.Owner {
		subject, subjectErr := identity.Subject(ctx, headers, h.verifier)
		recipient := h.service.ResolveRecipient(ctx)
		if subjectErr != nil || subject == "" || (recipient != "" && subject != recipient) {
			err = errors.New("owner does not match recipient")
		}
	}
	if err != nil {
		return hub.AskAudience{}, connect.NewError(connect.CodeUnauthenticated, errors.New("a verified owner, agent run or service identity is required to read asks"))
	}
	return audience, nil
}

func (h *handler) owner(ctx context.Context, headers http.Header) (string, error) {
	subject, err := identity.Subject(ctx, headers, h.verifier)
	if err != nil {
		return "", connect.NewError(connect.CodeUnauthenticated, errors.New("verified owner identity is required"))
	}
	return subject, nil
}

func (h *handler) Ask(ctx context.Context, req *connect.Request[v1.AskRequest]) (*connect.Response[v1.AskResponse], error) {
	producer, err := h.directProducer(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	spec, err := askSpecFromProto(req.Msg, producer, time.Now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	result, err := h.service.AskDecision(ctx, spec, producer.Key)
	if err != nil {
		return nil, connectError(err)
	}
	ask, notification := result.Ask, result.Notification
	if producer.Creator() != "" && !producer.Audience().SeesByKey(ask) {
		// The ask exists (same subject key or idempotency key) but belongs to
		// another conversation and another owner, so this caller can neither read
		// it back nor wait on it.
		result.Warnings = append(result.Warnings, hub.PacketWarning{
			Code: warnAskReadRestricted, Severity: hub.WarningAdvice,
			Message: "This ask was sent from another conversation and by another owner account, so get and wait will not find it for you.",
		})
	}
	return connect.NewResponse(&v1.AskResponse{AskId: ask.ID, Outcome: result.Outcome, State: ask.State, Kind: ask.Kind, Warnings: warningsToProto(result.Warnings), Notification: &shared.Notification{Id: notification.ID, RequestedBy: notification.RequestedBy, Title: notification.Title, Body: notification.Body, Urgency: notification.Urgency, SensitivityLabel: notification.SensitivityLabel, IdempotencyKey: notification.IdempotencyKey, DedupeKey: notification.DedupeKey, State: notificationState(notification.State), Reason: notification.Reason, CreatedAt: notification.CreatedAt, UpdatedAt: notification.UpdatedAt}}), nil
}

func notificationState(state string) shared.NotificationState {
	switch state {
	case "pending":
		return shared.NotificationState_NOTIFICATION_STATE_PENDING
	case "held":
		return shared.NotificationState_NOTIFICATION_STATE_HELD
	case "delivered":
		return shared.NotificationState_NOTIFICATION_STATE_DELIVERED
	case "failed":
		return shared.NotificationState_NOTIFICATION_STATE_FAILED
	default:
		return shared.NotificationState_NOTIFICATION_STATE_UNSPECIFIED
	}
}

func (h *handler) Answer(ctx context.Context, req *connect.Request[v1.AnswerRequest]) (*connect.Response[v1.AnswerResponse], error) {
	actor, err := h.owner(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	// Attribute the answer to the ACTUAL verified human (the Cloudflare Access
	// subject/email) while keeping the recipient-only authorization check keyed on
	// the recipient subject (actor). AnsweredBy is "" on the bearer/cookie path,
	// where AnswerAsk falls back to actor — preserving prior behavior.
	answerIdentityKind, answerIdentitySource := identity.AnswerIdentity(ctx)
	ask, err := h.service.AnswerAsk(ctx, hub.AnswerInput{
		AskID:                    req.Msg.GetAskId(),
		Answer:                   req.Msg.GetAnswer(),
		Actor:                    actor,
		AnsweredBy:               identity.AnsweredBy(ctx),
		AnsweredByIdentityKind:   answerIdentityKind,
		AnsweredByIdentitySource: answerIdentitySource,
		Note:                     req.Msg.GetNote(),
		Changes:                  structMap(req.Msg.GetChanges()),
		Images:                   answerImagesFromProto(req.Msg.GetImages()),
		Attachments:              answerAttachmentsFromProto(req.Msg.GetAttachments()),
		Responses:                responsesFromProto(req.Msg.GetResponses()),
		AttachmentNotes:          attachmentNotesFromProto(req.Msg.GetAttachmentNotes()),
	})
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&v1.AnswerResponse{AskId: ask.ID, Answer: ask.Answer, AnsweredAt: ask.AnsweredAt, Late: ask.Late, Changes: structValue(ask.AnswerChanges), Images: answerImagesToProto(ask.AnswerImages), Attachments: answerAttachmentsToProto(ask.AnswerAttachments)}), nil
}

func (h *handler) MarkRead(ctx context.Context, req *connect.Request[v1.MarkReadRequest]) (*connect.Response[v1.MarkReadResponse], error) {
	actor, err := h.owner(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	ask, err := h.service.MarkReadStatus(ctx, req.Msg.GetAskId(), actor)
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&v1.MarkReadResponse{AskId: ask.ID, State: ask.State}), nil
}

func (h *handler) Dismiss(ctx context.Context, req *connect.Request[v1.DismissRequest]) (*connect.Response[v1.DismissResponse], error) {
	actor, err := h.owner(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	ask, err := h.service.DismissAsk(ctx, req.Msg.GetAskId(), actor, req.Msg.GetNote())
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&v1.DismissResponse{AskId: ask.ID, State: ask.State}), nil
}

func (h *handler) Withdraw(ctx context.Context, req *connect.Request[v1.WithdrawRequest]) (*connect.Response[v1.WithdrawResponse], error) {
	requester, err := h.producer(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	// Withdrawing closes an ask for its sender, so it follows the rule reading
	// does: by id, the ask must be in the caller's conversation; by subject key,
	// naming the exact key is enough for a run of the same owner.
	audience := requester.Audience()
	var ask hub.AskRecord
	switch {
	case req.Msg.GetSubjectKey() != "" && req.Msg.GetAskId() != "":
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("give ask_id or subject_key, not both"))
	case req.Msg.GetSubjectKey() != "":
		ask, err = h.service.WithdrawDirectSubject(ctx, requester.AskSource(), req.Msg.GetSubjectKey(), req.Msg.GetReason(), audience)
	default:
		ask, err = h.service.WithdrawDirectAsk(ctx, req.Msg.GetAskId(), requester.AskSource(), req.Msg.GetReason(), audience)
	}
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&v1.WithdrawResponse{AskId: ask.ID, State: ask.State}), nil
}

// findAsk resolves the ask a read names: by id, which needs the reader to be in
// the ask's audience (the owner, its conversation lineage, or the service that
// raised it), or by exact subject key, which a run of the same owner may also
// name. Exactly one of the two is given. An ask the reader may not use is not
// found, exactly like one that does not exist.
func (h *handler) findAsk(ctx context.Context, askID, subjectKey string, audience hub.AskAudience) (hub.AskRecord, error) {
	askID, subjectKey = strings.TrimSpace(askID), strings.TrimSpace(subjectKey)
	switch {
	case askID != "" && subjectKey != "":
		return hub.AskRecord{}, connect.NewError(connect.CodeInvalidArgument, errors.New("give ask_id or subject_key, not both"))
	case askID == "" && subjectKey == "":
		return hub.AskRecord{}, connect.NewError(connect.CodeInvalidArgument, errors.New("give ask_id or subject_key"))
	case subjectKey != "" && (len(subjectKey) > 128 || !operatordecision.SubjectKeyRule.MatchString(subjectKey)):
		return hub.AskRecord{}, connect.NewError(connect.CodeInvalidArgument, errors.New("subject_key is invalid"))
	}
	recipient := h.service.ResolveRecipient(ctx)
	if recipient == "" {
		return hub.AskRecord{}, connect.NewError(connect.CodeNotFound, errors.New("ask not found"))
	}
	var (
		ask hub.AskRecord
		err error
	)
	if subjectKey != "" {
		ask, err = h.service.GetAskBySubjectKeyFor(ctx, subjectKey, recipient, audience)
	} else {
		ask, err = h.service.GetAskFor(ctx, askID, recipient, audience)
	}
	if err != nil {
		if errors.Is(err, hub.ErrNotFound) {
			h.logUnseenAsk(ctx, askID, subjectKey, recipient, audience)
		}
		return hub.AskRecord{}, connectError(err)
	}
	return ask, nil
}

func (h *handler) attestPortfolioContinuity(w http.ResponseWriter, r *http.Request) {
	owner, err := identity.Subject(r.Context(), r.Header, h.verifier)
	if err != nil {
		// A verified run may hand off only an ask that its existing creator or
		// conversation lineage can already read. Do not turn an owner subject or
		// a caller-selected ask ID into a broader migration capability.
		audience, runErr := identity.ReaderFor(r.Context(), r.Header, h.verifier, h.services)
		if runErr != nil || audience.Owner || !strings.HasPrefix(audience.CreatedBy, "run:") {
			http.Error(w, "verified owner or agent run identity required", http.StatusUnauthorized)
			return
		}
		ask, err := h.service.AttestPortfolioAskFromRun(r.Context(), mux.Vars(r)["ask_id"], audience)
		if err != nil {
			http.Error(w, "eligible pending ask not found", http.StatusNotFound)
			return
		}
		writePortfolioContinuity(w, ask)
		return
	}
	ask, err := h.service.AttestPortfolioAsk(r.Context(), mux.Vars(r)["ask_id"], owner)
	if err != nil {
		http.Error(w, "eligible pending ask not found", http.StatusNotFound)
		return
	}
	writePortfolioContinuity(w, ask)
}

// reportFollowthrough records what the owner's run did after the answer. It is
// the only write in the aftermath path: the projection on the ask is read-only
// and says "no owner reported" until this is called. The reporter must be the
// owner the ask was addressed to; a scenario service reports through the same
// owner on this host rather than gaining a second way to read asks.
func (h *handler) reportFollowthrough(w http.ResponseWriter, r *http.Request) {
	askID := mux.Vars(r)["ask_id"]
	owner, err := identity.Subject(r.Context(), r.Header, h.verifier)
	if err != nil {
		http.Error(w, "verified owner identity required", http.StatusUnauthorized)
		return
	}
	if _, err := h.service.GetAskForRecipient(r.Context(), askID, owner); err != nil {
		http.Error(w, "decision not found", http.StatusNotFound)
		return
	}
	var body struct {
		State        string `json:"state"`
		Owner        string `json:"owner"`
		RunRef       string `json:"run_ref"`
		StartedAt    string `json:"started_at"`
		FinishedAt   string `json:"finished_at"`
		BoundSeconds int    `json:"bound_seconds"`
		ResultURL    string `json:"result_url"`
		Summary      string `json:"summary"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&body); err != nil {
		http.Error(w, "follow-through body must be JSON", http.StatusBadRequest)
		return
	}
	reporter := strings.TrimSpace(body.Owner)
	if reporter == "" {
		reporter = owner
	}
	report, err := h.service.RecordFollowthrough(r.Context(), askID, hub.Followthrough{
		State: body.State, Owner: reporter, RunRef: body.RunRef,
		StartedAt: body.StartedAt, FinishedAt: body.FinishedAt, BoundSeconds: body.BoundSeconds,
		ResultURL: body.ResultURL, Summary: body.Summary,
	})
	if err != nil {
		switch {
		case errors.Is(err, hub.ErrNotFound):
			http.Error(w, "decision not found", http.StatusNotFound)
		case errors.Is(err, hub.ErrInvalidArgument):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "follow-through could not be recorded", http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ask_id": askID, "state": report.State, "owner": report.Owner, "run_ref": report.RunRef,
		"started_at": report.StartedAt, "finished_at": report.FinishedAt, "bound_seconds": report.BoundSeconds,
		"result_url": report.ResultURL, "summary": report.Summary, "reported_at": report.ReportedAt,
	})
}

func writePortfolioContinuity(w http.ResponseWriter, ask hub.AskRecord) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"ask_id": ask.ID, "source": "portfolio-steward", "recipient": ask.Recipient, "subject_key": ask.SubjectKey, "attested": "true"})
}

// logUnseenAsk records why a verified reader was told an ask does not exist,
// without telling the reader: whether the ask exists for the recipient at all,
// and which kind of reader asked. A scenario whose read-back of its own ask
// fails is diagnosed from this line, not by guessing at its token.
func (h *handler) logUnseenAsk(ctx context.Context, askID, subjectKey, recipient string, audience hub.AskAudience) {
	attrs := []any{"ask_id", askID, "subject_key", subjectKey, "reader", readerKind(audience), "recipient", recipient}
	if audience.Source != "" {
		attrs = append(attrs, "reader_source", audience.Source)
	}
	if askID != "" {
		record, err := h.service.GetAskForRecipient(ctx, askID, recipient)
		switch {
		case err != nil:
			attrs = append(attrs, "exists_for_recipient", false)
		default:
			attrs = append(attrs, "exists_for_recipient", true, "ask_source", record.Source, "event_sourced", record.SourceEventType != "")
		}
	}
	slog.Info("ask read refused as not found", attrs...)
}

// readerKind names the kind of verified reader an audience belongs to.
func readerKind(audience hub.AskAudience) string {
	switch {
	case audience.Owner:
		return "owner"
	case audience.Source != "":
		return "source"
	case strings.HasPrefix(audience.CreatedBy, "run:"):
		return "run"
	case audience.CreatedBy != "":
		return "sender"
	}
	return "none"
}

// maxKeyedWaitHops bounds how many replacements a wait by subject key follows.
const maxKeyedWaitHops = 10

// Wait blocks on one ask for a reader that may use it. An ask the reader may not
// see is reported as not found, before any waiting starts. Waiting by subject
// key follows the key: when the producer replaces the ask (a changed answer menu
// closes it and opens a successor), the wait continues on the successor, and
// returns when the key's ask is answered or otherwise ends.
func (h *handler) Wait(ctx context.Context, req *connect.Request[v1.WaitRequest]) (*connect.Response[v1.WaitResponse], error) {
	audience, err := h.reader(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	deadline, err := time.Parse(time.RFC3339, req.Msg.GetDeadline())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	ask, err := h.findAsk(ctx, req.Msg.GetAskId(), req.Msg.GetSubjectKey(), audience)
	if err != nil {
		return nil, err
	}
	options := hub.WaitOptions{Deadline: deadline, ReturnOnContextRequest: req.Msg.GetReturnOnContextRequest(), Yield: apilifecycle.Closing(ctx)}
	result, err := h.service.Wait(ctx, ask.ID, options)
	for hops := 0; err == nil && req.Msg.GetSubjectKey() != "" && result.Ask.SupersededByAskID != "" && hops < maxKeyedWaitHops; hops++ {
		if next, nextErr := h.findAsk(ctx, "", req.Msg.GetSubjectKey(), audience); nextErr == nil && next.ID != result.Ask.ID {
			result, err = h.service.Wait(ctx, next.ID, options)
			continue
		}
		break
	}
	if err != nil {
		return nil, connectError(err)
	}
	response := &v1.WaitResponse{
		AskId: result.Ask.ID, State: result.Ask.State, Answer: result.Ask.Answer, Reason: result.Reason, Note: result.Ask.Note,
		Responses: responsesToProto(result.Ask.Responses), AnswerSummary: result.Ask.AnswerSummary,
		AttachmentNotes: attachmentNotesToProto(result.Ask.AttachmentNotes), Packet: result.Ask.Packet,
		Changes: structValue(result.Ask.AnswerChanges), Images: answerImagesToProto(result.Ask.AnswerImages), Attachments: answerAttachmentsToProto(result.Ask.AnswerAttachments),
	}
	if result.ContextRequest != nil {
		response.ContextRequest = contextRequestToProto(*result.ContextRequest)
	}
	return connect.NewResponse(response), nil
}

func (h *handler) GetAsk(ctx context.Context, req *connect.Request[v1.GetAskRequest]) (*connect.Response[v1.GetAskResponse], error) {
	audience, err := h.reader(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	ask, err := h.findAsk(ctx, req.Msg.GetAskId(), req.Msg.GetSubjectKey(), audience)
	if err != nil {
		return nil, err
	}
	revisions, err := h.service.ListAskRevisions(ctx, ask.ID)
	if err != nil {
		return nil, connectError(err)
	}
	out := h.toProto(ask)
	recurrence, err := h.service.SubjectRecurrence(ctx, ask)
	if err != nil {
		return nil, connectError(err)
	}
	out.Recurrence = recurrenceToProto(recurrence)
	out.Questions = questionsToProto(ask.Questions)
	out.Responses = responsesToProto(ask.Responses)
	out.AttachmentNotes = attachmentNotesToProto(ask.AttachmentNotes)
	contextRequests, err := h.service.ListContextRequests(ctx, ask.ID)
	if err != nil {
		return nil, connectError(err)
	}
	out.ContextRequests = contextRequestsToProto(contextRequests)
	// The drift check reads the sender's host paths, so only the owner gets it.
	attachments, err := h.askAttachments(ctx, ask.ID, audience)
	if err != nil {
		return nil, connectError(err)
	}
	out.Attachments = attachmentsToProto(hub.WithCoverPage(ask, attachments))
	for _, revision := range revisions {
		options := make([]*v1.AskOption, 0, len(revision.Options))
		for _, option := range revision.Options {
			options = append(options, &v1.AskOption{Key: option.Key, Label: option.Label})
		}
		out.Revisions = append(out.Revisions, &v1.AskRevision{
			Revision: int32(revision.Revision), Title: revision.Title, Question: revision.Question,
			RecommendationReason: revision.RecommendationReason, ContextUrl: revision.ContextURL,
			BigPicture: revision.BigPicture, TechnicalDetails: revision.TechnicalDetails, Options: options,
			Deadline: revision.Deadline, RevisedAt: revision.RevisedAt,
		})
	}
	return connect.NewResponse(&v1.GetAskResponse{Ask: out}), nil
}

func (h *handler) ListAsks(ctx context.Context, req *connect.Request[v1.ListAsksRequest]) (*connect.Response[v1.ListAsksResponse], error) {
	audience, err := h.reader(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	recipient := h.service.ResolveRecipient(ctx)
	if recipient == "" {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("recipient not found"))
	}
	page, err := h.service.ListAsksPageFor(ctx, recipient, audience, hub.AskListOptions{
		OpenOnly: req.Msg.GetOpenOnly(), Limit: int(req.Msg.GetLimit()),
		Cursor: req.Msg.GetCursor(), Query: req.Msg.GetQuery(), StateFilter: req.Msg.GetStateFilter(),
		Order: askOrder(req.Msg.GetOrder()), Urgencies: req.Msg.GetUrgencies(), Senders: req.Msg.GetSenders(),
	})
	if err != nil {
		return nil, connectError(err)
	}
	out := make([]*v1.Ask, 0, len(page.Asks))
	for _, ask := range page.Asks {
		item := h.toProto(ask)
		// A list row carries the grade, never the whole report: the detail
		// read is where the findings belong, and a page of 50 stays small.
		item.QualityReport = ""
		out = append(out, item)
	}
	return connect.NewResponse(&v1.ListAsksResponse{Asks: out, NextCursor: page.NextCursor, TotalAwaiting: int32(page.TotalAwaiting), Facets: askFacetsToProto(page.Facets), ClosingWindowSeconds: int32(page.ClosingWindow / time.Second)}), nil
}

func (h *handler) ListPendingDecisionReferences(ctx context.Context, req *connect.Request[v1.ListPendingDecisionReferencesRequest]) (*connect.Response[v1.ListPendingDecisionReferencesResponse], error) {
	owner, err := h.pendingDecisionOwner(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	page, err := h.service.ListPendingDecisionReferences(ctx, owner, req.Msg.GetCursor(), int(req.Msg.GetLimit()))
	if err != nil {
		return nil, connectError(err)
	}
	out := &v1.ListPendingDecisionReferencesResponse{NextCursor: page.NextCursor, Decisions: make([]*v1.PendingDecisionReference, 0, len(page.Decisions))}
	for _, value := range page.Decisions {
		if err := hub.ValidatePendingDecisionReference(value); err != nil {
			return nil, connect.NewError(connect.CodeDataLoss, errors.New("pending decision projection is incomplete"))
		}
		out.Decisions = append(out.Decisions, pendingDecisionReferenceToProto(value))
	}
	return connect.NewResponse(out), nil
}

func (h *handler) ReadPendingDecisionReference(ctx context.Context, req *connect.Request[v1.ReadPendingDecisionReferenceRequest]) (*connect.Response[v1.ReadPendingDecisionReferenceResponse], error) {
	owner, err := h.pendingDecisionOwner(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	value, err := h.service.ReadPendingDecisionReference(ctx, owner, req.Msg.GetDecisionRef(), req.Msg.GetExpectedRevision())
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&v1.ReadPendingDecisionReferenceResponse{Decision: pendingDecisionReferenceToProto(value)}), nil
}

func (h *handler) pendingDecisionOwner(ctx context.Context, headers http.Header) (string, error) {
	owner, err := h.owner(ctx, headers)
	if err != nil {
		return "", err
	}
	recipient := h.service.ResolveRecipient(ctx)
	if recipient == "" || owner != recipient {
		return "", connect.NewError(connect.CodeUnauthenticated, errors.New("verified decision owner required"))
	}
	return owner, nil
}

func pendingDecisionReferenceToProto(value hub.PendingDecisionReference) *v1.PendingDecisionReference {
	return &v1.PendingDecisionReference{
		DecisionRef: value.DecisionRef, OwnerId: value.OwnerID, Source: value.Source,
		SourceEventType: value.SourceEventType, Revision: value.Revision, State: value.State,
		CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt, Deadline: value.Deadline,
		Urgency: value.Urgency,
	}
}

func askFacetsToProto(facets hub.AskFacets) *v1.AskFacets {
	convert := func(counts []hub.AskFacetCount) []*v1.AskFacetCount {
		out := make([]*v1.AskFacetCount, 0, len(counts))
		for _, count := range counts {
			out = append(out, &v1.AskFacetCount{Value: count.Value, Count: int32(count.Count)})
		}
		return out
	}
	return &v1.AskFacets{Urgencies: convert(facets.Urgencies), Senders: convert(facets.Senders)}
}

// askOrder maps the request's order to the hub's; an unspecified one is newest.
func askOrder(order v1.ListAsksRequest_Order) hub.AskOrder {
	switch order {
	case v1.ListAsksRequest_ORDER_UNSPECIFIED, v1.ListAsksRequest_ORDER_NEWEST:
		return hub.AskOrderNewest
	case v1.ListAsksRequest_ORDER_ATTENTION:
		return hub.AskOrderAttention
	case v1.ListAsksRequest_ORDER_DUE:
		return hub.AskOrderDue
	case v1.ListAsksRequest_ORDER_OLDEST:
		return hub.AskOrderOldest
	}
	return hub.AskOrder(order.String()) // an order this build does not know: the hub refuses it
}

// askAttachments lists an ask's attachments, with the source drift check only
// for the owner.
func (h *handler) askAttachments(ctx context.Context, askID string, audience hub.AskAudience) ([]hub.AskAttachment, error) {
	if audience.Owner {
		return h.service.AskAttachments(ctx, askID)
	}
	return h.service.AskAttachmentsUnchecked(ctx, askID)
}

func (h *handler) toProto(ask hub.AskRecord) *v1.Ask {
	options := make([]*v1.AskOption, 0, len(ask.Options)+1)
	for _, option := range ask.Options {
		options = append(options, &v1.AskOption{Key: option.Key, Label: option.Label})
	}
	if ask.Kind != "status" && !containsAskOption(ask.Options, "other") {
		options = append(options, &v1.AskOption{Key: "other", Label: "Other — I have a different answer"})
	}
	return &v1.Ask{
		Id: ask.ID, NotificationId: ask.NotificationID, Question: ask.Question, Title: ask.Title, Options: options,
		Recommended: ask.Recommended, RecommendationReason: ask.RecommendationReason, DefaultAnswer: ask.DefaultAnswer,
		Reversible: ask.Reversible, Urgency: ask.Urgency, ContextUrl: ask.ContextURL,
		BigPicture: ask.BigPicture, TechnicalDetails: ask.TechnicalDetails, Deadline: ask.Deadline,
		State: ask.State, Reason: ask.Reason, Answer: ask.Answer, AnswerLabel: ask.AnswerLabel, Note: ask.Note,
		AnsweredBy: ask.AnsweredBy, AnsweredAt: ask.AnsweredAt, Late: ask.Late, DefaultAppliedAt: ask.DefaultAppliedAt,
		AnsweredByIdentityKind: ask.AnsweredByIdentityKind, AnsweredByIdentitySource: ask.AnsweredByIdentitySource,
		FirstDeliveredAt: ask.FirstDeliveredAt, DefaultEligibleAt: h.service.DefaultEligibleAt(ask),
		Source: ask.Source, SourceEventType: ask.SourceEventType, RequesterRef: ask.RequesterRef,
		ResolvedAt: ask.ResolvedAt, CreatedAt: ask.CreatedAt, UpdatedAt: ask.UpdatedAt,
		ProviderDecisionContext: providerDecisionContextToProto(ask.ProviderDecisionContext),
		SubjectKey:              ask.SubjectKey, LastAssertedAt: ask.LastAssertedAt,
		ReassertWithinSeconds: int32(ask.ReassertWithinSeconds), RevisionCount: int32(ask.RevisionCount),
		SupersededByAskId: ask.SupersededByAskID, AttachmentCount: int32(ask.AttachmentCount),
		AnswerSummary: ask.AnswerSummary, QuestionCount: int32(len(ask.Questions)), OpenContextRequestCount: int32(ask.OpenContextRequestCount),
		CanRequestContext: ask.CanRequestContext(), Correlation: maps.Clone(ask.Correlation),
		Sender:      ask.Sender,
		ClosingSoon: ask.ClosingSoon,
		Packet:      ask.Packet, ApprovalUrl: ask.NativeApprovalURL(),
		Kind: ask.Kind, AnswerChanges: structValue(ask.AnswerChanges), AnswerImages: answerImagesToProto(ask.AnswerImages), AnswerAttachments: answerAttachmentsToProto(ask.AnswerAttachments),
		TemplateName: ask.TemplateName, TemplateData: structValue(ask.TemplateData),
		DecisionFamily: ask.DecisionFamily, DecisionCause: ask.DecisionCause, DecisionKind: ask.DecisionKind,
		QualityGrade: ask.QualityGrade, QualityReport: ask.QualityReport, QualitySummary: ask.QualitySummary,
		Related: relatedToProto(ask.Related), Followthrough: followthroughToProto(ask.Followthrough),
		Unblocks: ask.Unblocks, CostOfWaiting: ask.CostOfWaiting,
	}
}

func structMap(value *structpb.Struct) map[string]any {
	if value == nil {
		return nil
	}
	return value.AsMap()
}

func structValue(value map[string]any) *structpb.Struct {
	if len(value) == 0 {
		return nil
	}
	result, err := structpb.NewStruct(value)
	if err != nil {
		return nil
	}
	return result
}

func answerImagesFromProto(images []*v1.AnswerImage) []hub.AnswerImage {
	out := make([]hub.AnswerImage, 0, len(images))
	for _, image := range images {
		if image != nil {
			out = append(out, hub.AnswerImage{Name: image.GetName(), SHA256: image.GetSha256(), MediaType: image.GetMediaType(), SizeBytes: image.GetSizeBytes()})
		}
	}
	return out
}

func answerImagesToProto(images []hub.AnswerImage) []*v1.AnswerImage {
	out := make([]*v1.AnswerImage, 0, len(images))
	for _, image := range images {
		out = append(out, &v1.AnswerImage{Name: image.Name, Sha256: image.SHA256, MediaType: image.MediaType, SizeBytes: image.SizeBytes})
	}
	return out
}

func answerAttachmentsFromProto(files []*v1.AnswerAttachment) []hub.AnswerAttachment {
	out := make([]hub.AnswerAttachment, 0, len(files))
	for _, file := range files {
		if file != nil {
			out = append(out, hub.AnswerAttachment{Name: file.GetName(), SHA256: file.GetSha256(), MediaType: file.GetMediaType(), SizeBytes: file.GetSizeBytes()})
		}
	}
	return out
}

func answerAttachmentsToProto(files []hub.AnswerAttachment) []*v1.AnswerAttachment {
	out := make([]*v1.AnswerAttachment, 0, len(files))
	for _, file := range files {
		out = append(out, &v1.AnswerAttachment{Name: file.Name, Sha256: file.SHA256, MediaType: file.MediaType, SizeBytes: file.SizeBytes})
	}
	return out
}

func recurrenceToProto(recurrence hub.SubjectRecurrence) *v1.SubjectRecurrence {
	out := &v1.SubjectRecurrence{Count: int32(recurrence.Count), Dates: recurrence.Dates}
	for _, previous := range recurrence.PreviousAnswers {
		out.PreviousAnswers = append(out.PreviousAnswers, &v1.HistoricalAnswer{Answer: previous.Answer, Note: previous.Note, AnsweredAt: previous.AnsweredAt, Changes: structValue(previous.Changes)})
	}
	return out
}

func containsAskOption(options []hub.AskOption, key string) bool {
	for _, option := range options {
		if option.Key == key {
			return true
		}
	}
	return false
}

func providerDecisionContextFromProto(value *v1.ProviderDecisionContext) *hub.ProviderDecisionContext {
	if value == nil {
		return nil
	}
	scope := ""
	switch value.GetScope() {
	case v1.ProviderDecisionContext_SCOPE_PROVIDER_RULE:
		scope = hub.ProviderDecisionScopeRule
	case v1.ProviderDecisionContext_SCOPE_PROVIDER_AGREEMENT:
		scope = hub.ProviderDecisionScopeAgreement
	}
	return &hub.ProviderDecisionContext{SchemaVersion: value.GetSchemaVersion(), CanonicalTermsSHA256: value.GetCanonicalTermsSha256(), Scope: scope}
}

func providerDecisionContextToProto(value *hub.ProviderDecisionContext) *v1.ProviderDecisionContext {
	if value == nil {
		return nil
	}
	scope := v1.ProviderDecisionContext_SCOPE_UNSPECIFIED
	switch value.Scope {
	case hub.ProviderDecisionScopeRule:
		scope = v1.ProviderDecisionContext_SCOPE_PROVIDER_RULE
	case hub.ProviderDecisionScopeAgreement:
		scope = v1.ProviderDecisionContext_SCOPE_PROVIDER_AGREEMENT
	}
	return &v1.ProviderDecisionContext{SchemaVersion: value.SchemaVersion, CanonicalTermsSha256: value.CanonicalTermsSHA256, Scope: scope}
}

func connectError(err error) error {
	switch {
	case errors.Is(err, hub.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, hub.ErrAskResolved), errors.Is(err, hub.ErrAskWithdrawn), errors.Is(err, hub.ErrAskStale), errors.Is(err, hub.ErrDecisionRevisionConflict):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, hub.ErrRelatedDecision):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, hub.ErrAskChanged):
		// FailedPrecondition like a resolved ask, but a client tells them apart by
		// this reason: the ask is still open and should be reloaded, keeping the
		// operator's draft.
		changed := connect.NewError(connect.CodeFailedPrecondition, err)
		changed.Meta().Set(ErrorReasonHeader, ReasonAskChanged)
		return changed
	case errors.Is(err, hub.ErrContextNotAccepted):
		refused := connect.NewError(connect.CodeFailedPrecondition, err)
		refused.Meta().Set(ErrorReasonHeader, ReasonContextNotAccepted)
		return refused
	case errors.Is(err, hub.ErrAskRefused):
		refused := connect.NewError(connect.CodeFailedPrecondition, err)
		refused.Meta().Set(ErrorReasonHeader, ReasonAskRefused)
		return refused
	case errors.Is(err, hub.ErrContextRequestLimit):
		return connect.NewError(connect.CodeResourceExhausted, err)
	case errors.Is(err, hub.ErrAskIdentityConflict):
		return connect.NewError(connect.CodeAlreadyExists, err)
	case errors.Is(err, hub.ErrInvalidArgument):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "conversations_stat_attachment_blobs", Path: connectv1.ConversationsServiceStatAttachmentBlobsProcedure, Method: http.MethodPost, Summary: "Report which attachment content hashes the Hub does not hold (verified sender)", Category: "conversations"},
	{ID: "conversations_put_attachment_blob", Path: connectv1.ConversationsServicePutAttachmentBlobProcedure, Method: http.MethodPost, Summary: "Store one attachment file under its SHA-256 (at most 64 MiB; sender identity is optional)", Category: "conversations"},
	attachmentContentEndpoint,
	answerImageContentEndpoint,
	answerFileContentEndpoint,
	{ID: "conversations_linked_artifact_metadata", Path: linkedArtifactMetadataPath, Method: http.MethodGet, Summary: "Resolve one decision-linked artifact inside an allowed local root", Category: "conversations", RESTException: linkedArtifactRESTException},
	{ID: "conversations_linked_artifact_content", Path: linkedArtifactContentPath, Method: http.MethodGet, Summary: "Stream one owner-authorized decision-linked artifact", Category: "conversations", RESTException: linkedArtifactRESTException},
	{ID: "conversations_list_native_consents", Path: connectv1.ConversationsServiceListNativeConsentsProcedure, Method: http.MethodPost, Summary: "Read source-owned native consent without approving or creating an ask", Category: "conversations"},
	{ID: "conversations_ask", Path: connectv1.ConversationsServiceAskProcedure, Method: http.MethodPost, Summary: "Ask a question and persist its deadline", Category: "conversations"},
	{ID: "conversations_check_ask", Path: connectv1.ConversationsServiceCheckAskProcedure, Method: http.MethodPost, Summary: "Run the packet check and preview the notification for an ask without storing it", Category: "conversations"},
	{ID: "conversations_answer", Path: connectv1.ConversationsServiceAnswerProcedure, Method: http.MethodPost, Summary: "Record an allowed answer", Category: "conversations"},
	{ID: "conversations_answer_root_with_password", Path: connectv1.ConversationsServiceAnswerRootWithPasswordProcedure, Method: http.MethodPost, Summary: "Authenticate and answer one root decision", Category: "conversations"},
	{ID: "conversations_mark_read", Path: connectv1.ConversationsServiceMarkReadProcedure, Method: http.MethodPost, Summary: "Mark a status update as read without sending steering", Category: "conversations"},
	{ID: "conversations_dismiss", Path: connectv1.ConversationsServiceDismissProcedure, Method: http.MethodPost, Summary: "Dismiss an ask without answering", Category: "conversations"},
	{ID: "conversations_withdraw", Path: connectv1.ConversationsServiceWithdrawProcedure, Method: http.MethodPost, Summary: "Withdraw a direct ask you created", Category: "conversations"},
	{ID: "conversations_wait", Path: connectv1.ConversationsServiceWaitProcedure, Method: http.MethodPost, Summary: "Block until an ask is resolved or the caller's wait runs out", Category: "conversations"},
	{ID: "conversations_request_context", Path: connectv1.ConversationsServiceRequestContextProcedure, Method: http.MethodPost, Summary: "Ask the requester for more context on an open ask", Category: "conversations"},
	{ID: "conversations_get_ask", Path: connectv1.ConversationsServiceGetAskProcedure, Method: http.MethodPost, Summary: "Read one ask without changing it", Category: "conversations"},
	{ID: "conversations_list_asks", Path: connectv1.ConversationsServiceListAsksProcedure, Method: http.MethodPost, Summary: "List the owner's asks, optionally only open ones", Category: "conversations"},
	{ID: "conversations_list_pending_decision_references", Path: connectv1.ConversationsServiceListPendingDecisionReferencesProcedure, Method: http.MethodPost, Summary: "List safe references to pending owner decisions", Category: "conversations"},
	{ID: "conversations_read_pending_decision_reference", Path: connectv1.ConversationsServiceReadPendingDecisionReferenceProcedure, Method: http.MethodPost, Summary: "Read one safe pending owner decision reference", Category: "conversations"},
}

var linkedArtifactRESTException = &module.RESTException{
	Reason: module.RESTReasonBrowserSurface,
	Note:   "The read-only file viewer uses this route for metadata and byte-range content. Both require the verified owner and an ask addressed to that owner. File access is restricted to configured roots and 64 MiB; missing optional roots return 503.",
	ProtoPayloads: &module.RESTProtoPayloads{
		Request:  module.RESTPayload{Transport: "none", Conformance: "none"},
		Response: module.RESTPayload{Transport: "none", Conformance: "external_shape"},
		Error:    module.RESTPayload{Transport: "none", Conformance: "external_shape"},
	},
}
