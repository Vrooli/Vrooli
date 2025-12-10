import { Browser, BrowserContext } from 'playwright';
import type { SessionSpec } from '../types';
import type { Config } from '../config';
import { logger } from '../utils';

/**
 * Build BrowserContext from SessionSpec
 *
 * Handles:
 * - Viewport configuration
 * - HAR recording
 * - Video recording
 * - Tracing
 * - Base URL
 * - User agent and other browser options
 */
export async function buildContext(
  browser: Browser,
  spec: SessionSpec,
  config: Config
): Promise<{
  context: BrowserContext;
  harPath?: string;
  tracePath?: string;
  videoDir?: string;
}> {
  const contextOptions: Parameters<typeof browser.newContext>[0] = {
    viewport: {
      width: spec.viewport.width,
      height: spec.viewport.height,
    },
    // Use 2x device scale for crisp screenshots on HiDPI displays
    // This renders at 2x resolution (e.g., 1280x720 viewport = 2560x1440 pixels)
    deviceScaleFactor: 2,
    baseURL: spec.base_url,
    ignoreHTTPSErrors: config.browser.ignoreHTTPSErrors,
  };

  // Browser context configuration from spec
  if (spec.user_agent) {
    contextOptions.userAgent = spec.user_agent;
  }
  if (spec.locale) {
    contextOptions.locale = spec.locale;
  }
  if (spec.timezone) {
    contextOptions.timezoneId = spec.timezone;
  }
  if (spec.geolocation) {
    contextOptions.geolocation = spec.geolocation;
  }
  if (spec.permissions) {
    contextOptions.permissions = spec.permissions;
  }
  if (spec.storage_state) {
    contextOptions.storageState = spec.storage_state;
  }

  // HAR recording
  let harPath: string | undefined;
  if (spec.required_capabilities?.har && config.telemetry.har.enabled) {
    harPath = `/tmp/har-${spec.execution_id}-${Date.now()}.har`;
    contextOptions.recordHar = {
      path: harPath,
      mode: 'minimal',
    };
    logger.debug('HAR recording enabled', { harPath });
  }

  // Video recording
  let videoDir: string | undefined;
  if (spec.required_capabilities?.video && config.telemetry.video.enabled) {
    videoDir = `/tmp/videos-${spec.execution_id}-${Date.now()}`;
    contextOptions.recordVideo = {
      dir: videoDir,
      size: spec.viewport,
    };
    logger.debug('Video recording enabled', { videoDir });
  }

  // Tracing (if enabled)
  let tracePath: string | undefined;
  const shouldTrace = spec.required_capabilities?.tracing && config.telemetry.tracing.enabled;
  if (shouldTrace) {
    tracePath = `/tmp/trace-${spec.execution_id}-${Date.now()}.zip`;
  }

  // Create context
  const context = await browser.newContext(contextOptions);

  // Start tracing if enabled
  if (shouldTrace && tracePath) {
    await context.tracing.start({
      screenshots: true,
      snapshots: true,
      sources: true,
    });
    logger.debug('Tracing started', { tracePath });
  }

  logger.info('Browser context created', {
    executionId: spec.execution_id,
    viewport: spec.viewport,
    har: !!harPath,
    video: !!videoDir,
    tracing: !!tracePath,
  });

  return {
    context,
    harPath,
    tracePath,
    videoDir,
  };
}
