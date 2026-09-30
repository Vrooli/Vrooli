// Package protocol contains the bounded native-messaging contract used by
// the Secrets Manager browser host. It deliberately contains no browser or
// shell dependencies so the same checks run on Linux, macOS, and Windows.
package protocol

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"sync"
	"time"
)

const (
	ProtocolVersion = 1
	ProtocolName    = "vrooli.secrets.native/1"
	MaxFrameBytes   = 64 * 1024
	MaxPayloadBytes = 32 * 1024
	MaxLifetime     = 2 * time.Minute
)

var (
	errMalformed     = errors.New("malformed native-messaging request")
	errUnauthorized  = errors.New("native-messaging principal is not authorized")
	errReplay        = errors.New("native-messaging request nonce was already used")
	errExpired       = errors.New("native-messaging request expired")
	errScope         = errors.New("native-messaging request is outside its bound scope")
	errUnavailable   = errors.New("native-messaging authority adapter is unavailable")
	principalPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)
	requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)
)

type Scope struct {
	Origin     string `json:"origin"`
	TabID      string `json:"tab_id"`
	FrameID    string `json:"frame_id"`
	DocumentID string `json:"document_id"`
	ItemID     string `json:"item_id"`
	Revision   int    `json:"item_revision"`
}

type Request struct {
	Protocol    string            `json:"protocol"`
	RequestID   string            `json:"request_id"`
	Operation   string            `json:"operation"`
	ExtensionID string            `json:"extension_id"`
	SessionID   string            `json:"session_id,omitempty"`
	Nonce       string            `json:"nonce"`
	ExpiresAt   int64             `json:"expires_at"`
	Scope       Scope             `json:"scope"`
	Payload     map[string]string `json:"payload,omitempty"`
}

type Response struct {
	Protocol  string            `json:"protocol"`
	RequestID string            `json:"request_id"`
	OK        bool              `json:"ok"`
	Code      string            `json:"code,omitempty"`
	Message   string            `json:"message,omitempty"`
	SessionID string            `json:"session_id,omitempty"`
	Scope     *Scope            `json:"scope,omitempty"`
	Payload   map[string]string `json:"payload,omitempty"`
	Metadata  []MetadataRecord  `json:"metadata,omitempty"`
}

type FillRequest struct {
	ExtensionID string
	SessionID   string
	WorkspaceID string
	GrantID     string
	Field       string
	Scope       Scope
}

type EnrollmentRequest struct {
	ExtensionID string
	WorkspaceID string
	Origin      string
}

type UnlockRequest struct {
	ExtensionID string
	SessionID   string
	WorkspaceID string
	GrantID     string
	Field       string
	Scope       Scope
}

type MetadataRequest struct {
	ExtensionID string
	WorkspaceID string
	Scope       Scope
}

type MetadataRecord struct {
	GrantID  string `json:"grant_id"`
	ItemID   string `json:"item_id"`
	VaultID  string `json:"vault_id,omitempty"`
	Name     string `json:"name"`
	Username string `json:"username,omitempty"`
	URI      string `json:"uri,omitempty"`
	Type     string `json:"type"`
	Revision int    `json:"revision"`
}

type SaveRequest struct {
	ExtensionID string
	WorkspaceID string
	VaultID     string
	Name        string
	Username    string
	URI         string
	Password    string
	TOTP        string
	Scope       Scope
}

type UpdateRequest struct {
	SaveRequest
	ItemID   string
	Revision int
}

type Authority interface {
	Enroll(context.Context, EnrollmentRequest, string) error
	Metadata(context.Context, MetadataRequest, string) ([]MetadataRecord, error)
	Save(context.Context, SaveRequest, string) (MetadataRecord, error)
	Update(context.Context, UpdateRequest, string) (MetadataRecord, error)
	Unlock(context.Context, UnlockRequest, string) error
	Fill(context.Context, FillRequest) (string, error)
	Revoke(context.Context, FillRequest) error
}

var (
	ErrAuthorityUnavailable = errUnavailable
	ErrUnauthorized         = errUnauthorized
)

type unavailableAuthority struct{}

func (unavailableAuthority) Enroll(context.Context, EnrollmentRequest, string) error {
	return errUnavailable
}
func (unavailableAuthority) Metadata(context.Context, MetadataRequest, string) ([]MetadataRecord, error) {
	return nil, errUnavailable
}
func (unavailableAuthority) Save(context.Context, SaveRequest, string) (MetadataRecord, error) {
	return MetadataRecord{}, errUnavailable
}
func (unavailableAuthority) Update(context.Context, UpdateRequest, string) (MetadataRecord, error) {
	return MetadataRecord{}, errUnavailable
}
func (unavailableAuthority) Unlock(context.Context, UnlockRequest, string) error {
	return errUnavailable
}
func (unavailableAuthority) Fill(context.Context, FillRequest) (string, error) {
	return "", errUnavailable
}
func (unavailableAuthority) Revoke(context.Context, FillRequest) error { return errUnavailable }

type session struct {
	ID          string
	ExtensionID string
	WorkspaceID string
	GrantID     string
	Field       string
	Origin      string
	TabID       string
	FrameID     string
	DocumentID  string
	ItemID      string
	Revision    int
	ExpiresAt   time.Time
}

type Host struct {
	mu         sync.Mutex
	authority  Authority
	enrolled   map[string]EnrollmentRequest
	sessions   map[string]session
	usedNonces map[string]time.Time
}

func NewHost(authority Authority) *Host {
	if authority == nil {
		authority = unavailableAuthority{}
	}
	return &Host{authority: authority, enrolled: map[string]EnrollmentRequest{}, sessions: map[string]session{}, usedNonces: map[string]time.Time{}}
}

func (h *Host) Handle(ctx context.Context, request Request) Response {
	if err := validateRequest(request); err != nil {
		return failure(request, codeFor(err), err.Error())
	}
	h.mu.Lock()
	now := time.Now().UTC()
	for nonce, usedAt := range h.usedNonces {
		if now.Sub(usedAt) > MaxLifetime {
			delete(h.usedNonces, nonce)
		}
	}
	if _, used := h.usedNonces[request.Nonce]; used {
		h.mu.Unlock()
		return failure(request, "replay", errReplay.Error())
	}
	h.usedNonces[request.Nonce] = now
	h.mu.Unlock()

	switch request.Operation {
	case "enroll":
		return h.enroll(ctx, request)
	case "metadata":
		return h.metadata(ctx, request)
	case "save", "update":
		return h.write(ctx, request)
	case "unlock":
		return h.unlock(ctx, request)
	case "fill":
		return h.fill(ctx, request)
	case "revoke", "disconnect":
		return h.revoke(ctx, request)
	default:
		return failure(request, "unsupported_operation", "operation is unsupported")
	}
}

func (h *Host) write(ctx context.Context, request Request) Response {
	h.mu.Lock()
	enrollment, enrolled := h.enrolled[request.ExtensionID]
	h.mu.Unlock()
	if !enrolled || enrollment.Origin != request.Scope.Origin {
		return failure(request, "unauthorized", errUnauthorized.Error())
	}
	workspace := request.Payload["workspace_id"]
	vaultID := request.Payload["vault_id"]
	name := request.Payload["name"]
	uri := request.Payload["uri"]
	password := request.Payload["password"]
	if workspace == "" || !principalPattern.MatchString(workspace) || enrollment.WorkspaceID != workspace || vaultID == "" || name == "" || uri == "" || password == "" {
		return failure(request, "invalid_request", "workspace_id, vault_id, name, uri, and password are required")
	}
	base := SaveRequest{ExtensionID: request.ExtensionID, WorkspaceID: workspace, VaultID: vaultID, Name: name, Username: request.Payload["username"], URI: uri, Password: password, TOTP: request.Payload["totp"], Scope: request.Scope}
	var metadata MetadataRecord
	var err error
	if request.Operation == "update" {
		if request.Scope.ItemID == "" || request.Scope.Revision < 1 {
			return failure(request, "invalid_request", "update requires an item revision")
		}
		metadata, err = h.authority.Update(ctx, UpdateRequest{SaveRequest: base, ItemID: request.Scope.ItemID, Revision: request.Scope.Revision}, request.Payload["session_token"])
	} else {
		metadata, err = h.authority.Save(ctx, base, request.Payload["session_token"])
	}
	if err != nil {
		return failure(request, codeFor(err), err.Error())
	}
	response := success(request, nil)
	response.Metadata = []MetadataRecord{metadata}
	return response
}

func (h *Host) metadata(ctx context.Context, request Request) Response {
	h.mu.Lock()
	enrollment, enrolled := h.enrolled[request.ExtensionID]
	h.mu.Unlock()
	if !enrolled || enrollment.Origin != request.Scope.Origin {
		return failure(request, "unauthorized", errUnauthorized.Error())
	}
	workspace := request.Payload["workspace_id"]
	if workspace == "" || !principalPattern.MatchString(workspace) || enrollment.WorkspaceID != workspace {
		return failure(request, "unauthorized", errUnauthorized.Error())
	}
	metadata, err := h.authority.Metadata(ctx, MetadataRequest{ExtensionID: request.ExtensionID, WorkspaceID: workspace, Scope: request.Scope}, request.Payload["session_token"])
	if err != nil {
		return failure(request, codeFor(err), err.Error())
	}
	response := success(request, nil)
	response.Metadata = metadata
	return response
}

func (h *Host) enroll(ctx context.Context, request Request) Response {
	token := request.Payload["enrollment_token"]
	if token == "" {
		return failure(request, "unauthorized", errUnauthorized.Error())
	}
	h.mu.Lock()
	_, alreadyEnrolled := h.enrolled[request.ExtensionID]
	h.mu.Unlock()
	if alreadyEnrolled {
		return failure(request, "unauthorized", "extension is already enrolled")
	}
	workspace := request.Payload["workspace_id"]
	if workspace == "" || !principalPattern.MatchString(workspace) {
		return failure(request, "invalid_request", "workspace_id is required")
	}
	if err := h.authority.Enroll(ctx, EnrollmentRequest{ExtensionID: request.ExtensionID, WorkspaceID: workspace, Origin: request.Scope.Origin}, token); err != nil {
		return failure(request, codeFor(err), err.Error())
	}
	h.mu.Lock()
	h.enrolled[request.ExtensionID] = EnrollmentRequest{ExtensionID: request.ExtensionID, WorkspaceID: workspace, Origin: request.Scope.Origin}
	h.mu.Unlock()
	return success(request, map[string]string{"enrollment": "accepted"})
}

func (h *Host) unlock(ctx context.Context, request Request) Response {
	h.mu.Lock()
	enrollment, enrolled := h.enrolled[request.ExtensionID]
	h.mu.Unlock()
	if !enrolled {
		return failure(request, "unauthorized", errUnauthorized.Error())
	}
	id, err := randomID()
	if err != nil {
		return failure(request, "internal", err.Error())
	}
	workspace := request.Payload["workspace_id"]
	grantID := request.Payload["grant_id"]
	field := request.Payload["field"]
	if workspace == "" || !principalPattern.MatchString(workspace) || grantID == "" || !principalPattern.MatchString(grantID) || field == "" || len(field) > 64 || !principalPattern.MatchString(field) {
		return failure(request, "invalid_request", "workspace_id, grant_id, and field are required")
	}
	if enrollment.WorkspaceID != workspace || enrollment.Origin != request.Scope.Origin {
		return failure(request, "unauthorized", errUnauthorized.Error())
	}
	s := session{ID: id, ExtensionID: request.ExtensionID, WorkspaceID: workspace, GrantID: grantID, Field: field, Origin: request.Scope.Origin, TabID: request.Scope.TabID, FrameID: request.Scope.FrameID, DocumentID: request.Scope.DocumentID, ItemID: request.Scope.ItemID, Revision: request.Scope.Revision, ExpiresAt: time.Unix(request.ExpiresAt, 0).UTC()}
	if err := h.authority.Unlock(ctx, UnlockRequest{ExtensionID: request.ExtensionID, SessionID: id, WorkspaceID: workspace, GrantID: grantID, Field: field, Scope: request.Scope}, request.Payload["session_token"]); err != nil {
		return failure(request, codeFor(err), err.Error())
	}
	h.mu.Lock()
	h.sessions[id] = s
	h.mu.Unlock()
	response := success(request, nil)
	response.SessionID = id
	return response
}

func (h *Host) fill(ctx context.Context, request Request) Response {
	s, ok := h.authorizedSession(request)
	if !ok {
		return failure(request, "unauthorized", errUnauthorized.Error())
	}
	if field := request.Payload["field"]; field != "" && field != s.Field {
		return failure(request, "scope_denied", errScope.Error())
	}
	value, err := h.authority.Fill(ctx, FillRequest{ExtensionID: request.ExtensionID, SessionID: s.ID, WorkspaceID: s.WorkspaceID, GrantID: s.GrantID, Field: s.Field, Scope: request.Scope})
	if err != nil {
		return failure(request, codeFor(err), err.Error())
	}
	response := success(request, map[string]string{"credential": value})
	response.Scope = &request.Scope
	return response
}

func (h *Host) revoke(ctx context.Context, request Request) Response {
	if request.SessionID == "" {
		return failure(request, "unauthorized", errUnauthorized.Error())
	}
	if _, ok := h.authorizedSession(request); !ok {
		return failure(request, "unauthorized", errUnauthorized.Error())
	}
	s, _ := h.authorizedSession(request)
	_ = h.authority.Revoke(ctx, FillRequest{ExtensionID: request.ExtensionID, SessionID: s.ID, WorkspaceID: s.WorkspaceID, GrantID: s.GrantID, Field: s.Field, Scope: request.Scope})
	h.mu.Lock()
	if request.Payload["revoke_enrollment"] == "true" {
		delete(h.enrolled, request.ExtensionID)
	}
	if request.SessionID != "" {
		delete(h.sessions, request.SessionID)
	}
	if request.Operation == "disconnect" || request.Payload["revoke_enrollment"] == "true" {
		for id, session := range h.sessions {
			if session.ExtensionID == request.ExtensionID {
				delete(h.sessions, id)
			}
		}
	}
	h.mu.Unlock()
	return success(request, map[string]string{"status": "revoked"})
}

func (h *Host) authorizedSession(request Request) (session, bool) {
	h.mu.Lock()
	s, ok := h.sessions[request.SessionID]
	_, enrolled := h.enrolled[request.ExtensionID]
	h.mu.Unlock()
	if !ok || !enrolled || s.ExtensionID != request.ExtensionID || time.Now().UTC().After(s.ExpiresAt) {
		return session{}, false
	}
	if s.Origin != request.Scope.Origin || s.TabID != request.Scope.TabID || s.FrameID != request.Scope.FrameID || s.DocumentID != request.Scope.DocumentID || s.ItemID != request.Scope.ItemID || s.Revision != request.Scope.Revision {
		return session{}, false
	}
	return s, true
}

func validateRequest(request Request) error {
	if request.Protocol != ProtocolName || request.RequestID == "" || !requestIDPattern.MatchString(request.RequestID) || !principalPattern.MatchString(request.ExtensionID) || request.Nonce == "" || len(request.Nonce) > 128 {
		return errMalformed
	}
	if request.ExpiresAt <= time.Now().UTC().Unix() || time.Unix(request.ExpiresAt, 0).After(time.Now().UTC().Add(MaxLifetime)) {
		return errExpired
	}
	if len(request.Payload) > 32 {
		return errMalformed
	}
	for key, value := range request.Payload {
		if len(key) > 128 || len(value) > 16*1024 {
			return errMalformed
		}
	}
	if request.Operation != "enroll" && request.Operation != "metadata" && request.Operation != "save" && request.Operation != "update" && request.Operation != "unlock" && request.Operation != "fill" && request.Operation != "revoke" && request.Operation != "disconnect" {
		return errMalformed
	}
	if request.Operation == "enroll" && (request.Payload["workspace_id"] == "" || !principalPattern.MatchString(request.Payload["workspace_id"])) {
		return errMalformed
	}
	if request.Operation == "unlock" && (request.Payload["workspace_id"] == "" || request.Payload["grant_id"] == "" || request.Payload["field"] == "") {
		return errMalformed
	}
	if request.Operation == "metadata" && request.Payload["workspace_id"] == "" {
		return errMalformed
	}
	if (request.Operation == "save" || request.Operation == "update") && (request.Payload["workspace_id"] == "" || request.Payload["vault_id"] == "" || request.Payload["name"] == "" || request.Payload["uri"] == "" || request.Payload["password"] == "") {
		return errMalformed
	}
	if err := validateOrigin(request.Scope.Origin); err != nil {
		return err
	}
	if request.Operation == "enroll" {
		return nil
	}
	if request.Scope.TabID == "" || len(request.Scope.TabID) > 128 || request.Scope.FrameID == "" || len(request.Scope.FrameID) > 128 || request.Scope.DocumentID == "" || len(request.Scope.DocumentID) > 256 || request.Scope.ItemID == "" || len(request.Scope.ItemID) > 128 || request.Scope.Revision < 0 {
		return errMalformed
	}
	return nil
}

func validateOrigin(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errMalformed
	}
	return nil
}

func ReadFrame(reader io.Reader) ([]byte, error) {
	var length uint32
	if err := binary.Read(io.LimitReader(reader, 4), binary.LittleEndian, &length); err != nil {
		return nil, err
	}
	if length == 0 || length > MaxFrameBytes {
		return nil, fmt.Errorf("native-messaging frame exceeds %d bytes", MaxFrameBytes)
	}
	frame := make([]byte, length)
	if _, err := io.ReadFull(reader, frame); err != nil {
		return nil, err
	}
	return frame, nil
}

func WriteFrame(writer io.Writer, payload []byte) error {
	if len(payload) == 0 || len(payload) > MaxFrameBytes {
		return fmt.Errorf("native-messaging frame exceeds %d bytes", MaxFrameBytes)
	}
	var header [4]byte
	// MaxFrameBytes is a bounded 64 KiB protocol limit, so this conversion is
	// proven to fit before the length is written to the four-byte frame header.
	binary.LittleEndian.PutUint32(header[:], uint32(len(payload))) // #nosec G115 -- bounded by MaxFrameBytes
	if err := writeAll(writer, header[:]); err != nil {
		return err
	}
	return writeAll(writer, payload)
}

func writeAll(writer io.Writer, payload []byte) error {
	for len(payload) > 0 {
		written, err := writer.Write(payload)
		if err != nil {
			return err
		}
		if written <= 0 || written > len(payload) {
			return io.ErrShortWrite
		}
		payload = payload[written:]
	}
	return nil
}

func EncodeResponse(response Response) ([]byte, error) {
	payload, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}
	if len(payload) > MaxPayloadBytes {
		return nil, fmt.Errorf("native-messaging response exceeds %d bytes", MaxPayloadBytes)
	}
	return payload, nil
}

func DecodeRequest(payload []byte) (Request, error) {
	if len(payload) == 0 || len(payload) > MaxPayloadBytes {
		return Request{}, errMalformed
	}
	var request Request
	if err := json.Unmarshal(payload, &request); err != nil {
		return Request{}, errMalformed
	}
	return request, nil
}

func randomID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

func success(request Request, payload map[string]string) Response {
	return Response{Protocol: ProtocolName, RequestID: request.RequestID, OK: true, Payload: payload}
}

func failure(request Request, code, message string) Response {
	return Response{Protocol: ProtocolName, RequestID: request.RequestID, Code: code, Message: message}
}

func codeFor(err error) string {
	switch {
	case errors.Is(err, errReplay):
		return "replay"
	case errors.Is(err, errExpired):
		return "expired"
	case errors.Is(err, errScope):
		return "scope_denied"
	case errors.Is(err, errUnavailable):
		return "unavailable"
	case errors.Is(err, errUnauthorized):
		return "unauthorized"
	default:
		return "invalid_request"
	}
}

// ValidateSize is exposed for host adapters that receive a frame through a
// browser-specific transport before DecodeRequest is called.
func ValidateSize(payload []byte) error {
	if len(payload) == 0 || len(payload) > MaxPayloadBytes {
		return errMalformed
	}
	return nil
}

// Errors returns stable protocol classifications for callers that need to
// preserve a typed denial without exposing implementation details.
func Errors() (malformed, unauthorized, replay, expired, scope, unavailable error) {
	return errMalformed, errUnauthorized, errReplay, errExpired, errScope, errUnavailable
}
