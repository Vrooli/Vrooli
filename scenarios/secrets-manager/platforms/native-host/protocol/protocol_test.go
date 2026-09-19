package protocol

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

type fakeAuthority struct {
	enrolled bool
	secret   string
}

func (a *fakeAuthority) Enroll(context.Context, EnrollmentRequest, string) error {
	a.enrolled = true
	return nil
}
func (a *fakeAuthority) Metadata(context.Context, MetadataRequest, string) ([]MetadataRecord, error) {
	if !a.enrolled {
		return nil, errUnauthorized
	}
	return []MetadataRecord{{GrantID: "grant-a", ItemID: "item-1", Name: "Fixture"}}, nil
}
func (a *fakeAuthority) Save(context.Context, SaveRequest, string) (MetadataRecord, error) {
	if !a.enrolled {
		return MetadataRecord{}, errUnauthorized
	}
	return MetadataRecord{ItemID: "item-1", Name: "Fixture", Revision: 1}, nil
}
func (a *fakeAuthority) Update(context.Context, UpdateRequest, string) (MetadataRecord, error) {
	if !a.enrolled {
		return MetadataRecord{}, errUnauthorized
	}
	return MetadataRecord{ItemID: "item-1", Name: "Fixture", Revision: 2}, nil
}
func (a *fakeAuthority) Unlock(context.Context, UnlockRequest, string) error {
	if !a.enrolled {
		return errUnauthorized
	}
	return nil
}
func (a *fakeAuthority) Fill(context.Context, FillRequest) (string, error) { return a.secret, nil }
func (a *fakeAuthority) Revoke(context.Context, FillRequest) error         { return nil }

func request(operation, nonce string) Request {
	return Request{
		Protocol: ProtocolName, RequestID: "req-" + nonce, Operation: operation,
		ExtensionID: "abcdefghijklmnopabcdefghijklmnop", Nonce: nonce,
		ExpiresAt: time.Now().UTC().Add(time.Minute).Unix(),
		Scope:     Scope{Origin: "https://example.com", TabID: "tab-1", FrameID: "frame-1", DocumentID: "document-1", ItemID: "item-1", Revision: 3},
		Payload:   map[string]string{},
	}
}

func TestHostRequiresEnrollmentAndBindsFillToDocument(t *testing.T) {
	authority := &fakeAuthority{secret: "secret-value"}
	host := NewHost(authority)
	deniedRequest := request("unlock", "unlock-before-enroll")
	deniedRequest.Payload["workspace_id"] = "workspace-a"
	deniedRequest.Payload["grant_id"] = "grant-a"
	deniedRequest.Payload["field"] = "password"
	denied := host.Handle(context.Background(), deniedRequest)
	if denied.OK || denied.Code != "unauthorized" {
		t.Fatalf("unlock before enrollment = %+v", denied)
	}
	enroll := request("enroll", "enroll-1")
	enroll.Payload["enrollment_token"] = "one-time-enrollment"
	enroll.Payload["workspace_id"] = "workspace-a"
	if response := host.Handle(context.Background(), enroll); !response.OK {
		t.Fatalf("enrollment failed: %+v", response)
	}
	wrongOriginUnlock := request("unlock", "unlock-wrong-origin")
	wrongOriginUnlock.Payload["session_token"] = "fresh-session"
	wrongOriginUnlock.Payload["workspace_id"] = "workspace-a"
	wrongOriginUnlock.Payload["grant_id"] = "grant-a"
	wrongOriginUnlock.Payload["field"] = "password"
	wrongOriginUnlock.Scope.Origin = "https://other.example"
	if response := host.Handle(context.Background(), wrongOriginUnlock); response.OK || response.Code != "unauthorized" {
		t.Fatalf("cross-origin unlock = %+v", response)
	}
	unlock := request("unlock", "unlock-1")
	unlock.Payload["session_token"] = "fresh-session"
	unlock.Payload["workspace_id"] = "workspace-a"
	unlock.Payload["grant_id"] = "grant-a"
	unlock.Payload["field"] = "password"
	unlockResponse := host.Handle(context.Background(), unlock)
	if !unlockResponse.OK || unlockResponse.SessionID == "" {
		t.Fatalf("unlock = %+v", unlockResponse)
	}
	fill := request("fill", "fill-1")
	fill.SessionID = unlockResponse.SessionID
	fill.Payload["field"] = "password"
	fillResponse := host.Handle(context.Background(), fill)
	if !fillResponse.OK || fillResponse.Payload["credential"] != "secret-value" {
		t.Fatalf("fill = %+v", fillResponse)
	}
	replayed := host.Handle(context.Background(), fill)
	if replayed.OK || replayed.Code != "replay" {
		t.Fatalf("replayed fill = %+v", replayed)
	}
	navigated := request("fill", "fill-2")
	navigated.SessionID = unlockResponse.SessionID
	navigated.Scope.DocumentID = "document-2"
	if response := host.Handle(context.Background(), navigated); response.OK || response.Code != "unauthorized" {
		t.Fatalf("navigation replay = %+v", response)
	}
}

func TestHostRevocationAndExpiryFailClosed(t *testing.T) {
	authority := &fakeAuthority{secret: "secret-value"}
	host := NewHost(authority)
	enroll := request("enroll", "enroll-2")
	enroll.Payload["enrollment_token"] = "token"
	enroll.Payload["workspace_id"] = "workspace-a"
	host.Handle(context.Background(), enroll)
	unlock := request("unlock", "unlock-2")
	unlock.Payload["session_token"] = "token"
	unlock.Payload["workspace_id"] = "workspace-a"
	unlock.Payload["grant_id"] = "grant-a"
	unlock.Payload["field"] = "password"
	session := host.Handle(context.Background(), unlock).SessionID
	revoke := request("revoke", "revoke-1")
	revoke.SessionID = session
	if response := host.Handle(context.Background(), revoke); !response.OK {
		t.Fatalf("revoke = %+v", response)
	}
	fill := request("fill", "fill-after-revoke")
	fill.SessionID = session
	if response := host.Handle(context.Background(), fill); response.OK || response.Code != "unauthorized" {
		t.Fatalf("fill after revoke = %+v", response)
	}
	revokeEnrollment := request("revoke", "revoke-enrollment-1")
	revokeEnrollment.Payload["revoke_enrollment"] = "true"
	if response := host.Handle(context.Background(), revokeEnrollment); response.OK || response.Code != "unauthorized" {
		t.Fatalf("unauthenticated enrollment revoke = %+v", response)
	}
	unlockAgain := request("unlock", "unlock-for-enrollment-revoke")
	unlockAgain.Payload["session_token"] = "token"
	unlockAgain.Payload["workspace_id"] = "workspace-a"
	unlockAgain.Payload["grant_id"] = "grant-a"
	unlockAgain.Payload["field"] = "password"
	secondSession := host.Handle(context.Background(), unlockAgain).SessionID
	authenticatedRevoke := request("revoke", "authenticated-revoke-enrollment")
	authenticatedRevoke.SessionID = secondSession
	authenticatedRevoke.Payload["revoke_enrollment"] = "true"
	if response := host.Handle(context.Background(), authenticatedRevoke); !response.OK {
		t.Fatalf("authenticated revoke enrollment = %+v", response)
	}
	host.mu.Lock()
	if _, ok := host.enrolled[request("fill", "enrollment-check").ExtensionID]; ok {
		t.Fatal("revoked enrollment remained active")
	}
	host.mu.Unlock()
	expired := request("fill", "expired-1")
	expired.ExpiresAt = time.Now().Add(-time.Second).Unix()
	if response := host.Handle(context.Background(), expired); response.OK || response.Code != "expired" {
		t.Fatalf("expired = %+v", response)
	}
}

type shortWriter struct{}

func (shortWriter) Write(payload []byte) (int, error) {
	if len(payload) == 0 {
		return 0, nil
	}
	return 1, nil
}

func TestWriteFrameHandlesShortWrites(t *testing.T) {
	if err := WriteFrame(shortWriter{}, []byte("bounded")); err != nil {
		t.Fatalf("short writer frame = %v", err)
	}
}

func TestNativeFrameBoundsAndRoundTrip(t *testing.T) {
	payload := []byte(`{"protocol":"vrooli.secrets.native/1"}`)
	var framed bytes.Buffer
	if err := WriteFrame(&framed, payload); err != nil {
		t.Fatal(err)
	}
	decoded, err := ReadFrame(&framed)
	if err != nil || !bytes.Equal(decoded, payload) {
		t.Fatalf("frame round trip = %q, %v", decoded, err)
	}
	var oversized bytes.Buffer
	_ = binary.Write(&oversized, binary.LittleEndian, uint32(MaxFrameBytes+1))
	if _, err := ReadFrame(&oversized); err == nil {
		t.Fatal("oversized frame was accepted")
	}
	if _, err := DecodeRequest(make([]byte, MaxPayloadBytes+1)); err == nil {
		t.Fatal("oversized JSON payload was accepted")
	}
	if _, err := EncodeResponse(Response{Protocol: ProtocolName, Payload: map[string]string{"x": string(make([]byte, MaxPayloadBytes))}}); err == nil {
		t.Fatal("oversized response was accepted")
	}
}

func TestMalformedAndAuthorityUnavailableAreTyped(t *testing.T) {
	host := NewHost(nil)
	bad := request("fill", "bad-1")
	bad.Scope.Origin = "file:///tmp/secret"
	if response := host.Handle(context.Background(), bad); response.OK || response.Code != "invalid_request" {
		t.Fatalf("malformed origin = %+v", response)
	}
	enroll := request("enroll", "unavailable-1")
	enroll.Payload["enrollment_token"] = "token"
	enroll.Payload["workspace_id"] = "workspace-a"
	if response := host.Handle(context.Background(), enroll); response.OK || response.Code != "unavailable" {
		t.Fatalf("unavailable authority = %+v", response)
	}
	if errors.Is(errUnavailable, errMalformed) {
		t.Fatal("protocol errors collapsed")
	}
}

func FuzzDecodeRequest(f *testing.F) {
	seed, _ := json.Marshal(request("fill", "seed"))
	f.Add(seed)
	f.Add([]byte("not-json"))
	f.Fuzz(func(t *testing.T, payload []byte) {
		_, _ = DecodeRequest(payload)
	})
}
