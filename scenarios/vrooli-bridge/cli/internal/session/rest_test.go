package session

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type recordingDoer struct {
	req    *http.Request
	body   string
	status int
	reply  string
}

func (d *recordingDoer) Do(req *http.Request) (*http.Response, error) {
	d.req = req
	if req.Body != nil {
		data, _ := io.ReadAll(req.Body)
		d.body = string(data)
	}
	return &http.Response{StatusCode: d.status, Body: io.NopCloser(strings.NewReader(d.reply)), Header: http.Header{}}, nil
}

func TestRESTRequestSendsThroughTheSessionTransport(t *testing.T) {
	doer := &recordingDoer{status: http.StatusOK, reply: `{"ok":true}`}
	data, err := restRequest(doer, "http://localhost:18767/", "/api/v1/follow/node-1", http.MethodPut, url.Values{"x": {"1"}}, map[string]string{"branch": "agi"})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"ok":true}` {
		t.Fatalf("data = %s", data)
	}
	if doer.req.Method != http.MethodPut || doer.req.URL.String() != "http://localhost:18767/api/v1/follow/node-1?x=1" {
		t.Fatalf("request = %s %s", doer.req.Method, doer.req.URL)
	}
	if doer.body != `{"branch":"agi"}` || doer.req.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("body = %q", doer.body)
	}
}

func TestRESTRequestReportsTheStatusAndServerMessage(t *testing.T) {
	doer := &recordingDoer{status: http.StatusNotFound, reply: "node is not following a branch\n"}
	_, err := restRequest(doer, "http://localhost:18767", "/api/v1/follow/node-1/check", http.MethodPost, nil, nil)
	if err == nil || err.Error() != "api error (404): node is not following a branch" {
		t.Fatalf("err = %v", err)
	}
}

// A node pairing itself has no operator session, so its pairing calls must
// reach Bridge without an enrollment attempt.
func TestNodePairingCallsSkipOperatorEnrollment(t *testing.T) {
	for _, path := range []string{
		"/vrooli.vrooli_bridge.v1.pairing.PairingService/RedeemPairingCode",
		"/vrooli.vrooli_bridge.v1.pairing.PairingService/RequestPairing",
		"/vrooli.vrooli_bridge.v1.pairing.PairingService/GetPairingRequest",
	} {
		if !skipEnrollment(path) {
			t.Fatalf("%s must skip operator enrollment", path)
		}
	}
	if skipEnrollment("/vrooli.vrooli_bridge.v1.pairing.PairingService/IssuePairingCode") {
		t.Fatal("issuing a code is an owner operation and must stay enrolled")
	}
}
