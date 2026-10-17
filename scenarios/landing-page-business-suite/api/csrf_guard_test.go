package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSameOriginGuard(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	guard := sameOriginGuard(next)
	cases := []struct {
		name    string
		method  string
		cookie  bool
		headers map[string]string
		want    int
	}{
		{"cross-site cookie POST is blocked", http.MethodPost, true, map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusForbidden},
		{"sibling-subdomain cookie POST is blocked", http.MethodPost, true, map[string]string{"Sec-Fetch-Site": "same-site"}, http.StatusForbidden},
		{"same-origin cookie POST passes", http.MethodPost, true, map[string]string{"Sec-Fetch-Site": "same-origin"}, http.StatusNoContent},
		{"legacy browser foreign Origin is blocked", http.MethodPost, true, map[string]string{"Origin": "https://evil.example"}, http.StatusForbidden},
		{"legacy browser matching Origin passes", http.MethodPost, true, map[string]string{"Origin": "http://lpbs.test"}, http.StatusNoContent},
		{"opaque null Origin is blocked", http.MethodPost, true, map[string]string{"Origin": "null"}, http.StatusForbidden},
		{"cross-site GET is not a state change", http.MethodGet, true, map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusNoContent},
		{"cookie-less webhook passes", http.MethodPost, false, map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusNoContent},
		{"non-browser client without fetch metadata passes", http.MethodPost, true, nil, http.StatusNoContent},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(tc.method, "http://lpbs.test/api/v1/admin/mfa/disable", nil)
			request.Host = "lpbs.test"
			if tc.cookie {
				request.AddCookie(&http.Cookie{Name: "admin_session", Value: "x"})
			}
			for key, value := range tc.headers {
				request.Header.Set(key, value)
			}
			recorder := httptest.NewRecorder()
			guard.ServeHTTP(recorder, request)
			if recorder.Code != tc.want {
				t.Fatalf("status=%d want %d", recorder.Code, tc.want)
			}
		})
	}
}
