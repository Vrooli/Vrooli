package binaryfetch

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type authRoundTrip func(*http.Request) (*http.Response, error)

func (f authRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func authResponse(code int, body string, location string) *http.Response {
	return &http.Response{StatusCode: code, Status: fmt.Sprint(code), Header: http.Header{"Location": []string{location}}, Body: io.NopCloser(strings.NewReader(body))}
}
func useAuthClient(t *testing.T, c *http.Client) {
	t.Helper()
	old := HTTPClient
	HTTPClient = c
	t.Cleanup(func() { HTTPClient = old })
}

func TestAuthenticatedFetchScopesBearerAcrossRedirects(t *testing.T) {
	for _, target := range []string{"https://origin.test/final", "https://cdn.test/final", "https://sub.origin.test/final", "https://origin.test:444/final", "http://origin.test/final"} {
		t.Run(target, func(t *testing.T) {
			var got []string
			client := &http.Client{Transport: authRoundTrip(func(r *http.Request) (*http.Response, error) {
				got = append(got, r.Header.Get("Authorization"))
				if r.URL.Path == "/start" {
					return authResponse(302, "", target), nil
				}
				return authResponse(200, string(bigBinary()), ""), nil
			})}
			useAuthClient(t, client)
			_, err := Fetch(context.Background(), Target{Name: "model", URL: "https://origin.test/start", SHA256: sha256hex(bigBinary()), Authorization: NewBearerAuthorization("secret-token")}, t.TempDir(), nil)
			if err != nil {
				t.Fatal(err)
			}
			want := ""
			if target == "https://origin.test/final" {
				want = "Bearer secret-token"
			}
			if len(got) != 2 || got[0] != "Bearer secret-token" || got[1] != want {
				t.Fatalf("headers %q, want initial bearer/final %q", got, want)
			}
			if client.CheckRedirect != nil {
				t.Fatal("mutated shared client")
			}
		})
	}
}

func TestAuthenticatedFetchTLSAndSecretSafeErrors(t *testing.T) {
	const token = "secret-token-never-log"
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			t.Error("missing bearer")
		}
		_, _ = w.Write(bigBinary())
	}))
	defer srv.Close()
	useAuthClient(t, srv.Client())
	spec := Target{Name: "model", URL: srv.URL, SHA256: sha256hex(bigBinary()), Authorization: NewBearerAuthorization(token)}
	if _, err := Fetch(context.Background(), spec, t.TempDir(), nil); err != nil {
		t.Fatal(err)
	}
	for _, format := range []string{"%v", "%+v", "%#v"} {
		if strings.Contains(fmt.Sprintf(format, spec), token) {
			t.Fatal("secret in formatted spec")
		}
	}
	raw, _ := json.Marshal(spec)
	if bytes.Contains(raw, []byte(token)) {
		t.Fatal("secret serialized")
	}
	for _, injected := range []error{fmt.Errorf("transport failed %s", token), context.Canceled, context.DeadlineExceeded} {
		HTTPClient = &http.Client{Transport: authRoundTrip(func(*http.Request) (*http.Response, error) { return nil, injected })}
		_, err := Fetch(context.Background(), spec, t.TempDir(), nil)
		if err == nil || strings.Contains(fmt.Sprintf("%+v", err), token) {
			t.Fatalf("unsafe error %v", err)
		}
		if errors.Is(injected, context.Canceled) && !errors.Is(err, context.Canceled) {
			t.Fatal("lost cancellation")
		}
		if errors.Is(injected, context.DeadlineExceeded) && !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal("lost deadline")
		}
	}
}

func TestFetchDataAcceptsTinyMetadataWithoutWeakeningBinaryGuard(t *testing.T) {
	for _, body := range [][]byte{[]byte(`{"version":1}`), nil, []byte("<html>auth failed</html>")} {
		srv := serve(t, body)
		spec := Target{Name: "config.json", URL: srv.URL, SHA256: sha256hex(body)}
		dest := t.TempDir()
		prior := filepath.Join(dest, spec.Name)
		if err := os.WriteFile(prior, []byte("prior"), 0600); err != nil {
			t.Fatal(err)
		}
		_, err := FetchData(context.Background(), spec, dest, nil)
		if len(body) > 0 && body[0] == '{' {
			if err != nil {
				t.Fatal(err)
			}
			st, _ := os.Stat(prior)
			if st.Mode().Perm()&0111 != 0 {
				t.Fatal("data executable")
			}
			if _, err = Fetch(context.Background(), spec, t.TempDir(), nil); !errors.Is(err, ErrTooSmall) {
				t.Fatalf("binary floor lost %v", err)
			}
		} else {
			if err == nil {
				t.Fatal("invalid data accepted")
			}
			b, _ := os.ReadFile(prior)
			if string(b) != "prior" {
				t.Fatal("failed data replaced prior")
			}
		}
	}
}

func TestInspectDataBoundsGitIdentityAndRefusesOverwrite(t *testing.T) {
	body := []byte(`{"version":1}`)
	git := fmt.Sprintf("%x", sha1.Sum(append([]byte(fmt.Sprintf("blob %d%c", len(body), 0)), body...)))
	calls := 0
	useAuthClient(t, &http.Client{Transport: authRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Error("missing auth")
		}
		return authResponse(200, string(body), ""), nil
	})})
	spec := DataInspection{URL: "https://origin.test/config", GitBlobSHA1: git, MaxBytes: 1024, Authorization: NewBearerAuthorization("token")}
	destination := filepath.Join(t.TempDir(), "config.json")
	receipt, err := InspectData(context.Background(), spec, destination)
	if err != nil || receipt.SHA256 != sha256hex(body) || receipt.Bytes != int64(len(body)) || receipt.Origin != "https://origin.test" {
		t.Fatalf("receipt %+v err %v", receipt, err)
	}
	if _, err = InspectData(context.Background(), spec, destination); err == nil || calls != 1 {
		t.Fatalf("overwrite tried network: %v calls%d", err, calls)
	}
	for _, variant := range []string{"git", "limit", "oversize", "empty"} {
		t.Run(variant, func(t *testing.T) {
			s := spec
			switch variant {
			case "git":
				s.GitBlobSHA1 = strings.Repeat("a", 40)
			case "limit":
				s.MaxBytes = 1024*1024 + 1
			case "oversize":
				s.MaxBytes = 2
			case "empty":
				s.GitBlobSHA1 = ""
			}
			out := filepath.Join(t.TempDir(), "out")
			if _, err := InspectData(context.Background(), s, out); err == nil {
				t.Fatal("invalid inspection accepted")
			}
			if _, err := os.Stat(out); !os.IsNotExist(err) {
				t.Fatal("failed inspection published")
			}
		})
	}
}
