# Vrooli Database

**NOTE:** This folder is used exclusively by the database Docker container. The database schema, migrations, etc. are found [in the server](/packages/server/src/db).

## Technology Stack

- **Database**: PostgreSQL

## Directory Structure

```
db/
├── entrypoint/         # PostgreSQL initialization scripts (mounted to /docker-entrypoint-initdb.d/)
│   └── extensions.sh   # Installs required PostgreSQL extensions during first container initialization
```

The `entrypoint` directory is mounted as a Docker volume to PostgreSQL's `/docker-entrypoint-initdb.d/` directory. Any scripts in this directory are automatically executed when the database is first initialized. Currently, it includes:

- `extensions.sh`: Installs required PostgreSQL extensions:
  - `citext`: Case-insensitive text type for efficient text comparisons
  - `vector`: Vector similarity search support for AI/ML features

## Contributing

Please refer to the main project's [CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines.

## Documentation

- [API Documentation](../docs/api/README.md)
- [Database Schema](/packages/server/src/db/schema.prisma)
