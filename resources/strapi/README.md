# Strapi Resource

Open-source headless CMS built on Node.js that provides a customizable API and admin panel for content management.

## Quick Start

```bash
# Install Strapi
vrooli resource strapi manage install

# Start the service
vrooli resource strapi manage start --wait

# Check status
vrooli resource strapi status

# View admin credentials
vrooli resource strapi credentials
```

## Features

- **REST & GraphQL APIs**: Automatically generated from content types
- **Admin Panel**: Visual content management interface
- **Content Modeling**: Define custom content types through UI
- **Media Library**: Built-in file upload and management
- **Role-Based Access**: Granular permissions system
- **Plugin System**: Extend functionality with plugins
- **Multi-Database**: PostgreSQL, MySQL, SQLite support
- **Internationalization**: Multi-language content support

## Configuration

### Environment Variables

```bash
# Service Configuration
STRAPI_PORT=1337                    # Service port
STRAPI_HOST=0.0.0.0                # Bind address

# Database Configuration (Required)
POSTGRES_HOST=localhost            # PostgreSQL host
POSTGRES_PORT=5433                 # PostgreSQL port
POSTGRES_USER=postgres             # Database user
POSTGRES_PASSWORD=postgres         # Database password
STRAPI_DATABASE_NAME=strapi       # Database name

# Admin Configuration
STRAPI_ADMIN_EMAIL=admin@vrooli.local  # Admin email
STRAPI_ADMIN_PASSWORD=<auto>          # Admin password (auto-generated)

# Storage Configuration (Optional)
STORAGE_PROVIDER=local             # Storage provider (local/s3)
MINIO_ENDPOINT=localhost:9000     # MinIO endpoint for S3 storage
MINIO_ACCESS_KEY=                 # MinIO access key
MINIO_SECRET_KEY=                 # MinIO secret key
```

## Usage Examples

### Basic Content Management

```bash
# List content types
vrooli resource strapi content list

# Create content
vrooli resource strapi content add articles '{"title":"Hello World","content":"..."}'

# Get content
vrooli resource strapi content get articles 1

# Delete content
vrooli resource strapi content remove articles 1
```

### GraphQL Queries

```bash
# Execute GraphQL query
vrooli resource strapi content execute '{ articles { id title content } }'
```

### Lifecycle Management

```bash
# Install dependencies
vrooli resource strapi manage install

# Start service
vrooli resource strapi manage start --wait --timeout 60

# Stop service
vrooli resource strapi manage stop

# Restart service
vrooli resource strapi manage restart

# Uninstall (keep data)
vrooli resource strapi manage uninstall --keep-data
```

## Testing

```bash
# Run all tests
vrooli resource strapi test all

# Run specific test phase
vrooli resource strapi test smoke       # Quick health check
vrooli resource strapi test integration # Full functionality
vrooli resource strapi test unit        # Library functions
```

## API Endpoints

- **Admin Panel**: http://localhost:1337/admin
- **REST API**: http://localhost:1337/api
- **GraphQL**: http://localhost:1337/graphql
- **Health Check**: http://localhost:1337/health

## Integration with Other Resources

### PostgreSQL (Required)
Strapi requires PostgreSQL for data storage:
```bash
vrooli resource postgres manage start
```

### MinIO (Optional)
For S3-compatible media storage:
```bash
vrooli resource minio manage start
export STORAGE_PROVIDER=s3
export MINIO_ENDPOINT=localhost:9000
```

### Redis (Optional)
For caching and performance:
```bash
vrooli resource redis manage start
```

## Common Use Cases

### 1. Content API for Applications
```bash
# Create content types through admin panel
# Access via REST API
curl http://localhost:1337/api/articles
```

### 2. Multi-Channel Publishing
```bash
# Single content source for web, mobile, IoT
# GraphQL for flexible queries
```

### 3. Documentation Platform
```bash
# Markdown content with versioning
# Webhook integration for CI/CD
```

## Troubleshooting

### Service Won't Start
```bash
# Check logs
vrooli resource strapi logs --tail 100

# Verify PostgreSQL is running
vrooli resource postgres status

# Check port availability
lsof -i :1337
```

### Database Connection Issues
```bash
# Test PostgreSQL connection
PGPASSWORD=$POSTGRES_PASSWORD psql -h localhost -p 5433 -U postgres -d strapi

# Check environment variables
env | grep POSTGRES
```

### Admin Login Issues
```bash
# View credentials
vrooli resource strapi credentials

# Reset admin password (requires restart)
export STRAPI_ADMIN_PASSWORD=newpassword
vrooli resource strapi manage restart
```

## Performance Tuning

### Database Optimization
```bash
# Increase connection pool
export STRAPI_MAX_CONNECTIONS=20
```

### Caching
```bash
# Enable Redis caching
export STRAPI_CACHE_ENABLED=true
export REDIS_HOST=localhost
export REDIS_PORT=6380
```

### Rate Limiting
```bash
# Adjust rate limits
export STRAPI_RATE_LIMIT=200  # Requests per minute
```

## Security Considerations

1. **Change Default Credentials**: Always set custom admin password
2. **Enable HTTPS**: Use reverse proxy for SSL termination
3. **Configure CORS**: Restrict allowed origins in production
4. **API Permissions**: Set appropriate public/private endpoints
5. **File Upload Validation**: Configure allowed file types and sizes

## Advanced Configuration

### Custom Plugins
Place custom plugins in the Strapi project directory:
```bash
~/.vrooli/strapi/app/plugins/
```

### Environment-Specific Config
```bash
# Development
export NODE_ENV=development

# Production
export NODE_ENV=production
```

### Backup and Restore
```bash
# Backup database
pg_dump -h localhost -p 5433 -U postgres strapi > backup.sql

# Backup uploads
tar -czf uploads.tar.gz ~/.vrooli/strapi/uploads/

# Restore database
psql -h localhost -p 5433 -U postgres strapi < backup.sql
```

## Resource Metadata

- **Port**: 1337
- **Version**: 5.0
- **Dependencies**: PostgreSQL (required), Redis (optional), MinIO (optional)
- **Memory**: ~512MB baseline
- **Startup Time**: 20-30 seconds
- **Health Check**: GET /health

## Support

For issues or questions:
1. Check logs: `vrooli resource strapi logs`
2. Review configuration: `vrooli resource strapi info`
3. Consult [Strapi Documentation](https://docs.strapi.io/)
4. File issues in Vrooli repository