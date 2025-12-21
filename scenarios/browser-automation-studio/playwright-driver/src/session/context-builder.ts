import { mkdir } from 'fs/promises';
import path from 'path';
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
  const artifactPaths = spec.artifact_paths ?? {};
  const artifactRoot = (artifactPaths.root ?? '').trim();

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
  if (spec.required_capabilities?.har) {
    harPath =
      artifactPaths.har_path ||
      (artifactRoot ? path.join(artifactRoot, 'har', `execution-${spec.execution_id}.har`) : undefined) ||
      `/tmp/har-${spec.execution_id}-${Date.now()}.har`;
    await ensureDir(path.dirname(harPath), 'har');
    contextOptions.recordHar = {
      path: harPath,
      mode: 'minimal',
    };
    logger.debug('HAR recording enabled', { harPath });
  }

  // Video recording
  let videoDir: string | undefined;
  if (spec.required_capabilities?.video) {
    videoDir =
      artifactPaths.video_dir ||
      (artifactRoot ? path.join(artifactRoot, 'videos') : undefined) ||
      `/tmp/videos-${spec.execution_id}-${Date.now()}`;
    await ensureDir(videoDir, 'video');
    contextOptions.recordVideo = {
      dir: videoDir,
      size: spec.viewport,
    };
    logger.debug('Video recording enabled', { videoDir });
  }

  // Tracing (if enabled)
  let tracePath: string | undefined;
  const shouldTrace = Boolean(spec.required_capabilities?.tracing);
  if (shouldTrace) {
    tracePath =
      artifactPaths.trace_path ||
      (artifactRoot ? path.join(artifactRoot, 'traces', `execution-${spec.execution_id}.zip`) : undefined) ||
      `/tmp/trace-${spec.execution_id}-${Date.now()}.zip`;
    await ensureDir(path.dirname(tracePath), 'trace');
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

async function ensureDir(dir: string, label: string): Promise<void> {
  try {
    await mkdir(dir, { recursive: true });
  } catch (error) {
    logger.warn('Failed to ensure artifact directory', {
      label,
      dir,
      error: error instanceof Error ? error.message : String(error),
    });
  }
}
