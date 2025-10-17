# ElectionGuard Resource

End-to-end verifiable voting infrastructure for secure, transparent elections.

## Overview

ElectionGuard provides cryptographic guarantees for democratic processes through homomorphic encryption, enabling vote secrecy while maintaining complete verifiability. This resource packages Microsoft's ElectionGuard SDK for use in governance scenarios, civic engagement simulations, and policy decision systems.

## Quick Start

```bash
# Install the resource
vrooli resource electionguard manage install

# Start ElectionGuard service
vrooli resource electionguard manage start

# Run a mock election
vrooli resource electionguard content execute mock-election

# Verify health
vrooli resource electionguard test smoke
```

## Core Features

- **End-to-End Verifiability**: Mathematical proof that votes are counted correctly
- **Homomorphic Encryption**: Vote secrecy with public verification
- **Guardian Key Ceremonies**: Distributed trust through threshold cryptography
- **Audit Capabilities**: Complete verification chain from ballot to tally
- **Vault Integration**: Secure key management through HashiCorp Vault
- **Database Export**: Export election data to PostgreSQL or QuestDB for analytics
- **REST API**: Full-featured HTTP API for all election operations

## CLI Commands

### Management Commands
```bash
vrooli resource electionguard help                    # Show help
vrooli resource electionguard manage install          # Install dependencies
vrooli resource electionguard manage start            # Start service
vrooli resource electionguard manage stop             # Stop service
vrooli resource electionguard manage restart          # Restart service
vrooli resource electionguard status                  # Check status
```

### Content Operations
```bash
# Create a new election
vrooli resource electionguard content create-election \
  --name "Sample Election" \
  --guardians 5 \
  --threshold 3

# Generate guardian keys
vrooli resource electionguard content generate-keys \
  --election-id <id>

# Encrypt a ballot
vrooli resource electionguard content encrypt-ballot \
  --election-id <id> \
  --ballot-file ballot.json

# Compute tally
vrooli resource electionguard content compute-tally \
  --election-id <id>

# Verify results
vrooli resource electionguard content verify \
  --election-id <id> \
  --receipt <voter-receipt>
```

### Testing
```bash
vrooli resource electionguard test smoke        # Quick health check
vrooli resource electionguard test integration  # Full functionality test
vrooli resource electionguard test all         # Run all tests
```

## API Endpoints

The ElectionGuard API is available at `http://localhost:18250` (configurable):

### Core Operations
- `GET /health` - Service health status with integration status
- `POST /api/v1/election/create` - Create new election
- `POST /api/v1/keys/generate` - Generate guardian keys (stored in Vault)
- `POST /api/v1/ballot/encrypt` - Encrypt ballot
- `POST /api/v1/tally/compute` - Compute election tally
- `GET /api/v1/verify/ballot` - Verify ballot receipt
- `GET /api/v1/election/<id>` - Get election details

### Vault Integration
- `GET /api/v1/vault/status` - Check Vault connection status
- `GET /api/v1/keys/retrieve` - Retrieve guardian keys from Vault

### Database Export
- `POST /api/v1/export/postgres` - Export election data to PostgreSQL
- `POST /api/v1/export/questdb` - Export metrics to QuestDB

## Configuration

Environment variables:
```bash
ELECTIONGUARD_PORT=18250           # API port
ELECTIONGUARD_VAULT_ENABLED=true   # Use Vault for secrets
ELECTIONGUARD_DB_ENABLED=false     # Database export (optional)
ELECTIONGUARD_LOG_LEVEL=info       # Logging level
```

## Integration Examples

### With Vault (Secret Management)
```bash
# Guardian keys are automatically stored in Vault
vrooli resource electionguard content generate-keys \
  --vault-path /electionguard/guardians
```

### With PostgreSQL (Analytics)
```bash
# Export election data for analysis
vrooli resource electionguard content export \
  --election-id <id> \
  --format postgres \
  --connection $DATABASE_URL
```

### With n8n (Automation)
```yaml
# n8n workflow for automated election processing
- Trigger: Election created
- Encrypt ballots as received
- Compute tally at close
- Publish verification receipts
```

## Mock Election Example

```bash
# Run complete mock election
vrooli resource electionguard content execute mock-election \
  --voters 1000 \
  --candidates 5 \
  --export-results
```

## Security Considerations

- Guardian keys are stored encrypted in Vault
- All API endpoints require authentication tokens
- Ballots are encrypted end-to-end
- Verification proofs are publicly auditable
- No plaintext votes are ever stored

## Troubleshooting

### Service won't start
```bash
# Check Python version (requires 3.9+)
python3 --version

# Verify dependencies
vrooli resource electionguard manage install --force

# Check logs
vrooli resource electionguard logs
```

### Verification fails
```bash
# Ensure all guardians are available
vrooli resource electionguard status --guardians

# Check threshold requirements
vrooli resource electionguard info
```

## Resources

- [ElectionGuard Documentation](https://www.electionguard.vote/)
- [Python SDK Reference](https://github.com/microsoft/electionguard-python)
- [Cryptographic Specification](https://github.com/microsoft/electionguard/wiki)