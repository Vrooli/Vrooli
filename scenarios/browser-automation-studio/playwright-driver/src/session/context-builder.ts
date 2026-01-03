import { mkdir } from 'fs/promises';
import path from 'path';
import type { Browser, BrowserContext } from 'rebrowser-playwright';
import type { SessionSpec, BehaviorSettings } from '../types';
import type { Config } from '../config';
import { logger } from '../utils';
import {
  mergeWithPreset,
  resolveUserAgent,
  applyAntiDetection,
  applyAdBlocking,
  isAdBlockingEnabled,
  generateClientHints,
  mergeClientHintsWithHeaders,
} from '../browser-profile';

// Symbol to store behavior settings on context for handler access
export const BEHAVIOR_SETTINGS_KEY = Symbol('behaviorSettings');

// Extend BrowserContext type to include behavior settings
declare module 'rebrowser-playwright' {
  interface BrowserContext {
    [BEHAVIOR_SETTINGS_KEY]?: BehaviorSettings;
  }
}

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
 * - Browser profile (fingerprint, behavior, anti-detection)
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

  // Merge browser profile with preset defaults
  const { fingerprint, behavior, antiDetection, proxy } = mergeWithPreset(spec.browser_profile);

  // Determine viewport: browser profile overrides spec if set
  const viewportWidth = fingerprint.viewport_width || spec.viewport.width;
  const viewportHeight = fingerprint.viewport_height || spec.viewport.height;

  // Determine device scale factor from profile
  const deviceScaleFactor = fingerprint.device_scale_factor || 2;

  // Resolve user agent: profile > spec > default
  const userAgent = spec.user_agent || resolveUserAgent(fingerprint);

  // Generate Client Hints headers for Chromium browsers
  const clientHints = generateClientHints(userAgent);

  // Merge Client Hints with user-provided headers (user headers take precedence)
  const finalHeaders = mergeClientHintsWithHeaders(clientHints, spec.browser_profile?.extra_headers);

  const contextOptions: Parameters<typeof browser.newContext>[0] = {
    viewport: {
      width: viewportWidth,
      height: viewportHeight,
    },
    deviceScaleFactor,
    baseURL: spec.base_url,
    ignoreHTTPSErrors: config.browser.ignoreHTTPSErrors,
    userAgent,
    locale: spec.locale || fingerprint.locale || undefined,
    timezoneId: spec.timezone || fingerprint.timezone_id || undefined,
    colorScheme: fingerprint.color_scheme || undefined,
    extraHTTPHeaders: Object.keys(finalHeaders).length > 0 ? finalHeaders : undefined,
  };

  // Geolocation: spec overrides profile
  if (spec.geolocation) {
    contextOptions.geolocation = spec.geolocation;
    contextOptions.permissions = [...(spec.permissions || []), 'geolocation'];
  } else if (fingerprint.geolocation_enabled) {
    contextOptions.geolocation = {
      latitude: fingerprint.latitude,
      longitude: fingerprint.longitude,
      accuracy: fingerprint.accuracy,
    };
    contextOptions.permissions = [...(spec.permissions || []), 'geolocation'];
  } else if (spec.permissions) {
    contextOptions.permissions = spec.permissions;
  }

  if (spec.storage_state) {
    contextOptions.storageState = spec.storage_state;
  }

  // Proxy configuration
  if (proxy.enabled && proxy.server) {
    contextOptions.proxy = {
      server: proxy.server,
      ...(proxy.bypass && { bypass: proxy.bypass }),
      ...(proxy.username && { username: proxy.username }),
      ...(proxy.password && { password: proxy.password }),
    };
    logger.debug('Proxy configured', {
      executionId: spec.execution_id,
      server: proxy.server,
      hasBypass: !!proxy.bypass,
      hasAuth: !!proxy.username,
    });
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

  // Apply anti-detection patches
  const hasAntiDetection = Object.values(antiDetection).some(Boolean);
  if (hasAntiDetection) {
    await applyAntiDetection(context, antiDetection, fingerprint);
    logger.debug('Anti-detection patches applied', {
      executionId: spec.execution_id,
      patches: Object.entries(antiDetection)
        .filter(([, v]) => v)
        .map(([k]) => k),
    });
  }

  // Apply ad blocking if configured
  if (isAdBlockingEnabled(antiDetection.ad_blocking_mode)) {
    await applyAdBlocking(
      context,
      antiDetection.ad_blocking_mode as 'ads_only' | 'ads_and_tracking',
      antiDetection.ad_blocking_whitelist
    );
    logger.debug('Ad blocking enabled', {
      executionId: spec.execution_id,
      mode: antiDetection.ad_blocking_mode,
      whitelistDomains: antiDetection.ad_blocking_whitelist?.length ?? 0,
    });
  }

  // Store behavior settings on context for handler access
  const behaviorSettings: BehaviorSettings = {
    typing_delay_min: behavior.typing_delay_min,
    typing_delay_max: behavior.typing_delay_max,
    typing_start_delay_min: behavior.typing_start_delay_min,
    typing_start_delay_max: behavior.typing_start_delay_max,
    typing_paste_threshold: behavior.typing_paste_threshold,
    typing_variance_enabled: behavior.typing_variance_enabled,
    mouse_movement_style: behavior.mouse_movement_style,
    mouse_jitter_amount: behavior.mouse_jitter_amount,
    click_delay_min: behavior.click_delay_min,
    click_delay_max: behavior.click_delay_max,
    scroll_style: behavior.scroll_style,
    scroll_speed_min: behavior.scroll_speed_min,
    scroll_speed_max: behavior.scroll_speed_max,
    micro_pause_enabled: behavior.micro_pause_enabled,
    micro_pause_min_ms: behavior.micro_pause_min_ms,
    micro_pause_max_ms: behavior.micro_pause_max_ms,
    micro_pause_frequency: behavior.micro_pause_frequency,
  };
  (context as any)[BEHAVIOR_SETTINGS_KEY] = behaviorSettings;

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
    viewport: { width: viewportWidth, height: viewportHeight },
    har: !!harPath,
    video: !!videoDir,
    tracing: !!tracePath,
    browserProfile: spec.browser_profile?.preset || 'none',
    hasAntiDetection,
    adBlocking: antiDetection.ad_blocking_mode || 'none',
    proxy: proxy.enabled ? 'enabled' : 'disabled',
    clientHints: clientHints ? 'enabled' : 'none',
    extraHeaders: Object.keys(finalHeaders).length,
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
