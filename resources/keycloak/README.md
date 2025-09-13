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
- **Database**: PostgreSQL (production), H2 (fallback)
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

## Social Login Providers

```bash
# Add GitHub provider
vrooli resource keycloak social add-github \
  --client-id <your-github-client-id> \
  --client-secret <your-github-client-secret> \
  --realm master

# Add Google provider
vrooli resource keycloak social add-google \
  --client-id <your-google-client-id> \
  --client-secret <your-google-client-secret> \
  --realm master

# List configured providers
vrooli resource keycloak social list --realm master

# Test provider configuration
vrooli resource keycloak social test --alias github --realm master

# Remove a provider
vrooli resource keycloak social remove --alias github --realm master
```

## LDAP/AD Federation

```bash
# Add LDAP provider
vrooli resource keycloak ldap add \
  --url ldap://ldap.example.com:389 \
  --users-dn ou=users,dc=example,dc=com \
  --bind-dn cn=admin,dc=example,dc=com \
  --bind-password admin-password \
  --name my-ldap \
  --realm master

# Add Active Directory provider
vrooli resource keycloak ldap add \
  --url ldap://ad.example.com:389 \
  --users-dn CN=Users,DC=example,DC=com \
  --bind-dn admin@example.com \
  --bind-password admin-password \
  --name my-ad \
  --type ad \
  --realm master

# List configured LDAP/AD providers
vrooli resource keycloak ldap list --realm master

# Test connection
vrooli resource keycloak ldap test --name my-ldap --realm master

# Sync users (incremental)
vrooli resource keycloak ldap sync --name my-ldap --realm master

# Sync users (full)
vrooli resource keycloak ldap sync --name my-ldap --realm master --full

# Remove provider
vrooli resource keycloak ldap remove --name my-ldap --realm master
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