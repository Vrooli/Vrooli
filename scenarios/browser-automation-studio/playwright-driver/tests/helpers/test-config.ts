import type { Config } from '../../src/config';

/**
 * Recursive partial type for deep object overrides
 */
type DeepPartial<T> = T extends object
  ? {
      [P in keyof T]?: DeepPartial<T[P]>;
    }
  : T;

/**
 * Create a test configuration with sensible defaults
 */
export function createTestConfig(overrides?: DeepPartial<Config>): Config {
  const defaultConfig: Config = {
    server: {
      port: 39400,
      host: '127.0.0.1',
      requestTimeout: 300000,
      maxRequestSize: 5 * 1024 * 1024,
    },
    browser: {
      headless: true,
      executablePath: undefined,
      args: [],
      ignoreHTTPSErrors: false,
    },
    session: {
      maxConcurrent: 10,
      idleTimeoutMs: 300000,
      poolSize: 5,
      cleanupIntervalMs: 60000,
    },
    execution: {
      defaultTimeoutMs: 30000,
      navigationTimeoutMs: 45000,
      waitTimeoutMs: 30000,
      assertionTimeoutMs: 15000,
      replayActionTimeoutMs: 10000,
    },
    recording: {
      maxBufferSize: 10000,
      minSelectorConfidence: 0.3,
      defaultSwipeDistance: 300,
      debounce: {
        inputMs: 500,
        scrollMs: 150,
      },
      selector: {
        maxCssDepth: 5,
        includeXPath: true,
      },
    },
    telemetry: {
      screenshot: {
        enabled: true,
        fullPage: true,
        quality: 80,
        maxSizeBytes: 512000,
      },
      dom: {
        enabled: true,
        maxSizeBytes: 524288,
      },
      console: {
        enabled: true,
        maxEntries: 100,
      },
      network: {
        enabled: true,
        maxEvents: 200,
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
    frameStreaming: {
      useScreencast: true,
      fallbackToPolling: true,
    },
    performance: {
      enabled: false,
      includeTimingHeaders: true,
      logSummaryInterval: 60,
      bufferSize: 100,
    },
  };

  return {
    ...defaultConfig,
    ...overrides,
    server: { ...defaultConfig.server, ...overrides?.server },
    browser: {
      ...defaultConfig.browser,
      ...overrides?.browser,
      args: (overrides?.browser?.args as string[]) ?? defaultConfig.browser.args,
    },
    session: { ...defaultConfig.session, ...overrides?.session },
    execution: { ...defaultConfig.execution, ...overrides?.execution },
    recording: {
      ...defaultConfig.recording,
      ...overrides?.recording,
      debounce: { ...defaultConfig.recording.debounce, ...overrides?.recording?.debounce },
      selector: { ...defaultConfig.recording.selector, ...overrides?.recording?.selector },
    },
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
    frameStreaming: { ...defaultConfig.frameStreaming, ...overrides?.frameStreaming },
    performance: { ...defaultConfig.performance, ...overrides?.performance },
  };
}
