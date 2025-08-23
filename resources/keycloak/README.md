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
vrooli resource keycloak install
vrooli resource keycloak start

# Check status
resource-keycloak status

# Access admin console
# http://localhost:8080
# Default: admin/admin
```

## Architecture

- **Port**: 8080 (HTTP), 8443 (HTTPS)
- **Database**: PostgreSQL (shared with platform)
- **Storage**: `/data/keycloak`
- **Config**: `/data/keycloak/config`

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