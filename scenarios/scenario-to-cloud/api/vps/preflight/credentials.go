package preflight

import (
	"context"
	"strings"

	"scenario-to-cloud/credentials"
	"scenario-to-cloud/domain"

	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

// CredentialValidator validates credentials for a specific resource type.
// Implement this interface to add support for new database/service types.
type CredentialValidator interface {
	ResourceName() string
	CheckID() string
	Title() string
	Validate(ctx context.Context, obs observer, manifest domain.CloudManifest, client credentialclient.Client) domain.PreflightCheck
}

var credentialValidators = []CredentialValidator{
	&PostgresCredentialValidator{},
}

// RunCredentialValidation checks credential authority state without reading a
// plaintext secrets file. The remote authority is the only source of truth;
// preflight may inspect presence and provider health, but never materializes a
// credential into the cloud API process or a command string. The authority is
// asked through the bound reach's typed credential verbs.
func RunCredentialValidation(ctx context.Context, obs observer, manifest domain.CloudManifest) []domain.PreflightCheck {
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

	client, err := credentials.NewReachCredentialClient(obs.reach, obs.target)
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
			checks = append(checks, validator.Validate(ctx, obs, manifest, client))
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

// PostgresCredentialValidator checks the database password binding and
// whether the database is listening.
type PostgresCredentialValidator struct{}

func (v *PostgresCredentialValidator) ResourceName() string { return "postgres" }
func (v *PostgresCredentialValidator) CheckID() string      { return "postgres_credentials" }
func (v *PostgresCredentialValidator) Title() string        { return "PostgreSQL credentials" }

// postgresPort is the resource's listening port on the target.
const postgresPort = 5433

func (v *PostgresCredentialValidator) Validate(ctx context.Context, obs observer, manifest domain.CloudManifest, client credentialclient.Client) domain.PreflightCheck {
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
	res, err := obs.listeningSockets(ctx, postgresPort)
	if ok(res, err) && strings.TrimSpace(res.Stdout) != "" {
		return domain.PreflightCheck{ID: v.CheckID(), Title: v.Title(), Status: domain.PreflightPass, Details: "PostgreSQL credential is configured and the database port is listening", Data: map[string]string{"database": dbName}}
	}
	if ok(res, err) {
		return domain.PreflightCheck{ID: v.CheckID(), Title: v.Title(), Status: domain.PreflightWarn, Details: "PostgreSQL credential is configured but the database is not listening", Hint: "Deployment will start PostgreSQL before connecting"}
	}
	return domain.PreflightCheck{ID: v.CheckID(), Title: v.Title(), Status: domain.PreflightWarn, Details: "PostgreSQL credential is configured; database readiness could not be confirmed", Data: map[string]string{"database": dbName}}
}
