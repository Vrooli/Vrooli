# Privacy & Terms Generator

Generate legally compliant privacy policies, terms of service, and other legal documents for your business.

## Features

- **Multi-jurisdiction Support**: GDPR (EU), CCPA (US), UK-GDPR, PIPEDA (Canada), and more
- **Document Types**: Privacy Policy, Terms of Service, Cookie Policy, EULA
- **Template Freshness Tracking**: Know when templates were last updated
- **AI-Powered Customization**: Uses Ollama to adapt templates to your specific needs
- **Multiple Output Formats**: Markdown, HTML, PDF
- **Version History Tracking**: Track changes to generated documents over time
- **Semantic Search**: Find relevant legal clauses using Qdrant or PostgreSQL full-text search
- **Programmatic Access**: Full CLI and API for automation

## Quick Start

### Using the CLI

```bash
# Install the CLI
cd cli && ./install.sh

# Generate a privacy policy
privacy-terms-generator generate privacy \
  --business-name "My Company" \
  --jurisdiction EU \
  --email legal@company.com

# Generate in different formats
privacy-terms-generator generate terms \
  --business-name "SaaS Co" \
  --format pdf \
  --output terms.pdf

# Check template freshness
privacy-terms-generator list-templates --stale

# Update templates from web
privacy-terms-generator update-templates --force

# Generate cookie policy
privacy-terms-generator generate cookie \
  --business-name "Web App" \
  --jurisdiction US

# Search for relevant legal clauses
privacy-terms-generator search "data collection" \
  --type privacy \
  --limit 5

# View document version history
privacy-terms-generator history <document-id> \
  --limit 10
```

### Using the Web UI

1. Start the UI server: `vrooli scenario run privacy-terms-generator`
2. Open http://localhost:21150 in your browser
3. Fill in your business details
4. Generate your legal documents

## How It Works

1. **Templates**: Pre-built legal templates stored in PostgreSQL, sourced from authoritative legal resources
2. **Customization**: Ollama AI adapts templates to your specific business needs
3. **Compliance**: Tracks jurisdiction-specific requirements (GDPR, CCPA, etc.)
4. **Freshness**: Monitors template age and fetches updates from web when needed
5. **Storage**: All generated documents saved with version history

## Integration with Other Scenarios

This scenario provides legal document generation as a service to other Vrooli scenarios:

- **SaaS Billing Hub**: Auto-generates billing terms
- **App Deployment**: Provides required compliance docs
- **Multi-tenant Platforms**: Per-tenant legal documents

## Value Proposition

- **Cost Savings**: $5-15K in legal fees per business
- **Time Savings**: 100+ hours of legal research
- **Always Current**: Templates updated regularly from authoritative sources
- **Compound Value**: Every business scenario gets professional legal docs

## Architecture

- **Backend**: Shell scripts with direct resource CLI integration (no n8n dependency)
- **Storage**: PostgreSQL for templates and documents
- **AI**: Ollama for intelligent customization
- **UI**: Clean, professional web interface
- **CLI**: Full-featured command-line tool

## Important Note

⚠️ **Legal Disclaimer**: Generated documents should always be reviewed by a qualified attorney before use. This tool provides templates and guidance but is not a substitute for legal advice.

## Resources Used

- **PostgreSQL**: Template and document storage
- **Ollama**: AI-powered customization
- **Qdrant** (optional): Semantic search for clauses
- **Browserless** (optional): PDF generation