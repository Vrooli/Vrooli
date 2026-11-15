import { randomUUID } from 'node:crypto';
import type {
  Browser,
  BrowserContext,
  Locator,
  Page,
  Request,
  Response,
} from 'rebrowser-playwright';
import { createPublicCaptureContext } from './public-egress';
import { collectFrozenPageInPage, serializeFrozenPage, type FrozenFrameTile } from './frozen-page';
import { collectOutlineInPage, type OutlineResult } from './outline';
import { collectComponentIdentityInPage, type ComponentIdentityResult } from './identity';
import { redactPageInPage, type RedactionReport } from './redaction';
import {
  isBlankPixels,
  measurePng,
  waitForPaintReadinessInPage,
  type PaintReadiness,
  type PixelStats,
} from './paint';
import {
  collectRclStyleTokensInPage,
  collectStyleCensusInPage,
  type StyleCensusResult,
  type StyleToken,
} from './style-census';
import { applyStrictTokenSentinelInPage, type StrictTokenDiagnostic } from './strict-tokens';

export type PublicPageCaptureOptions = {
  url: string;
  scenario?: string;
  route?: string;
  steps?: string[];
  /** Wait for a visible selector after navigation; scenario captures only. */
  waitSelector?: string;
  /** Wait for a selector to be hidden after navigation; scenario captures only. */
  waitHiddenSelector?: string;
  theme?: 'light' | 'dark' | 'no-preference';
  strictTokens?: boolean;
  /** Opt-in bounded RCL component test evidence; scenario captures only. */
  evidenceProfile?: 'componenttests';
  regionSelector?: string;
  /** Optional bounded full-page image for consumers that compare whole documents. */
  fullPagePixels?: boolean;
  allowedOrigins?: string[];
  width?: number;
  height?: number;
  outlineTokenBudget?: number;
  frozenPageByteBudget?: number;
  styleTokens?: StyleToken[];
  maxScreenshotBytes?: number;
  maxLensJsonBytes?: number;
  waitUntil?: 'domcontentloaded' | 'load';
  timeoutMs?: number;
  html?: string;
  filename?: string;
};

export type LensResult = { status: 'present' | 'partial' | 'absent'; reason?: string };
export type PublicPageCaptureResult = {
  schema_version: 'ux-capture.v1';
  id: string;
  captured_at: string;
  source:
    | { kind: 'url'; url: string }
    | { kind: 'file'; filename: string }
    | { kind: 'scenario'; scenario: string; route: string };
  session: { id: string; browser: string };
  viewport: { width: number; height: number };
  theme: 'light' | 'dark' | 'no-preference';
  lenses: Record<'L0' | 'L1' | 'L2' | 'L3' | 'L4' | 'L5' | 'L6', LensResult>;
  privacy: {
    redaction_status: 'complete' | 'partial';
    redaction_reason?: string;
    redacted_classes: string[];
    external: boolean;
    ephemeral: boolean;
  };
  artifacts: {
    screenshot_png_base64?: string;
    full_page_pixels_png_base64?: string;
    full_page_pixels_status?: LensResult;
    frozen_page_html?: string;
    outline?: OutlineResult;
    style_census?: StyleCensusResult;
    network?: Array<{ origin: string; resource_type: string; method: string; status?: number }>;
    component_identity?: ComponentIdentityResult;
    strict_tokens?: StrictTokenDiagnostic;
    region_png_base64?: string;
    region_style_sample?: {
      schema: 'bas-region-style-sample/v1';
      computed_style: {
        color: string;
        background_color: string;
        border_color: string;
        border_style: string;
        border_width: string;
        visibility: string;
        display: string;
        opacity: string;
      };
    };
    accessibility_snapshot?: { schema: 'bas-accessibility-snapshot/v1'; snapshot: string };
    console_summary?: {
      schema: 'bas-console-summary/v1';
      error_count: number;
      warning_count: number;
    };
    performance_metrics?: {
      schema: 'bas-performance-metrics/v1';
      dom_content_loaded_ms: number | null;
      load_ms: number | null;
      resource_count: number;
    };
  };
};

type CapturePageDependencies = {
  id?: () => string;
  now?: () => Date;
  redact?: (page: Page) => Promise<RedactionReport>;
  capturePixels?: (page: Page) => Promise<Buffer>;
  waitForPaint?: (page: Page) => Promise<PaintReadiness>;
  /** Pause between blank-pixel retries; tests pass 0. */
  blankRetryDelayMs?: number;
  collectFrozen?: (page: Page) => Promise<ReturnType<typeof serializeFrozenPage>>;
  collectOutline?: (page: Page, budget: number) => Promise<OutlineResult>;
  collectIdentity?: (page: Page) => Promise<ComponentIdentityResult>;
  collectStyle?: (page: Page, tokens?: StyleToken[]) => Promise<StyleCensusResult>;
};

type NetworkRecord = { origin: string; resource_type: string; method: string; status?: number };
const safeCaptureResourceTypes = new Set([
  'document',
  'stylesheet',
  'image',
  'media',
  'font',
  'script',
  'texttrack',
  'xhr',
  'fetch',
  'eventsource',
  'websocket',
  'manifest',
  'other',
]);

/**
 * Captures a public page through the fail-closed unauthenticated browser
 * context. The function returns only in-memory, redacted lens artifacts; the
 * caller owns private storage and ephemeral external-bundle cleanup.
 */
export async function capturePublicPageFromURL(
  browser: Browser,
  options: PublicPageCaptureOptions,
  dependencies: CapturePageDependencies = {}
): Promise<PublicPageCaptureResult> {
  const opened = await createPublicCaptureContext(browser, options.url);
  try {
    return await capturePublicPageInContext(
      opened.context,
      opened.initialUrl,
      options,
      dependencies
    );
  } finally {
    await opened.close();
  }
}

/** File captures use an isolated script-disabled context and a deny-all request route. */
export async function captureHTMLFile(
  browser: Browser,
  options: PublicPageCaptureOptions,
  dependencies: CapturePageDependencies = {}
): Promise<PublicPageCaptureResult> {
  if (
    typeof options.html !== 'string' ||
    Buffer.byteLength(options.html, 'utf8') < 1 ||
    Buffer.byteLength(options.html, 'utf8') > 2 * 1024 * 1024
  )
    throw new Error('HTML file must be between 1 byte and 2 MiB');
  const filename = sanitizeFilename(options.filename ?? 'capture.html');
  // Reject URL-like active protocols before they enter Chromium. Normal relative and https links remain inert;
  // all requests are denied below and scripts are disabled at the context level.
  if (
    /\b(?:href|src|action|formaction|xlink:href)\s*=\s*["']?\s*(?:javascript|vbscript|file):/i.test(
      options.html
    )
  )
    throw new Error('HTML file contains an unsafe active URL scheme');
  const context = await browser.newContext({ javaScriptEnabled: false });
  try {
    await context.route('**/*', (route) => route.abort('blockedbyclient'));
    return await capturePublicPageInContext(
      context,
      '',
      options,
      dependencies,
      { kind: 'file', filename },
      true
    );
  } finally {
    await context.close();
  }
}

/** Synthetic origin for file captures. `.invalid` never resolves, so relative
 * links in the artifact cannot reach a real host even if a route were missed. */
export const FILE_CAPTURE_URL = 'http://frozen.capture.invalid/page.html';

/** Remove parser-blocking script source before Chromium parses a local artifact. */
export function inertHTMLFileBeforeParse(html: string): string {
  // Script elements are raw-text in HTML. Removing them before setContent is
  // necessary because a denied external script can hold DOMContentLoaded open
  // even when JavaScript is disabled for the context. The post-parse DOM pass
  // below still removes other active surfaces and event attributes.
  return html.replace(/<script\b[^>]*>[\s\S]*?(?:<\/script\s*>|$)/gi, '');
}

/** Captures a control-plane-resolved local scenario UI with network egress denied. */
export async function captureScenarioPage(
  browser: Browser,
  options: PublicPageCaptureOptions,
  dependencies: CapturePageDependencies = {}
): Promise<PublicPageCaptureResult> {
  const url = validateScenarioTarget(options.url, options.allowedOrigins ?? []);
  const context = await browser.newContext({ colorScheme: options.theme ?? 'light' });
  const allowed = new Set(
    (options.allowedOrigins?.length ? options.allowedOrigins : [url.origin]).map(
      (value) => new URL(value).origin
    )
  );
  await context.route('**/*', (route) => {
    try {
      const requestUrl = new URL(route.request().url());
      if (
        (requestUrl.protocol === 'http:' || requestUrl.protocol === 'https:') &&
        allowed.has(requestUrl.origin)
      )
        return route.continue();
    } catch {
      /* deny malformed and non-network requests */
    }
    return route.abort('blockedbyclient');
  });
  if (typeof context.routeWebSocket === 'function') {
    await context.routeWebSocket('**/*', (socket) => {
      try {
        const socketURL = new URL(socket.url());
        const origin =
          socketURL.protocol === 'ws:'
            ? `http://${socketURL.host}`
            : socketURL.protocol === 'wss:'
              ? `https://${socketURL.host}`
              : socketURL.origin;
        if (allowed.has(origin)) void socket.connectToServer();
        else
          void socket.close({ code: 1008, reason: 'scenario capture origin is not allowlisted' });
      } catch {
        void socket.close({ code: 1008, reason: 'invalid websocket origin' });
      }
    });
  }
  try {
    return await capturePublicPageInContext(context, url.href, options, dependencies, {
      kind: 'scenario',
      scenario: options.scenario ?? 'unknown',
      route: options.route ?? `${url.pathname}${url.search}`,
    });
  } finally {
    await context.close();
  }
}

function validateScenarioTarget(raw: string, allowedOrigins: string[]): URL {
  let url: URL;
  try {
    url = new URL(raw);
  } catch {
    throw new Error('Resolved scenario URL is invalid');
  }
  if (url.protocol !== 'http:' && url.protocol !== 'https:')
    throw new Error('Scenario URL must use HTTP or HTTPS');
  if (url.username || url.password) throw new Error('Scenario URL cannot contain credentials');
  const host = url.hostname.toLowerCase().replace(/^\[|\]$/g, '');
  if (!(host === 'localhost' || host === '127.0.0.1' || host === '::1'))
    throw new Error('Scenario URL must resolve to loopback');
  const allowed = allowedOrigins
    .map((value) => {
      try {
        return new URL(value).origin;
      } catch {
        return '';
      }
    })
    .filter(Boolean);
  if (allowed.length && !allowed.includes(url.origin))
    throw new Error('Scenario URL origin is not allowlisted');
  return url;
}

function sanitizeFilename(value: string): string {
  const leaf = value.replace(/\\/g, '/').split('/').pop() ?? 'capture.html';
  const clean = leaf
    .replace(/[^a-zA-Z0-9._ -]/g, '_')
    .slice(0, 120)
    .trim();
  return clean || 'capture.html';
}

/** Exposed separately so route tests can exercise the one-session lens flow. */
export async function capturePublicPageInContext(
  context: BrowserContext,
  safeUrl: string,
  options: PublicPageCaptureOptions,
  dependencies: CapturePageDependencies = {},
  source:
    | { kind: 'url'; url: string }
    | { kind: 'file'; filename: string }
    | { kind: 'scenario'; scenario: string; route: string } = { kind: 'url', url: safeUrl },
  networkBlocked = false
): Promise<PublicPageCaptureResult> {
  const id = dependencies.id ?? randomUUID;
  const now = dependencies.now ?? (() => new Date());
  const page = await context.newPage();
  const width = boundedDimension(options.width, 1280);
  const height = boundedHeight(options.height, 720);
  await page.setViewportSize({ width, height });
  const theme = options.theme ?? 'light';
  await page.emulateMedia({ colorScheme: theme });

  // Retain only request handles until the privacy prepass finishes. Network
  // metadata is projected to an origin-only report after redaction succeeds.
  const requests = new Map<Request, NetworkRecord>();
  const responses: Response[] = [];
  const failedResourceCounts = new Map<string, number>();
  const removedFileScripts =
    source.kind === 'file' ? (options.html?.match(/<script\b/gi)?.length ?? 0) : 0;
  let consoleErrors = 0;
  let consoleWarnings = 0;
  page.on('console', (message) => {
    if (message.type() === 'error') consoleErrors++;
    else if (message.type() === 'warning') consoleWarnings++;
  });
  let networkOverflow = false;
  const recordRequest = (request: Request): void => {
    try {
      if (!requests.has(request) && requests.size >= 500) {
        networkOverflow = true;
        return;
      }
      requests.set(request, { origin: '', resource_type: '', method: '' });
    } catch {
      // Malformed and non-HTTP URLs are not included in the redacted summary.
    }
  };
  const recordResponse = (response: Response): void => {
    responses.push(response);
  };
  page.on('request', recordRequest);
  page.on('response', recordResponse);
  // A page can look like a valid pixel capture while a required chart or app
  // bundle was denied by the scenario egress policy. Keep only a count here;
  // request URLs remain in the existing post-redaction projection path.
  page.on('requestfailed', (request) => {
    try {
      const resourceType = request.resourceType();
      const key = safeCaptureResourceTypes.has(resourceType) ? resourceType : 'other';
      failedResourceCounts.set(key, (failedResourceCounts.get(key) ?? 0) + 1);
    } catch {
      // Diagnostics must not fail a capture when a browser request is unusual.
    }
  });

  let paint: PaintReadiness | undefined;
  if (source.kind === 'file') {
    // Serve the inert document from a synthetic URL rather than setContent:
    // under rebrowser-playwright, setContent timed out for every document
    // (even `<h1>Hi</h1>`, scripts on or off), so every file capture failed. The page-level route wins over the context's deny-all route
    // for this one URL; every other request is still denied.
    const body = inertHTMLFileBeforeParse(options.html ?? '');
    await page.route(FILE_CAPTURE_URL, (route) =>
      route.fulfill({ status: 200, contentType: 'text/html; charset=utf-8', body })
    );
    await page.goto(FILE_CAPTURE_URL, { waitUntil: 'domcontentloaded', timeout: 15_000 });
    await page.evaluate(() => {
      document.querySelectorAll('script, iframe, object, embed').forEach((node) => node.remove());
      for (const element of Array.from(document.querySelectorAll('*')))
        for (const attribute of Array.from(element.attributes)) {
          if (/^on/i.test(attribute.name) || attribute.name.toLowerCase() === 'srcdoc')
            element.removeAttribute(attribute.name);
          if (
            /^(href|src|action|formaction|xlink:href)$/i.test(attribute.name) &&
            /^(?:javascript|vbscript|file):/i.test(attribute.value.trim())
          )
            element.removeAttribute(attribute.name);
        }
    });
  } else {
    await page.goto(safeUrl, {
      // Scenario showcases are local application pages whose main CSS and
      // document scripts must finish before we freeze pixels. Waiting only
      // for DOMContentLoaded can capture a transient unstyled shell.
      waitUntil: options.waitUntil ?? (source.kind === 'scenario' ? 'load' : 'domcontentloaded'),
      timeout: Math.max(1_000, Math.min(options.timeoutMs ?? 45_000, 120_000)),
    });
    // `load` fires before a single-page app commits its first render. Wait for
    // visible content and a quiet DOM before state steps (their targets must
    // exist) and again before the privacy prepass, so redaction, pixels and
    // every later lens see the same rendered page.
    const waitForPaint = async (): Promise<PaintReadiness | undefined> => {
      try {
        return await (
          dependencies.waitForPaint ??
          ((target) =>
            target.evaluate(waitForPaintReadinessInPage, { timeoutMs: 10_000, quietMs: 300 }))
        )(page);
      } catch {
        return undefined;
      }
    };
    paint = await waitForPaint();
    if (source.kind === 'scenario') {
      if (options.waitSelector)
        await page.waitForSelector(options.waitSelector, { state: 'visible', timeout: 10_000 });
      if (options.waitHiddenSelector)
        await page.waitForSelector(options.waitHiddenSelector, {
          state: 'hidden',
          timeout: 15_000,
        });
      if (options.steps?.length) {
        await runScenarioSteps(page, options.steps);
        paint = await waitForPaint();
      }
    }
  }
  // Fail closed: a prepass that throws blocks every content lens.
  const redact = async (target: Page): Promise<RedactionReport> => {
    try {
      return await (dependencies.redact ?? ((value) => value.evaluate(redactPageInPage)))(target);
    } catch (error) {
      return {
        schema: 'bas-capture-redaction/v1',
        status: 'blocked',
        redacted: {
          formValues: 0,
          markedRegions: 0,
          assignments: 0,
          attributes: 0,
          urlParameters: 0,
          personalData: 0,
        },
        unsupported: [
          {
            surface: 'redaction',
            reason: error instanceof Error ? error.message : 'privacy prepass failed',
          },
        ],
      };
    }
  };
  let redaction = await redact(page);

  let strictTokens: StrictTokenDiagnostic | undefined;
  let strictTokenError = '';
  if (options.strictTokens && redaction.status === 'safe') {
    try {
      strictTokens = await page.evaluate(applyStrictTokenSentinelInPage);
    } catch (error) {
      strictTokenError = error instanceof Error ? error.message : 'strict-token diagnostic failed';
    }
  }

  // Component-test evidence is opt-in and collected only after the shared
  // redaction prepass. Console output is count-only to avoid leaking messages.
  let region: Buffer | undefined;
  let regionStyleSample: PublicPageCaptureResult['artifacts']['region_style_sample'];
  let accessibility: { schema: 'bas-accessibility-snapshot/v1'; snapshot: string } | undefined;
  let consoleSummary:
    | { schema: 'bas-console-summary/v1'; error_count: number; warning_count: number }
    | undefined;
  let performanceMetrics:
    | {
        schema: 'bas-performance-metrics/v1';
        dom_content_loaded_ms: number | null;
        load_ms: number | null;
        resource_count: number;
      }
    | undefined;
  if (
    options.evidenceProfile === 'componenttests' &&
    redaction.status === 'safe' &&
    source.kind === 'scenario'
  ) {
    if (options.regionSelector) {
      try {
        // Validate as browser CSS and require an unambiguous region before
        // asking Playwright to screenshot it.
        const matches = await page.evaluate(
          (selector) => document.querySelectorAll(selector).length,
          options.regionSelector
        );
        if (matches === 1) {
          const target = page.locator(options.regionSelector);
          if ((await target.count()) === 1) {
            region = await target.screenshot({
              type: 'png',
              animations: 'disabled',
              timeout: 5000,
            });
            regionStyleSample = await target.evaluate((element) => {
              const style = getComputedStyle(element);
              return {
                schema: 'bas-region-style-sample/v1' as const,
                computed_style: {
                  color: style.color.slice(0, 128),
                  background_color: style.backgroundColor.slice(0, 128),
                  border_color: style.borderColor.slice(0, 128),
                  border_style: style.borderStyle.slice(0, 128),
                  border_width: style.borderWidth.slice(0, 128),
                  visibility: style.visibility.slice(0, 128),
                  display: style.display.slice(0, 128),
                  opacity: style.opacity.slice(0, 128),
                },
              };
            });
          }
        }
      } catch {
        /* invalid, absent, or oversized crop remains omitted */
      }
    }
    try {
      const snapshot = await page.locator('body').ariaSnapshot({ timeout: 5000 });
      accessibility = {
        schema: 'bas-accessibility-snapshot/v1',
        snapshot: snapshot.slice(0, 32 * 1024),
      };
    } catch {
      /* accessibility snapshot can be unavailable on older drivers */
    }
    try {
      let errors = 0,
        warnings = 0;
      // Counts are incremented by listeners installed before navigation below.
      errors = consoleErrors;
      warnings = consoleWarnings;
      consoleSummary = {
        schema: 'bas-console-summary/v1',
        error_count: errors,
        warning_count: warnings,
      };
    } catch {
      /* count-only summary is optional evidence */
    }
    try {
      performanceMetrics = await page.evaluate(() => {
        const nav = performance.getEntriesByType('navigation')[0] as
          | PerformanceNavigationTiming
          | undefined;
        return {
          schema: 'bas-performance-metrics/v1' as const,
          dom_content_loaded_ms: nav ? Math.round(nav.domContentLoadedEventEnd) : null,
          load_ms: nav ? Math.round(nav.loadEventEnd) : null,
          resource_count: Math.min(performance.getEntriesByType('resource').length, 500),
        };
      });
    } catch {
      /* performance API is optional evidence */
    }
  }

  const screenshotLimit = Math.max(64 * 1024, options.maxScreenshotBytes ?? 20 * 1024 * 1024);
  const lensJsonLimit = Math.max(8 * 1024, options.maxLensJsonBytes ?? 1024 * 1024);
  let screenshot: Buffer | undefined;
  let screenshotError = '';
  let pixelStats: PixelStats | undefined;
  let blankAttempts = 0;
  if (redaction.status === 'safe') {
    const shoot =
      dependencies.capturePixels ??
      // EX9 L0 is an exact viewport slice. Full-page capture can allocate
      // unbounded image memory and mislabel a different canvas as pixels.
      ((target: Page) =>
        target.screenshot({ type: 'png', fullPage: false, animations: 'disabled' }));
    try {
      // A flat image is never reported as pixels. Late-painting pages get two
      // more chances; each retry re-runs redaction on the newly painted DOM.
      for (let attempt = 1; attempt <= 3; attempt++) {
        screenshot = await shoot(page);
        try {
          pixelStats = measurePng(screenshot);
        } catch {
          pixelStats = undefined;
        }
        if (!pixelStats || !isBlankPixels(pixelStats)) break;
        blankAttempts = attempt;
        if (attempt === 3 || source.kind === 'file') break;
        await new Promise((resolve) =>
          setTimeout(resolve, dependencies.blankRetryDelayMs ?? 1_000)
        );
        redaction = await redact(page);
        if (redaction.status !== 'safe') {
          screenshot = undefined;
          break;
        }
      }
    } catch (error) {
      screenshotError = error instanceof Error ? error.message : 'screenshot failed';
    }
  }
  const blankPixels = pixelStats !== undefined && isBlankPixels(pixelStats);
  const screenshotFits =
    !blankPixels && screenshot !== undefined && screenshot.byteLength <= screenshotLimit;
  const blankReason = blankPixels
    ? `Pixels were one flat colour (${((pixelStats?.dominantShare ?? 1) * 100).toFixed(1)}% of the image) ` +
      `after ${blankAttempts} attempt${blankAttempts === 1 ? '' : 's'}; the page painted no visible content` +
      (paint && !paint.content ? ' (no visible text or media appeared within 10 s).' : '.')
    : '';

  let fullPageScreenshot: Buffer | undefined;
  let fullPageScreenshotError = '';
  const fullPagePixelLimit = 20_000_000;
  if (options.fullPagePixels && redaction.status === 'safe') {
    try {
      const dimensions = await page.evaluate(() => ({
        width: Math.max(document.documentElement.scrollWidth, document.body?.scrollWidth ?? 0),
        height: Math.max(document.documentElement.scrollHeight, document.body?.scrollHeight ?? 0),
      }));
      if (dimensions.width <= 0 || dimensions.height <= 0) {
        fullPageScreenshotError = 'Full-page image has no measurable document dimensions.';
      } else if (dimensions.width * dimensions.height > fullPagePixelLimit) {
        fullPageScreenshotError = `Full-page image exceeds the ${fullPagePixelLimit.toLocaleString()}-pixel area limit.`;
      } else {
        const candidate = await page.screenshot({
          type: 'png',
          fullPage: true,
          animations: 'disabled',
        });
        if (candidate.byteLength > screenshotLimit)
          fullPageScreenshotError = `Full-page image exceeded the ${screenshotLimit}-byte artifact limit.`;
        else fullPageScreenshot = candidate;
      }
    } catch (error) {
      fullPageScreenshotError =
        error instanceof Error ? error.message : 'Full-page screenshot failed.';
    }
  } else if (options.fullPagePixels) {
    fullPageScreenshotError =
      'Privacy prepass did not clear page surfaces; full-page pixels were not emitted.';
  }
  // The outline runs before the frozen page so its node ids are stamped into
  // the frozen markup (`data-ux-node`), which makes regions editable by id.
  let outline: OutlineResult | undefined;
  let outlineError = '';
  if (redaction.status === 'safe') {
    try {
      outline = await (
        dependencies.collectOutline ??
        ((target, budget) =>
          target.evaluate(collectOutlineInPage, {
            tokenBudget: budget,
            stampAttribute: 'data-ux-node',
          }))
      )(page, options.outlineTokenBudget ?? 8000);
    } catch (error) {
      outlineError = error instanceof Error ? error.message : 'outline collection failed';
    }
  } else
    outlineError =
      'Privacy prepass did not clear page surfaces; text and actions were not emitted.';
  const frameOnlyRedactionBlock =
    redaction.status === 'blocked' &&
    redaction.unsupported.length > 0 &&
    redaction.unsupported.every((surface) => surface.surface === 'iframe');
  let frozen: ReturnType<typeof serializeFrozenPage> | undefined;
  let frozenError = '';
  if (redaction.status === 'safe' || frameOnlyRedactionBlock) {
    try {
      frozen = await (
        dependencies.collectFrozen ??
        (async (target) => {
          const snapshot = await target.evaluate(collectFrozenPageInPage);
          snapshot.frameTiles = await collectFrameScreenshotTiles(target);
          return serializeFrozenPage(snapshot, { byteBudget: options.frozenPageByteBudget });
        })
      )(page);
    } catch (error) {
      frozenError = error instanceof Error ? error.message : 'frozen page collection failed';
    }
  } else frozenError = 'Privacy prepass did not clear page surfaces; frozen HTML was not emitted.';
  let identity: ComponentIdentityResult | undefined;
  let identityError = '';
  if (redaction.status === 'safe') {
    try {
      identity = await (
        dependencies.collectIdentity ??
        ((target) => target.evaluate(collectComponentIdentityInPage))
      )(page);
    } catch (error) {
      identityError =
        error instanceof Error ? error.message : 'component identity collection failed';
    }
  } else
    identityError =
      'Privacy prepass did not clear page surfaces; component identity was not emitted.';
  let style: StyleCensusResult | undefined;
  let styleError = '';
  if (redaction.status === 'safe') {
    try {
      const tokens = options.styleTokens ?? (await page.evaluate(collectRclStyleTokensInPage));
      style = dependencies.collectStyle
        ? await dependencies.collectStyle(page, tokens)
        : await page.evaluate(collectStyleCensusInPage, { tokens });
    } catch (error) {
      styleError = error instanceof Error ? error.message : 'style census collection failed';
    }
  } else
    styleError = 'Privacy prepass did not clear page surfaces; style selectors were not emitted.';
  const outlineFits = outline !== undefined && JSON.stringify(outline).length <= lensJsonLimit;
  const styleFits = style !== undefined && JSON.stringify(style).length <= lensJsonLimit;
  const identityFits = identity !== undefined && JSON.stringify(identity).length <= lensJsonLimit;
  const network: NetworkRecord[] = [];
  if (!networkBlocked && redaction.status === 'safe') {
    for (const request of requests.keys()) {
      try {
        const parsed = new URL(request.url());
        if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') continue;
        const record: NetworkRecord = {
          origin: parsed.origin,
          resource_type: request.resourceType(),
          method: request.method(),
        };
        const matchingResponse: Response | undefined = responses.find((candidate: Response) => {
          try {
            const responseRequest = candidate.request();
            return (
              responseRequest === request ||
              (candidate.url() === request.url() && responseRequest.method() === request.method())
            );
          } catch {
            return false;
          }
        });
        if (matchingResponse) record.status = matchingResponse.status();
        network.push(record);
      } catch {
        // Malformed and non-HTTP URLs are not included in the redacted summary.
      }
    }
  }
  const serializedNetwork = JSON.stringify(network);
  const networkFits = serializedNetwork.length <= lensJsonLimit;
  const safeUrlForManifest = source.kind === 'url' ? redactUrlForManifest(safeUrl) : '';
  const failedResourceCount = Array.from(failedResourceCounts.values()).reduce(
    (sum, count) => sum + count,
    0
  );
  const missingResources = failedResourceCount + removedFileScripts;
  const omissionReasons = [
    ...(removedFileScripts > 0
      ? [
          `${removedFileScripts} script${removedFileScripts === 1 ? '' : 's'} removed by the safe file import`,
        ]
      : []),
    ...Array.from(failedResourceCounts.entries())
      .sort(([left], [right]) => left.localeCompare(right))
      .map(([type, count]) => {
        const noun = `${type} request${count === 1 ? '' : 's'}`;
        const failure = count === 1 ? 'failed or was blocked' : 'failed or were blocked';
        return `${count} ${noun} ${failure}`;
      }),
  ];
  const missingResourceReason =
    `${omissionReasons.join('; ')} before pixels were captured; ` +
    'rendered content may be incomplete.';

  const lenses: PublicPageCaptureResult['lenses'] = {
    L0:
      redaction.status !== 'safe'
        ? {
            status: 'absent',
            reason: `Redaction prepass blocked pixel output: ${redaction.unsupported.map((item) => item.surface).join(', ')}`,
          }
        : blankPixels
          ? { status: 'absent', reason: blankReason }
          : !screenshotFits
            ? {
                status: 'partial',
                reason:
                  screenshotError ||
                  `Screenshot exceeded ${screenshotLimit} bytes or was unavailable.`,
              }
            : missingResources > 0
              ? {
                  status: 'partial',
                  reason: missingResourceReason,
                }
              : { status: 'present' },
    L1:
      redaction.status !== 'safe' && !frameOnlyRedactionBlock
        ? {
            status: 'absent',
            reason: `Redaction prepass blocked frozen-page output: ${redaction.unsupported.map((item) => item.surface).join(', ')}`,
          }
        : !frozen
          ? {
              status: 'absent',
              reason: frozenError || 'Frozen-page serializer did not return a result.',
            }
          : frozen.status === 'partial'
            ? {
                status: 'partial',
                reason:
                  frozen.unsupported.length > 0
                    ? frozen.unsupported
                        .slice(0, 5)
                        .map((item) => `${item.surface}: ${item.reason}`)
                        .join('; ')
                    : frozen.reasons.join(' ') || 'Frozen page is partial.',
              }
            : { status: 'present' },
    L2: !outline
      ? { status: 'absent', reason: outlineError || 'Outline collection failed.' }
      : outline.overBudget
        ? {
            status: 'partial',
            reason: `All ${outline.actionCount} actions were retained, but the outline estimate (${outline.estimatedTokens}) exceeded ${outline.tokenBudget} tokens.`,
          }
        : !outlineFits
          ? {
              status: 'partial',
              reason: `Outline exceeded ${lensJsonLimit} bytes; output omitted.`,
            }
          : outline.unsupported.length
            ? {
                status: 'partial',
                reason: `${outline.unsupported.length} unsupported page surface(s) are described in the outline.`,
              }
            : { status: 'present' },
    L3: !style
      ? { status: 'absent', reason: styleError || 'Style census collection failed.' }
      : !styleFits
        ? {
            status: 'partial',
            reason: `Style census exceeded ${lensJsonLimit} bytes; output omitted.`,
          }
        : (failedResourceCounts.get('stylesheet') ?? 0) > 0
          ? {
              status: 'partial',
              reason: `${failedResourceCounts.get('stylesheet')} stylesheet request${failedResourceCounts.get('stylesheet') === 1 ? ' failed or was blocked' : 's failed or were blocked'}; measured styles may be incomplete.`,
            }
          : style.tokenCatalogStatus !== 'rcl'
            ? {
                status: 'partial',
                reason:
                  'RCL token stylesheet was not present; measured styles are retained without an RCL token match.',
              }
            : style.clusters.some((cluster) => cluster.token === null)
              ? {
                  status: 'partial',
                  reason:
                    'Some measured styles have no comparable RCL token; unmatched values are identified in the census.',
                }
              : style.sampledElementCount < style.elementCount
                ? {
                    status: 'partial',
                    reason: `Style census sampled ${style.sampledElementCount} of ${style.elementCount} elements.`,
                  }
                : { status: 'present' },
    L4: !identity
      ? { status: 'absent', reason: identityError || 'Component identity collection failed.' }
      : !identityFits
        ? {
            status: 'partial',
            reason: `Component identity exceeded ${lensJsonLimit} bytes; output omitted.`,
          }
        : identity.stamped_count === 0
          ? {
              status: 'absent',
              reason: 'No data-rcl-asset and data-rcl-version component stamps were found.',
            }
          : identity.stamped_count < identity.element_count
            ? {
                status: 'partial',
                reason: `Exact RCL identity found for ${identity.stamped_count} of ${identity.element_count} stamped component elements; unstamped custom components are not identified.`,
              }
            : { status: 'present' },
    L5:
      redaction.status !== 'safe'
        ? {
            status: 'absent',
            reason:
              'Redaction prepass did not clear page surfaces; network metadata was not emitted.',
          }
        : networkBlocked
          ? {
              status: 'absent',
              reason:
                'Network access is disabled for local file captures; no network lens was collected.',
            }
          : !networkFits || networkOverflow
            ? {
                status: 'partial',
                reason:
                  'Redacted request metadata exceeded the capture budget; bodies and headers are not retained.',
              }
            : { status: 'present' },
    L6:
      source.kind === 'scenario' && (options.steps?.length ?? 0) > 0
        ? { status: 'present' }
        : {
            status: 'absent',
            reason:
              source.kind === 'scenario'
                ? 'No interaction state steps were requested.'
                : 'State steps are available only for a running scenario capture.',
          },
  };
  if (options.strictTokens && strictTokenError) {
    lenses.L3 = {
      status: 'partial',
      reason: `Strict-token diagnostic failed: ${strictTokenError}`,
    };
  }

  const artifacts: PublicPageCaptureResult['artifacts'] = {
    ...(screenshotFits ? { screenshot_png_base64: screenshot?.toString('base64') } : {}),
    ...(fullPageScreenshot
      ? { full_page_pixels_png_base64: fullPageScreenshot.toString('base64') }
      : {}),
    ...(options.fullPagePixels
      ? {
          full_page_pixels_status: fullPageScreenshot
            ? { status: 'present' as const }
            : {
                status: 'absent' as const,
                reason: fullPageScreenshotError || 'Full-page image was unavailable.',
              },
        }
      : {}),
    ...(frozen && frozen.html ? { frozen_page_html: frozen.html } : {}),
    ...(outlineFits && outline ? { outline } : {}),
    ...(styleFits && style ? { style_census: style } : {}),
    ...(!networkBlocked && networkFits ? { network } : {}),
    ...(identityFits && identity ? { component_identity: identity } : {}),
    ...(strictTokens ? { strict_tokens: strictTokens } : {}),
    ...(region && region.byteLength <= 5 * 1024 * 1024
      ? { region_png_base64: region.toString('base64') }
      : {}),
    ...(regionStyleSample ? { region_style_sample: regionStyleSample } : {}),
    ...(accessibility ? { accessibility_snapshot: accessibility } : {}),
    ...(consoleSummary ? { console_summary: consoleSummary } : {}),
    ...(performanceMetrics ? { performance_metrics: performanceMetrics } : {}),
  };
  return {
    schema_version: 'ux-capture.v1',
    id: id(),
    captured_at: now().toISOString(),
    source:
      source.kind === 'url'
        ? { kind: 'url', url: redactUrlForManifest(safeUrlForManifest) }
        : source.kind === 'file'
          ? { kind: 'file', filename: sanitizeFilename(source.filename) }
          : { kind: 'scenario', scenario: source.scenario, route: source.route },
    session: { id: id(), browser: 'chromium' },
    viewport: { width, height },
    theme,
    lenses,
    privacy: {
      redaction_status: redaction.status === 'safe' ? 'complete' : 'partial',
      ...(redaction.status === 'blocked'
        ? {
            redaction_reason: redaction.unsupported
              .map((item) => `${item.surface}: ${item.reason}`)
              .join('; '),
          }
        : {}),
      redacted_classes: [
        ...(redaction.redacted.formValues ? ['form-values'] : []),
        ...(redaction.redacted.markedRegions ? ['marked-regions'] : []),
        ...(redaction.redacted.assignments ? ['secret-assignments'] : []),
        ...(redaction.redacted.attributes ? ['sensitive-attributes'] : []),
        ...(redaction.redacted.urlParameters ? ['url-parameters-and-fragments'] : []),
        ...(redaction.redacted.personalData ? ['email-and-phone-values'] : []),
      ],
      external: source.kind === 'url',
      ephemeral: source.kind === 'url',
    },
    artifacts,
  };
}

async function runScenarioSteps(page: Page, steps: string[]): Promise<void> {
  for (const raw of steps) {
    const match = /^click:(.+)$/i.exec(raw.trim());
    if (!match?.[1]?.trim())
      throw new Error(
        `Unsupported scenario capture step ${JSON.stringify(raw)}; use click:<accessible label>`
      );
    const label = match[1].trim();
    const roles = ['button', 'link', 'tab', 'menuitem', 'option'] as const;
    let selected: Locator | undefined;
    // The target may render a moment after the previous step; look for 5 s.
    for (let tries = 0; !selected && tries < 20; tries++) {
      if (tries) await new Promise((resolve) => setTimeout(resolve, 250));
      for (const role of roles) {
        const candidate = page.getByRole(role, { name: label, exact: true });
        if ((await candidate.count()) === 1) {
          selected = candidate;
          break;
        }
      }
    }
    if (!selected) throw new Error(`click:${label} did not match exactly one accessible action`);
    await selected.click({ timeout: 10_000 });
    await page.waitForLoadState('domcontentloaded', { timeout: 1_500 }).catch(() => undefined);
  }
}

/** Collects only frame pixels whose own DOM passed the same fail-closed privacy prepass. */
async function collectFrameScreenshotTiles(page: Page): Promise<FrozenFrameTile[]> {
  const tiles: FrozenFrameTile[] = [];
  const frames = page.frames().filter((frame) => frame !== page.mainFrame());
  for (const [index, frame] of frames.slice(0, 16).entries()) {
    const label = `Embedded frame ${index + 1}`;
    try {
      const privacy = await frame.evaluate(redactPageInPage);
      if (privacy.status !== 'safe') {
        tiles.push({
          label,
          dataUrl: '',
          reason: 'frame content did not pass the privacy prepass',
        });
        continue;
      }
      const element = await frame.frameElement();
      const png = await element.screenshot({ type: 'png', animations: 'disabled' });
      if (png.byteLength > 1_000_000) {
        tiles.push({
          label,
          dataUrl: '',
          reason: 'frame screenshot omitted: per-tile byte limit exceeded',
        });
        continue;
      }
      tiles.push({ label, dataUrl: `data:image/png;base64,${png.toString('base64')}` });
    } catch {
      tiles.push({
        label,
        dataUrl: '',
        reason: 'frame is inaccessible or could not be screenshotted',
      });
    }
  }
  if (frames.length > 16)
    tiles.push({
      label: 'Embedded frames',
      dataUrl: '',
      reason: 'frame tile limit reached; remaining frames omitted',
    });
  return tiles;
}

function boundedDimension(value: number | undefined, fallback: number): number {
  const selected = value ?? fallback;
  if (!Number.isInteger(selected) || selected < 320 || selected > 2560)
    throw new Error('Capture viewport width must be an integer between 320 and 2560');
  return selected;
}

function boundedHeight(value: number | undefined, fallback: number): number {
  const selected = value ?? fallback;
  if (!Number.isInteger(selected) || selected < 240 || selected > 1600)
    throw new Error('Capture viewport height must be an integer between 240 and 1600');
  return selected;
}

function redactUrlForManifest(raw: string): string {
  const parsed = new URL(raw);
  for (const key of [...parsed.searchParams.keys()]) {
    parsed.searchParams.set(key, '[REDACTED]');
  }
  if (parsed.hash) parsed.hash = '#[REDACTED]';
  parsed.username = '';
  parsed.password = '';
  return parsed.href;
}
