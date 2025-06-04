# Vrooli Jobs

Background job (cron job) processing for Vrooli.

## Overview

This package handles all background processing, scheduled tasks, and long-running operations for the Vrooli platform. It manages tasks such as data processing, notifications, scheduled operations, and integration with external services.

## Technology Stack

- **Runtime**: Node.js
- **Language**: TypeScript
- **Database**: Redis
- **Testing**: Mocha, Chai

## Directory Structure

```
jobs/
└── src/
    └── schedules/    # Cron jobs
```

## Getting Started

1. Install dependencies:
   ```bash
   pnpm install
   ```

2. Set up environment variables:
   ```bash
   cp ../../.env-example .env-dev
   ```

3. Start development server:
   ```bash
   pnpm dev
   ```

4. Run tests:
   ```bash
   pnpm test
   ```

## Job Types

### Background Jobs

- Cleaning up revoked sessions
- Verifying calculated database fields
- Updating stale search embeddings
- Generating sitemaps
- Scheduled moderation tools
- Credit issuing
- Payment failures/expirations
- Scheduled notifications

## Building and Deployment

### Development

```bash
pnpm dev
```

### Production

```bash
pnpm build
pnpm start
```

### Docker

Development:
```bash
docker-compose up jobs
```

Production:
```bash
docker-compose -f docker-compose-prod.yml up jobs
```

## Contributing

Please refer to the main project's [CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines.

## Documentation

- [Architecture Overview](../../ARCHITECTURE.md)
- [API Documentation](../docs/api/README.md)

## Best Practices

- Implement proper error reporting
- Implement proper logging 