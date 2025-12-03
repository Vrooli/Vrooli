// Package secrets provides secret identification, templating, and validation.
package secrets

import "fmt"

// TemplateFormat represents supported template output formats.
type TemplateFormat string

const (
	TemplateFormatEnv   TemplateFormat = "env"
	TemplateFormatVault TemplateFormat = "vault"
	TemplateFormatAWS   TemplateFormat = "aws"
)

// TemplateParams holds parameters for generating secret templates.
type TemplateParams struct {
	ProfileID string
	Scenario  string
	TierName  string
}

// GenerateTemplate generates a secret template in the specified format.
// Returns the template string and an error if the format is unsupported.
func GenerateTemplate(format TemplateFormat, params TemplateParams) (string, error) {
	switch format {
	case TemplateFormatEnv:
		return generateEnvTemplate(params), nil
	case TemplateFormatVault:
		return generateVaultTemplate(params), nil
	case TemplateFormatAWS:
		return generateAWSTemplate(params), nil
	default:
		return "", fmt.Errorf("unsupported format %q (supported: env, vault, aws)", format)
	}
}

func generateEnvTemplate(params TemplateParams) string {
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

func generateVaultTemplate(params TemplateParams) string {
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

func generateAWSTemplate(params TemplateParams) string {
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
func IsEnvFormat(format TemplateFormat) bool {
	return format == TemplateFormatEnv
}
