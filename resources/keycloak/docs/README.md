# Keycloak Resource

Enterprise-grade identity and access management solution for modern applications and services.

## Overview

Keycloak provides comprehensive identity management capabilities including user authentication, authorization, single sign-on (SSO), and identity federation. This resource runs Keycloak in a Docker container with full REST API access for automation and integration.

## Features

- **User Management**: Create, manage, and authenticate users
- **Single Sign-On (SSO)**: Centralized authentication across applications
- **Identity Federation**: Integration with LDAP, Active Directory, and social providers
- **Role-Based Access Control**: Fine-grained permissions and authorization
- **Multi-tenancy**: Support for multiple realms (organizations)
- **Standards Compliance**: OpenID Connect, OAuth 2.0, SAML 2.0
- **Admin REST API**: Full programmatic control
- **Realm Import/Export**: Configuration management and deployment

## Usage

### Start the service
```bash
vrooli resource keycloak start
# or
resource-keycloak start
```

### Check status
```bash
vrooli resource keycloak status
# JSON format
vrooli resource keycloak status --format json
# Verbose details
vrooli resource keycloak status --verbose
```

### Access Keycloak
- **Application**: http://localhost:8070
- **Admin Console**: http://localhost:8070/admin
- **Default Credentials**: admin/admin

### Realm Management

#### Import realm configuration
```bash
# Import a realm from JSON file
resource-keycloak inject my-realm.json
```

#### List realms and statistics
```bash
resource-keycloak list
```

#### Clear all data
```bash
resource-keycloak clear
```

### REST API Usage

Once running, you can interact with Keycloak's Admin REST API:

#### Get admin token
```bash
curl -X POST http://localhost:8070/realms/master/protocol/openid-connect/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "username=admin" \
  -d "password=admin" \
  -d "grant_type=password" \
  -d "client_id=admin-cli"
```

#### Create a new realm
```bash
curl -X POST http://localhost:8070/admin/realms \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "realm": "my-app",
    "enabled": true,
    "displayName": "My Application"
  }'
```

#### Create a user
```bash
curl -X POST http://localhost:8070/admin/realms/my-app/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "enabled": true,
    "firstName": "Test",
    "lastName": "User",
    "email": "test@example.com",
    "credentials": [{
      "type": "password",
      "value": "password123",
      "temporary": false
    }]
  }'
```

## Configuration Files

### Realm Configuration Format
```json
{
  "realm": "my-application",
  "enabled": true,
  "displayName": "My Application",
  "registrationAllowed": true,
  "passwordPolicy": "length(8)",
  "sslRequired": "external",
  "users": [
    {
      "username": "admin",
      "enabled": true,
      "firstName": "Admin",
      "lastName": "User",
      "email": "admin@example.com",
      "credentials": [{
        "type": "password",
        "value": "admin123",
        "temporary": false
      }],
      "realmRoles": ["admin"]
    }
  ],
  "clients": [
    {
      "clientId": "my-app-client",
      "enabled": true,
      "publicClient": false,
      "secret": "client-secret-here",
      "redirectUris": ["http://localhost:3000/*"],
      "webOrigins": ["http://localhost:3000"]
    }
  ],
  "roles": {
    "realm": [
      {
        "name": "admin",
        "description": "Administrator role"
      },
      {
        "name": "user",
        "description": "Regular user role"
      }
    ]
  }
}
```

### Example Client Configuration
```json
{
  "clientId": "vrooli-web-app",
  "name": "Vrooli Web Application",
  "enabled": true,
  "publicClient": true,
  "protocol": "openid-connect",
  "redirectUris": [
    "http://localhost:3000/*",
    "https://app.vrooli.com/*"
  ],
  "webOrigins": [
    "http://localhost:3000",
    "https://app.vrooli.com"
  ],
  "attributes": {
    "post.logout.redirect.uris": "http://localhost:3000/logout"
  }
}
```

## Integration with Vrooli

Keycloak integrates seamlessly with other Vrooli resources:

### Authentication for Web Apps
- Protect **React/Next.js** applications with OpenID Connect
- Secure **FastAPI/Node.js** backends with JWT token validation
- Enable SSO across multiple applications

### API Gateway Integration
- Use with **Kong** or **NGINX** for API authentication
- Implement OAuth 2.0 flows for third-party integrations
- Token introspection for microservices

### Database Integration
- Store user sessions in **Redis**
- Audit logs in **PostgreSQL**
- User analytics in **QuestDB**

### Automation Workflows
- **N8n** workflows for user onboarding
- **Huginn** agents for security monitoring
- Automated provisioning with **Ansible**

## Security Considerations

### Development Settings
- Default admin credentials (change in production)
- HTTP enabled (use HTTPS in production)
- Permissive hostname settings
- H2 database (use PostgreSQL in production)

### Production Recommendations
1. **Database**: Use PostgreSQL or MySQL
2. **HTTPS**: Enable SSL/TLS encryption
3. **Credentials**: Use strong admin passwords
4. **Network**: Restrict access with firewalls
5. **Backup**: Regular realm exports
6. **Monitoring**: Enable audit logging

## Troubleshooting

### Common Issues

#### Container won't start
```bash
# Check Docker daemon
docker info

# Check container logs
docker logs vrooli-keycloak

# Check port conflicts
lsof -i :8070
```

#### Can't access admin console
```bash
# Verify service is running
resource-keycloak status

# Check health endpoint
curl http://localhost:8070/health/ready

# Reset admin password
docker exec vrooli-keycloak /opt/keycloak/bin/kc.sh export --help
```

#### Import fails
```bash
# Validate JSON format
jq . realm-config.json

# Check file permissions
ls -la realm-config.json

# Manual import via admin console
# Use the web interface at http://localhost:8070/admin
```

### Performance Tuning

#### Memory Settings
Default JVM settings: `-Xms512m -Xmx1024m`

For high-load environments:
```bash
# Edit config/defaults.sh
KEYCLOAK_JVM_OPTS="-Xms1g -Xmx2g -XX:+UseG1GC"
```

#### Database Connection Pool
For production workloads, configure connection pooling:
```bash
# Environment variables in container
KC_DB_POOL_INITIAL_SIZE=20
KC_DB_POOL_MAX_SIZE=100
KC_DB_POOL_MIN_SIZE=10
```

## Health Endpoints

- `GET /health/ready` - Readiness check
- `GET /health/live` - Liveness check  
- `GET /metrics` - Prometheus metrics (if enabled)

## API Documentation

Full API documentation available at:
- **Admin REST API**: http://localhost:8070/admin/master/console/#/
- **OpenAPI Spec**: Available in the admin console

## Port Configuration

Default port: 8070 (configured via port-registry)

To change the port, update `/scripts/resources/port_registry.sh` and restart the service.

## Support

For Keycloak-specific issues:
- [Official Documentation](https://www.keycloak.org/documentation)
- [Admin REST API Guide](https://www.keycloak.org/docs-api/latest/rest-api/)
- [Server Administration Guide](https://www.keycloak.org/docs/latest/server_admin/)

For Vrooli integration issues:
- Check logs in `~/.vrooli/keycloak/logs/`
- Use `resource-keycloak status --verbose` for diagnostics
- Review container logs with `docker logs vrooli-keycloak`