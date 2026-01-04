import { mkdir } from 'fs/promises';
import type { Browser, BrowserContext } from 'rebrowser-playwright';
import type { SessionSpec, BehaviorSettings } from '../types';
import type { Config } from '../config';
import { logger } from '../utils';
import { ServiceWorkerController } from '../service-worker';
import {
  RecordingContextInitializer,
  createRecordingContextInitializer,
} from '../recording/context-initializer';
import {
  mergeWithPreset,
  resolveUserAgent,
  applyAntiDetection,
  applyAdBlocking,
  isAdBlockingEnabled,
  generateClientHints,
  mergeClientHintsWithHeaders,
  BEHAVIOR_SETTINGS_KEY,
} from '../browser-profile';
import {
  resolveArtifactPaths,
  getArtifactDir,
} from './artifact-paths';
import {
  logContextOptions,
  logClientHints,
  logAntiDetectionApplied,
  logAdBlockerConfig,
} from './diagnostic-logger';

// Re-export for backward compatibility (canonical location is browser-profile)
export { BEHAVIOR_SETTINGS_KEY } from '../browser-profile';

/**
 * Build BrowserContext from SessionSpec
 *
 * ARCHITECTURE:
 * - Viewport: fingerprint overrides spec
 * - User agent: profile > spec > default
 * - Artifacts: resolved via artifact-paths.ts
 * - Anti-detection: applied via browser-profile module
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
  serviceWorkerController: ServiceWorkerController;
  recordingInitializer: RecordingContextInitializer;
}> {
  // Resolve artifact paths using dedicated module (explicit decision logic)
  const resolvedArtifacts = resolveArtifactPaths(
    spec.artifact_paths,
    spec.required_capabilities,
    spec.execution_id
  );

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

  // HAR recording - path resolved via artifact-paths.ts
  const harPath = resolvedArtifacts.harPath;
  if (harPath) {
    await ensureDir(getArtifactDir(harPath), 'har');
    contextOptions.recordHar = {
      path: harPath,
      mode: 'minimal',
    };
    logger.debug('HAR recording enabled', { harPath });
  }

  // Video recording - path resolved via artifact-paths.ts
  const videoDir = resolvedArtifacts.videoDir;
  if (videoDir) {
    await ensureDir(videoDir, 'video');
    contextOptions.recordVideo = {
      dir: videoDir,
      size: spec.viewport,
    };
    logger.debug('Video recording enabled', { videoDir });
  }

  // Tracing - path resolved via artifact-paths.ts
  const tracePath = resolvedArtifacts.tracePath;
  if (tracePath) {
    await ensureDir(getArtifactDir(tracePath), 'trace');
  }

  // Create context
  const context = await browser.newContext(contextOptions);

  // Log context options for diagnostic debugging
  logContextOptions(spec.execution_id, {
    viewport: contextOptions.viewport,
    userAgent: userAgent.slice(0, 100),
    locale: contextOptions.locale,
    timezoneId: contextOptions.timezoneId,
    colorScheme: contextOptions.colorScheme,
    extraHTTPHeaders: finalHeaders,
    hasProxy: !!contextOptions.proxy,
    hasGeolocation: !!contextOptions.geolocation,
  });

  // Log Client Hints configuration
  logClientHints(spec.execution_id, userAgent, clientHints);

  // Apply anti-detection patches
  const hasAntiDetection = Object.values(antiDetection).some(Boolean);
  if (hasAntiDetection) {
    const enabledPatches = Object.entries(antiDetection)
      .filter(([, v]) => v)
      .map(([k]) => k);
    await applyAntiDetection(context, antiDetection, fingerprint);
    logger.debug('Anti-detection patches applied', {
      executionId: spec.execution_id,
      patches: enabledPatches,
    });
    // Log for diagnostic debugging
    logAntiDetectionApplied(spec.execution_id, enabledPatches);
  }

  // Apply ad blocking if configured (uses script injection, not route interception)
  if (isAdBlockingEnabled(antiDetection.ad_blocking_mode)) {
    const adBlockWhitelist = antiDetection.ad_blocking_whitelist ?? [];
    await applyAdBlocking(
      context,
      antiDetection.ad_blocking_mode as 'ads_only' | 'ads_and_tracking',
      adBlockWhitelist
    );
    logger.debug('Ad blocking enabled', {
      executionId: spec.execution_id,
      mode: antiDetection.ad_blocking_mode,
      whitelistDomains: adBlockWhitelist.length,
    });
    // Log for diagnostic debugging
    logAdBlockerConfig(spec.execution_id, antiDetection.ad_blocking_mode as string, adBlockWhitelist);
  }

  // Initialize service worker controller
  const swControl = spec.service_worker_control || { mode: 'allow' as const };
  const serviceWorkerController = new ServiceWorkerController(spec.execution_id, swControl);

  // Setup blocking scripts if mode requires it
  await serviceWorkerController.setupBlockingForContext(context);

  if (swControl.mode !== 'allow' || swControl.domainOverrides?.length) {
    logger.debug('Service worker control configured', {
      executionId: spec.execution_id,
      mode: swControl.mode,
      blockedDomains: swControl.blockedDomains?.length || 0,
      domainOverrides: swControl.domainOverrides?.length || 0,
      unregisterOnStart: swControl.unregisterOnStart || false,
    });
  }

  // Initialize recording context (binding and init script)
  // This sets up the recording infrastructure that runs in MAIN context
  // The actual recording is activated/deactivated per-session via messages
  const recordingInitializer = createRecordingContextInitializer({ logger });
  await recordingInitializer.initialize(context);
  logger.debug('Recording context initialized', {
    executionId: spec.execution_id,
  });

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

  // Start tracing if enabled (tracePath only set when tracing is requested)
  if (tracePath) {
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
    serviceWorkerController,
    recordingInitializer,
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
