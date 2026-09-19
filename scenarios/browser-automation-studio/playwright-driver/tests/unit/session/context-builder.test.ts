import { buildContext } from '../../../src/session/context-builder';
import type { SessionSpec } from '../../../src/types';
import type { BrowserContextOptions } from 'rebrowser-playwright';
import { createMockBrowser, createTestConfig } from '../../helpers';
import { interactionStateForContext } from '../../../src/session/interaction-state';

describe('ContextBuilder', () => {
  let mockBrowser: ReturnType<typeof createMockBrowser>;
  let sessionSpec: SessionSpec;
  let config: ReturnType<typeof createTestConfig>;

  const getNewContextOptions = (
    browser: ReturnType<typeof createMockBrowser>
  ): BrowserContextOptions | undefined => {
    const calls = (browser.newContext as jest.MockedFunction<typeof browser.newContext>)
      .mock.calls as Array<[BrowserContextOptions?]>;
    return calls[0]?.[0];
  };

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

      const options = getNewContextOptions(mockBrowser);
      expect(options).toEqual(
        expect.objectContaining({
          viewport: { width: 1280, height: 720 },
          baseURL: 'https://example.com',
        })
      );
      expect(result.context).toBeDefined();
    });

    it.each(['rest', 'hover', 'focus-visible', 'pressed', 'disabled'] as const)(
      'should apply declared interaction state %s to the browser context',
      async (interactionState) => {
        const specWithInteractionState: SessionSpec = {
          ...sessionSpec,
          browser_profile: { interaction_state: interactionState },
        };

        const result = await buildContext(mockBrowser, specWithInteractionState, config);

        expect(interactionStateForContext(result.context)).toBe(interactionState);
      }
    );

    it('should create context without HAR when disabled', async () => {
      const configNoHAR = createTestConfig({
        telemetry: { har: { enabled: false } },
      });

      const result = await buildContext(mockBrowser, sessionSpec, configNoHAR);

      const options = getNewContextOptions(mockBrowser);
      expect(options?.recordHar).toBeUndefined();
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

      const options = getNewContextOptions(mockBrowser);
      const recordHar = options?.recordHar as { path?: string; mode?: string } | undefined;
      expect(recordHar?.path).toContain('har-exec-123');
      expect(recordHar?.mode).toBe('minimal');
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
      const configWithVideo = createTestConfig();
      const specWithVideo: SessionSpec = {
        ...sessionSpec,
        required_capabilities: { video: true },
      };

      const result = await buildContext(mockBrowser, specWithVideo, configWithVideo);

      const options = getNewContextOptions(mockBrowser);
      const recordVideo = options?.recordVideo as { dir?: string; size?: { width: number; height: number } } | undefined;
      expect(recordVideo?.dir).toContain('videos-exec-123');
      expect(recordVideo?.size).toBeDefined();
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

      const options = getNewContextOptions(mockBrowser);
      expect(options).toEqual(
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

      const options = getNewContextOptions(mockBrowser);
      expect(options).toEqual(
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

      const options = getNewContextOptions(mockBrowser);
      expect(options).toEqual(
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

      const options = getNewContextOptions(mockBrowser);
      expect(options).toEqual(
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

      const options = getNewContextOptions(mockBrowser);
      expect(options).toEqual(
        expect.objectContaining({
          permissions: ['geolocation', 'notifications'],
        })
      );
    });

    it('grants microphone only in the explicit deterministic media driver', async () => {
      const fixtureConfig = createTestConfig({
        browser: { fakeMicrophoneFile: '/fixtures/reference.wav' },
      });
      const result = await buildContext(mockBrowser, sessionSpec, fixtureConfig);

      expect(result.context.grantPermissions).toHaveBeenCalledWith(['microphone']);
    });

    it('should apply ignoreHTTPSErrors from config', async () => {
      const configWithHTTPS = createTestConfig({
        browser: { headless: true, ignoreHTTPSErrors: true },
      });

      await buildContext(mockBrowser, sessionSpec, configWithHTTPS);

      const options = getNewContextOptions(mockBrowser);
      expect(options).toEqual(
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

      const [options] = mockBrowser.newContext.mock.calls[0] ?? [];
      expect(options).toEqual(
        expect.objectContaining({
          storageState: specWithStorage.storage_state,
        })
      );
    });
  });

  describe('viewport source attribution', () => {
    it('emulates coarse no-hover input for a mobile capture', async () => {
      const mobileSpec: SessionSpec = {
        ...sessionSpec,
        viewport: { width: 390, height: 844 },
      };

      await buildContext(mockBrowser, mobileSpec, config);

      const [options] = mockBrowser.newContext.mock.calls[0] ?? [];
      expect(options).toEqual(expect.objectContaining({ hasTouch: true, isMobile: true }));
    });

    it('retains mobile input emulation when a phone rotates to landscape', async () => {
      const mobileLandscapeSpec: SessionSpec = {
        ...sessionSpec,
        viewport: { width: 844, height: 390 },
      };

      await buildContext(mockBrowser, mobileLandscapeSpec, config);

      const [options] = mockBrowser.newContext.mock.calls[0] ?? [];
      expect(options).toEqual(expect.objectContaining({ hasTouch: true, isMobile: true }));
    });

    it('retains fine hover input for a desktop capture', async () => {
      await buildContext(mockBrowser, sessionSpec, config);

      const [options] = mockBrowser.newContext.mock.calls[0] ?? [];
      expect(options).toEqual(expect.objectContaining({ hasTouch: false, isMobile: false }));
    });

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
        browser_profile: {
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
        browser_profile: {
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
        browser_profile: {
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
        browser_profile: {
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
