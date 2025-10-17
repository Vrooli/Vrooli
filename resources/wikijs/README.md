# Wiki.js Resource

Modern wiki platform with Git storage backend for structured knowledge management and documentation.

## 🎯 Purpose

Wiki.js provides a centralized, version-controlled documentation platform that enables:
- **Scenario Documentation**: Structured docs for all scenarios with templates and automation
- **Knowledge Base**: Searchable reference for agents and developers
- **API Documentation**: Auto-generated docs from OpenAPI specs
- **Integration Guides**: Step-by-step guides for resource integration
- **Git Backend**: All content version-controlled and reviewable

## 🚀 Quick Start

```bash
# Install Wiki.js
vrooli resource wikijs install

# Check status
vrooli resource wikijs status

# Start Wiki.js
vrooli resource wikijs start

# Run tests
vrooli resource wikijs test
```

## 📦 Features

### Core Capabilities
- **GraphQL API**: Full programmatic access to create, update, and query content
- **REST API**: Alternative API for simpler integrations
- **Git Storage**: All content stored in Git for version control
- **Search**: Full-text search with ElasticSearch or built-in DB search
- **Authentication**: Multiple auth providers (local, LDAP, OAuth2, SAML)
- **Markdown Support**: Write in Markdown with live preview
- **Multi-language**: Support for internationalization

### Vrooli Integration
- **Auto-documentation**: Scenarios can auto-generate their docs
- **Agent Access**: Agents can query and update documentation
- **Workflow Integration**: n8n/Huginn workflows can maintain docs
- **Template System**: Consistent documentation across scenarios

## 📚 Documentation

See the following guides for detailed information:
- [Installation Guide](docs/installation.md)
- [Configuration Guide](docs/configuration.md)
- [API Reference](docs/api.md)
- [Integration Patterns](docs/integration.md)
- [Troubleshooting](docs/troubleshooting.md)

## 🔧 Configuration

Wiki.js uses environment variables for configuration:
- `WIKI_PORT`: HTTP port (default: from port_registry)
- `WIKI_DB_TYPE`: Database type (postgres)
- `WIKI_DB_HOST`: Database host
- `WIKI_DB_PORT`: Database port
- `WIKI_DB_NAME`: Database name
- `WIKI_DB_USER`: Database user
- `WIKI_DB_PASS`: Database password
- `WIKI_GIT_URL`: Git repository URL for content storage
- `WIKI_GIT_BRANCH`: Git branch (default: main)

## 🧪 Testing

Run the test suite:
```bash
vrooli resource wikijs test
```

Tests cover:
- Installation verification
- API connectivity (GraphQL and REST)
- Authentication flows
- Content CRUD operations
- Git synchronization
- Search functionality

## 📝 Examples

See the `examples/` directory for:
- `create-page.sh`: Create a new wiki page via API
- `bulk-import.sh`: Import multiple documents
- `search-content.sh`: Search wiki content
- `git-sync.sh`: Sync with Git repository

## 🔍 Troubleshooting

### Common Issues

**Wiki.js not starting**
- Check PostgreSQL is running: `vrooli resource postgres status`
- Verify port availability: `./scripts/resources/port_registry.sh list`
- Check logs: `docker logs wikijs`

**Git sync failing**
- Verify Git credentials in configuration
- Check network access to Git repository
- Ensure branch exists and is accessible

**Search not working**
- Rebuild search index from admin panel
- Check ElasticSearch if using external search
- Verify database permissions

## 📊 Resource Impact

Wiki.js enables these scenario types:
- **Documentation Automation**: Auto-generate and maintain docs
- **Knowledge Management**: Centralized information repository
- **Training Material**: Create and manage training content
- **API Documentation**: Auto-generate from OpenAPI specs
- **Runbooks**: Operational procedures and guides
- **Research Notes**: Collaborative research documentation

## 🔗 Dependencies

- **PostgreSQL**: Primary data storage
- **Git**: Content version control
- **Redis** (optional): Session storage and caching
- **ElasticSearch** (optional): Advanced search capabilities