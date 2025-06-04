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
chmod +x ./scripts/*
pnpm cache clean && ./scripts/setup.sh
```
**Read /scripts/setup.sh for available flags**

## Start the development environment:
```bash
# Normal start or restart
./scripts/develop.sh
# Rebuild and restart. Useful when you change config files like package.json
./scripts/develop.sh --build
# Force restart. Redownloads and rebuilds containers, keeping only volumes
./scripts/develop.sh --build --force-recreate
```


## Access local environment
- UI: http://localhost:{PORT_UI:3000}
- API: http://localhost:{PORT_API:5329}

## Build for deployment:
```bash
./scripts/develop.sh
```
**Read /scripts/develop.sh for available flags**

## Deploy (typically run in remote server), after sending the build to it:
```bash
./scripts/deploy.sh
```
**Read /scripts/deploy.sh for available flags**


# Testing commands

## Package-level unit/integration tests:
```bash
# Testing /packages/server
cd packages/server && pnpm build
# Testing /packages/jobs
cd packages/jobs && pnpm build
# Coverage tests on /packages/shared
cd packages/shared && pnpm test-coverage
```

## File-level unit-integration tests:
```bash
# Testing /packages/server/src/services/stripe.test.ts
clear && pnpm build-tests && npx dotenv -e ../../.env-test -- mocha --file dist/__test/setup.js dist/services/stripe.test.js
```

## Storybook:
```bash
cd packages/ui && pnpm storybook
```


# Docker

## Container logs
```bash
# View development logs from all containers
docker-compose --env-file .env-dev logs

# View production logs from a specific container
docker-compose --env-file .env-prod logs server

# Follow logs in real-time
docker-compose --env-file .env-dev logs -f
```

## Restarting services
```bash
# Restart a specific service
docker-compose --env-file .env-dev restart server

# Restart all services
docker-compose --env-file .env-dev restart
```

## Accessing Container Shell
```bash
# Access shell in server container
docker exec -it server bash

# Access shell in UI container
docker exec -it ui sh
```
