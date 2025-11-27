# Phase 1 Implementation Guide

Quick start guide for migrating the Playwright driver to TypeScript.

## Step 1: Project Setup

### Initialize TypeScript Project

```bash
cd api/automation/playwright-driver

# Initialize package.json
npm init -y

# Install dependencies
npm install playwright@^1.40.0 winston@^3.11.0 prom-client@^15.1.0 zod@^3.22.4

# Install dev dependencies
npm install -D \
  typescript@^5.3.0 \
  @types/node@^20.10.0 \
  eslint@^8.55.0 \
  @typescript-eslint/eslint-plugin@^6.15.0 \
  @typescript-eslint/parser@^6.15.0 \
  prettier@^3.1.1 \
  jest@^29.7.0 \
  @types/jest@^29.5.11 \
  ts-jest@^29.1.1 \
  ts-node@^10.9.2
```

### Configuration Files

#### `tsconfig.json`
```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "commonjs",
    "lib": ["ES2022"],
    "outDir": "./dist",
    "rootDir": "./src",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "resolveJsonModule": true,
    "declaration": true,
    "declarationMap": true,
    "sourceMap": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noImplicitReturns": true,
    "noFallthroughCasesInSwitch": true
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules", "dist", "tests"]
}
```

#### `tsconfig.build.json`
```json
{
  "extends": "./tsconfig.json",
  "exclude": ["node_modules", "dist", "tests", "**/*.test.ts"]
}
```

#### `.eslintrc.js`
```javascript
module.exports = {
  parser: '@typescript-eslint/parser',
  parserOptions: {
    ecmaVersion: 2022,
    sourceType: 'module',
    project: './tsconfig.json',
  },
  extends: [
    'eslint:recommended',
    'plugin:@typescript-eslint/recommended',
    'plugin:@typescript-eslint/recommended-requiring-type-checking',
  ],
  rules: {
    '@typescript-eslint/no-explicit-any': 'error',
    '@typescript-eslint/explicit-function-return-type': 'warn',
    '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
    '@typescript-eslint/no-floating-promises': 'error',
  },
};
```

#### `.prettierrc`
```json
{
  "semi": true,
  "trailingComma": "es5",
  "singleQuote": true,
  "printWidth": 100,
  "tabWidth": 2
}
```

#### `jest.config.js`
```javascript
module.exports = {
  preset: 'ts-jest',
  testEnvironment: 'node',
  roots: ['<rootDir>/tests'],
  testMatch: ['**/*.test.ts'],
  collectCoverageFrom: [
    'src/**/*.ts',
    '!src/**/*.d.ts',
    '!src/types/**',
  ],
  coverageThreshold: {
    global: {
      branches: 80,
      functions: 80,
      lines: 80,
      statements: 80,
    },
  },
};
```

#### Update `package.json` scripts
```json
{
  "scripts": {
    "build": "tsc -p tsconfig.build.json",
    "dev": "ts-node src/server.ts",
    "start": "node dist/server.js",
    "test": "jest",
    "test:watch": "jest --watch",
    "test:coverage": "jest --coverage",
    "lint": "eslint src tests --ext .ts",
    "lint:fix": "eslint src tests --ext .ts --fix",
    "format": "prettier --write 'src/**/*.ts' 'tests/**/*.ts'",
    "typecheck": "tsc --noEmit"
  }
}
```

## Step 2: Core Type Definitions

### `src/types/contracts.ts`

```typescript
/**
 * Type definitions matching Go automation/contracts package.
 * These must stay in sync with the Go contract definitions.
 */

export const STEP_OUTCOME_SCHEMA_VERSION = 'automation-step-outcome-v1';
export const PAYLOAD_VERSION = '1';

/**
 * CompiledInstruction - matches Go contracts.CompiledInstruction
 */
export interface CompiledInstruction {
  index: number;
  node_id: string;
  type: string;
  params: Record<string, unknown>;
  preload_html?: string;
  context?: Record<string, unknown>;
  metadata?: Record<string, string>;
}

/**
 * StepOutcome - matches Go contracts.StepOutcome
 */
export interface StepOutcome {
  schema_version: string;
  payload_version: string;
  execution_id?: string;
  correlation_id?: string;
  step_index: number;
  attempt: number;
  node_id: string;
  step_type: string;
  instruction?: string;
  success: boolean;
  started_at: string; // ISO 8601
  completed_at?: string; // ISO 8601
  duration_ms?: number;
  final_url?: string;
  screenshot?: Screenshot;
  dom_snapshot?: DOMSnapshot;
  console_logs?: ConsoleLogEntry[];
  network?: NetworkEvent[];
  extracted_data?: Record<string, unknown>;
  assertion?: AssertionOutcome;
  condition?: ConditionOutcome;
  failure?: StepFailure;
  notes?: Record<string, string>;
}

/**
 * StepFailure - matches Go contracts.StepFailure
 */
export interface StepFailure {
  kind: FailureKind;
  code?: string;
  message?: string;
  fatal?: boolean;
  retryable?: boolean;
  occurred_at?: string; // ISO 8601
  details?: Record<string, unknown>;
  source?: FailureSource;
}

export type FailureKind =
  | 'engine'
  | 'infra'
  | 'orchestration'
  | 'user'
  | 'timeout'
  | 'cancelled';

export type FailureSource = 'engine' | 'executor' | 'recorder';

export interface Screenshot {
  media_type?: string;
  capture_time?: string; // ISO 8601
  width?: number;
  height?: number;
  hash?: string;
  from_cache?: boolean;
  truncated?: boolean;
  source?: string;
}

export interface DOMSnapshot {
  html?: string;
  preview?: string;
  hash?: string;
  collected_at?: string; // ISO 8601
  truncated?: boolean;
}

export interface ConsoleLogEntry {
  type: 'log' | 'warn' | 'error' | 'info' | 'debug';
  text: string;
  timestamp: string; // ISO 8601
  stack?: string;
  location?: string;
}

export interface NetworkEvent {
  type: 'request' | 'response' | 'failure';
  url: string;
  method?: string;
  resource_type?: string;
  status?: number;
  ok?: boolean;
  failure?: string;
  timestamp: string; // ISO 8601
  request_headers?: Record<string, string>;
  response_headers?: Record<string, string>;
  request_body_preview?: string;
  response_body_preview?: string;
  truncated?: boolean;
}

export interface AssertionOutcome {
  mode?: string;
  selector?: string;
  expected?: unknown;
  actual?: unknown;
  success: boolean;
  negated?: boolean;
  case_sensitive?: boolean;
  message?: string;
}

export interface ConditionOutcome {
  type?: string;
  outcome: boolean;
  negated?: boolean;
  operator?: string;
  variable?: string;
  selector?: string;
  expression?: string;
  actual?: unknown;
  expected?: unknown;
}

export interface BoundingBox {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface Point {
  x: number;
  y: number;
}
```

### `src/types/session.ts`

```typescript
import { Page, BrowserContext, Browser } from 'playwright';

export type ReuseMode = 'fresh' | 'clean' | 'reuse';

export interface SessionSpec {
  execution_id: string;
  workflow_id: string;
  viewport: {
    width: number;
    height: number;
  };
  reuse_mode: ReuseMode;
  base_url?: string;
  labels?: Record<string, string>;
  required_capabilities?: {
    tabs?: boolean;
    iframes?: boolean;
    uploads?: boolean;
    downloads?: boolean;
    har?: boolean;
    video?: boolean;
    tracing?: boolean;
    viewport_width?: number;
    viewport_height?: number;
  };
}

export interface SessionState {
  id: string;
  browser: Browser;
  context: BrowserContext;
  page: Page;
  spec: SessionSpec;
  createdAt: Date;
  lastUsedAt: Date;
  tracing: boolean;
  video: boolean;
  harPath?: string;
  tracePath?: string;
  videoDir?: string;
  frameStack: string[]; // Stack of frame selectors for frame-switch
  tabStack: Page[]; // Stack of pages for multi-tab support
}

export interface SessionMetrics {
  totalSessions: number;
  activeSessions: number;
  idleSessions: number;
  avgSessionDuration: number;
  peakSessions: number;
}
```

### `src/types/instruction.ts`

```typescript
import { z } from 'zod';

/**
 * Parameter types for each instruction.
 * Uses Zod for runtime validation.
 */

export const NavigateParamsSchema = z.object({
  url: z.string(),
  timeoutMs: z.number().optional(),
  waitUntil: z.enum(['load', 'domcontentloaded', 'networkidle']).optional(),
});
export type NavigateParams = z.infer<typeof NavigateParamsSchema>;

export const ClickParamsSchema = z.object({
  selector: z.string(),
  timeoutMs: z.number().optional(),
  clickCount: z.number().optional(),
  button: z.enum(['left', 'right', 'middle']).optional(),
  modifiers: z.array(z.string()).optional(),
});
export type ClickParams = z.infer<typeof ClickParamsSchema>;

export const TypeParamsSchema = z.object({
  selector: z.string(),
  text: z.string().optional(),
  value: z.string().optional(),
  timeoutMs: z.number().optional(),
  delay: z.number().optional(),
});
export type TypeParams = z.infer<typeof TypeParamsSchema>;

export const WaitParamsSchema = z.object({
  selector: z.string().optional(),
  timeoutMs: z.number().optional(),
  ms: z.number().optional(),
  state: z.enum(['attached', 'detached', 'visible', 'hidden']).optional(),
});
export type WaitParams = z.infer<typeof WaitParamsSchema>;

export const AssertParamsSchema = z.object({
  selector: z.string(),
  mode: z.string().optional(),
  kind: z.string().optional(),
  expected: z.unknown().optional(),
  text: z.string().optional(),
  attribute: z.string().optional(),
  value: z.string().optional(),
  contains: z.boolean().optional(),
  timeoutMs: z.number().optional(),
});
export type AssertParams = z.infer<typeof AssertParamsSchema>;

// Add more schemas for other instruction types...
```

## Step 3: Configuration

### `src/config.ts`

```typescript
import { z } from 'zod';

const ConfigSchema = z.object({
  server: z.object({
    port: z.number().default(39400),
    host: z.string().default('127.0.0.1'),
    requestTimeout: z.number().default(120000),
    maxRequestSize: z.number().default(5 * 1024 * 1024),
  }),
  browser: z.object({
    headless: z.boolean().default(true),
    executablePath: z.string().optional(),
    args: z.array(z.string()).default([]),
  }),
  session: z.object({
    maxConcurrent: z.number().default(10),
    idleTimeoutMs: z.number().default(300000),
    poolSize: z.number().default(5),
    cleanupIntervalMs: z.number().default(60000),
  }),
  telemetry: z.object({
    screenshot: z.object({
      enabled: z.boolean().default(true),
      fullPage: z.boolean().default(true),
      quality: z.number().default(80),
      maxSizeBytes: z.number().default(512000),
    }),
    dom: z.object({
      enabled: z.boolean().default(true),
      maxSizeBytes: z.number().default(524288),
    }),
    console: z.object({
      enabled: z.boolean().default(true),
      maxEntries: z.number().default(100),
    }),
    network: z.object({
      enabled: z.boolean().default(true),
      maxEvents: z.number().default(200),
    }),
  }),
  logging: z.object({
    level: z.enum(['debug', 'info', 'warn', 'error']).default('info'),
    format: z.enum(['json', 'text']).default('json'),
  }),
  metrics: z.object({
    enabled: z.boolean().default(true),
    port: z.number().default(9090),
  }),
});

export type Config = z.infer<typeof ConfigSchema>;

export function loadConfig(): Config {
  const config = {
    server: {
      port: parseInt(process.env.PLAYWRIGHT_DRIVER_PORT || '39400', 10),
      host: process.env.PLAYWRIGHT_DRIVER_HOST || '127.0.0.1',
      requestTimeout: parseInt(process.env.REQUEST_TIMEOUT_MS || '120000', 10),
      maxRequestSize: parseInt(process.env.MAX_REQUEST_SIZE || '5242880', 10),
    },
    browser: {
      headless: process.env.HEADLESS !== 'false',
      executablePath: process.env.BROWSER_EXECUTABLE_PATH,
      args: process.env.BROWSER_ARGS?.split(',') || [],
    },
    session: {
      maxConcurrent: parseInt(process.env.MAX_SESSIONS || '10', 10),
      idleTimeoutMs: parseInt(process.env.SESSION_IDLE_TIMEOUT_MS || '300000', 10),
      poolSize: parseInt(process.env.SESSION_POOL_SIZE || '5', 10),
      cleanupIntervalMs: parseInt(process.env.CLEANUP_INTERVAL_MS || '60000', 10),
    },
    telemetry: {
      screenshot: {
        enabled: process.env.SCREENSHOT_ENABLED !== 'false',
        fullPage: process.env.SCREENSHOT_FULL_PAGE !== 'false',
        quality: parseInt(process.env.SCREENSHOT_QUALITY || '80', 10),
        maxSizeBytes: parseInt(process.env.SCREENSHOT_MAX_SIZE || '512000', 10),
      },
      dom: {
        enabled: process.env.DOM_ENABLED !== 'false',
        maxSizeBytes: parseInt(process.env.DOM_MAX_SIZE || '524288', 10),
      },
      console: {
        enabled: process.env.CONSOLE_ENABLED !== 'false',
        maxEntries: parseInt(process.env.CONSOLE_MAX_ENTRIES || '100', 10),
      },
      network: {
        enabled: process.env.NETWORK_ENABLED !== 'false',
        maxEvents: parseInt(process.env.NETWORK_MAX_EVENTS || '200', 10),
      },
    },
    logging: {
      level: (process.env.LOG_LEVEL || 'info') as 'debug' | 'info' | 'warn' | 'error',
      format: (process.env.LOG_FORMAT || 'json') as 'json' | 'text',
    },
    metrics: {
      enabled: process.env.METRICS_ENABLED !== 'false',
      port: parseInt(process.env.METRICS_PORT || '9090', 10),
    },
  };

  return ConfigSchema.parse(config);
}
```

## Step 4: Logging & Metrics

### `src/utils/logger.ts`

```typescript
import winston from 'winston';
import { Config } from '../config';

export function createLogger(config: Config): winston.Logger {
  const format =
    config.logging.format === 'json'
      ? winston.format.combine(
          winston.format.timestamp(),
          winston.format.errors({ stack: true }),
          winston.format.json()
        )
      : winston.format.combine(
          winston.format.timestamp(),
          winston.format.errors({ stack: true }),
          winston.format.printf(
            ({ level, message, timestamp, ...meta }) =>
              `${timestamp} [${level}]: ${message} ${Object.keys(meta).length ? JSON.stringify(meta) : ''}`
          )
        );

  return winston.createLogger({
    level: config.logging.level,
    format,
    transports: [new winston.transports.Console()],
  });
}
```

### `src/utils/metrics.ts`

```typescript
import { Registry, Counter, Histogram, Gauge } from 'prom-client';

export class Metrics {
  private registry: Registry;

  public sessionCount: Gauge;
  public instructionDuration: Histogram;
  public instructionErrors: Counter;
  public screenshotSize: Histogram;

  constructor() {
    this.registry = new Registry();

    this.sessionCount = new Gauge({
      name: 'playwright_driver_sessions_total',
      help: 'Current number of active sessions',
      registers: [this.registry],
    });

    this.instructionDuration = new Histogram({
      name: 'playwright_driver_instruction_duration_ms',
      help: 'Instruction execution duration in milliseconds',
      labelNames: ['type', 'success'],
      buckets: [10, 50, 100, 250, 500, 1000, 2500, 5000, 10000],
      registers: [this.registry],
    });

    this.instructionErrors = new Counter({
      name: 'playwright_driver_instruction_errors_total',
      help: 'Total number of instruction errors',
      labelNames: ['type', 'error_kind'],
      registers: [this.registry],
    });

    this.screenshotSize = new Histogram({
      name: 'playwright_driver_screenshot_size_bytes',
      help: 'Screenshot size in bytes',
      buckets: [10000, 50000, 100000, 250000, 500000, 1000000],
      registers: [this.registry],
    });
  }

  getRegistry(): Registry {
    return this.registry;
  }

  async getMetrics(): Promise<string> {
    return this.registry.metrics();
  }
}
```

## Step 5: Next Steps

After setting up the foundation:

1. **Implement SessionManager** (`src/session/manager.ts`)
2. **Implement base handler** (`src/handlers/base.ts`)
3. **Migrate existing handlers** one by one
4. **Write tests** as you go
5. **Set up HTTP routes** (`src/routes/*.ts`)
6. **Create main server** (`src/server.ts`)

See the detailed plan in `docs/plans/playwright-driver-completion.md` for full implementation details.

## Verification

Before moving to Phase 2:

- [ ] TypeScript compiles without errors
- [ ] All tests pass
- [ ] Test coverage >80%
- [ ] ESLint passes
- [ ] Existing 13 handlers migrated
- [ ] Integration tests pass
- [ ] No regressions from original server.js
