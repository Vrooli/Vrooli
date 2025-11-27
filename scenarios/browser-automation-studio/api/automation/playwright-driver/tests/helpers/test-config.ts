import type { Config } from '../../src/config';

/**
 * Create a test configuration with sensible defaults
 */
export function createTestConfig(overrides?: Partial<Config>): Config {
  const defaultConfig: Config = {
    server: {
      port: 39400,
      host: '127.0.0.1',
    },
    browser: {
      headless: true,
      executablePath: undefined,
      ignoreHTTPSErrors: false,
    },
    session: {
      maxConcurrent: 10,
      idleTimeoutMs: 300000,
      cleanupIntervalMs: 60000,
    },
    telemetry: {
      screenshot: {
        enabled: true,
        format: 'png',
        quality: 80,
        fullPage: false,
        maxSizeBytes: 5 * 1024 * 1024,
      },
      console: {
        enabled: true,
        maxEntries: 100,
      },
      network: {
        enabled: true,
        maxEntries: 100,
      },
      dom: {
        enabled: true,
        maxSizeBytes: 1 * 1024 * 1024,
      },
      har: {
        enabled: false,
      },
      video: {
        enabled: false,
      },
      tracing: {
        enabled: false,
      },
    },
    logging: {
      level: 'info',
      format: 'json',
    },
    metrics: {
      enabled: true,
      port: 9090,
    },
  };

  return {
    ...defaultConfig,
    ...overrides,
    server: { ...defaultConfig.server, ...overrides?.server },
    browser: { ...defaultConfig.browser, ...overrides?.browser },
    session: { ...defaultConfig.session, ...overrides?.session },
    telemetry: {
      screenshot: { ...defaultConfig.telemetry.screenshot, ...overrides?.telemetry?.screenshot },
      console: { ...defaultConfig.telemetry.console, ...overrides?.telemetry?.console },
      network: { ...defaultConfig.telemetry.network, ...overrides?.telemetry?.network },
      dom: { ...defaultConfig.telemetry.dom, ...overrides?.telemetry?.dom },
      har: { ...defaultConfig.telemetry.har, ...overrides?.telemetry?.har },
      video: { ...defaultConfig.telemetry.video, ...overrides?.telemetry?.video },
      tracing: { ...defaultConfig.telemetry.tracing, ...overrides?.telemetry?.tracing },
    },
    logging: { ...defaultConfig.logging, ...overrides?.logging },
    metrics: { ...defaultConfig.metrics, ...overrides?.metrics },
  };
}
