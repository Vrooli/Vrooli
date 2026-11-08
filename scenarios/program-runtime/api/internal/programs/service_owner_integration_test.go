package programs

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	"program-runtime/internal/localworkbridge"
)

type ownerJWKSFixture struct{ key *rsa.PrivateKey }

func (d ownerJWKSFixture) Do(r *http.Request) (*http.Response, error) {
	if r.URL.String() != "https://fixture.invalid/jwks" {
		return nil, localworkbridge.ErrSourceAuthority
	}
	body := fmt.Sprintf(`{"keys":[{"kid":"fixture","kty":"RSA","alg":"RS256","n":%q,"e":"AQAB"}]}`, base64.RawURLEncoding.EncodeToString(d.key.N.Bytes()))
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
}

type ownerBindingFixture struct{ b localworkbridge.SourceBinding }

func (b *ownerBindingFixture) ReadSourceBinding(context.Context, string) (localworkbridge.SourceBinding, error) {
	return b.b, nil
}

type ownerBoundaryFunc func(context.Context, func(context.Context) error) error

func (f ownerBoundaryFunc) Within(ctx context.Context, fn func(context.Context) error) error {
	return f(ctx, fn)
}

func TestServiceAdmissionCanonicalPermitComposition(t *testing.T) {
	now := time.Now()
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	require.NoError(t, err)
	policy := localworkbridge.SourcePolicy{Purpose: localworkbridge.SourceCollector, BindingID: "binding-1", BindingVersion: "v1", Subject: "bridge-service", Owner: "owner-1", SourceInstance: "swarminator", SourceEpoch: "epoch-1", CollectorDigest: strings.Repeat("a", 64), CollectorInputs: json.RawMessage(`{"row_limit":4,"team_ids":[]}`)}
	authority := &ownerBindingFixture{}
	verifier := authn.NewScenarioAuthenticatorProvider(authn.JWTConfig{Audience: "vrooli-local-work:collector", ExclusiveAudience: true, JWKSURL: "https://fixture.invalid/jwks", Doer: ownerJWKSFixture{key: key}})
	gate, err := localworkbridge.NewSourceGate(localworkbridge.SourceGateConfig{Enabled: true, Policy: policy, Verifier: verifier, Bindings: authority})
	require.NoError(t, err)
	authority.b = localworkbridge.SourceBinding{ID: policy.BindingID, Version: policy.BindingVersion, Subject: policy.Subject, Owner: policy.Owner, Audience: "vrooli-local-work:collector", ScopeSHA256: gate.ScopeSHA256(), NotBefore: now.Add(-time.Minute), ExpiresAt: now.Add(2 * time.Hour), Grant: &identity.OperationGrant{ID: "fixture-grant", Approver: policy.Owner, ExpiresAt: now.Add(2 * time.Hour), Operations: []string{"local-work.collector.admit", "local-work.collector.observe", "local-work.collector.wait"}, Targets: []string{"program-runtime:local-work/" + gate.ScopeSHA256() + "/*"}}}
	payload, err := json.Marshal(map[string]any{"sub": policy.Subject, "iss": "scenario-authenticator", "actor_kind": "service", "scope": []string{"local-work:collector"}, "aud": "vrooli-local-work:collector", "exp": now.Add(time.Hour).Unix()})
	require.NoError(t, err)
	input := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","kid":"fixture"}`)) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	require.NoError(t, err)
	token := input + "." + base64.RawURLEncoding.EncodeToString(signature)
	admitPermit := func(path, body string) *localworkbridge.SourcePermit {
		r, err := http.NewRequest("POST", path, strings.NewReader(body))
		require.NoError(t, err)
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Connect-Protocol-Version", "1")
		p, err := gate.Admit(context.Background(), r)
		require.NoError(t, err)
		return p
	}
	requestKey := "local-work-" + strings.Repeat("b", 64)
	deadline := now.Add(5 * time.Minute).UTC().Format(time.RFC3339Nano)
	body := fmt.Sprintf(`{"name":%q,"inputs":{"row_limit":4,"team_ids":[]},"expectedDigest":%q,"provenance":"PROVENANCE_AGENT","async":true,"idempotencyKey":%q,"admissionDeadline":%q}`, localworkbridge.SourceCollectorName, policy.CollectorDigest, requestKey, deadline)
	permit := admitPermit("/vrooli.program_runtime.v1.library.LibraryService/RunDeclaredProgram", body)
	boundary := &ownerBoundaryFixture{}
	ctx, err := WithSourcePermit(context.Background(), permit, boundary)
	require.NoError(t, err)
	declared := Identity{ProgramName: localworkbridge.SourceCollectorName, ProgramDigest: policy.CollectorDigest, IdempotencyKey: requestKey, AdmissionDeadline: deadline}
	d := newProgramsTestDB(t)
	runner := &admissionRunner{}
	s := NewService(Options{Store: d, Runner: runner})
	program, _, err := s.SubmitDeclared(ctx, "session", "print(1)", programsv1.Provenance_PROVENANCE_TEST, false, false, declared, Caller{})
	require.NoError(t, err)
	require.EqualValues(t, 1, runner.calls.Load())
	require.False(t, program.CallerIdentityVerified)
	_, _, err = s.SubmitDeclared(ctx, "session", "print(1)", programsv1.Provenance_PROVENANCE_TEST, false, false, declared, Caller{})
	require.NoError(t, err)
	require.EqualValues(t, 1, runner.calls.Load())
	observe := admitPermit("/vrooli.program_runtime.v1.library.LibraryService/GetDeclaredExecution", fmt.Sprintf(`{"name":%q,"idempotencyKey":%q}`, localworkbridge.SourceCollectorName, requestKey))
	observedCtx, err := WithSourcePermit(context.Background(), observe, boundary)
	require.NoError(t, err)
	result, err := s.SourceProgram(observedCtx, program.Id)
	require.NoError(t, err)
	require.True(t, result.Found)
	require.Equal(t, policy.Owner, result.Identity.Owner)
	require.Equal(t, policy.BindingVersion, result.Identity.BindingVersion)
	wait := admitPermit("/vrooli.program_runtime.v1.programs.ProgramService/WaitForProgram", fmt.Sprintf(`{"id":%q,"timeoutMillis":1000}`, program.Id))
	waitCtx, err := WithSourcePermit(context.Background(), wait, boundary)
	require.NoError(t, err)
	_, terminal, err := s.Wait(waitCtx, program.Id, time.Second)
	require.NoError(t, err)
	require.True(t, terminal)
	authority.b.Version = "v2"
	_, err = s.SourceProgram(observedCtx, program.Id)
	require.Error(t, err)
	authority.b.Version = "v1"
	authority.b.Owner = "other"
	_, err = s.SourceProgram(waitCtx, program.Id)
	require.Error(t, err)
	authority.b.Owner = policy.Owner
	// Final authority changes occur after permit construction, inside the
	// serialization immediately before insertion, never after a durable row.
	for _, reason := range []string{"revoked", "expired"} {
		t.Run(reason, func(t *testing.T) {
			authority.b.Revoked = false
			authority.b.ExpiresAt = now.Add(2 * time.Hour)
			denying := ownerBoundaryFunc(func(ctx context.Context, fn func(context.Context) error) error {
				if reason == "revoked" {
					authority.b.Revoked = true
				} else {
					authority.b.ExpiresAt = time.Now().Add(-time.Second)
				}
				return fn(ctx)
			})
			deniedCtx, err := WithSourcePermit(context.Background(), permit, denying)
			require.NoError(t, err)
			otherDB := newProgramsTestDB(t)
			other := NewRepository(otherDB)
			p := &programsv1.Program{Id: program.Id, ProgramName: declared.ProgramName, ProgramDigest: declared.ProgramDigest, RequestDigest: "intent"}
			created, err := other.Create(deniedCtx, p)
			require.Error(t, err)
			require.False(t, created)
			var count int
			require.NoError(t, otherDB.QueryRow("SELECT count(*) FROM programs").Scan(&count))
			require.Zero(t, count)
		})
	}
}
