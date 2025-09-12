# Keycloak Resource

Enterprise-grade identity and access management with SSO, OIDC, SAML, and social login support.

## Overview

Keycloak provides centralized authentication and authorization for Vrooli scenarios requiring:
- Single Sign-On (SSO)
- Multi-tenant authentication
- OAuth2/OIDC integration
- SAML support
- Social login providers
- User federation (LDAP/AD)

## Quick Start

```bash
# Install and start
vrooli resource keycloak manage install
vrooli resource keycloak manage start --wait

# Check status
vrooli resource keycloak status

# Access admin console
# http://localhost:8070/admin
# Default: admin/admin

# Run tests
vrooli resource keycloak test all
```

## Architecture

- **Port**: 8070 (HTTP), 8443 (HTTPS - when configured)
- **Database**: H2 (development), PostgreSQL (production - planned)
- **Container**: keycloak/keycloak:latest
- **Network**: vrooli-network

## Content Management

```bash
# Create a realm
vrooli resource keycloak content add --file realm.json --type realm

# Add users to a realm
vrooli resource keycloak content add --file users.json --type user --name test-realm

# Add OAuth2 clients
vrooli resource keycloak content add --file client.json --type client --name test-realm

# List all realms and their content
vrooli resource keycloak content list

# Export a realm configuration
vrooli resource keycloak content get --name test-realm --output realm-export.json

# Remove a realm
vrooli resource keycloak content remove --name test-realm
```

## Documentation

- [Installation Guide](docs/installation.md)
- [Configuration](docs/configuration.md)
- [Integration Guide](docs/integration.md)
- [API Reference](docs/api.md)

## Use Cases

1. **Multi-tenant SaaS**: Isolate customer data with realms
2. **Enterprise SSO**: Connect LDAP/AD directories
3. **API Security**: OAuth2 token validation
4. **B2B Scenarios**: SAML federation
5. **Social Login**: GitHub, Google, Facebook integration