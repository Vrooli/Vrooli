import { mkdir } from 'fs/promises';
import type { Browser, BrowserContext } from 'rebrowser-playwright';
import type { SessionSpec, BehaviorSettings } from '../types';
import type { Config } from '../config';
import { logger } from '../utils';
import { ServiceWorkerController } from '../service-worker';
import {
  RecordingContextInitializer,
  createRecordingContextInitializer,
} from '../recording';
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
import { generateSilentSinkPatch, type AudioStrategy } from './audio';

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
/**
 * Describes what determined the actual viewport dimensions.
 * This attribution helps users understand why dimensions may differ from requested.
 */
export type ViewportSource =
  | 'requested'           // Used the UI-requested dimensions
  | 'fingerprint'         // Browser profile fingerprint override
  | 'fingerprint_partial' // Fingerprint set one dimension, requested used for other
  | 'default';            // Fallback defaults used

export interface ActualViewport {
  width: number;
  height: number;
  source: ViewportSource;
  /** Human-readable explanation of why this source was used */
  reason: string;
}

export async function buildContext(
  browser: Browser,
  spec: SessionSpec,
  config: Config,
  audioStrategy: AudioStrategy = 'host_device',
): Promise<{
  context: BrowserContext;
  harPath?: string;
  tracePath?: string;
  videoDir?: string;
  serviceWorkerController: ServiceWorkerController;
  recordingInitializer: RecordingContextInitializer;
  actualViewport: ActualViewport;
}> {
  // Resolve artifact paths using dedicated module (explicit decision logic)
  const effectiveCapabilities = spec.required_capabilities
    ? { ...spec.required_capabilities }
    : undefined;
  if (effectiveCapabilities?.har && !config.telemetry.har.enabled) {
    effectiveCapabilities.har = false;
  }

  const resolvedArtifacts = resolveArtifactPaths(
    spec.artifact_paths,
    effectiveCapabilities,
    spec.execution_id
  );

  // Merge browser profile with preset defaults
  const { fingerprint, behavior, antiDetection, proxy } = mergeWithPreset(spec.browser_profile);

  // Determine viewport with explicit source attribution
  // We check the EXPLICIT browser_profile fingerprint, not the merged fingerprint,
  // because mergeWithPreset always adds default values (1280x720).
  // Fingerprint viewport is ONLY used when the user explicitly provided a browser_profile
  // with BOTH dimensions set and > 0. This prevents default fingerprint values from
  // overriding the UI-requested viewport.
  const explicitFingerprint = spec.browser_profile?.fingerprint;
  const explicitWidth = explicitFingerprint?.viewport_width;
  const explicitHeight = explicitFingerprint?.viewport_height;
  const explicitHasWidth = typeof explicitWidth === 'number' && explicitWidth > 0;
  const explicitHasHeight = typeof explicitHeight === 'number' && explicitHeight > 0;

  let viewportWidth: number;
  let viewportHeight: number;
  let viewportSource: ViewportSource;
  let viewportReason: string;

  if (explicitWidth !== undefined && explicitHeight !== undefined && explicitWidth > 0 && explicitHeight > 0) {
    // User explicitly provided a browser_profile with complete viewport override
    viewportWidth = explicitWidth;
    viewportHeight = explicitHeight;
    viewportSource = 'fingerprint';
    viewportReason = `Browser profile specifies ${viewportWidth}x${viewportHeight}`;
  } else if (explicitHasWidth || explicitHasHeight) {
    // User provided partial fingerprint override - warn but use requested viewport
    // This is likely unintentional, so we don't apply partial overrides
    logger.warn('Partial viewport override detected in session profile - using requested viewport instead', {
      explicitWidth: explicitFingerprint?.viewport_width,
      explicitHeight: explicitFingerprint?.viewport_height,
      requestedWidth: spec.viewport.width,
      requestedHeight: spec.viewport.height,
    });
    // Fall through to use requested viewport
    if (spec.viewport.width > 0 && spec.viewport.height > 0) {
      viewportWidth = spec.viewport.width;
      viewportHeight = spec.viewport.height;
      viewportSource = 'requested';
      viewportReason = `Using requested ${viewportWidth}x${viewportHeight} (profile had partial override)`;
    } else {
      viewportWidth = 1280;
      viewportHeight = 720;
      viewportSource = 'default';
      viewportReason = 'No valid viewport specified, using default 1280x720';
    }
  } else if (spec.viewport.width > 0 && spec.viewport.height > 0) {
    // Use requested viewport from UI (most common case)
    viewportWidth = spec.viewport.width;
    viewportHeight = spec.viewport.height;
    viewportSource = 'requested';
    viewportReason = `Using requested ${viewportWidth}x${viewportHeight}`;
  } else {
    // Fallback defaults (should rarely happen)
    viewportWidth = 1280;
    viewportHeight = 720;
    viewportSource = 'default';
    viewportReason = 'No viewport specified, using default 1280x720';
    logger.warn('Using default viewport - no valid dimensions provided', {
      specViewport: spec.viewport,
    });
  }

  const actualViewport: ActualViewport = {
    width: viewportWidth,
    height: viewportHeight,
    source: viewportSource,
    reason: viewportReason,
  };

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

  if (audioStrategy === 'synthetic_sink') {
    await context.addInitScript({ content: generateSilentSinkPatch() });
    logger.info('Synthetic Web Audio sink enabled for host without output device', {
      executionId: spec.execution_id,
    });
  }

  // A fixture is a driver-level test-only opt-in. Chromium still obtains a
  // real getUserMedia stream; BrowserManager supplies its deterministic fake
  // device only when BAS_FAKE_MICROPHONE_FILE is configured. The isolated
  // qualification driver grants this permission to its sessions; normal BAS
  // instances do neither.
  if (config.browser.fakeMicrophoneFile) {
    await context.grantPermissions(['microphone']);
    logger.debug('Deterministic microphone permission granted', {
      executionId: spec.execution_id,
    });
  }

  // Fix gray bar in video recording caused by window/viewport size mismatch.
  //
  // ROOT CAUSE (Playwright issue #36032): When launched with headless:false +
  // --headless=new, Playwright calculates the window size as viewport + browser
  // chrome height (assuming a headed browser). But --headless=new means there's
  // no actual chrome, so the window is taller than the viewport. Chrome's video
  // encoder captures the full window area, but only viewport-height pixels have
  // rendered page content — the rest fills with gray (VP8 framebuffer default
  // RGB 128,128,128).
  //
  // FIX: Override the screen dimensions via CDP so the compositor knows the
  // true renderable area. This must be applied to each page since it's a
  // per-target CDP command. The 'page' event fires before any navigation,
  // giving us time to set the override before frames are captured.
  //
  // See: docs/bugs/VIDEO_BOTTOM_FLICKER.md for full investigation.
  if (videoDir) {
    context.on('page', (page) => {
      context.newCDPSession(page).then((session) => {
        return session.send('Emulation.setDeviceMetricsOverride', {
          width: viewportWidth,
          height: viewportHeight,
          deviceScaleFactor,
          mobile: false,
          screenWidth: viewportWidth,
          screenHeight: viewportHeight,
        });
      }).catch((err) => {
        // Non-fatal: video may have gray bar but functionality is unaffected
        logger.warn('Failed to apply video recording screen metrics override', {
          executionId: spec.execution_id,
          error: err instanceof Error ? err.message : String(err),
        });
      });
    });
    logger.debug('Video recording screen metrics fix registered', {
      executionId: spec.execution_id,
      viewport: `${viewportWidth}x${viewportHeight}`,
      deviceScaleFactor,
    });

    // Stabilize viewport layout for video recording. When page content changes
    // during execution, a vertical scrollbar can appear/disappear, changing the
    // content width by ~17px. CSS reflow cascades downward, so the bottom ~50px
    // of the recorded video flickers visibly as elements shift horizontally.
    // Forcing a permanent scrollbar and hiding horizontal overflow prevents the
    // content width from ever changing. This is the same fix applied in the
    // export/render capture path (playwright_client.go viewportStabilizationScript).
    await context.addInitScript(() => {
      const style = document.createElement('style');
      style.id = 'bas-viewport-stabilize';
      style.textContent = 'html{overflow-y:scroll!important}body{overflow-x:hidden!important}';
      (document.head || document.documentElement).appendChild(style);
    });
    logger.debug('Viewport stabilization injected for video recording', {
      executionId: spec.execution_id,
    });
  }

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
    await applyAntiDetection(context, antiDetection, fingerprint, {
      deterministicAudio: Boolean(spec.fake_media?.microphone_wav),
    });
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
  Object.defineProperty(context, BEHAVIOR_SETTINGS_KEY, {
    value: behaviorSettings,
    writable: true,
    configurable: true,
  });

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
    viewport: {
      width: viewportWidth,
      height: viewportHeight,
      source: viewportSource,
      requestedWidth: spec.viewport.width,
      requestedHeight: spec.viewport.height,
    },
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
    actualViewport,
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
