# Development Environment Setup

This guide covers setting up a development environment for Vrooli, ensuring a clean and isolated space for making and testing changes before they reach production.

## Overview

A proper development environment should:
1. Mirror production as closely as possible
2. Allow for rapid iteration and testing
3. Be isolated to prevent conflicts with other environments
4. Support debugging and monitoring

## Local Development Setup

### Prerequisites

- Git
- Node.js (v16.x or later)
- Yarn
- Docker and Docker Compose
- PostgreSQL client (for connecting to the database)
- Redis client (optional, for debugging cache)

### Setup Steps

1. **Clone the repository**:
   ```bash
   git clone https://github.com/Vrooli/Vrooli.git
   cd Vrooli
   ```

2. **Create development environment file**:
   ```bash
   cp .env-example .env-dev
   ```

3. **Edit the environment variables**:
   ```bash
   vim .env-dev
   ```
   
   Important variables to configure:
   - `DB_USER` and `DB_PASSWORD`: Database credentials
   - `JWT_PRIV` and `JWT_PUB`: JWT keys (generate with `./scripts/genJwt.sh`)
   - `OPENAI_API_KEY`: If using OpenAI features
   - `SITE_EMAIL_*`: For email functionality

4. **Start the development environment**:
   ```bash
   chmod +x ./scripts/*
   ./scripts/develop.sh
   ```

5. **Access the application**:
   - UI: http://localhost:3000
   - API: http://localhost:5329
   - GraphQL Playground: http://localhost:5329/graphql

## Visual Studio Code Configuration

Using VSCode with the proper extensions can enhance your development experience:

1. **Install recommended extensions**:
   - ESLint
   - Prettier
   - Docker
   - PostgreSQL
   - GitLens
   - Peacock (for visual differentiation between environments)

2. **Configure Peacock**:
   
   To visually distinguish between development and production environments, set up Peacock:
   
   - Open VSCode settings (File > Preferences > Settings)
   - Search for "Peacock"
   - Add color presets:
     - Development: `#42b883` (green)
     - Staging: `#f5a623` (orange)
     - Production: `#ff4444` (red)
   
   When working in different environments, set the color:
   - Command Palette (Ctrl+Shift+P)
   - "Peacock: Change to a Favorite Color"
   - Select the appropriate environment color

3. **Workspace Settings**:
   
   Create a `.vscode` folder with recommended settings:
   ```json
   // .vscode/settings.json
   {
     "editor.formatOnSave": true,
     "editor.codeActionsOnSave": {
       "source.fixAll.eslint": true
     },
     "typescript.tsdk": "node_modules/typescript/lib",
     "peacock.favoriteColors": [
       { "name": "Development", "value": "#42b883" },
       { "name": "Staging", "value": "#f5a623" },
       { "name": "Production", "value": "#ff4444" }
     ]
   }
   ```

## Database Management

### Connecting to Development Database

```bash
# Connect to PostgreSQL database
psql -h localhost -U site -d postgres
# Enter password from DB_PASSWORD in .env-dev
```

### Running Migrations

```bash
# Enter the server container
docker exec -it server bash

# Run migrations
cd packages/server
yarn migrate
```

### Creating Test Data

```bash
# Set CREATE_MOCK_DATA=true in .env-dev before starting the environment
# Or run the mock data script directly
docker exec -it server bash -c "cd packages/server && yarn create-mock-data"
```

## Docker Containers Management

### Viewing Logs

```bash
# View logs from all containers
docker-compose --env-file .env-dev logs

# View logs from a specific container
docker-compose --env-file .env-dev logs server

# Follow logs in real-time
docker-compose --env-file .env-dev logs -f
```

### Restarting Services

```bash
# Restart a specific service
docker-compose --env-file .env-dev restart server

# Restart all services
docker-compose --env-file .env-dev restart
```

### Accessing Container Shell

```bash
# Access shell in server container
docker exec -it server bash

# Access shell in UI container
docker exec -it ui sh
```

## Testing

### Running Tests Locally

```bash
# Run shared package tests
cd packages/shared
yarn test

# Run server tests
cd packages/server
yarn test

# Run UI tests
cd packages/ui
yarn test
```

### Testing with Different Environments

For testing with different configurations:

1. Create environment-specific files:
   ```bash
   cp .env-dev .env-dev-test
   ```

2. Edit the test environment:
   ```bash
   nano .env-dev-test
   ```

3. Start using the test environment:
   ```bash
   docker-compose --env-file .env-dev-test up -d
   ```

## Debugging

### Server Debugging

The server container exposes port 9229 for Node.js debugging:

1. **Start with debugging enabled**:
   ```bash
   # This is already configured in develop.sh
   ```

2. **Connect from VSCode**:
   Add this to `.vscode/launch.json`:
   ```json
   {
     "version": "0.2.0",
     "configurations": [
       {
         "type": "node",
         "request": "attach",
         "name": "Attach to Docker",
         "port": 9229,
         "address": "localhost",
         "localRoot": "${workspaceFolder}/packages/server",
         "remoteRoot": "/srv/app/packages/server",
         "outFiles": ["${workspaceFolder}/packages/server/dist/**/*.js"]
       }
     ]
   }
   ```

3. **Start debugging**:
   - Open the Debug panel in VSCode
   - Select "Attach to Docker"
   - Click the play button

### UI Debugging

Use browser developer tools:

1. Open Chrome DevTools (F12)
2. Navigate to Sources tab
3. Find files under webpack:// directory

## Best Practices

1. **Use feature branches**:
   ```bash
   git checkout -b feature/my-new-feature
   ```

2. **Commit often with clear messages**:
   ```bash
   git commit -m "Add user authentication for API endpoints"
   ```

3. **Keep .env files secure**:
   - Never commit .env files to git
   - Use .env-example as a template

4. **Regularly pull from the main branch**:
   ```bash
   git pull origin main
   ```

5. **Document changes**:
   - Update README.md when adding features
   - Comment complex code
   - Add TypeScript interfaces and JSDoc comments

6. **Test thoroughly**:
   - Write unit tests for new features
   - Manually test in the development environment

## Troubleshooting Development Environment

### Common Issues

1. **Container won't start**:
   ```bash
   # Check for port conflicts
   sudo lsof -i :3000
   sudo lsof -i :5329
   
   # Check Docker logs
   docker-compose --env-file .env-dev logs
   ```

2. **Database connection issues**:
   ```bash
   # Verify database is running
   docker ps | grep db
   
   # Check database logs
   docker logs db
   ```

3. **Node modules issues**:
   ```bash
   # Rebuild node modules
   docker-compose --env-file .env-dev down
   rm -rf node_modules
   rm -rf packages/*/node_modules
   docker-compose --env-file .env-dev up -d
   ```

4. **Hot reload not working**:
   ```bash
   # Restart the affected container
   docker-compose --env-file .env-dev restart ui
   ```

### Resetting the Environment

If you need a clean start:

```bash
# Stop all containers
docker-compose --env-file .env-dev down

# Remove volumes (WARNING: This deletes database data)
docker-compose --env-file .env-dev down -v

# Remove images
docker-compose --env-file .env-dev down --rmi all

# Start fresh
./scripts/develop.sh
```

## Remote Development Environment

For team collaboration, you might want to set up a shared development server:

1. Follow the [Single Server Deployment](./single_server.md) guide, but use the development branch
2. Use `./scripts/develop.sh` instead of `./scripts/deploy.sh`
3. Configure your domain with a development subdomain (e.g., dev.yourdomain.com)
4. Set environment variables appropriately for a shared environment

This allows team members to test changes in a shared space before pushing to production. 