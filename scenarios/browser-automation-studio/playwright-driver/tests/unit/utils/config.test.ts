import { getObservabilityConfigSummary, loadConfig } from '../../../src/config';
import { ObservabilityCache } from '../../../src/observability/cache';

describe('Config', () => {
  const originalEnv = process.env;

  beforeEach(() => {
    jest.resetModules();
    process.env = { ...originalEnv };
    process.env.PLAYWRIGHT_DRIVER_PORT = originalEnv.PLAYWRIGHT_DRIVER_PORT ?? '39400';
  });

  afterAll(() => {
    process.env = originalEnv;
  });

  describe('observability summary', () => {
    it('marks every process-start option as unavailable for runtime editing', () => {
      const summary = getObservabilityConfigSummary();
      const options = Object.values(summary.all_options ?? {}).flat();

      expect(options).toHaveLength(68);
      expect(options.every(option => option.editable === false)).toBe(true);
    });

    it.each([
      ['unset', undefined, false],
      ['set to default', '', false],
      ['modified', 'synthetic-recovery-credential', true],
    ])('redacts recovery credential in serialized %s summary while preserving metadata', (_state, value, modified) => {
      if (value === undefined) delete process.env.PLAYWRIGHT_DRIVER_ADMIN_SECRET;
      else process.env.PLAYWRIGHT_DRIVER_ADMIN_SECRET = value;

      const summary = getObservabilityConfigSummary();
      const internal = summary.all_options?.internal.find(option => option.env_var === 'PLAYWRIGHT_DRIVER_ADMIN_SECRET');
      expect(internal).toMatchObject({
        current_value: modified ? '[REDACTED]' : '',
        default_value: '',
        is_modified: modified,
        description: expect.any(String),
      });
      const serialized = JSON.stringify(summary);
      expect(serialized).not.toContain('synthetic-recovery-credential');
      expect(loadConfig().server.adminSecret).toBe(value?.trim() ?? '');
      if (modified) {
        expect(summary.modified_options?.find(option => option.env_var === 'PLAYWRIGHT_DRIVER_ADMIN_SECRET'))
          .toMatchObject({ current_value: '[REDACTED]', default_value: '' });
      }

      for (const depth of ['standard', 'deep']) {
        const response = { depth, config: summary };
        const cache = new ObservabilityCache(1000);
        cache.set(depth, response as any);
        const completeResponse = JSON.stringify(cache.get(depth));
        expect(completeResponse).not.toContain('synthetic-recovery-credential');
        expect(completeResponse).toContain('PLAYWRIGHT_DRIVER_ADMIN_SECRET');
        expect(completeResponse).toContain('Shared secret required for loopback administrative session recovery');
      }
    });
  });

  describe('loadConfig', () => {
    it('should load default configuration', () => {
      const config = loadConfig();

      expect(config.server.port).toBe(39400);
      expect(config.server.host).toBe('127.0.0.1');
      expect(config.browser.headless).toBe(false);
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
      process.env.HEADLESS = 'true';
      const config = loadConfig();

      expect(config.browser.headless).toBe(true);
    });

    it('should set browser executable path from environment', () => {
      process.env.BROWSER_EXECUTABLE_PATH = '/path/to/chrome';
      const config = loadConfig();

      expect(config.browser.executablePath).toBe('/path/to/chrome');
    });

    it('loads an opt-in deterministic microphone fixture path', () => {
      process.env.BAS_FAKE_MICROPHONE_FILE = ' /fixtures/reference.wav ';
      const config = loadConfig();

      expect(config.browser.fakeMicrophoneFile).toBe('/fixtures/reference.wav');
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

    it('should throw for invalid required port number', () => {
      process.env.PLAYWRIGHT_DRIVER_PORT = 'invalid';

      expect(() => loadConfig()).toThrow(
        'Environment variable PLAYWRIGHT_DRIVER_PORT has invalid value "invalid" - expected an integer.'
      );
    });

    it('should use default max sessions for invalid value', () => {
      // Invalid max sessions falls back to default with warning (graceful degradation)
      process.env.MAX_SESSIONS = 'invalid';
      const warnSpy = jest.spyOn(console, 'warn').mockImplementation();

      const config = loadConfig();

      expect(warnSpy).toHaveBeenCalledWith(expect.stringContaining('Invalid numeric config value'));
      expect(config.session.maxConcurrent).toBe(10); // Default max sessions
      warnSpy.mockRestore();
    });
  });
});
