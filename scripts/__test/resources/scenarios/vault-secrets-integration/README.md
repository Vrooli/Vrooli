# Vault Secrets Integration

## 🎯 Overview

Technical integration test for secure secrets management between Vault and n8n workflow automation. This scenario validates enterprise-grade credential handling for automated workflows.

## 📋 Prerequisites

- Required resources: `vault`, `n8n`
- Optional resources: `postgres` (for audit trail logging)
- Vault must be unsealed and initialized
- n8n must have network access to Vault

## 🚀 Quick Start

```bash
# Run the integration test
./test.sh

# Run with custom Vault token
VAULT_TOKEN=your-token ./test.sh
```

## 💼 Business Value

### Use Cases
- Secure API key management for workflows
- Dynamic credential rotation
- Compliance-driven secret handling
- Zero-trust automation architectures

### Target Market
- DevOps and security teams
- Enterprises with strict security requirements
- Workflow automation platforms
- Regulated industries

### Revenue Potential
- Integration projects: $2,000 - $8,000
- Enterprise support: $500 - $2,000/month

## 🔧 Technical Details

### Architecture
1. **Vault Setup**: AppRole authentication for n8n
2. **Secret Storage**: Hierarchical secret paths
3. **n8n Integration**: Dynamic secret retrieval
4. **Audit Trail**: Complete access logging

### Integration Flow
1. n8n authenticates to Vault using AppRole
2. Retrieves required secrets for workflow
3. Uses secrets in API calls or integrations
4. Audit trail captures all access
5. Optional: Automatic secret rotation

## 🔌 Resource Integration

### Directory Structure
```
vault-secrets-integration/
├── metadata.yaml
├── test.sh
├── README.md
└── resources/
    ├── n8n/
    │   └── vault-integration-workflow.json    # Pre-configured n8n workflow
    ├── postgres/
    │   ├── schema.sql                         # Audit logging schema
    │   └── seed.sql                           # Sample audit data
    └── config/
        └── .env.template                      # Configuration template
```

### n8n Workflow
The included workflow (`vault-integration-workflow.json`) provides:
- Webhook trigger for testing Vault integration
- HTTP request node configured for Vault API
- Secret data processing with security masking
- Success/error response handling

**Import Instructions:**
1. Open n8n web interface
2. Navigate to Workflows → Import
3. Select `resources/n8n/vault-integration-workflow.json`
4. Update environment variables in workflow settings
5. Activate the workflow

### PostgreSQL Audit Database
The optional PostgreSQL integration adds comprehensive audit logging:
- **Schema**: Creates `vault_audit` schema with access and rotation logs
- **Features**: Compliance reporting, performance metrics, retention policies
- **Setup**:
  ```bash
  # Apply schema
  psql -h $POSTGRES_HOST -U $POSTGRES_USER -d $POSTGRES_DB -f resources/postgres/schema.sql
  
  # Load sample data (optional)
  psql -h $POSTGRES_HOST -U $POSTGRES_USER -d $POSTGRES_DB -f resources/postgres/seed.sql
  ```

### Configuration
Copy and customize the environment template:
```bash
cp resources/config/.env.template resources/config/.env
# Edit with your Vault token, n8n API key, and database credentials
```

## 🧪 Test Coverage

This scenario validates:
- ✅ Vault authentication mechanisms
- ✅ Secret retrieval in workflows
- ✅ Error handling for missing secrets
- ✅ Audit trail functionality
- ✅ Secret rotation capabilities

## 📊 Success Metrics

- Authentication success rate: 100%
- Secret retrieval latency: <100ms
- Audit completeness: 100%
- Rotation automation: Zero-downtime

## 🚧 Troubleshooting

| Issue | Solution |
|-------|----------|
| Auth failures | Check AppRole credentials and policies |
| Network errors | Verify Vault URL and connectivity |
| Permission denied | Review Vault policies for secret paths |
| Seal status | Ensure Vault is unsealed before testing |

## 🔒 Security Best Practices

1. Use AppRole authentication (not root tokens)
2. Implement least-privilege policies
3. Enable audit logging
4. Rotate credentials regularly
5. Use TLS for all communications

## 🏷️ Tags

`secrets-management`, `security`, `devops`, `basic-scenario`, `integration-test`