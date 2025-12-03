package main

import "fmt"

// SecretTemplateFormat represents supported template output formats.
type SecretTemplateFormat string

const (
	SecretTemplateFormatEnv   SecretTemplateFormat = "env"
	SecretTemplateFormatVault SecretTemplateFormat = "vault"
	SecretTemplateFormatAWS   SecretTemplateFormat = "aws"
)

// SecretTemplateParams holds parameters for generating secret templates.
type SecretTemplateParams struct {
	ProfileID string
	Scenario  string
	TierName  string
}

// GenerateSecretTemplate generates a secret template in the specified format.
// Returns the template string and an error if the format is unsupported.
func GenerateSecretTemplate(format SecretTemplateFormat, params SecretTemplateParams) (string, error) {
	switch format {
	case SecretTemplateFormatEnv:
		return generateEnvTemplate(params), nil
	case SecretTemplateFormatVault:
		return generateVaultTemplate(params), nil
	case SecretTemplateFormatAWS:
		return generateAWSTemplate(params), nil
	default:
		return "", fmt.Errorf("unsupported format %q (supported: env, vault, aws)", format)
	}
}

func generateEnvTemplate(params SecretTemplateParams) string {
	return `# Deployment Manager Secret Template
# Generated for profile: ` + params.ProfileID + `
# Scenario: ` + params.Scenario + `
# Tier: ` + params.TierName + `

# Database connection string (required)
# Example: postgresql://user:pass@localhost:5432/dbname
DATABASE_URL=

# API key for third-party services (optional)
API_KEY=

# Enable debug mode (dev-only, set to 'true' for verbose logs)
DEBUG_MODE=false
`
}

func generateVaultTemplate(params SecretTemplateParams) string {
	return `{
  "secrets": [
    {
      "path": "secret/data/deployment-manager/` + params.ProfileID + `/database",
      "key": "DATABASE_URL",
      "description": "PostgreSQL connection string"
    },
    {
      "path": "secret/data/deployment-manager/` + params.ProfileID + `/api",
      "key": "API_KEY",
      "description": "Third-party API key"
    }
  ]
}`
}

func generateAWSTemplate(params SecretTemplateParams) string {
	return `{
  "secrets": [
    {
      "arn": "arn:aws:secretsmanager:us-east-1:123456789012:secret:deployment-manager/` + params.ProfileID + `/database",
      "key": "DATABASE_URL",
      "description": "PostgreSQL connection string"
    },
    {
      "arn": "arn:aws:secretsmanager:us-east-1:123456789012:secret:deployment-manager/` + params.ProfileID + `/api",
      "key": "API_KEY",
      "description": "Third-party API key"
    }
  ]
}`
}

// IsEnvFormat returns true if the format is the env template format.
func IsEnvFormat(format SecretTemplateFormat) bool {
	return format == SecretTemplateFormatEnv
}
