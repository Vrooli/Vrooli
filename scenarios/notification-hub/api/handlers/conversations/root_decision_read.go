package conversations

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"notification-hub/internal/hub"
	"notification-hub/internal/identity"
)

type rootReadRequest struct {
	AskID string `json:"askId"`
	enrollmentCredentials
}

// rootDecisionRead discloses only an eligible root ask after fresh account
// proof and current binding checks. It issues no session or reusable token.
func (h *handler) rootDecisionRead(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" || h.stepUp == nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	var input rootReadRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&input); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	record, err := h.service.GetAsk(r.Context(), input.AskID)
	if err != nil || !rootPasswordDecisionEligible(record, record.RevisionCount, time.Now()) || record.Recipient != h.service.ResolveRecipient(r.Context()) {
		http.Error(w, "decision unavailable", http.StatusNotFound)
		return
	}
	revision := int64(record.RevisionCount)
	snapshot, err := h.service.RootEnrollmentSnapshot(r.Context(), record.Recipient)
	if err != nil {
		http.Error(w, "enrollment unavailable", http.StatusServiceUnavailable)
		return
	}
	if input.MFAChallenge != "" {
		c, err := unwrapChallengeWithGrants(input.MFAChallenge, "root-read:"+record.ID, record.Recipient)
		if err != nil || c.Generation != revision {
			http.Error(w, "decision unavailable", http.StatusUnauthorized)
			return
		}
		input.MFAChallenge = c.Challenge
		snapshot = c.Grants
	}
	data, _ := json.Marshal([]string{"notification-hub/root-decision-read/v1", record.ID, record.Recipient, strconv.FormatInt(revision, 10)})
	hash := sha256.Sum256(data)
	proof, err := h.stepUp.VerifyDecision(r.Context(), input.Email, input.Password, input.MFAChallenge, input.TOTPCode, input.RecoveryCode, hex.EncodeToString(hash[:]))
	if err != nil {
		if errors.Is(err, identity.ErrRateLimited) {
			http.Error(w, "try again later", http.StatusTooManyRequests)
			return
		}
		if errors.Is(err, identity.ErrAuthUnavailable) {
			http.Error(w, "sign-in unavailable", http.StatusServiceUnavailable)
			return
		}
		http.Error(w, "sign-in failed", http.StatusUnauthorized)
		return
	}
	if proof.MFARequired {
		challenge, err := wrapChallengeWithGrants(proof.MFAChallenge, "root-read:"+record.ID, record.Recipient, revision, snapshot)
		if err != nil {
			http.Error(w, "sign-in unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"mfaRequired": true, "mfaChallenge": challenge})
		return
	}
	bindings, err := identity.OwnerBindings()
	if err != nil {
		http.Error(w, "owner policy unavailable", http.StatusServiceUnavailable)
		return
	}
	allowed, err := h.checkRootPasswordAllowed(r.Context(), bindings, strings.TrimSpace(proof.Subject), record.Recipient)
	if err != nil {
		http.Error(w, "enrollment unavailable", http.StatusServiceUnavailable)
		return
	}
	if !allowed || proof.Subject == "" {
		http.Error(w, "sign-in failed", http.StatusUnauthorized)
		return
	}
	record, err = h.service.GetAsk(r.Context(), input.AskID)
	if err != nil || record.Recipient != h.service.ResolveRecipient(r.Context()) || !rootPasswordDecisionEligible(record, int(revision), time.Now()) {
		http.Error(w, "decision unavailable", http.StatusConflict)
		return
	}
	// This narrow read does not grant access to protected attachments or the
	// general answer form. Refuse rather than show a partial decision.
	attachments, err := h.service.AskAttachmentsUnchecked(r.Context(), record.ID)
	if err != nil {
		http.Error(w, "decision unavailable", http.StatusServiceUnavailable)
		return
	}
	if !rootReadComplete(record, len(attachments)) {
		http.Error(w, "decision requires full owner access", http.StatusConflict)
		return
	}
	ask, err := protojson.Marshal(h.toProto(record))
	if err != nil {
		http.Error(w, "decision unavailable", http.StatusServiceUnavailable)
		return
	}
	// This final durable check is the disclosure's linearization point. A
	// revocation that commits first is observed; one that commits afterward is
	// ordered after this read, even if response bytes arrive later.
	if h.beforeRootReadDisclosure != nil {
		h.beforeRootReadDisclosure()
	}
	if !rootPasswordOwnerAllowed(bindings, proof.Subject, record.Recipient) {
		active, err := h.service.RootEnrollmentActive(r.Context(), record.Recipient, proof.Subject, eligibleRootGrants(bindings, record.Recipient, snapshot))
		if err != nil {
			http.Error(w, "enrollment unavailable", http.StatusServiceUnavailable)
			return
		}
		if !active {
			http.Error(w, "sign-in failed", http.StatusUnauthorized)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(append(append([]byte(`{"mfaRequired":false,"ask":`), ask...), '}'))
}

func rootReadComplete(record hub.AskRecord, attachmentCount int) bool {
	return attachmentCount == 0 && record.AttachmentCount == 0 && len(record.Questions) == 0
}
