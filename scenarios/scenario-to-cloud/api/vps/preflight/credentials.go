package preflight

import (
	"context"
	"fmt"
	"io"
	"strings"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/shellutil"
	"scenario-to-cloud/ssh"

	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

// CredentialValidator validates credentials for a specific resource type.
// Implement this interface to add support for new database/service types.
type CredentialValidator interface {
	ResourceName() string
	CheckID() string
	Title() string
	Validate(ctx context.Context, cfg ssh.Config, sshRunner ssh.Runner,
		manifest domain.CloudManifest, client credentialclient.Client) domain.PreflightCheck
}

var credentialValidators = []CredentialValidator{
	&PostgresCredentialValidator{},
}

// credentialSSHRunner adapts the cloud SSH seam to the typed credential
// client. Arguments are kept separate until this adapter must hand a command
// to the existing SSH runner; values, when provisioned elsewhere, travel only
// through stdin and never through this command string.
type credentialSSHRunner struct {
	runner ssh.Runner
	cfg    ssh.Config
}

func (r credentialSSHRunner) Run(ctx context.Context, _ string, args []string, stdin io.Reader) ([]byte, error) {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, shellutil.QuoteSingle(arg))
	}
	opts := ssh.DefaultRunOptions()
	if stdin != nil {
		data, err := io.ReadAll(stdin)
		if err != nil {
			return nil, err
		}
		opts.Stdin = data
	}
	result, err := r.runner.Run(ctx, r.cfg, strings.Join(quoted, " "), opts)
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return []byte(result.Stdout), fmt.Errorf("remote command exited %d: %s", result.ExitCode, result.Stderr)
	}
	return []byte(result.Stdout), nil
}

// RunCredentialValidation checks credential authority state without reading a
// plaintext secrets file. The remote authority is the only source of truth;
// preflight may inspect presence and provider health, but never materializes a
// credential into the cloud API process or an SSH command.
func RunCredentialValidation(
	ctx context.Context,
	cfg ssh.Config,
	sshRunner ssh.Runner,
	manifest domain.CloudManifest,
	_ string,
) []domain.PreflightCheck {
	requiredResources := make(map[string]bool, len(manifest.Dependencies.Resources))
	for _, resource := range manifest.Dependencies.Resources {
		requiredResources[resource] = true
	}
	hasValidatable := false
	for _, validator := range credentialValidators {
		if requiredResources[validator.ResourceName()] {
			hasValidatable = true
			break
		}
	}
	if !hasValidatable {
		return nil
	}

	target := strings.TrimSpace(cfg.User) + "@" + strings.TrimSpace(cfg.Host)
	client, err := credentialclient.NewClient(credentialclient.ClientOptions{
		RemoteTarget: target,
		RemoteRunner: credentialSSHRunner{runner: sshRunner, cfg: cfg},
	})
	if err != nil {
		return []domain.PreflightCheck{credentialStoreWarning("Unable to create credential client", err.Error())}
	}
	diagnosis, err := client.Doctor(ctx)
	if err != nil {
		return []domain.PreflightCheck{credentialStoreWarning("Credential authority unavailable", err.Error())}
	}
	if diagnosis.Provider.Condition != "available" {
		detail := diagnosis.Provider.Explanation
		if detail == "" {
			detail = diagnosis.Provider.Condition
		}
		return []domain.PreflightCheck{credentialStoreWarning("Credential authority unavailable", detail)}
	}

	checks := make([]domain.PreflightCheck, 0, len(credentialValidators))
	for _, validator := range credentialValidators {
		if requiredResources[validator.ResourceName()] {
			checks = append(checks, validator.Validate(ctx, cfg, sshRunner, manifest, client))
		}
	}
	return checks
}

func credentialStoreWarning(title, details string) domain.PreflightCheck {
	return domain.PreflightCheck{
		ID:      "credentials_authority",
		Title:   title,
		Status:  domain.PreflightWarn,
		Details: details,
		Hint:    "Initialize the target host credential authority before deployment",
	}
}

// ============================================
// PostgreSQL Credential Validator
// ============================================

type PostgresCredentialValidator struct{}

func (v *PostgresCredentialValidator) ResourceName() string { return "postgres" }
func (v *PostgresCredentialValidator) CheckID() string      { return "postgres_credentials" }
func (v *PostgresCredentialValidator) Title() string        { return "PostgreSQL credentials" }

func (v *PostgresCredentialValidator) Validate(
	ctx context.Context,
	cfg ssh.Config,
	sshRunner ssh.Runner,
	manifest domain.CloudManifest,
	client credentialclient.Client,
) domain.PreflightCheck {
	identity := "vrooli/" + strings.TrimSpace(manifest.Scenario.ID)
	status, err := client.Status(ctx, identity, "postgres-password")
	if err != nil {
		return domain.PreflightCheck{ID: v.CheckID(), Title: v.Title(), Status: domain.PreflightWarn, Details: "Could not inspect PostgreSQL credential status", Hint: err.Error()}
	}
	if status.ProviderState != "available" {
		return domain.PreflightCheck{ID: v.CheckID(), Title: v.Title(), Status: domain.PreflightWarn, Details: "Credential authority did not confirm PostgreSQL credential state", Hint: status.ProviderState}
	}
	if !status.Configured {
		return domain.PreflightCheck{ID: v.CheckID(), Title: v.Title(), Status: domain.PreflightWarn, Details: "POSTGRES_PASSWORD is not configured in the credential authority", Hint: "Password will be provisioned during deployment"}
	}

	dbName := "vrooli_" + strings.ReplaceAll(manifest.Scenario.ID, "-", "_")
	probe := fmt.Sprintf("pg_isready -h localhost -p 5433 -U vrooli -d %s 2>&1 || true", shellutil.QuoteSingle(dbName))
	result, _ := sshRunner.Run(ctx, cfg, probe, ssh.DefaultRunOptions())
	output := result.Stdout + "\n" + result.Stderr
	if strings.Contains(output, "accepting connections") {
		return domain.PreflightCheck{ID: v.CheckID(), Title: v.Title(), Status: domain.PreflightPass, Details: "PostgreSQL credential is configured and database is accepting connections", Data: map[string]string{"database": dbName}}
	}
	if strings.Contains(output, "no response") || strings.Contains(output, "Connection refused") || strings.Contains(output, "not found") {
		return domain.PreflightCheck{ID: v.CheckID(), Title: v.Title(), Status: domain.PreflightWarn, Details: "PostgreSQL credential is configured but database is not ready", Hint: "Deployment will start PostgreSQL before connecting"}
	}
	return domain.PreflightCheck{ID: v.CheckID(), Title: v.Title(), Status: domain.PreflightWarn, Details: "PostgreSQL credential is configured; database readiness could not be confirmed", Data: map[string]string{"database": dbName}}
}
