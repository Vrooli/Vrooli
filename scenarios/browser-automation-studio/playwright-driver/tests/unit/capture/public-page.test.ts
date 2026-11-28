import type { BrowserContext, Page, Request, Response } from 'rebrowser-playwright';
import {
  capturePublicPageInContext,
  FILE_CAPTURE_URL,
  inertHTMLFileBeforeParse,
} from '../../../src/capture/public-page';
import type { RedactionReport } from '../../../src/capture/redaction';
import type { OutlineResult } from '../../../src/capture/outline';
import type { FrozenPageResult } from '../../../src/capture/frozen-page';
import type { StyleCensusResult } from '../../../src/capture/style-census';
import type { ComponentIdentityResult } from '../../../src/capture/identity';
import { rgbPng } from './paint.test';

function createContext() {
  const page = {
    setViewportSize: jest.fn(async () => undefined),
    emulateMedia: jest.fn(async () => undefined),
    on: jest.fn((event: string, listener: (value: Request | Response) => void) => {
      if (event === 'request') {
        listener({
          url: () => 'https://public.example/?access_token=secret-value',
          method: () => 'GET',
          resourceType: () => 'document',
        } as unknown as Request);
      } else if (event === 'response') {
        listener({
          url: () => 'https://public.example/?access_token=secret-value',
          request: () => ({ method: () => 'GET' }) as unknown as Request,
          status: () => 200,
        } as unknown as Response);
      }
      return page;
    }),
    goto: jest.fn(async () => null),
    route: jest.fn(async () => undefined),
    setContent: jest.fn(async () => undefined),
    evaluate: jest.fn(async () => ({
      schema: 'bas-strict-token-diagnostic/v1',
      token_count: 12,
      sentinel_element_count: 4,
      non_sentinel_color_count: 3,
    })),
    screenshot: jest.fn(async () => Buffer.from('viewport-pixels')),
  };
  const context = { newPage: jest.fn(async () => page as unknown as Page) };
  return { context: context as unknown as BrowserContext, page };
}

const outline: OutlineResult = {
  schema: 'bas-agent-outline/v1',
  nodes: [],
  unsupported: [],
  tokenBudget: 8000,
  estimatedTokens: 0,
  overBudget: false,
  actionCount: 0,
};

const style: StyleCensusResult = {
  schema: 'bas-agent-style-census/v1',
  tokenCatalogStatus: 'rcl',
  elementCount: 1,
  sampledElementCount: 1,
  clusters: [],
};

const frozen: FrozenPageResult = {
  schema: 'bas-frozen-page/v1',
  html: '<!doctype html><html><body>safe</body></html>',
  status: 'present',
  bytes: 45,
  byteBudget: 5_000_000,
  reasons: [],
  unsupported: [],
};

const safeReport: RedactionReport = {
  schema: 'bas-capture-redaction/v1',
  status: 'safe',
  redacted: {
    formValues: 1,
    markedRegions: 0,
    assignments: 0,
    attributes: 0,
    urlParameters: 0,
    personalData: 0,
  },
  unsupported: [],
};

const componentIdentity: ComponentIdentityResult = {
  schema: 'bas-component-identity/v1',
  element_count: 1,
  stamped_count: 1,
  components: [
    {
      selector: '#main-action',
      asset: 'controls.button',
      version: '2.1.0',
      stamp: 'vite',
      source_slot: 'primitives.button',
    },
  ],
};

describe('public URL page capture composition', () => {
  it('marks pixel fidelity partial when a script request fails before capture', async () => {
    const { context, page } = createContext();
    (page.on as jest.Mock).mockImplementation(
      (event: string, listener: (value: Request) => void) => {
        if (event === 'requestfailed')
          listener({ resourceType: () => 'script' } as unknown as Request);
        return page;
      }
    );

    const result = await capturePublicPageInContext(
      context,
      'http://localhost:4300/',
      { styleTokens: [] },
      {
        redact: async () => safeReport,
        collectFrozen: async () => frozen,
        collectOutline: async () => outline,
        collectStyle: async () => style,
      }
    );

    expect(result.artifacts.screenshot_png_base64).toBeDefined();
    expect(result.lenses.L0).toEqual({
      status: 'partial',
      reason:
        '1 script request failed or was blocked before pixels were captured; rendered content may be incomplete.',
    });
    expect(JSON.stringify(result.lenses)).not.toContain('localhost:4300');
  });

  it('marks pixel fidelity partial when a stylesheet request is blocked before capture', async () => {
    const { context, page } = createContext();
    (page.on as jest.Mock).mockImplementation(
      (event: string, listener: (value: Request) => void) => {
        if (event === 'requestfailed')
          listener({ resourceType: () => 'stylesheet' } as unknown as Request);
        return page;
      }
    );

    const result = await capturePublicPageInContext(
      context,
      'http://localhost:4300/',
      { styleTokens: [] },
      {
        redact: async () => safeReport,
        collectFrozen: async () => frozen,
        collectOutline: async () => outline,
        collectStyle: async () => style,
      }
    );

    expect(result.lenses.L0).toEqual({
      status: 'partial',
      reason:
        '1 stylesheet request failed or was blocked before pixels were captured; rendered content may be incomplete.',
    });
    expect(result.lenses.L3).toEqual({
      status: 'partial',
      reason: '1 stylesheet request failed or was blocked; measured styles may be incomplete.',
    });
    expect(JSON.stringify(result.lenses)).not.toContain('localhost:4300');
  });

  it('adds optional bounded full-page pixels after redaction without changing L0 viewport pixels', async () => {
    const { context, page } = createContext();
    const order: string[] = [];
    (page as unknown as { evaluate: jest.Mock }).evaluate = jest.fn(async () => ({
      width: 1280,
      height: 720,
    }));
    (page as unknown as { screenshot: jest.Mock }).screenshot = jest.fn(
      async (options?: { fullPage?: boolean }) => {
        order.push(options?.fullPage ? 'full-page' : 'unexpected-page-screenshot');
        return Buffer.from('full-page-pixels');
      }
    );
    const viewport = Buffer.from('viewport-pixels');
    const result = await capturePublicPageInContext(
      context,
      'http://localhost:4300/',
      {
        url: 'http://localhost:4300/',
        scenario: 'git-control-tower',
        route: '/',
        styleTokens: [],
        fullPagePixels: true,
      },
      {
        redact: async () => {
          order.push('redaction');
          return safeReport;
        },
        capturePixels: async () => {
          order.push('viewport');
          return viewport;
        },
        collectFrozen: async () => frozen,
        collectOutline: async () => outline,
        collectIdentity: async () => componentIdentity,
        collectStyle: async () => style,
      },
      { kind: 'scenario', scenario: 'git-control-tower', route: '/' }
    );
    expect(order).toEqual(['redaction', 'viewport', 'full-page']);
    expect(result.artifacts.screenshot_png_base64).toBe(viewport.toString('base64'));
    expect(result.artifacts.full_page_pixels_png_base64).toBe(
      Buffer.from('full-page-pixels').toString('base64')
    );
    expect(result.artifacts.full_page_pixels_status).toEqual({ status: 'present' });
  });

  it('adds bounded component evidence only on opted-in redacted scenario captures', async () => {
    const { context, page } = createContext();
    const order: string[] = [];
    const crop = Buffer.from('crop-png');
    (page as unknown as { locator: jest.Mock }).locator = jest.fn((selector: string) => ({
      count: async () => (selector === '[data-story]' ? 1 : 0),
      screenshot: async () => crop,
      ariaSnapshot: async () => 'main\n  button "Save"',
      evaluate: async () => {
        order.push('region-style');
        return {
          schema: 'bas-region-style-sample/v1',
          computed_style: {
            color: 'rgb(0, 0, 0)',
            background_color: 'rgb(255, 255, 255)',
            border_color: 'rgb(0, 0, 0)',
            border_style: 'solid',
            border_width: '1px',
            visibility: 'visible',
            display: 'block',
            opacity: '1',
          },
        };
      },
    }));
    (page as unknown as { evaluate: jest.Mock }).evaluate = jest.fn(
      async (_fn: unknown, selector?: string) =>
        selector
          ? 1
          : {
              schema: 'bas-performance-metrics/v1',
              dom_content_loaded_ms: 20,
              load_ms: 30,
              resource_count: 4,
            }
    );
    const result = await capturePublicPageInContext(
      context,
      'http://localhost:4300/',
      {
        url: 'http://localhost:4300/',
        scenario: 'storybook',
        route: '/',
        styleTokens: [],
        evidenceProfile: 'componenttests',
        regionSelector: '[data-story]',
      },
      {
        redact: async () => {
          order.push('redaction');
          return safeReport;
        },
        collectFrozen: async () => frozen,
        collectOutline: async () => outline,
        collectStyle: async () => style,
        collectIdentity: async () => componentIdentity,
      },
      { kind: 'scenario', scenario: 'storybook', route: '/' }
    );
    expect(result.artifacts.region_png_base64).toBe(crop.toString('base64'));
    expect(result.artifacts.region_style_sample).toEqual({
      schema: 'bas-region-style-sample/v1',
      computed_style: {
        color: 'rgb(0, 0, 0)',
        background_color: 'rgb(255, 255, 255)',
        border_color: 'rgb(0, 0, 0)',
        border_style: 'solid',
        border_width: '1px',
        visibility: 'visible',
        display: 'block',
        opacity: '1',
      },
    });
    expect(order).toEqual(['redaction', 'region-style']);
    expect(Object.keys(result.artifacts.region_style_sample!.computed_style).sort()).toEqual([
      'background_color',
      'border_color',
      'border_style',
      'border_width',
      'color',
      'display',
      'opacity',
      'visibility',
    ]);
    expect(result.artifacts.accessibility_snapshot?.snapshot).toContain('button "Save"');
    expect(result.artifacts.console_summary).toEqual({
      schema: 'bas-console-summary/v1',
      error_count: 0,
      warning_count: 0,
    });
    expect(result.artifacts.performance_metrics?.resource_count).toBe(4);
    expect(result.lenses.L0.status).toBe('present');
  });

  it('removes local-file scripts before Chromium parses the document', () => {
    const html =
      '<!doctype html><body><main>safe</main><script type="module" src="/slow.js"></script></body>';

    const inert = inertHTMLFileBeforeParse(html);

    expect(inert).toContain('<main>safe</main>');
    expect(inert).not.toMatch(/<script\b/i);
  });

  it('marks file pixel fidelity partial when active script content is removed', async () => {
    const { context, page } = createContext();
    const result = await capturePublicPageInContext(
      context,
      '',
      { html: '<!doctype html><body><main>shell</main><script>render()</script></body>' },
      {
        redact: async () => safeReport,
        collectFrozen: async () => frozen,
        collectOutline: async () => outline,
        collectStyle: async () => style,
      },
      { kind: 'file', filename: 'artifact.html' },
      true
    );

    // The artifact is served inert from a synthetic URL; setContent never
    // reaches DOMContentLoaded under the patched driver.
    expect(page.setContent).not.toHaveBeenCalled();
    expect(page.goto).toHaveBeenCalledWith(FILE_CAPTURE_URL, expect.anything());
    const [routed, handler] = page.route.mock.calls[0] as unknown as [
      string,
      (route: { fulfill: (response: { body: string }) => Promise<void> }) => Promise<void>,
    ];
    expect(routed).toBe(FILE_CAPTURE_URL);
    let served = '';
    await handler({
      fulfill: (response) => {
        served = response.body;
        return Promise.resolve();
      },
    });
    expect(served).toContain('<main>shell</main>');
    expect(served).not.toContain('render()');
    expect(result.source).toEqual({ kind: 'file', filename: 'artifact.html' });
    expect(result.lenses.L0).toEqual({
      status: 'partial',
      reason:
        '1 script removed by the safe file import before pixels were captured; rendered content may be incomplete.',
    });
    expect(JSON.stringify(result)).not.toContain('render()');
  });

  it('captures pixels from one bounded viewport rather than an unbounded full-page image', async () => {
    const { context, page } = createContext();
    const result = await capturePublicPageInContext(
      context,
      'https://public.example/',
      { styleTokens: [] },
      {
        redact: async () => safeReport,
        collectFrozen: async () => frozen,
        collectOutline: async () => outline,
        collectStyle: async () => style,
      }
    );

    expect(page.setViewportSize).toHaveBeenCalledWith({ width: 1280, height: 720 });
    expect(page.emulateMedia).toHaveBeenCalledWith({ colorScheme: 'light' });
    expect(page.screenshot).toHaveBeenCalledWith({
      type: 'png',
      fullPage: false,
      animations: 'disabled',
    });
    expect(result.artifacts.screenshot_png_base64).toBe(
      Buffer.from('viewport-pixels').toString('base64')
    );
  });

  it('captures all supported lenses in one page and redacts URL query values in metadata', async () => {
    const { context, page } = createContext();
    const result = await capturePublicPageInContext(
      context,
      'https://public.example/?access_token=secret-value&user=alice#private-state',
      {
        url: 'https://public.example/?access_token=secret-value&user=alice#private-state',
        width: 1200,
        height: 800,
        theme: 'dark',
        styleTokens: [],
      },
      {
        id: (() => {
          let index = 0;
          return () => `id-${++index}`;
        })(),
        now: () => new Date('2026-11-14T05:00:00Z'),
        redact: async () => safeReport,
        capturePixels: jest.fn(async () => Buffer.from('png')),
        collectFrozen: async () => frozen,
        collectOutline: async () => outline,
        collectStyle: async () => style,
        collectIdentity: async () => componentIdentity,
      }
    );

    expect(page.setViewportSize).toHaveBeenCalledWith({ width: 1200, height: 800 });
    expect(page.emulateMedia).toHaveBeenCalledWith({ colorScheme: 'dark' });
    expect(result.theme).toBe('dark');
    expect(page.goto).toHaveBeenCalledWith(
      'https://public.example/?access_token=secret-value&user=alice#private-state',
      expect.any(Object)
    );
    expect(result.source.url).toContain('access_token=%5BREDACTED%5D');
    expect(result.source.url).toContain('user=%5BREDACTED%5D');
    expect(result.source.url).toContain('#[REDACTED]');
    expect(result.source.url).not.toMatch(/secret-value|alice|private-state/);
    expect(result.artifacts.screenshot_png_base64).toBe(Buffer.from('png').toString('base64'));
    expect(result.artifacts.frozen_page_html).toBe(frozen.html);
    expect(result.artifacts.network).toEqual([
      { origin: 'https://public.example', resource_type: 'document', method: 'GET', status: 200 },
    ]);
    expect(result.lenses.L0.status).toBe('present');
    expect(result.lenses.L1.status).toBe('present');
    expect(result.lenses.L2.status).toBe('present');
    expect(result.lenses.L3.status).toBe('present');
    expect(result.lenses.L4.status).toBe('present');
    expect(result.artifacts.component_identity).toEqual(componentIdentity);
    expect(result.lenses.L5.status).toBe('present');
    expect(result.lenses.L6.status).toBe('absent');
    expect(JSON.stringify(result)).not.toMatch(/secret-value|alice|private-state/);
  });

  it('includes the specific unsupported surfaces in partial frozen-page fidelity', async () => {
    const { context } = createContext();
    const partialFrozen: FrozenPageResult = {
      ...frozen,
      status: 'partial',
      reasons: ['One or more page surfaces or assets could not be frozen completely.'],
      unsupported: [{ surface: 'asset:/fonts/app.woff2', reason: 'asset could not be fetched' }],
    };
    const result = await capturePublicPageInContext(
      context,
      'https://public.example/',
      {},
      {
        redact: async () => safeReport,
        collectFrozen: async () => partialFrozen,
        collectOutline: async () => outline,
        collectStyle: async () => style,
        collectIdentity: async () => componentIdentity,
      }
    );

    expect(result.lenses.L1).toEqual({
      status: 'partial',
      reason: 'asset:/fonts/app.woff2: asset could not be fetched',
    });
  });

  it('runs the strict-token sentinel after redaction in the same scenario session', async () => {
    const { context, page } = createContext();
    const result = await capturePublicPageInContext(
      context,
      'http://localhost:4321/',
      {
        scenario: 'git-control-tower',
        route: '/',
        strictTokens: true,
        allowedOrigins: ['http://localhost:4321'],
        styleTokens: [],
      },
      {
        redact: async () => safeReport,
        collectFrozen: async () => frozen,
        collectOutline: async () => outline,
        collectStyle: async () => style,
        collectIdentity: async () => componentIdentity,
      }
    );
    expect(page.evaluate).toHaveBeenCalledWith(expect.any(Function));
    expect(result.artifacts.strict_tokens).toEqual({
      schema: 'bas-strict-token-diagnostic/v1',
      token_count: 12,
      sentinel_element_count: 4,
      non_sentinel_color_count: 3,
    });
  });

  it('waits for the scenario document load event before collecting showcase pixels', async () => {
    const { context, page } = createContext();
    await capturePublicPageInContext(
      context,
      'http://localhost:4321/',
      { scenario: 'git-control-tower', route: '/', allowedOrigins: ['http://localhost:4321'] },
      {
        redact: async () => safeReport,
        collectFrozen: async () => frozen,
        collectOutline: async () => outline,
        collectStyle: async () => style,
        collectIdentity: async () => componentIdentity,
      },
      { kind: 'scenario', scenario: 'git-control-tower', route: '/' }
    );

    expect(page.goto).toHaveBeenCalledWith(
      'http://localhost:4321/',
      expect.objectContaining({ waitUntil: 'load' })
    );
  });

  it('fails closed across data-bearing lenses when privacy cannot inspect a canvas', async () => {
    const { context } = createContext();
    const unsafe: RedactionReport = {
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
      unsupported: [{ surface: 'canvas', reason: 'Cannot inspect canvas content' }],
    };
    const capturePixels = jest.fn(async () => Buffer.from('must-not-be-read'));
    const collectFrozen = jest.fn(async () => frozen);
    const result = await capturePublicPageInContext(
      context,
      'https://public.example/',
      {
        url: 'https://public.example/',
      },
      {
        id: () => 'stable-id',
        now: () => new Date('2026-11-14T05:00:00Z'),
        redact: async () => unsafe,
        capturePixels,
        collectFrozen,
        collectOutline: async () => outline,
        collectStyle: async () => style,
      }
    );

    expect(capturePixels).not.toHaveBeenCalled();
    expect(collectFrozen).not.toHaveBeenCalled();
    expect(result.artifacts).not.toHaveProperty('screenshot_png_base64');
    expect(result.artifacts).not.toHaveProperty('frozen_page_html');
    expect(result.lenses.L0).toEqual({
      status: 'absent',
      reason: 'Redaction prepass blocked pixel output: canvas',
    });
    expect(result.lenses.L1).toEqual({
      status: 'absent',
      reason: 'Redaction prepass blocked frozen-page output: canvas',
    });
    expect(result.lenses.L2.status).toBe('absent');
    expect(result.lenses.L3.status).toBe('absent');
    expect(result.lenses.L4.status).toBe('absent');
    expect(result.lenses.L5.status).toBe('absent');
  });

  it('fails closed when a blocked privacy prepass has no classified surface', async () => {
    const { context } = createContext();
    const collectFrozen = jest.fn(async () => frozen);
    const result = await capturePublicPageInContext(
      context,
      'https://public.example/',
      { url: 'https://public.example/' },
      {
        redact: async () => ({ ...safeReport, status: 'blocked', unsupported: [] }),
        collectFrozen,
        collectOutline: async () => outline,
        collectStyle: async () => style,
      }
    );
    expect(collectFrozen).not.toHaveBeenCalled();
    expect(result.lenses.L1.status).toBe('absent');
  });

  it('marks lenses partial and omits data that exceeds per-lens budgets', async () => {
    const { context } = createContext();
    const result = await capturePublicPageInContext(
      context,
      'https://public.example/',
      {
        url: 'https://public.example/',
        maxScreenshotBytes: 64 * 1024,
        maxLensJsonBytes: 8 * 1024,
      },
      {
        redact: async () => safeReport,
        capturePixels: jest.fn(async () => Buffer.alloc(80 * 1024)),
        collectFrozen: async () => frozen,
        collectOutline: async () =>
          ({ ...outline, nodes: [{ text: 'x'.repeat(9000) }] }) as unknown as OutlineResult,
        collectStyle: async () => style,
      }
    );

    expect(result.lenses.L0.status).toBe('partial');
    expect(result.lenses.L2.status).toBe('partial');
    expect(result.artifacts).not.toHaveProperty('screenshot_png_base64');
    expect(result.artifacts).not.toHaveProperty('outline');
  });

  it('captures only frame tiles that pass their own privacy prepass', async () => {
    const { context, page } = createContext();
    const mainFrame = {};
    const element = { screenshot: jest.fn(async () => Buffer.from('frame-png')) };
    const childFrame = {
      evaluate: jest.fn(async () => safeReport),
      frameElement: jest.fn(async () => element),
    };
    Object.assign(page, {
      evaluate: jest.fn(async () => ({
        url: 'https://public.example/',
        title: 'Example',
        html: '<!doctype html><html><head></head><body><main>before<!--bas-frame:0-->after</main></body></html>',
        css: '',
        assets: [],
        unsupported: [],
      })),
      frames: () => [mainFrame, childFrame],
      mainFrame: () => mainFrame,
    });
    const result = await capturePublicPageInContext(
      context,
      'https://public.example/',
      {},
      {
        redact: async () => ({
          ...safeReport,
          status: 'blocked',
          unsupported: [
            {
              surface: 'iframe',
              reason: 'embedded frame content is outside the inspected document',
            },
          ],
        }),
        collectOutline: async () => outline,
        collectStyle: async () => style,
      }
    );

    expect(childFrame.evaluate).toHaveBeenCalledTimes(1);
    expect(element.screenshot).toHaveBeenCalledWith({ type: 'png', animations: 'disabled' });
    expect(result.artifacts.frozen_page_html).toContain('data:image/png;base64,');
    expect(result.artifacts.frozen_page_html).toContain('Embedded frame 1 · captured pixels');
    expect(result.artifacts.frozen_page_html).toContain('before<figure');
    expect(result.artifacts.frozen_page_html).toContain('</figure>after');
    expect(result.lenses.L1.status).toBe('present');
  });
});

describe('L0 never reports blank pixels as present', () => {
  const flat = rgbPng(64, 64, () => [249, 248, 244]);
  const painted = rgbPng(64, 64, (x) => (x < 32 ? [20, 20, 20] : [249, 248, 244]));

  it('marks L0 absent and emits no image when every attempt is one flat colour', async () => {
    const { context } = createContext();
    const redact = jest.fn(async () => safeReport);
    const result = await capturePublicPageInContext(
      context,
      'http://localhost:4300/',
      { styleTokens: [] },
      {
        redact,
        capturePixels: async () => flat,
        waitForPaint: async () => ({ content: false, settled: true, waitedMs: 10_000 }),
        blankRetryDelayMs: 0,
        collectFrozen: async () => frozen,
        collectOutline: async () => outline,
        collectStyle: async () => style,
      }
    );
    expect(result.artifacts.screenshot_png_base64).toBeUndefined();
    expect(result.lenses.L0.status).toBe('absent');
    expect(result.lenses.L0.reason).toContain('one flat colour');
    expect(result.lenses.L0.reason).toContain('after 3 attempts');
    // Each retry re-runs the privacy prepass on the newly painted page.
    expect(redact).toHaveBeenCalledTimes(3);
  });

  it('retries a late-painting page and reports the painted image', async () => {
    const { context } = createContext();
    const shots = [flat, painted];
    const result = await capturePublicPageInContext(
      context,
      'http://localhost:4300/',
      { styleTokens: [] },
      {
        redact: async () => safeReport,
        capturePixels: async () => shots.shift() ?? painted,
        waitForPaint: async () => ({ content: true, settled: true, waitedMs: 40 }),
        blankRetryDelayMs: 0,
        collectFrozen: async () => frozen,
        collectOutline: async () => outline,
        collectStyle: async () => style,
      }
    );
    expect(result.lenses.L0).toEqual({ status: 'present' });
    expect(Buffer.from(result.artifacts.screenshot_png_base64!, 'base64').equals(painted)).toBe(
      true
    );
  });
});
