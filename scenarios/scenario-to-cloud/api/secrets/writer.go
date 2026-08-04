package secrets

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"scenario-to-cloud/internal/shellutil"
	"scenario-to-cloud/ssh"

	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

// WriteToVPS provisions generated credentials to the target authority before
// resource startup. It does not create a plaintext secrets file.
// IMPORTANT: Preserves existing secrets to avoid breaking database connections on redeploy.
func WriteToVPS(
	ctx context.Context,
	sshRunner ssh.Runner,
	cfg ssh.Config,
	workdir string,
	secrets []GeneratedSecret,
	userSecrets map[string]string,
	scenarioID string,
) error {
	if len(secrets) == 0 && len(userSecrets) == 0 {
		return nil // Nothing to write
	}
	// Validated before any remote call. The identity is built from this, so an
	// empty value would otherwise send a run of `vrooli//<field>` queries to the
	// target before anything noticed.
	if strings.TrimSpace(scenarioID) == "" {
		return fmt.Errorf("provision remote credentials: scenario id is required to name the credential identity")
	}

	// A generated secret is written only when the remote store does not already
	// hold one. Regenerating a database password on redeploy is what produced
	// "password authentication failed" against a database that still had the
	// old one, so preservation is load-bearing rather than an optimization.
	//
	// Preservation now asks the remote authority whether a value exists instead
	// of reading a remote plaintext file — and it never reads the value back.
	// Knowing that something is stored is the whole question here.
	secretsMap := make(map[string]string)
	client, err := newRemoteCredentialClient(sshRunner, cfg)
	if err != nil {
		return err
	}
	for _, s := range secrets {
		configured, err := remoteCredentialConfiguredWithClient(ctx, client, scenarioID, s.Key)
		if err != nil {
			return err
		}
		if configured {
			continue
		}
		secretsMap[s.Key] = s.Value
	}
	// An operator-supplied value is an explicit instruction for this deploy and
	// overrides what is stored.
	for key, value := range userSecrets {
		if strings.TrimSpace(value) == "" {
			continue
		}
		secretsMap[key] = value
	}

	// Each value is provisioned into the remote host's own credential
	// authority, one SSH call per secret, with the value on standard input.
	//
	// It used to be one call that embedded the whole plaintext payload in the
	// command string. A command string becomes argv for the local ssh process
	// and the argument to the remote shell, so every value was visible in both
	// process listings for the duration of the deploy — the precise exposure
	// every credential-store adapter in this platform is built to avoid. It
	// also left a plaintext secrets.json on the VPS afterwards.
	//
	// The remote store is the same seam as the local one: on a headless VPS
	// that is the encrypted file store, so the values land encrypted at rest
	// under a key the file alone does not contain.
	identity := "vrooli/" + strings.TrimSpace(scenarioID)
	if err := ensureRemoteCredentialStoreWithClient(ctx, client); err != nil {
		return err
	}

	keys := make([]string, 0, len(secretsMap))
	for key := range secretsMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := secretsMap[key]
		if strings.TrimSpace(value) == "" {
			continue
		}
		field := CredentialField(key)
		if field == "" {
			continue
		}
		if _, err := client.Provision(ctx, credentialclient.ProvisionRequest{Identity: identity, Field: field, Value: value}); err != nil {
			// The value is never echoed back, so neither is it in this error.
			return fmt.Errorf("provision remote credential %s/%s: %w", identity, field, err)
		}
	}

	return nil
}

type credentialSSHRunner struct {
	runner ssh.Runner
	cfg    ssh.Config
}

func (r credentialSSHRunner) Run(ctx context.Context, _ string, args []string, stdin io.Reader) ([]byte, error) {
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		parts = append(parts, shellutil.QuoteSingle(arg))
	}
	options := ssh.DefaultRunOptions()
	if stdin != nil {
		data, err := io.ReadAll(stdin)
		if err != nil {
			return nil, err
		}
		options.Stdin = data
	}
	result, err := r.runner.Run(ctx, r.cfg, strings.Join(parts, " "), options)
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return []byte(result.Stdout), fmt.Errorf("remote command exited %d: %s", result.ExitCode, result.Stderr)
	}
	return []byte(result.Stdout), nil
}

func newRemoteCredentialClient(runner ssh.Runner, cfg ssh.Config) (credentialclient.Client, error) {
	target := fmt.Sprintf("%s@%s", cfg.User, cfg.Host)
	return credentialclient.NewClient(credentialclient.ClientOptions{RemoteTarget: target, RemoteRunner: credentialSSHRunner{runner: runner, cfg: cfg}})
}

func remoteCredentialConfiguredWithClient(ctx context.Context, client credentialclient.Client, scenarioID, key string) (bool, error) {
	field := CredentialField(key)
	if field == "" {
		return false, nil
	}
	identity := "vrooli/" + strings.TrimSpace(scenarioID)
	status, err := client.Status(ctx, identity, field)
	if err != nil {
		return false, err
	}
	if status.ProviderState != "available" {
		return false, fmt.Errorf("remote credential store is %s; refusing to decide whether %s/%s already exists", status.ProviderState, identity, field)
	}
	return status.Configured, nil
}

func ensureRemoteCredentialStoreWithClient(ctx context.Context, client credentialclient.Client) error {
	diagnosis, err := client.Doctor(ctx)
	if err != nil {
		return fmt.Errorf("remote credential diagnosis failed: %w", err)
	}
	if diagnosis.Provider.Condition != "available" {
		return fmt.Errorf("remote credential store is %s: %s", diagnosis.Provider.Condition, diagnosis.Provider.Fix)
	}
	return nil
}

func listRemoteCredentials(ctx context.Context, client credentialclient.Client, scenarioID string) ([]credentialclient.CredentialRef, error) {
	refs, err := client.List(ctx)
	if err != nil {
		return nil, err
	}
	prefix := "vrooli/" + strings.TrimSpace(scenarioID)
	filtered := make([]credentialclient.CredentialRef, 0, len(refs))
	for _, ref := range refs {
		if ref.LogicalID == prefix {
			filtered = append(filtered, ref)
		}
	}
	return filtered, nil
}

// CredentialField converts a bundle secret key into the durable field name the
// credential authority addresses. It matches the normalization the deploy path
// uses, so a value written here is a value the scenario can read back.
func CredentialField(key string) string {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return ""
	}
	return strings.ToLower(strings.NewReplacer("_", "-", ".", "-").Replace(trimmed))
}

// AddSecretToVPS adds a new secret to the target credential authority.
// Returns an error if the key already exists.
func AddSecretToVPS(
	ctx context.Context,
	sshRunner ssh.Runner,
	cfg ssh.Config,
	workdir string,
	key string,
	value string,
	scenarioID string,
) error {
	client, err := newRemoteCredentialClient(sshRunner, cfg)
	if err != nil {
		return err
	}
	identity := "vrooli/" + strings.TrimSpace(scenarioID)
	field := CredentialField(key)
	status, err := client.Status(ctx, identity, field)
	if err != nil {
		return err
	}
	if status.Configured {
		return fmt.Errorf("secret key %q already exists", key)
	}
	_, err = client.Provision(ctx, credentialclient.ProvisionRequest{Identity: identity, Field: field, Value: value})
	return err
}

// UpdateSecretOnVPS updates an existing secret value on the VPS.
// Returns an error if the key doesn't exist.
func UpdateSecretOnVPS(
	ctx context.Context,
	sshRunner ssh.Runner,
	cfg ssh.Config,
	workdir string,
	key string,
	value string,
	scenarioID string,
) error {
	client, err := newRemoteCredentialClient(sshRunner, cfg)
	if err != nil {
		return err
	}
	identity := "vrooli/" + strings.TrimSpace(scenarioID)
	field := CredentialField(key)
	status, err := client.Status(ctx, identity, field)
	if err != nil {
		return err
	}
	if !status.Configured {
		return fmt.Errorf("secret key %q not found", key)
	}
	_, err = client.Provision(ctx, credentialclient.ProvisionRequest{Identity: identity, Field: field, Value: value})
	return err
}

// DeleteSecretFromVPS removes a secret from the target credential authority.
// Returns an error if the key doesn't exist.
func DeleteSecretFromVPS(
	ctx context.Context,
	sshRunner ssh.Runner,
	cfg ssh.Config,
	workdir string,
	key string,
	scenarioID string,
) error {
	client, err := newRemoteCredentialClient(sshRunner, cfg)
	if err != nil {
		return err
	}
	identity := "vrooli/" + strings.TrimSpace(scenarioID)
	field := CredentialField(key)
	status, err := client.Status(ctx, identity, field)
	if err != nil {
		return err
	}
	if !status.Configured {
		return fmt.Errorf("secret key %q not found", key)
	}
	return client.Delete(ctx, identity, field)
}
