# UI Scripts Directory

This directory contains utility scripts and build tools for the Scenario Generator V1 user interface.

## Contents

- **build-scripts/**: Production build and optimization scripts
- **dev-tools/**: Development utilities and helpers  
- **test-utils/**: Testing utilities and mock data generators
- **deployment/**: UI deployment and configuration scripts

## Usage

Most scripts are automatically invoked by npm/yarn commands defined in package.json:

```bash
# Development server
npm start

# Production build  
npm run build

# Run tests
npm test

# Lint and format
npm run lint
npm run format
```

## Available Scripts

### Development
- `start-dev.sh` - Start development server with hot reload
- `mock-api.sh` - Start mock API server for development
- `test-watch.sh` - Run tests in watch mode

### Build & Deployment  
- `build-prod.sh` - Production build with optimizations
- `analyze-bundle.sh` - Bundle size analysis and optimization tips
- `deploy-staging.sh` - Deploy to staging environment

### Utilities
- `generate-types.sh` - Generate TypeScript types from API schema
- `update-deps.sh` - Update dependencies and check for security issues  
- `clean.sh` - Clean build artifacts and node_modules

## Environment Variables

Scripts may reference these environment variables:

- `REACT_APP_API_URL` - API base URL (default: http://localhost:${API_PORT})
- `REACT_APP_N8N_URL` - N8N webhook URL (default: http://localhost:${N8N_PORT}) 
- `NODE_ENV` - Environment (development/production)
- `PORT` - Development server port (default: 3000)

## Custom Script Guidelines

When adding new scripts:

1. Make scripts executable: `chmod +x script-name.sh`
2. Include error handling with `set -euo pipefail` 
3. Add descriptive help text with `--help` flag
4. Use environment variables for configuration
5. Log important actions with timestamps
6. Clean up temporary files on exit

## Integration Points

UI scripts integrate with:

- **API Server**: Go backend on port 8080
- **N8N Workflows**: Webhook endpoints for generation
- **Database**: PostgreSQL for data persistence  
- **Build System**: React build tools and bundlers