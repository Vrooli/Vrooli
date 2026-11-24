package conversations

import (
	"bytes"
	"context"
	"errors"
	"mime"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"

	"notification-hub/internal/hub"
	"notification-hub/internal/identity"
	"notification-hub/internal/module"
)

// Decision-packet attachments: the blob upload RPCs, the proto mapping of
// attachments and packet warnings, and the content route the browser opens
// files through. Rules: docs/concepts/DECISION_PACKETS.md.

const (
	// attachmentContentPath serves one stored file of an ask to its recipient.
	attachmentContentPath      = "/api/v1/asks/{ask_id}/attachments/{name}"
	answerImageContentPath     = "/api/v1/asks/{ask_id}/answer-images/{name}"
	answerFileContentPath      = "/api/v1/asks/{ask_id}/answer-files/{name}"
	linkedArtifactMetadataPath = "/api/v1/asks/{ask_id}/linked-artifact/metadata"
	linkedArtifactContentPath  = "/api/v1/asks/{ask_id}/linked-artifact/content"
	// putMessageLimit is what the upload RPC may receive: a 64 MiB file, which
	// the JSON codec carries as base64 (4/3 larger), plus its small envelope.
	putMessageLimit = hub.MaxAttachmentBlobBytes/3*4 + 64<<10
	// otherMessageLimit bounds every other procedure of the service. Connect
	// applies a read limit to the message after decompression too, so a small
	// gzip body cannot expand past it.
	otherMessageLimit = 8 << 20
	// maxConcurrentUploads bounds the memory of file-sized uploads in flight: each
	// holds its request body, the decoded file and a copy for the write.
	maxConcurrentUploads = 2
	// uploadSlotWait is how long an upload waits for a free slot before the
	// Hub answers that it is busy.
	uploadSlotWait = 30 * time.Second
	// transferTimeout extends the server's per-request deadlines for one upload
	// or file download, which can outlast the defaults on a slow link.
	transferTimeout = 10 * time.Minute
)

// attachmentContentCSP stops a file opened directly (not through the viewer)
// from running as a page of the Hub's origin.
const attachmentContentCSP = "sandbox; default-src 'none'; img-src data:; media-src data:; style-src 'unsafe-inline'"

// connectHandlers builds the service's Connect handlers. The upload procedure is
// the only one that may receive a file-sized message, and it is served by its
// own handler so that limit is not shared with the rest: Connect's read limit is
// per handler, and it is the limit on the decompressed message.
func (h *handler) connectHandlers() (path string, general, upload http.Handler) {
	path, general = connectv1.NewConversationsServiceHandler(h, connect.WithReadMaxBytes(otherMessageLimit))
	_, upload = connectv1.NewConversationsServiceHandler(h, connect.WithReadMaxBytes(putMessageLimit))
	return path, general, h.gateUploads(upload)
}

// gateUploads admits an upload before its body is read when a slot is free.
// Sending a decision packet does not require an identity; reading and answering
// the resulting ask still require the configured owner identity. Reading is
// what costs memory, so the slot check happens before Connect decodes the body.
func (h *handler) gateUploads(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wait := time.NewTimer(h.uploadWait)
		defer wait.Stop()
		select {
		case h.uploads <- struct{}{}:
			defer func() { <-h.uploads }()
		case <-wait.C:
			w.Header().Set("Retry-After", "5")
			http.Error(w, "the Hub is receiving other uploads; try again shortly", http.StatusServiceUnavailable)
			return
		case <-r.Context().Done():
			return
		}
		// The server's write deadline starts when the request headers are read, so
		// a slow upload needs both extended to answer at the end.
		controller := http.NewResponseController(w)
		_ = controller.SetReadDeadline(time.Now().Add(transferTimeout))
		_ = controller.SetWriteDeadline(time.Now().Add(transferTimeout))
		next.ServeHTTP(w, r)
	})
}

// Stat is a read of the Hub's content inventory, so it is limited to verified
// producers. Anonymous senders can skip the lookup and upload content directly.
func (h *handler) StatAttachmentBlobs(ctx context.Context, req *connect.Request[v1.StatAttachmentBlobsRequest]) (*connect.Response[v1.StatAttachmentBlobsResponse], error) {
	if _, err := h.producer(ctx, req.Header()); err != nil {
		return nil, err
	}
	missing, err := h.service.StatAttachmentBlobs(ctx, req.Msg.GetSha256())
	if err != nil {
		return nil, attachmentError(err)
	}
	return connect.NewResponse(&v1.StatAttachmentBlobsResponse{Missing: missing}), nil
}

func (h *handler) PutAttachmentBlob(ctx context.Context, req *connect.Request[v1.PutAttachmentBlobRequest]) (*connect.Response[v1.PutAttachmentBlobResponse], error) {
	result, err := h.service.PutAttachmentBlob(ctx, req.Msg.GetSha256(), req.Msg.GetMediaType(), req.Msg.GetContent())
	if err != nil {
		return nil, attachmentError(err)
	}
	return connect.NewResponse(&v1.PutAttachmentBlobResponse{Sha256: result.SHA256, SizeBytes: result.SizeBytes, Existed: result.Existed}), nil
}

// attachmentError maps attachment failures; everything else is connectError's.
func attachmentError(err error) error {
	switch {
	case errors.Is(err, hub.ErrAttachmentsUnavailable):
		return connect.NewError(connect.CodeUnavailable, err)
	case errors.Is(err, hub.ErrBlobTooLarge):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, hub.ErrBlobQuota):
		return connect.NewError(connect.CodeResourceExhausted, err)
	default:
		return connectError(err)
	}
}

// serveAttachment serves one stored file of an ask to its recipient: the
// attachment itself, or ?path= inside a folder. Owner access keeps the existing
// rule. The portfolio steward may also read bytes from an exact ask with a
// stored continuity attestation, because its authenticated producer needs to
// inspect and revise that same page. No other service audience gets this path.
func (h *handler) serveAttachment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entryPath := r.URL.Query().Get("path")
	owner, ownerErr := identity.Subject(r.Context(), r.Header, h.verifier)
	var content hub.AttachmentContent
	var err error
	if ownerErr == nil && owner != "" {
		content, err = h.service.OpenAskAttachment(r.Context(), vars["ask_id"], owner, vars["name"], entryPath)
	} else {
		audience, audienceErr := identity.ReaderFor(r.Context(), r.Header, h.verifier, h.services)
		if audienceErr != nil {
			http.Error(w, "verified owner or portfolio steward identity is required", http.StatusUnauthorized)
			return
		}
		if audience.DirectSubjectSource != "portfolio-steward" {
			notFound(w)
			return
		}
		recipient := h.service.ResolveRecipient(r.Context())
		if recipient == "" {
			notFound(w)
			return
		}
		content, err = h.service.OpenAskAttachmentFor(r.Context(), vars["ask_id"], recipient, vars["name"], entryPath, audience)
	}
	switch {
	case errors.Is(err, hub.ErrNotFound):
		notFound(w)
		return
	case errors.Is(err, hub.ErrAttachmentsUnavailable):
		http.Error(w, "attachments are unavailable", http.StatusServiceUnavailable)
		return
	case err != nil:
		http.Error(w, "attachment could not be read", http.StatusInternalServerError)
		return
	}
	defer content.Content.Close()

	header := w.Header()
	header.Set("Content-Type", content.MediaType)
	header.Set("X-Content-Type-Options", "nosniff")
	// The URL names the ask and attachment, not the bytes: a revision can attach
	// different content under the same name. So the response is never reused
	// without asking, and the content address is the validator (http.ServeContent
	// answers a matching If-None-Match with 304 and no body).
	header.Set("ETag", `"sha256-`+content.SHA256+`"`)
	header.Set("Cache-Control", "private, no-cache")
	header.Set("Content-Security-Policy", attachmentContentCSP)
	disposition := mime.FormatMediaType("inline", map[string]string{"filename": content.FileName})
	if disposition == "" {
		disposition = "inline"
	}
	header.Set("Content-Disposition", disposition)
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(transferTimeout))
	http.ServeContent(w, r, "", time.Time{}, content.Content)
}

// serveAnswerImage streams an image attached to the owner's sent answer.
func (h *handler) serveAnswerImage(w http.ResponseWriter, r *http.Request) {
	owner, err := identity.Subject(r.Context(), r.Header, h.verifier)
	if err != nil || owner == "" {
		http.Error(w, "verified owner identity is required", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	image, content, err := h.service.AnswerImage(r.Context(), vars["ask_id"], owner, vars["name"])
	if errors.Is(err, hub.ErrNotFound) {
		notFound(w)
		return
	}
	if err != nil {
		http.Error(w, "answer image could not be read", http.StatusInternalServerError)
		return
	}
	header := w.Header()
	header.Set("Content-Type", image.MediaType)
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Cache-Control", "private, no-cache")
	header.Set("ETag", `"sha256-`+image.SHA256+`"`)
	header.Set("Content-Security-Policy", attachmentContentCSP)
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(transferTimeout))
	http.ServeContent(w, r, image.Name, time.Time{}, bytes.NewReader(content))
}

func (h *handler) serveAnswerFile(w http.ResponseWriter, r *http.Request) {
	owner, err := identity.Subject(r.Context(), r.Header, h.verifier)
	if err != nil || owner == "" {
		http.Error(w, "verified owner identity is required", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	file, content, err := h.service.AnswerFile(r.Context(), vars["ask_id"], owner, vars["name"])
	if errors.Is(err, hub.ErrNotFound) {
		notFound(w)
		return
	}
	if err != nil {
		http.Error(w, "answer file could not be read", http.StatusInternalServerError)
		return
	}
	header := w.Header()
	header.Set("Content-Type", file.MediaType)
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Cache-Control", "private, no-cache")
	header.Set("ETag", `"sha256-`+file.SHA256+`"`)
	header.Set("Content-Security-Policy", attachmentContentCSP)
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": file.Name})
	if disposition != "" {
		header.Set("Content-Disposition", disposition)
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(transferTimeout))
	http.ServeContent(w, r, file.Name, time.Time{}, bytes.NewReader(content))
}

func notFound(w http.ResponseWriter) { http.Error(w, "not found", http.StatusNotFound) }

// attachmentContentEndpoint documents the content route. It serves raw bytes
// with range support, which a Connect procedure cannot.
var attachmentContentEndpoint = module.EndpointDescriptor{
	ID:          "conversations_attachment_content",
	Path:        attachmentContentPath,
	Method:      http.MethodGet,
	Summary:     "Open one attached file of an ask",
	Description: "Streams the Hub's stored copy of an attachment (or ?path= inside a folder attachment) to the ask's recipient, with byte ranges for video and audio. Requires a verified owner identity or the registered portfolio-steward sender with an exact stored continuity attestation (401 without a supported identity). Unknown asks, foreign asks, attachments and paths are all 404.",
	Category:    "conversations",
	RESTException: &module.RESTException{
		Reason: module.RESTReasonBrowserSurface,
		Note:   "Raw file bytes with Range support for the Hub's viewer; a Connect procedure cannot stream or seek. Metadata for the same attachments is the proto AskAttachment on GetAsk. The portfolio-steward exception is scoped to an exact per-ask source attestation; other service identities remain denied. Errors are plain-text HTTP statuses (401, 404).",
		ProtoPayloads: &module.RESTProtoPayloads{
			Request:  module.RESTPayload{Transport: "none", Conformance: "none"},
			Response: module.RESTPayload{Transport: "none", Conformance: "none"},
			Error:    module.RESTPayload{Transport: "none", Conformance: "none"},
		},
	},
}

var answerImageContentEndpoint = module.EndpointDescriptor{
	ID: "conversations_answer_image_content", Path: answerImageContentPath, Method: http.MethodGet,
	Summary: "Read one image sent with an answer", Category: "conversations",
	RESTException: &module.RESTException{Reason: module.RESTReasonBrowserSurface, Note: "Streams one image attached to the owner's stored answer. Requires verified owner identity; unknown or foreign answers return 404."},
}

var answerFileContentEndpoint = module.EndpointDescriptor{
	ID: "conversations_answer_file_content", Path: answerFileContentPath, Method: http.MethodGet,
	Summary: "Download one file sent with an answer", Category: "conversations",
	RESTException: &module.RESTException{Reason: module.RESTReasonBrowserSurface, Note: "Downloads one file attached to the owner's stored answer. Requires verified owner identity; unknown or foreign answers return 404."},
}

func attachmentsFromProto(in []*v1.AskAttachment) []hub.AskAttachment {
	if len(in) == 0 {
		return nil
	}
	out := make([]hub.AskAttachment, 0, len(in))
	for _, a := range in {
		item := hub.AskAttachment{
			Name: a.GetName(), Label: a.GetLabel(), SourcePath: a.GetSourcePath(), SourceHost: a.GetSourceHost(),
			SHA256: a.GetSha256(), SizeBytes: a.GetSizeBytes(), MediaType: a.GetMediaType(), Truncated: a.GetTruncated(),
		}
		switch a.GetKind() {
		case v1.AskAttachment_KIND_FILE:
			item.Kind = hub.AttachmentFile
		case v1.AskAttachment_KIND_FOLDER:
			item.Kind = hub.AttachmentFolder
		}
		switch a.GetRole() {
		case v1.AskAttachment_ROLE_EVIDENCE:
			item.Role = hub.AttachmentRoleEvidence
		case v1.AskAttachment_ROLE_EXPLAINER:
			item.Role = hub.AttachmentRoleExplainer
		}
		for _, e := range a.GetEntries() {
			item.Entries = append(item.Entries, hub.AskAttachmentEntry{Path: e.GetPath(), SHA256: e.GetSha256(), SizeBytes: e.GetSizeBytes(), MediaType: e.GetMediaType()})
		}
		out = append(out, item)
	}
	return out
}

func attachmentsToProto(in []hub.AskAttachment) []*v1.AskAttachment {
	out := make([]*v1.AskAttachment, 0, len(in))
	for _, a := range in {
		item := &v1.AskAttachment{
			Name: a.Name, Label: a.Label, SourcePath: a.SourcePath, SourceHost: a.SourceHost,
			Sha256: a.SHA256, SizeBytes: a.SizeBytes, MediaType: a.MediaType, Truncated: a.Truncated,
			PreviewKind: a.PreviewKind, SourceStatus: a.SourceStatus, SourceCheckedAt: a.SourceCheckedAt,
		}
		switch a.Kind {
		case hub.AttachmentFile:
			item.Kind = v1.AskAttachment_KIND_FILE
		case hub.AttachmentFolder:
			item.Kind = v1.AskAttachment_KIND_FOLDER
		}
		switch a.Role {
		case hub.AttachmentRoleEvidence:
			item.Role = v1.AskAttachment_ROLE_EVIDENCE
		case hub.AttachmentRoleExplainer:
			item.Role = v1.AskAttachment_ROLE_EXPLAINER
		}
		for _, e := range a.Entries {
			item.Entries = append(item.Entries, &v1.AskAttachmentEntry{Path: e.Path, Sha256: e.SHA256, SizeBytes: e.SizeBytes, MediaType: e.MediaType, PreviewKind: e.PreviewKind})
		}
		out = append(out, item)
	}
	return out
}

func warningsToProto(in []hub.PacketWarning) []*v1.PacketWarning {
	if len(in) == 0 {
		return nil
	}
	out := make([]*v1.PacketWarning, 0, len(in))
	for _, w := range in {
		out = append(out, &v1.PacketWarning{Code: w.Code, Field: w.Field, Message: w.Message, Severity: w.Severity, CarriedOver: w.CarriedOver})
	}
	return out
}
