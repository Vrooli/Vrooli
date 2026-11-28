package main

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/reach"
	"scenario-to-cloud/reach/sshadapter"
	"scenario-to-cloud/secrets"

	"github.com/vrooli/api-core/identity"
)

// =============================================================================
// Test Fakes for Seam Interfaces
//
// These fakes enable testing without live external services. Use them in tests
// by injecting into the Server struct or passing to domain functions.
//
// Example usage:
//
//	srv := &Server{
//	    sshRunner:        &FakeSSHRunner{Responses: map[string]sshadapter.Result{"echo ok": {ExitCode: 0}}},
//	    scpRunner:        &FakeSCPRunner{},
//	    secretsFetcher:   &FakeSecretsFetcher{Response: testSecrets},
//	    secretsGenerator: &FakeSecretsGenerator{Values: map[string]string{"key": "value"}}, // implements secrets.GeneratorFunc
//	    dnsService:       dns.NewService(&dns.FakeResolver{Hosts: map[string][]string{"example.com": {"1.2.3.4"}}}),
//	}
// =============================================================================

// FakeSSHRunner provides a controllable sshadapter.Runner for testing.
// Configure expected command responses, or set default error behavior.
type FakeSSHRunner struct {
	// Handler is called first for every command. If nil or returns false,
	// falls through to map-based matching.
	Handler func(command string) (sshadapter.Result, error, bool)
	// Responses maps commands to their sshadapter.Result responses
	Responses map[string]sshadapter.Result
	// Errs maps commands to errors (takes precedence over Responses)
	Errs map[string]error
	// DefaultErr is returned for unknown commands if set
	DefaultErr error
	// Calls records all commands executed (for verification)
	Calls []string
	mu    sync.Mutex
}

// Ensure FakeSSHRunner implements sshadapter.Runner at compile time.
var _ sshadapter.Runner = (*FakeSSHRunner)(nil)

// Run returns the configured response for a command, or an error.
func (f *FakeSSHRunner) Run(_ context.Context, _ sshadapter.ConnectionConfig, command string, _ sshadapter.RunOptions) (sshadapter.Result, error) {
	f.mu.Lock()
	f.Calls = append(f.Calls, command)
	f.mu.Unlock()

	// Try the legacy semantic spelling first for typed observations so old
	// handlers remain meaningful while the production transport changes.
	if f.Handler != nil {
		if legacy, ok := f.typedObservationLegacy(command); ok {
			if res, err, handled := f.Handler(legacy); handled {
				return res, err
			}
		}
		// Try handler first for all other commands (supports prefix/regex matching).
		if res, err, handled := f.Handler(command); handled {
			return res, err
		}
	}

	// Fall back to exact-match maps (backward compatible)
	if err, ok := f.Errs[command]; ok {
		return sshadapter.Result{ExitCode: 255}, err
	}
	if res, ok := f.Responses[command]; ok {
		return res, nil
	}
	if res, ok := f.typedObservationResponse(command); ok {
		return res, nil
	}
	if f.DefaultErr != nil {
		return sshadapter.Result{ExitCode: 1}, f.DefaultErr
	}
	return sshadapter.Result{Stdout: "", ExitCode: 127}, fmt.Errorf("unknown command: %s", command)
}

var quotedToken = regexp.MustCompile(`'([^']*)'`)

// typedObservationResponse keeps legacy semantic fixtures reusable while the
// SSH adapter now invokes the target-owned cloud-target host observe verb.
// It is test compatibility only; production never parses remote shell text.
func (f *FakeSSHRunner) typedObservationResponse(command string) (sshadapter.Result, bool) {
	legacy, ok := f.typedObservationLegacy(command)
	if !ok {
		return sshadapter.Result{}, false
	}
	if result, ok := f.Responses[legacy]; ok {
		return result, true
	}
	return sshadapter.Result{}, false
}

func (f *FakeSSHRunner) typedObservationLegacy(command string) (string, bool) {
	if !strings.Contains(command, "'cloud-target' 'host' 'observe'") {
		return "", false
	}
	matches := quotedToken.FindAllStringSubmatch(command, -1)
	var tokens []string
	for _, match := range matches {
		tokens = append(tokens, match[1])
	}
	kindIndex := -1
	for i, token := range tokens {
		if token == "--kind" && i+1 < len(tokens) {
			kindIndex = i + 1
			break
		}
	}
	if kindIndex < 0 {
		return "", false
	}
	kind := tokens[kindIndex]
	var args []string
	for i := kindIndex + 1; i+1 < len(tokens); i++ {
		if tokens[i] == "--arg" {
			args = append(args, tokens[i+1])
			i++
		}
	}
	for program, mapped := range reach.ObservationKinds {
		if mapped != kind {
			continue
		}
		legacy := sshadapter.ObservationCommand(append([]string{program}, args...))
		return legacy, true
	}
	return "", false
}

// FakeSCPRunner provides a controllable sshadapter.SCPRunner for testing.
// Configure specific path errors or use default success behavior.
type FakeSCPRunner struct {
	// Errs maps "localPath->remotePath" to errors
	Errs map[string]error
	// DefaultErr is returned for all copies if set
	DefaultErr error
	// Calls records all copy operations (for verification)
	Calls []struct{ Local, Remote string }
	mu    sync.Mutex
}

// Ensure FakeSCPRunner implements sshadapter.SCPRunner at compile time.
var _ sshadapter.SCPRunner = (*FakeSCPRunner)(nil)

// Copy records the operation and returns any configured error.
func (f *FakeSCPRunner) Copy(_ context.Context, _ sshadapter.ConnectionConfig, localPath, remotePath string, _ sshadapter.SCPOptions) error {
	f.mu.Lock()
	f.Calls = append(f.Calls, struct{ Local, Remote string }{localPath, remotePath})
	f.mu.Unlock()

	key := localPath + "->" + remotePath
	if err, ok := f.Errs[key]; ok {
		return err
	}
	return f.DefaultErr
}

// FakeSecretsFetcher provides a controllable secrets.Fetcher for testing.
// Configure a response or error for FetchBundleSecrets calls.
type FakeSecretsFetcher struct {
	// Response is returned by FetchBundleSecrets if Err is nil
	Response *secrets.ManagerResponse
	// Err is returned by FetchBundleSecrets if set
	Err error
	// HealthErr is returned by HealthCheck if set
	HealthErr error
	// Calls records all FetchBundleSecrets calls (for verification)
	Calls []struct {
		Scenario  string
		Tier      string
		Resources []string
	}
	mu sync.Mutex
}

// Ensure FakeSecretsFetcher implements secrets.Fetcher at compile time.
var _ secrets.Fetcher = (*FakeSecretsFetcher)(nil)

// FetchBundleSecrets returns the configured response or error.
func (f *FakeSecretsFetcher) FetchBundleSecrets(_ context.Context, scenario, tier string, resources []string) (*secrets.ManagerResponse, error) {
	f.mu.Lock()
	f.Calls = append(f.Calls, struct {
		Scenario  string
		Tier      string
		Resources []string
	}{scenario, tier, resources})
	f.mu.Unlock()

	if f.Err != nil {
		return nil, f.Err
	}
	return f.Response, nil
}

// HealthCheck returns HealthErr if set, nil otherwise.
func (f *FakeSecretsFetcher) HealthCheck(_ context.Context) error {
	return f.HealthErr
}

// FakeSecretsGenerator provides a controllable secrets.GeneratorFunc for testing.
// Configure deterministic values for reproducible test results.
type FakeSecretsGenerator struct {
	// Values maps secret keys to their generated values
	Values map[string]string
	// Err is returned by GenerateSecrets if set
	Err error
	// DefaultValue is used for keys not in Values
	DefaultValue string
	// Calls records all GenerateSecrets calls (for verification)
	Calls [][]domain.BundleSecretPlan
	mu    sync.Mutex
}

// FakeDeploymentRepo provides a controllable DeploymentRepository for testing.
type FakeDeploymentRepo struct {
	Deployment *domain.Deployment
	Err        error
}

// GetDeployment returns the configured deployment or error.
func (f *FakeDeploymentRepo) GetDeployment(_ context.Context, _ string) (*domain.Deployment, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	return f.Deployment, nil
}

// Ensure FakeSecretsGenerator implements secrets.GeneratorFunc at compile time.
var _ secrets.GeneratorFunc = (*FakeSecretsGenerator)(nil)

// GenerateSecrets returns configured values for per_install_generated secrets.
func (f *FakeSecretsGenerator) GenerateSecrets(plans []domain.BundleSecretPlan) ([]secrets.GeneratedSecret, error) {
	f.mu.Lock()
	f.Calls = append(f.Calls, plans)
	f.mu.Unlock()

	if f.Err != nil {
		return nil, f.Err
	}

	var generated []secrets.GeneratedSecret
	for _, plan := range plans {
		if plan.Class != "per_install_generated" {
			continue
		}

		value := f.DefaultValue
		if v, ok := f.Values[plan.Target.Name]; ok {
			value = v
		}
		if value == "" {
			value = "fake-generated-" + plan.Target.Name
		}

		generated = append(generated, secrets.GeneratedSecret{
			ID:    plan.ID,
			Key:   plan.Target.Name,
			Value: value,
		})
	}

	return generated, nil
}

// FakePrincipalProvider is an authn.Provider that returns a fixed principal
// (or a fixed failure). Tests mutate it between requests to model expiry and
// revocation; it is never wired outside newTestEnforcer.
type FakePrincipalProvider struct {
	mu        sync.Mutex
	Principal identity.Principal
	Fail      error
	// Anonymous makes the provider report a missing credential.
	Anonymous bool
	// RevokeAfterCalls, when > 0, makes every verification after that many
	// calls fail as an invalid credential (revoked between admission and effect).
	RevokeAfterCalls int
	Calls            int
}

func (f *FakePrincipalProvider) Source() identity.AuthSource { return identity.SourcePersonalLocal }

func (f *FakePrincipalProvider) VerifyRequest(_ context.Context, _ *http.Request) (identity.Principal, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls++
	if f.Anonymous {
		return identity.Principal{}, identity.NewFailure(identity.FailureMissing, f.Source())
	}
	if f.Fail != nil {
		return identity.Principal{}, f.Fail
	}
	if f.RevokeAfterCalls > 0 && f.Calls > f.RevokeAfterCalls {
		return identity.Principal{}, identity.NewFailure(identity.FailureInvalid, f.Source())
	}
	return f.Principal, nil
}

// Set replaces the principal under the lock.
func (f *FakePrincipalProvider) Set(principal identity.Principal) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Principal = principal
	f.Fail = nil
	f.Anonymous = false
}

// Revoke makes every later verification fail as an invalid credential.
func (f *FakePrincipalProvider) Revoke() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Fail = identity.NewFailure(identity.FailureInvalid, f.Source())
}
