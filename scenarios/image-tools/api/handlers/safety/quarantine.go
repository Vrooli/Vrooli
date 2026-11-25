package safety

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"google.golang.org/protobuf/types/known/timestamppb"

	safetyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/safety"
	internalquarantine "image-tools/internal/quarantine"
)

// ListQuarantinedOutputs returns the refused outputs retained for bounded admin
// review. Local deployments are ungated; a public deployment requires the
// configured admin token.
func (h *connectHandler) ListQuarantinedOutputs(ctx context.Context, req *connect.Request[safetyv1.ListQuarantinedOutputsRequest]) (*connect.Response[safetyv1.ListQuarantinedOutputsResponse], error) {
	if h.quarantine == nil || !h.quarantine.Enabled() {
		return connect.NewResponse(&safetyv1.ListQuarantinedOutputsResponse{}), nil
	}
	if err := h.authorizeQuarantine(req.Header()); err != nil {
		return nil, err
	}
	items, total, err := h.quarantine.List(ctx, int(req.Msg.GetLimit()), int(req.Msg.GetOffset()))
	if err != nil {
		return nil, h.feedbackError(err, "ListQuarantinedOutputs")
	}
	resp := &safetyv1.ListQuarantinedOutputsResponse{Total: int32(total)}
	for _, item := range items {
		out := &safetyv1.QuarantinedOutput{
			Id:           item.ID,
			JobId:        item.JobID,
			OriginalRef:  item.OriginalRef,
			ReasonCode:   item.ReasonCode,
			SafetyReview: item.SafetyReview,
			Score:        item.Score,
			EvidenceJson: item.Evidence,
			FindingJson:  item.Finding,
			Mime:         item.MIME,
		}
		if !item.CreatedAt.IsZero() {
			out.CreatedAt = timestamppb.New(item.CreatedAt)
		}
		if !item.ExpiresAt.IsZero() {
			out.ExpiresAt = timestamppb.New(item.ExpiresAt)
		}
		resp.Outputs = append(resp.Outputs, out)
	}
	return connect.NewResponse(resp), nil
}

// quarantineImage serves one quarantined blob. It is deliberately separate from
// /api/v1/blobs/: the ordinary route refuses the quarantine namespace, so a
// refused output is only reachable through this bounded review surface.
func (h *connectHandler) quarantineImage(w http.ResponseWriter, r *http.Request) {
	if h.quarantine == nil || !h.quarantine.Enabled() {
		http.NotFound(w, r)
		return
	}
	if !h.quarantineAuthorized(r.Header) {
		http.Error(w, "admin review is not authorized", http.StatusForbidden)
		return
	}
	id, err := strconv.ParseInt(strings.TrimSpace(mux.Vars(r)["id"]), 10, 64)
	if err != nil || id <= 0 {
		http.NotFound(w, r)
		return
	}
	rc, item, err := h.quarantine.Open(r.Context(), id)
	switch {
	case errors.Is(err, internalquarantine.ErrExpired):
		http.Error(w, "quarantined output has expired", http.StatusGone)
		return
	case err != nil:
		http.NotFound(w, r)
		return
	}
	defer func() { _ = rc.Close() }()

	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	if mime := strings.TrimSpace(item.MIME); mime != "" {
		w.Header().Set("Content-Type", mime)
	}
	if r.URL.Query().Get("download") != "" {
		filename := fmt.Sprintf("quarantine-%d%s", item.ID, path.Ext(item.Key))
		w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	}
	if _, err := io.Copy(w, rc); err != nil {
		h.logf("quarantine image %d copy: %v", id, err)
	}
}

func (h *connectHandler) authorizeQuarantine(header http.Header) error {
	if !h.requireAdmin {
		return nil
	}
	if strings.TrimSpace(h.adminToken) == "" {
		return connect.NewError(connect.CodeFailedPrecondition, errors.New("quarantine review on a public deployment requires IMAGE_TOOLS_SAFETY_QUARANTINE_TOKEN"))
	}
	if !h.quarantineAuthorized(header) {
		return connect.NewError(connect.CodePermissionDenied, errors.New("admin review token is missing or incorrect"))
	}
	return nil
}

func (h *connectHandler) quarantineAuthorized(header http.Header) bool {
	if !h.requireAdmin {
		return true
	}
	if strings.TrimSpace(h.adminToken) == "" {
		return false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header.Get("Authorization"), "Bearer "))
	if token == "" {
		token = strings.TrimSpace(header.Get("X-Image-Tools-Admin-Token"))
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(h.adminToken)) == 1
}
