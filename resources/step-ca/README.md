# Step-CA Resource

Private certificate authority with ACME protocol support for automated certificate lifecycle management.

## Overview

Step-CA provides a modern, open-source certificate authority that enables:
- Automated certificate issuance via ACME protocol
- X.509 and SSH certificate generation
- Multiple authentication methods (OIDC, tokens, cloud APIs)
- Zero-trust architectures with mutual TLS
- HSM/KMS integration for secure key storage

## Quick Start

### Installation

```bash
# Install Step-CA
resource-step-ca manage install

# Start the service
resource-step-ca manage start --wait

# Check status
resource-step-ca status
```

### Basic Usage

```bash
# View connection details
resource-step-ca credentials

# Issue a certificate
resource-step-ca content add --cn service.local --duration 30d

# List certificates
resource-step-ca content list

# Get certificate details
resource-step-ca content get --cn service.local
```

## ACME Integration

Step-CA provides an ACME server compatible with standard clients:

```bash
# ACME directory URL
https://localhost:9010/acme/acme/directory

# Using certbot
certbot certonly \
  --server https://localhost:9010/acme/acme/directory \
  --standalone \
  -d example.local

# Using acme.sh
acme.sh --issue \
  --server https://localhost:9010/acme/acme/directory \
  -d example.local \
  --standalone
```

## Configuration

Configuration is managed through environment variables in `config/defaults.sh`:

- `STEPCA_CA_NAME`: Certificate Authority name (default: "Vrooli CA")
- `STEPCA_DEFAULT_DURATION`: Default certificate lifetime (default: "24h")
- `STEPCA_MAX_DURATION`: Maximum certificate lifetime (default: "720h")
- `STEPCA_DB_TYPE`: Database backend (badger, bbolt, postgresql, mysql)

## Authentication Methods

### JWK (Default)
Token-based authentication using JSON Web Keys.

### OIDC
Integration with identity providers like Keycloak:
```bash
export STEPCA_REQUIRE_OIDC=true
export STEPCA_OIDC_CLIENT_ID="step-ca"
export STEPCA_OIDC_ISSUER="https://keycloak.local/auth/realms/vrooli"
```

### Cloud Provider
Support for AWS, GCP, and Azure instance identity.

## Testing

```bash
# Run smoke tests (quick validation)
resource-step-ca test smoke

# Run integration tests
resource-step-ca test integration

# Run all tests
resource-step-ca test all
```

## Directory Structure

```
step-ca/
├── cli.sh                 # Main CLI interface
├── config/
│   ├── defaults.sh        # Configuration defaults
│   ├── runtime.json       # Runtime configuration
│   └── schema.json        # Configuration schema
├── lib/
│   ├── core.sh           # Core functionality
│   └── test.sh           # Test functions
├── test/
│   ├── run-tests.sh      # Test runner
│   └── phases/           # Test phases
├── docs/                 # Documentation
├── examples/             # Usage examples
├── PRD.md               # Product requirements
└── README.md            # This file
```

## Security Considerations

1. **Root Certificate**: The root certificate is stored in `data/step-ca/certs/root_ca.crt` and must be distributed to all clients.

2. **Password Protection**: The CA password is generated automatically and stored in `data/step-ca/config/password.txt`.

3. **Network Security**: Step-CA runs on port 9010 (configurable) and uses TLS for all communications.

4. **Certificate Lifetime**: Default is 24 hours for security. Adjust `STEPCA_DEFAULT_DURATION` for different environments.

## Integration with Other Resources

Step-CA integrates with:
- **PostgreSQL**: For certificate storage (optional)
- **Vault**: For HSM key protection (optional)
- **Keycloak**: For OIDC authentication (optional)

## Troubleshooting

### Health Check Fails
```bash
# Check container logs
resource-step-ca logs

# Verify network connectivity
docker network ls | grep vrooli-network

# Check port availability
netstat -tlnp | grep 9010
```

### Certificate Issuance Fails
```bash
# Check CA initialization
ls -la ~/Vrooli/data/step-ca/config/

# Verify provisioner configuration
cat ~/Vrooli/data/step-ca/config/ca.json | jq .authority.provisioners
```

### ACME Errors
```bash
# Test ACME directory
curl -sk https://localhost:9010/acme/acme/directory

# Check ACME provisioner
resource-step-ca content execute list-provisioners
```

## Advanced Usage

### Custom Certificate Templates
Create custom certificate templates for specific use cases:
```bash
# Add custom template
docker exec vrooli-step-ca step ca provisioner add \
  --type JWK \
  --name custom \
  --x509-template custom.tpl
```

### Certificate Revocation
```bash
# Revoke a certificate
resource-step-ca content remove --cn service.local

# Generate CRL
resource-step-ca content execute generate-crl
```

### Multi-Provisioner Setup
Configure different authentication methods for different use cases:
```bash
# Add OIDC provisioner
resource-step-ca content execute add-provisioner \
  --type OIDC \
  --name keycloak \
  --client-id step-ca
```

## Performance Tuning

For high-volume certificate issuance:
1. Use PostgreSQL backend instead of embedded database
2. Increase cache size: `STEPCA_CACHE_SIZE=50000`
3. Adjust connection limits: `STEPCA_MAX_CONNECTIONS=500`
4. Use dedicated hardware/container resources

## Support

For issues or questions:
- Check the [Step-CA documentation](https://smallstep.com/docs/step-ca)
- Review the PRD.md for detailed requirements
- Run diagnostic tests: `resource-step-ca test all --verbose`