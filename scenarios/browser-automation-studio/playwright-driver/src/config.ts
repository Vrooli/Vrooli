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
    ignoreHTTPSErrors: z.boolean().default(false),
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
      args: process.env.BROWSER_ARGS?.split(',').filter((arg) => arg.trim()) || [],
      ignoreHTTPSErrors: process.env.IGNORE_HTTPS_ERRORS === 'true',
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
