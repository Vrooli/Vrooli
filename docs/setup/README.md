# Getting Started with Vrooli

This guide will walk you through setting up any Vrooli repository for development. If you want to deploy the project instead, follow the [deployment guide](/docs/deployment/README.md).

## Setup Process Overview

Follow these guides in sequence for a complete setup:

1. [Prerequisites](prerequisites.md) - Install required software and configure your environment
2. [Repo Setup](repo_setup.md) - Clone and configure the repository
3. [Working with Docker](working_with_docker.md) - Learn how to use Docker for development

## Additional Services

Depending on your needs, you may need to configure these additional services:

| Service | Description |
|---------|-------------|
| [Remote Development](remote_setup.md) | Set up development on a remote server |
| [S3 Storage](s3_setup.md) | Configure S3 or compatible service for file storage |
| [Messaging Services](messenger_setup.md) | Set up email and push notifications |
| [Stripe Integration](stripe_setup.md) | Enable payment processing |
| [OAuth Authentication](oauth_setup.md) | Configure social login providers |

## Quick Docker Commands

For rapid development, here are the most commonly used Docker commands:

```bash
# Start all services in development mode
docker-compose up -d

# View logs of all containers
docker-compose logs -f

# Stop all services
docker-compose down

# Rebuild and restart with a fresh database
docker-compose down && rm -rf ./data/postgres && docker-compose up --build -d
```

For more detailed Docker instructions, see the [Working with Docker](working_with_docker.md) guide. 