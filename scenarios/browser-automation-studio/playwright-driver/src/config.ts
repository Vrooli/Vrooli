import { z } from 'zod';

const ConfigSchema = z.object({
  server: z.object({
    port: z.number().min(1).max(65535).default(39400),
    host: z.string().default('127.0.0.1'),
    requestTimeout: z.number().min(1000).max(600000).default(300000), // 5 minutes - playwright operations can be slow
    maxRequestSize: z.number().min(1024).max(50 * 1024 * 1024).default(5 * 1024 * 1024),
  }),
  browser: z.object({
    headless: z.boolean().default(true),
    executablePath: z.string().optional(),
    args: z.array(z.string()).default([]),
    ignoreHTTPSErrors: z.boolean().default(false),
  }),
  session: z.object({
    maxConcurrent: z.number().min(1).max(100).default(10),
    idleTimeoutMs: z.number().min(10000).max(3600000).default(300000),
    poolSize: z.number().min(1).max(50).default(5),
    cleanupIntervalMs: z.number().min(5000).max(600000).default(60000),
  }),
  telemetry: z.object({
    screenshot: z.object({
      enabled: z.boolean().default(true),
      fullPage: z.boolean().default(true),
      quality: z.number().min(1).max(100).default(80),
      maxSizeBytes: z.number().min(1024).max(10 * 1024 * 1024).default(512000),
    }),
    dom: z.object({
      enabled: z.boolean().default(true),
      maxSizeBytes: z.number().min(1024).max(10 * 1024 * 1024).default(524288),
    }),
    console: z.object({
      enabled: z.boolean().default(true),
      maxEntries: z.number().min(1).max(10000).default(100),
    }),
    network: z.object({
      enabled: z.boolean().default(true),
      maxEvents: z.number().min(1).max(10000).default(200),
    }),
    har: z.object({
      enabled: z.boolean().default(false),
    }),
    video: z.object({
      enabled: z.boolean().default(false),
    }),
    tracing: z.object({
      enabled: z.boolean().default(false),
    }),
  }),
  logging: z.object({
    level: z.enum(['debug', 'info', 'warn', 'error']).default('info'),
    format: z.enum(['json', 'text']).default('json'),
  }),
  metrics: z.object({
    enabled: z.boolean().default(true),
    port: z.number().min(1).max(65535).default(9090),
  }),
});

export type Config = z.infer<typeof ConfigSchema>;

/**
 * Parse integer from environment variable with validation
 *
 * Hardened assumptions:
 * - Environment variable may be undefined, empty, or invalid
 * - parseInt may return NaN for invalid input
 * - Default value must be returned for any invalid input
 */
function parseEnvInt(envVar: string | undefined, defaultValue: number): number {
  if (!envVar || envVar.trim() === '') {
    return defaultValue;
  }

  const parsed = parseInt(envVar, 10);

  // Check for NaN and return default if invalid
  if (Number.isNaN(parsed)) {
    // Log warning about invalid config (but don't fail startup)
    console.warn(`Invalid numeric config value: "${envVar}", using default: ${defaultValue}`);
    return defaultValue;
  }

  return parsed;
}

/**
 * Validate log level from environment variable
 */
function parseLogLevel(envVar: string | undefined): 'debug' | 'info' | 'warn' | 'error' {
  const validLevels = ['debug', 'info', 'warn', 'error'];
  const level = (envVar || 'info').toLowerCase();

  if (!validLevels.includes(level)) {
    console.warn(`Invalid LOG_LEVEL "${envVar}", using default: info`);
    return 'info';
  }

  return level as 'debug' | 'info' | 'warn' | 'error';
}

/**
 * Validate log format from environment variable
 */
function parseLogFormat(envVar: string | undefined): 'json' | 'text' {
  const validFormats = ['json', 'text'];
  const format = (envVar || 'json').toLowerCase();

  if (!validFormats.includes(format)) {
    console.warn(`Invalid LOG_FORMAT "${envVar}", using default: json`);
    return 'json';
  }

  return format as 'json' | 'text';
}

export function loadConfig(): Config {
  const config = {
    server: {
      port: parseEnvInt(process.env.PLAYWRIGHT_DRIVER_PORT, 39400),
      host: process.env.PLAYWRIGHT_DRIVER_HOST || '127.0.0.1',
      requestTimeout: parseEnvInt(process.env.REQUEST_TIMEOUT_MS, 300000), // 5 minutes
      maxRequestSize: parseEnvInt(process.env.MAX_REQUEST_SIZE, 5242880),
    },
    browser: {
      headless: process.env.HEADLESS !== 'false',
      executablePath: process.env.BROWSER_EXECUTABLE_PATH,
      // Hardened: Simple comma-split can break args containing commas (rare but possible)
      // Browser args that contain commas are uncommon, but for robustness we trim each arg
      // and filter out empty strings that might result from trailing/leading commas
      // Note: If you need args with commas, use a different delimiter in the env var
      // e.g., BROWSER_ARGS="--arg1;;--arg2=value" with split(';;')
      args: process.env.BROWSER_ARGS
        ? process.env.BROWSER_ARGS.split(',')
            .map((arg) => arg.trim())
            .filter((arg) => arg.length > 0)
        : [],
      ignoreHTTPSErrors: process.env.IGNORE_HTTPS_ERRORS === 'true',
    },
    session: {
      maxConcurrent: parseEnvInt(process.env.MAX_SESSIONS, 10),
      idleTimeoutMs: parseEnvInt(process.env.SESSION_IDLE_TIMEOUT_MS, 300000),
      poolSize: parseEnvInt(process.env.SESSION_POOL_SIZE, 5),
      cleanupIntervalMs: parseEnvInt(process.env.CLEANUP_INTERVAL_MS, 60000),
    },
    telemetry: {
      screenshot: {
        enabled: process.env.SCREENSHOT_ENABLED !== 'false',
        fullPage: process.env.SCREENSHOT_FULL_PAGE !== 'false',
        quality: parseEnvInt(process.env.SCREENSHOT_QUALITY, 80),
        maxSizeBytes: parseEnvInt(process.env.SCREENSHOT_MAX_SIZE, 512000),
      },
      dom: {
        enabled: process.env.DOM_ENABLED !== 'false',
        maxSizeBytes: parseEnvInt(process.env.DOM_MAX_SIZE, 524288),
      },
      console: {
        enabled: process.env.CONSOLE_ENABLED !== 'false',
        maxEntries: parseEnvInt(process.env.CONSOLE_MAX_ENTRIES, 100),
      },
      network: {
        enabled: process.env.NETWORK_ENABLED !== 'false',
        maxEvents: parseEnvInt(process.env.NETWORK_MAX_EVENTS, 200),
      },
      har: {
        enabled: process.env.HAR_ENABLED === 'true',
      },
      video: {
        enabled: process.env.VIDEO_ENABLED === 'true',
      },
      tracing: {
        enabled: process.env.TRACING_ENABLED === 'true',
      },
    },
    logging: {
      level: parseLogLevel(process.env.LOG_LEVEL),
      format: parseLogFormat(process.env.LOG_FORMAT),
    },
    metrics: {
      enabled: process.env.METRICS_ENABLED !== 'false',
      port: parseEnvInt(process.env.METRICS_PORT, 9090),
    },
  };

  const parsed = ConfigSchema.parse(config);

  // Hardened: Deep freeze config to prevent accidental mutation during runtime
  return deepFreeze(parsed);
}

/**
 * Deep freeze an object to prevent any mutation.
 * Hardening measure to ensure config immutability.
 */
function deepFreeze<T extends Record<string, unknown>>(obj: T): T {
  const propNames = Object.getOwnPropertyNames(obj);

  for (const name of propNames) {
    const value = obj[name];
    if (value && typeof value === 'object' && !Object.isFrozen(value)) {
      deepFreeze(value as Record<string, unknown>);
    }
  }

  return Object.freeze(obj);
}
