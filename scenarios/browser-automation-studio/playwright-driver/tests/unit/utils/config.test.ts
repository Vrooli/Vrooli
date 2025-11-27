import { loadConfig } from '../../../src/config';

describe('Config', () => {
  const originalEnv = process.env;

  beforeEach(() => {
    jest.resetModules();
    process.env = { ...originalEnv };
  });

  afterAll(() => {
    process.env = originalEnv;
  });

  describe('loadConfig', () => {
    it('should load default configuration', () => {
      const config = loadConfig();

      expect(config.server.port).toBe(39400);
      expect(config.server.host).toBe('127.0.0.1');
      expect(config.browser.headless).toBe(true);
      expect(config.session.maxConcurrent).toBe(10);
      expect(config.telemetry.screenshot.enabled).toBe(true);
      expect(config.logging.level).toBe('info');
      expect(config.metrics.enabled).toBe(true);
    });

    it('should override port from environment', () => {
      process.env.PLAYWRIGHT_DRIVER_PORT = '9999';
      const config = loadConfig();

      expect(config.server.port).toBe(9999);
    });

    it('should override host from environment', () => {
      process.env.PLAYWRIGHT_DRIVER_HOST = '0.0.0.0';
      const config = loadConfig();

      expect(config.server.host).toBe('0.0.0.0');
    });

    it('should set headless to false when HEADLESS=false', () => {
      process.env.HEADLESS = 'false';
      const config = loadConfig();

      expect(config.browser.headless).toBe(false);
    });

    it('should set headless to true by default', () => {
      const config = loadConfig();

      expect(config.browser.headless).toBe(true);
    });

    it('should set browser executable path from environment', () => {
      process.env.BROWSER_EXECUTABLE_PATH = '/path/to/chrome';
      const config = loadConfig();

      expect(config.browser.executablePath).toBe('/path/to/chrome');
    });

    it('should set ignoreHTTPSErrors when enabled', () => {
      process.env.IGNORE_HTTPS_ERRORS = 'true';
      const config = loadConfig();

      expect(config.browser.ignoreHTTPSErrors).toBe(true);
    });

    it('should override max sessions from environment', () => {
      process.env.MAX_SESSIONS = '20';
      const config = loadConfig();

      expect(config.session.maxConcurrent).toBe(20);
    });

    it('should override session idle timeout from environment', () => {
      process.env.SESSION_IDLE_TIMEOUT_MS = '600000';
      const config = loadConfig();

      expect(config.session.idleTimeoutMs).toBe(600000);
    });

    it('should disable screenshots when SCREENSHOT_ENABLED=false', () => {
      process.env.SCREENSHOT_ENABLED = 'false';
      const config = loadConfig();

      expect(config.telemetry.screenshot.enabled).toBe(false);
    });

    it('should set screenshot quality from environment', () => {
      process.env.SCREENSHOT_QUALITY = '90';
      const config = loadConfig();

      expect(config.telemetry.screenshot.quality).toBe(90);
    });

    it('should enable HAR recording when HAR_ENABLED=true', () => {
      process.env.HAR_ENABLED = 'true';
      const config = loadConfig();

      expect(config.telemetry.har.enabled).toBe(true);
    });

    it('should enable video recording when VIDEO_ENABLED=true', () => {
      process.env.VIDEO_ENABLED = 'true';
      const config = loadConfig();

      expect(config.telemetry.video.enabled).toBe(true);
    });

    it('should enable tracing when TRACING_ENABLED=true', () => {
      process.env.TRACING_ENABLED = 'true';
      const config = loadConfig();

      expect(config.telemetry.tracing.enabled).toBe(true);
    });

    it('should set log level from environment', () => {
      process.env.LOG_LEVEL = 'debug';
      const config = loadConfig();

      expect(config.logging.level).toBe('debug');
    });

    it('should disable metrics when METRICS_ENABLED=false', () => {
      process.env.METRICS_ENABLED = 'false';
      const config = loadConfig();

      expect(config.metrics.enabled).toBe(false);
    });

    it('should override metrics port from environment', () => {
      process.env.METRICS_PORT = '8080';
      const config = loadConfig();

      expect(config.metrics.port).toBe(8080);
    });

    it('should throw error for invalid port number', () => {
      process.env.PLAYWRIGHT_DRIVER_PORT = 'invalid';

      expect(() => loadConfig()).toThrow();
    });

    it('should throw error for invalid max sessions', () => {
      process.env.MAX_SESSIONS = 'invalid';

      expect(() => loadConfig()).toThrow();
    });
  });
});
