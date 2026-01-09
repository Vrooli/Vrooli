package main

import (
	"context"
	"errors"
	"sync"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/secrets"
	"scenario-to-cloud/ssh"
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
//	    sshRunner:        &FakeSSHRunner{Responses: map[string]ssh.Result{"echo ok": {ExitCode: 0}}},
//	    scpRunner:        &FakeSCPRunner{},
//	    secretsFetcher:   &FakeSecretsFetcher{Response: testSecrets},
//	    secretsGenerator: &FakeSecretsGenerator{Values: map[string]string{"key": "value"}}, // implements secrets.GeneratorFunc
//	    dnsService:       dns.NewService(&dns.FakeResolver{Hosts: map[string][]string{"example.com": {"1.2.3.4"}}}),
//	}
// =============================================================================

// FakeSSHRunner provides a controllable ssh.Runner for testing.
// Configure expected command responses, or set default error behavior.
type FakeSSHRunner struct {
	// Responses maps commands to their ssh.Result responses
	Responses map[string]ssh.Result
	// Errs maps commands to errors (takes precedence over Responses)
	Errs map[string]error
	// DefaultErr is returned for unknown commands if set
	DefaultErr error
	// Calls records all commands executed (for verification)
	Calls []string
	mu    sync.Mutex
}

// Ensure FakeSSHRunner implements ssh.Runner at compile time.
var _ ssh.Runner = (*FakeSSHRunner)(nil)

// Run returns the configured response for a command, or an error.
func (f *FakeSSHRunner) Run(_ context.Context, _ ssh.Config, command string) (ssh.Result, error) {
	f.mu.Lock()
	f.Calls = append(f.Calls, command)
	f.mu.Unlock()

	if err, ok := f.Errs[command]; ok {
		return ssh.Result{ExitCode: 255}, err
	}
	if res, ok := f.Responses[command]; ok {
		return res, nil
	}
	if f.DefaultErr != nil {
		return ssh.Result{ExitCode: 255}, f.DefaultErr
	}
	return ssh.Result{ExitCode: 127, Stderr: "unknown command: " + command}, errors.New("unknown command")
}

// FakeSCPRunner provides a controllable ssh.SCPRunner for testing.
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

// Ensure FakeSCPRunner implements ssh.SCPRunner at compile time.
var _ ssh.SCPRunner = (*FakeSCPRunner)(nil)

// Copy records the operation and returns any configured error.
func (f *FakeSCPRunner) Copy(_ context.Context, _ ssh.Config, localPath, remotePath string) error {
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
