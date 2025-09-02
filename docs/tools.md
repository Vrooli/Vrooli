# Project commands

## Clone the repository:
```bash
git clone https://github.com/Vrooli/Vrooli.git
cd Vrooli
```

## Edit the environment variables:
```bash
vim .env-dev
```

## Setup full project based on environment variables:
```bash
pnpm cache clean && vrooli setup
```
**Run `vrooli setup --help` for available options**

## Start the development environment:
```bash
# Start development environment
vrooli develop
# Note: For rebuild options, use vrooli clean first, then vrooli develop
```


## Access local environment
- UI: http://localhost:{PORT_UI:3000}
- API: http://localhost:{PORT_API:5329}

## Build for deployment:
```bash
vrooli build
```
**Run `vrooli build --help` for available options**

## Deploy (typically run in remote server), after sending the build to it:
```bash
vrooli deploy
```
**Run `vrooli deploy --help` for available options**


# Testing commands

## Scenario Testing:
```bash
# Test individual scenarios
vrooli scenario test research-assistant
vrooli scenario test invoice-generator

# Test all scenarios
vrooli test scenarios
```

## Resource Testing:
```bash
# Test individual resources
resource-ollama test
resource-postgres test

# Test all resources
vrooli test resources
```

## Integration Testing:
```bash
# Run comprehensive integration tests
vrooli test integration
```


# Docker

## Container logs
```bash
# View development logs from all containers
docker compose --env-file .env-dev logs

# View production logs from a specific container
docker compose --env-file .env-prod logs server

# Follow logs in real-time
docker compose --env-file .env-dev logs -f
```

## Restarting services
```bash
# Restart a specific service
docker compose --env-file .env-dev restart server

# Restart all services
docker compose --env-file .env-dev restart
```

## Accessing Container Shell
```bash
# Access shell in server container
docker exec -it server bash

# Access shell in UI container
docker exec -it ui sh
```
