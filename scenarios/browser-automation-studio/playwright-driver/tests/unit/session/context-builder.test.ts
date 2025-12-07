import { buildContext } from '../../../src/session/context-builder';
import type { SessionSpec } from '../../../src/types';
import { createMockBrowser, createTestConfig } from '../../helpers';

describe('ContextBuilder', () => {
  let mockBrowser: ReturnType<typeof createMockBrowser>;
  let sessionSpec: SessionSpec;
  let config: ReturnType<typeof createTestConfig>;

  beforeEach(() => {
    mockBrowser = createMockBrowser();
    config = createTestConfig();

    sessionSpec = {
      execution_id: 'exec-123',
      workflow_id: 'workflow-123',
      base_url: 'https://example.com',
      viewport: { width: 1280, height: 720 },
      reuse_mode: 'fresh',
      required_capabilities: {},
    };
  });

  describe('buildContext', () => {
    it('should create browser context with viewport', async () => {
      const result = await buildContext(mockBrowser, sessionSpec, config);

      expect(mockBrowser.newContext).toHaveBeenCalledWith(
        expect.objectContaining({
          viewport: { width: 1280, height: 720 },
          baseURL: 'https://example.com',
        })
      );
      expect(result.context).toBeDefined();
    });

    it('should create context without HAR when disabled', async () => {
      const configNoHAR = createTestConfig({
        telemetry: { har: { enabled: false } },
      });

      const result = await buildContext(mockBrowser, sessionSpec, configNoHAR);

      expect(mockBrowser.newContext).toHaveBeenCalledWith(
        expect.not.objectContaining({
          recordHar: expect.anything(),
        })
      );
      expect(result.harPath).toBeUndefined();
    });

    it('should create context with HAR when enabled and required', async () => {
      const configWithHAR = createTestConfig({
        telemetry: { har: { enabled: true } },
      });
      const specWithHAR: SessionSpec = {
        ...sessionSpec,
        required_capabilities: { har: true },
      };

      const result = await buildContext(mockBrowser, specWithHAR, configWithHAR);

      expect(mockBrowser.newContext).toHaveBeenCalledWith(
        expect.objectContaining({
          recordHar: expect.objectContaining({
            path: expect.stringContaining('har-exec-123'),
            mode: 'minimal',
          }),
        })
      );
      expect(result.harPath).toBeDefined();
      expect(result.harPath).toContain('har-exec-123');
    });

    it('should not enable HAR when required but config disabled', async () => {
      const configNoHAR = createTestConfig({
        telemetry: { har: { enabled: false } },
      });
      const specWithHAR: SessionSpec = {
        ...sessionSpec,
        required_capabilities: { har: true },
      };

      const result = await buildContext(mockBrowser, specWithHAR, configNoHAR);

      expect(result.harPath).toBeUndefined();
    });

    it('should create context with video when enabled and required', async () => {
      const configWithVideo = createTestConfig({
        telemetry: { video: { enabled: true } },
      });
      const specWithVideo: SessionSpec = {
        ...sessionSpec,
        required_capabilities: { video: true },
      };

      const result = await buildContext(mockBrowser, specWithVideo, configWithVideo);

      expect(mockBrowser.newContext).toHaveBeenCalledWith(
        expect.objectContaining({
          recordVideo: expect.objectContaining({
            dir: expect.stringContaining('videos-exec-123'),
            size: expect.anything(),
          }),
        })
      );
      expect(result.videoDir).toBeDefined();
      expect(result.videoDir).toContain('videos-exec-123');
    });

    it('should create context with tracing when enabled and required', async () => {
      const configWithTracing = createTestConfig({
        telemetry: { tracing: { enabled: true } },
      });
      const specWithTracing: SessionSpec = {
        ...sessionSpec,
        required_capabilities: { tracing: true },
      };

      const result = await buildContext(mockBrowser, specWithTracing, configWithTracing);

      expect(result.tracePath).toBeDefined();
      expect(result.tracePath).toContain('trace-exec-123');
    });

    it('should handle custom user agent', async () => {
      const specWithUA: SessionSpec = {
        ...sessionSpec,
        user_agent: 'Mozilla/5.0 Custom',
      };

      await buildContext(mockBrowser, specWithUA, config);

      expect(mockBrowser.newContext).toHaveBeenCalledWith(
        expect.objectContaining({
          userAgent: 'Mozilla/5.0 Custom',
        })
      );
    });

    it('should handle custom locale', async () => {
      const specWithLocale: SessionSpec = {
        ...sessionSpec,
        locale: 'fr-FR',
      };

      await buildContext(mockBrowser, specWithLocale, config);

      expect(mockBrowser.newContext).toHaveBeenCalledWith(
        expect.objectContaining({
          locale: 'fr-FR',
        })
      );
    });

    it('should handle custom timezone', async () => {
      const specWithTZ: SessionSpec = {
        ...sessionSpec,
        timezone: 'America/New_York',
      };

      await buildContext(mockBrowser, specWithTZ, config);

      expect(mockBrowser.newContext).toHaveBeenCalledWith(
        expect.objectContaining({
          timezoneId: 'America/New_York',
        })
      );
    });

    it('should handle geolocation', async () => {
      const specWithGeo: SessionSpec = {
        ...sessionSpec,
        geolocation: { latitude: 40.7128, longitude: -74.006 },
      };

      await buildContext(mockBrowser, specWithGeo, config);

      expect(mockBrowser.newContext).toHaveBeenCalledWith(
        expect.objectContaining({
          geolocation: { latitude: 40.7128, longitude: -74.006 },
        })
      );
    });

    it('should handle permissions', async () => {
      const specWithPerms: SessionSpec = {
        ...sessionSpec,
        permissions: ['geolocation', 'notifications'],
      };

      await buildContext(mockBrowser, specWithPerms, config);

      expect(mockBrowser.newContext).toHaveBeenCalledWith(
        expect.objectContaining({
          permissions: ['geolocation', 'notifications'],
        })
      );
    });

    it('should apply ignoreHTTPSErrors from config', async () => {
      const configWithHTTPS = createTestConfig({
        browser: { headless: true, ignoreHTTPSErrors: true },
      });

      await buildContext(mockBrowser, sessionSpec, configWithHTTPS);

      expect(mockBrowser.newContext).toHaveBeenCalledWith(
        expect.objectContaining({
          ignoreHTTPSErrors: true,
        })
      );
    });

    it('should handle storage state', async () => {
      const specWithStorage: SessionSpec = {
        ...sessionSpec,
        storage_state: {
          cookies: [
            {
              name: 'test',
              value: '123',
              domain: 'example.com',
              path: '/',
              expires: -1,
              httpOnly: false,
              secure: false,
              sameSite: 'Lax',
            },
          ],
          origins: [],
        },
      };

      await buildContext(mockBrowser, specWithStorage, config);

      expect(mockBrowser.newContext).toHaveBeenCalledWith(
        expect.objectContaining({
          storageState: specWithStorage.storage_state,
        })
      );
    });
  });
});
