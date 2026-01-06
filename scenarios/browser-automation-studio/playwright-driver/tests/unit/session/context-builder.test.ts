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

  describe('viewport source attribution', () => {
    it('should return actualViewport with requested source when using spec viewport', async () => {
      const result = await buildContext(mockBrowser, sessionSpec, config);

      expect(result.actualViewport).toEqual({
        width: 1280,
        height: 720,
        source: 'requested',
        reason: 'Using requested 1280x720',
      });
    });

    it('should return actualViewport with fingerprint source when profile has both dimensions', async () => {
      const specWithProfile: SessionSpec = {
        ...sessionSpec,
        session_profile: {
          id: 'profile-1',
          name: 'Test Profile',
          fingerprint: {
            viewport_width: 1920,
            viewport_height: 1080,
          },
        },
      };

      const result = await buildContext(mockBrowser, specWithProfile, config);

      expect(result.actualViewport).toEqual({
        width: 1920,
        height: 1080,
        source: 'fingerprint',
        reason: 'Browser profile specifies 1920x1080',
      });
    });

    it('should use requested viewport when fingerprint has only one dimension (width)', async () => {
      const specWithPartialProfile: SessionSpec = {
        ...sessionSpec,
        viewport: { width: 1280, height: 720 },
        session_profile: {
          id: 'profile-1',
          name: 'Test Profile',
          fingerprint: {
            viewport_width: 1920,
            // viewport_height intentionally omitted
          },
        },
      };

      const result = await buildContext(mockBrowser, specWithPartialProfile, config);

      // Should use requested viewport, not fingerprint width
      expect(result.actualViewport.source).toBe('requested');
      expect(result.actualViewport.width).toBe(1280);
      expect(result.actualViewport.height).toBe(720);
    });

    it('should use requested viewport when fingerprint has only one dimension (height)', async () => {
      const specWithPartialProfile: SessionSpec = {
        ...sessionSpec,
        viewport: { width: 1280, height: 720 },
        session_profile: {
          id: 'profile-1',
          name: 'Test Profile',
          fingerprint: {
            // viewport_width intentionally omitted
            viewport_height: 1080,
          },
        },
      };

      const result = await buildContext(mockBrowser, specWithPartialProfile, config);

      // Should use requested viewport, not fingerprint height
      expect(result.actualViewport.source).toBe('requested');
      expect(result.actualViewport.width).toBe(1280);
      expect(result.actualViewport.height).toBe(720);
    });

    it('should use requested viewport when fingerprint dimensions are zero', async () => {
      const specWithZeroProfile: SessionSpec = {
        ...sessionSpec,
        viewport: { width: 1280, height: 720 },
        session_profile: {
          id: 'profile-1',
          name: 'Test Profile',
          fingerprint: {
            viewport_width: 0,
            viewport_height: 0,
          },
        },
      };

      const result = await buildContext(mockBrowser, specWithZeroProfile, config);

      expect(result.actualViewport.source).toBe('requested');
      expect(result.actualViewport.width).toBe(1280);
      expect(result.actualViewport.height).toBe(720);
    });

    it('should use default viewport when both requested and fingerprint are missing', async () => {
      const specWithNoViewport: SessionSpec = {
        ...sessionSpec,
        viewport: { width: 0, height: 0 },
      };

      const result = await buildContext(mockBrowser, specWithNoViewport, config);

      expect(result.actualViewport.source).toBe('default');
      expect(result.actualViewport.width).toBe(1280);
      expect(result.actualViewport.height).toBe(720);
      expect(result.actualViewport.reason).toContain('default');
    });
  });
});
