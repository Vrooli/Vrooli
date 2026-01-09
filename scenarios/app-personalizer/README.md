# App Personalizer

## Purpose
Customize generated apps using prompts or digital twins for personalized experiences. This scenario enables dynamic modification of apps based on user personas, brand assets, and behavioral preferences.

## Core Capabilities
- **Backup & Versioning**: Creates backups before modifications
- **Digital Twin Integration**: Fetches persona data for personalization
- **Brand Asset Application**: Applies brand themes and assets
- **AI-Powered Modifications**: Uses Claude Code to intelligently modify app code
- **Multi-Deployment Modes**: Supports copy, patch, and multi-tenant deployments
- **Validation & Testing**: Ensures modified apps remain functional

- **Required Resources**: Ollama (llama3.2, codellama), PostgreSQL
- **Optional Resources**: Claude Code, MinIO
- **External Scenarios**: Leverages personal-digital-twin and brand-manager when available

## How It Helps Other Scenarios
- Provides personalization API that other scenarios can call
- Enables white-labeling and branding for any generated app
- Creates tenant-specific versions for SaaS scenarios
- Stores personalization history for analytics scenarios

## Architecture
- **API**: Go-based REST API for orchestration
- **CLI**: Shell-based CLI for local operations
- **Workflows**: In-API personalization pipeline without an external workflow engine
- **Storage**: PostgreSQL for metadata, MinIO for app versions

## Usage
```bash
# CLI usage
app-personalizer personalize --app /path/to/app --persona digital-twin-id
app-personalizer brand --app /path/to/app --brand brand-id
app-personalizer list-personalizations

# API endpoints
POST /api/personalize - Start personalization process
GET /api/apps - List personalized apps
POST /api/backup - Create app backup
POST /api/validate - Validate personalized app
```

## Personalization Types
1. **UI Theme**: Colors, fonts, logos from brand assets
2. **Content**: Default text, prompts, welcome messages
3. **Behavior**: AI personality, interaction styles
4. **Structure**: Menu items, feature toggles

## UX Style
Professional business application with clean, modern interface focused on configuration and monitoring rather than end-user interaction.
