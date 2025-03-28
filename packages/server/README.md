# Vrooli Server

The backend service for Vrooli, built with Node.js, Express, and TypeScript.

## Overview

This package implements the core backend services for Vrooli, providing RESTful APIs, WebSocket support, and integration with various AI providers. It handles authentication, data persistence, and business logic for the platform.

## Technology Stack

- **Runtime**: Node.js
- **Framework**: Express
- **Language**: TypeScript
- **Database**: PostgreSQL
- **Caching**: Redis
- **Testing**: Mocha, Chai, Sinon
- **Documentation**: OpenAPI/Swagger
- **Task Queue**: Bull
- **WebSocket**: Socket.io

## Directory Structure

```
server/
├── src/
│   ├── actions/      # Business logic actions
│   ├── auth/         # Authentication logic
│   ├── builders/     # Query and object builders
│   ├── db/           # Database operations
│   ├── endpoints/    # API route handlers
│   ├── events/       # Event handlers
│   ├── getters/      # Data retrieval functions
│   ├── middleware/   # Express middleware
│   ├── models/       # Data models
│   ├── notify/       # Notification system
│   ├── services/     # External service integrations
│   ├── sockets/      # WebSocket handlers
│   ├── tasks/        # Bull queue tasks
│   ├── utils/        # Utility functions
│   ├── validators/   # Input validation
│   └── __test/      # Test files
├── config/          # Configuration files
└── scripts/         # Utility scripts
```

## Getting Started

1. Install dependencies:
   ```bash
   yarn install
   ```

2. Set up environment variables:
   ```bash
   cp ../../.env-example .env
   ```

3. Start development server:
   ```bash
   yarn dev
   ```

4. Run tests:
   ```bash
   yarn test
   ```

## Development

### Key Features

- **RESTful APIs**: Well-documented endpoints
- **WebSocket Support**: Real-time communication
- **Authentication**: JWT-based auth system
- **Rate Limiting**: API protection
- **Error Handling**: Standardized error responses
- **Logging**: Structured logging system
- **Monitoring**: Health checks and metrics
- **Background Tasks**: Queue-based processing
- **Notifications**: Real-time and email notifications
- **Data Validation**: Input and output validation
- **Database Operations**: 
  - Query building
  - Transaction management
  - Connection pooling
  - Migration support

### Testing

```bash
# Run all tests
yarn test

# Run unit tests
yarn test:unit

# Run integration tests
yarn test:integration

# Run specific test file
yarn test:file path/to/test

# Run tests with coverage
yarn test:coverage

# Run API endpoint tests
yarn test:api

# Run database tests
yarn test:db
```

## Building and Deployment

### Development

```bash
yarn dev
```

### Production

```bash
yarn build
yarn start
```

### Docker

Development:
```bash
docker-compose up server
```

Production:
```bash
docker-compose -f docker-compose-prod.yml up server
```

## Environment Variables

Key environment variables (see `.env-example` for full list):

- `PORT_SERVER`: Server port
- `DB_PASSWORD`: SQL Database password
- `REDIS_PASSWORD`: Redis password

## Contributing

Please refer to the main project's [CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines.

## Documentation

- [Architecture Overview](../../ARCHITECTURE.md)
- [API Documentation](../docs/api/README.md)
- [Database Schema](../db/README.md)

## Best Practices

- Use TypeScript decorators for routes
- Implement proper error handling
- Follow REST conventions
- Document all endpoints
- Write comprehensive tests
- Use dependency injection
- Implement proper logging
- Monitor performance metrics
- Handle edge cases
- Follow security best practices 