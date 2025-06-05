# syntax=docker/dockerfile:1.4

ARG PROJECT_DIR=/srv/app

### Base stage: install dependencies and build workspace
FROM node:18-alpine3.20 AS base
ARG PROJECT_DIR
ENV PROJECT_DIR=${PROJECT_DIR}
ENV PNPM_HOME="/pnpm"
ENV PATH="$PNPM_HOME:$PATH"
WORKDIR ${PROJECT_DIR}
ENV NODE_ENV=development

# Enable Corepack and prepare pnpm
RUN corepack enable && corepack prepare pnpm@latest --activate

# Copy manifest files and prime cache
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
# Copy workspace package.json files for workspace detection
COPY packages/server/package.json packages/server/package.json
COPY packages/jobs/package.json packages/jobs/package.json
COPY packages/ui/package.json packages/ui/package.json
COPY packages/shared/package.json packages/shared/package.json
RUN --mount=type=cache,id=pnpm-store,target=/usr/local/share/pnpm-store pnpm fetch

# Install all dependencies, including devDependencies, across all workspaces
RUN --mount=type=cache,id=pnpm-store,target=/usr/local/share/pnpm-store \
    pnpm install --frozen-lockfile

# Copy full source and build all packages
COPY . .
# Generate Prisma client for correct schema
RUN pnpm --filter @vrooli/prisma run generate -- --schema=packages/server/src/db/schema.prisma

# Build shared utilities
RUN pnpm --filter @vrooli/shared run build

# Build UI package (produces dist folder)
RUN pnpm --filter @vrooli/ui run build

# Build jobs package
RUN pnpm --filter @vrooli/jobs run build

# Build server package: shared, compile, and copy static files
RUN pnpm --filter @vrooli/server run build:shared \
    && pnpm --filter @vrooli/server run build:compile \
    && pnpm --filter @vrooli/server run copy


### Server image: run compiled server code
FROM base AS server
ENV NODE_ENV=development
WORKDIR ${PROJECT_DIR}
# Inherit all built code and dependencies from base
EXPOSE ${PORT_SERVER}
CMD ["node", "packages/server/dist/index.js"]


### Jobs image: run compiled jobs code
FROM base AS jobs
ENV NODE_ENV=development
WORKDIR ${PROJECT_DIR}
# Inherit all built code and dependencies from base
EXPOSE ${PORT_JOBS}
CMD ["node", "packages/jobs/dist/index.js"]


### UI development image: run Vite dev server (inherits pnpm from base)
FROM base AS ui-dev
WORKDIR ${PROJECT_DIR}
COPY --from=base ${PROJECT_DIR}/node_modules ./node_modules
COPY --from=base ${PROJECT_DIR} .
EXPOSE ${PORT_UI}
RUN corepack enable && corepack prepare pnpm@latest --activate
CMD ["pnpm", "--filter", "@vrooli/ui", "run", "dev", "--", "--host", "0.0.0.0", "--port", "${PORT_UI}"] 