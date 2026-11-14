import {
  collectFrozenPageInPage,
  serializeFrozenPage,
  type FrozenPageSnapshot,
} from '../../../src/capture/frozen-page';

const snapshot = (overrides: Partial<FrozenPageSnapshot> = {}): FrozenPageSnapshot => ({
  url: 'https://example.test/page',
  title: 'Example',
  html: '<!DOCTYPE html><html><head><title>Example</title><style>body{color:red}</style></head><body><main>Hello world</main></body></html>',
  css: 'body{color:red}',
  assets: [],
  unsupported: [],
  ...overrides,
});

function expectWellFormedHtml(html: string): void {
  const voidTags = new Set([
    'area',
    'base',
    'br',
    'col',
    'embed',
    'hr',
    'img',
    'input',
    'link',
    'meta',
    'param',
    'source',
    'track',
    'wbr',
  ]);
  const stack: string[] = [];
  const tags = html.matchAll(/<!--[\s\S]*?-->|<!DOCTYPE[^>]*>|<\/?([a-zA-Z][\w:-]*)\b[^>]*>/gi);
  for (const match of tags) {
    const token = match[0];
    const name = match[1]?.toLowerCase();
    if (
      !name ||
      token.startsWith('<!--') ||
      /^<!DOCTYPE/i.test(token) ||
      /\/\s*>$/.test(token) ||
      voidTags.has(name)
    )
      continue;
    if (token.startsWith('</')) expect(stack.pop()).toBe(name);
    else stack.push(name);
  }
  expect(stack).toEqual([]);
}

describe('serializeFrozenPage', () => {
  it('returns a complete frozen document when it fits the budget', () => {
    const result = serializeFrozenPage(snapshot());
    expect(result.schema).toBe('bas-frozen-page/v1');
    expect(result.status).toBe('present');
    expect(result.html).toContain('<main>Hello world</main>');
    expect(result.bytes).toBe(new TextEncoder().encode(result.html).byteLength);
    expect(result.unsupported).toEqual([]);
    if (result.html) expectWellFormedHtml(result.html);
  });

  it('reports unavailable assets and page surfaces as partial fidelity', () => {
    const result = serializeFrozenPage(
      snapshot({
        assets: [
          {
            url: 'https://cdn.example.test/font.woff2?email=private@example.test#account',
            reason: 'CORS blocked',
          },
        ],
        unsupported: [{ surface: 'frame', reason: 'cross-origin frame omitted' }],
      })
    );
    expect(result.status).toBe('partial');
    expect(result.unsupported).toEqual(
      expect.arrayContaining([
        {
          surface: 'asset:https://cdn.example.test/font.woff2?email=%5BREDACTED%5D#[REDACTED]',
          reason: 'CORS blocked',
        },
        { surface: 'frame', reason: 'cross-origin frame omitted' },
      ])
    );
    expect(result.reasons).toContain(
      'One or more page surfaces or assets could not be frozen completely.'
    );
  });

  it('omits oversized HTML instead of returning malformed markup', () => {
    const result = serializeFrozenPage(
      snapshot({ html: `<html><body>${'rendered content '.repeat(100)}</body></html>` }),
      { byteBudget: 300 }
    );
    expect(result.status).toBe('partial');
    expect(result.html).toBe('');
    expect(result.bytes).toBe(0);
    expect(result.byteBudget).toBe(300);
    expect(result.unsupported).toContainEqual({
      surface: 'document',
      reason: 'document not emitted: byte budget exceeded',
    });
    expect(result.reasons).toContain(
      'Document not emitted because it exceeded the byte budget; other capture lenses can still be present.'
    );
    if (result.html) expectWellFormedHtml(result.html);
  });

  it('embeds sanitized frame screenshot tiles and counts them against the aggregate budget', () => {
    const tile = { label: 'Embedded frame 1', dataUrl: 'data:image/png;base64,cG5n' };
    const result = serializeFrozenPage(
      snapshot({
        html: snapshot().html.replace('<main>', '<main><!--bas-frame:0-->'),
        frameTiles: [tile],
      })
    );
    expect(result.html).toContain('Embedded frame 1 · captured pixels');
    expect(result.html).toContain(tile.dataUrl);
    expect(result.status).toBe('present');
    expect(result.bytes).toBe(new TextEncoder().encode(result.html).byteLength);

    const tooLarge = serializeFrozenPage(
      snapshot({
        html: snapshot().html.replace('<main>', '<main><!--bas-frame:0-->'),
        frameTiles: [tile],
      }),
      { byteBudget: 256 }
    );
    expect(tooLarge.html).not.toContain(tile.dataUrl);
    expect(tooLarge.status).toBe('partial');
    expect(tooLarge.unsupported).toContainEqual({
      surface: 'Embedded frame 1',
      reason: 'frame screenshot omitted: aggregate byte budget exceeded',
    });
  });

  it('keeps blocked frame tiles out of HTML and explains the partial capture', () => {
    const result = serializeFrozenPage(
      snapshot({
        html: snapshot().html.replace('<main>', '<main><!--bas-frame:0-->'),
        frameTiles: [
          {
            label: 'Embedded frame 2',
            dataUrl: '',
            reason: 'frame is inaccessible or could not be screenshotted',
          },
        ],
      })
    );
    expect(result.html).toContain(
      'Embedded frame 2 · screenshot unavailable: frame is inaccessible or could not be screenshotted'
    );
    expect(result.status).toBe('partial');
    expect(result.unsupported).toContainEqual({
      surface: 'Embedded frame 2',
      reason: 'frame is inaccessible or could not be screenshotted',
    });
  });
});

describe('collectFrozenPageInPage', () => {
  it('keeps used rules and active conditions while dropping unused CSS', async () => {
    const previous = {
      document: globalThis.document,
      nodeFilter: globalThis.NodeFilter,
      css: globalThis.CSS,
      style: globalThis.getComputedStyle,
      matchMedia: globalThis.matchMedia,
      cssStyleRule: globalThis.CSSStyleRule,
      cssMediaRule: globalThis.CSSMediaRule,
      cssFontFaceRule: globalThis.CSSFontFaceRule,
      cssKeyframesRule: globalThis.CSSKeyframesRule,
      cssSupportsRule: globalThis.CSSSupportsRule,
    };
    class StyleRule {
      constructor(
        public selectorText: string,
        public cssText: string
      ) {}
    }
    class MediaRule {
      constructor(
        public conditionText: string,
        public cssRules: CSSRuleList,
        public cssText: string
      ) {}
    }
    class FontFaceRule {
      style = { getPropertyValue: () => '"Arial"' };
      cssText = '@font-face { font-family: Arial; src: url(font.woff2); }';
    }
    class KeyframesRule {
      name = 'unused-animation';
      cssText = '@keyframes unused-animation { from { opacity: 0; } to { opacity: 1; } }';
    }
    class SupportsRule {
      constructor(
        public conditionText: string,
        public cssRules: CSSRuleList,
        public cssText: string
      ) {}
    }
    const usedElement = { isConnected: true } as Element;
    const usedRule = new StyleRule('.used', '.used { color: red; }');
    const unusedRule = new StyleRule('.unused', '.unused { color: blue; }');
    const activeMedia = new MediaRule(
      'screen',
      [usedRule] as unknown as CSSRuleList,
      '@media screen { .used { color: red; } }'
    );
    const inactiveSupports = new SupportsRule(
      'future-feature',
      [unusedRule] as unknown as CSSRuleList,
      '@supports future-feature { .unused { color: blue; } }'
    );
    const rules = [
      usedRule,
      unusedRule,
      new FontFaceRule(),
      new KeyframesRule(),
      activeMedia,
      inactiveSupports,
    ];
    let injectedStyle: { textContent: string; remove(): void } | undefined;
    const oldStyle = {
      removed: false,
      remove() {
        this.removed = true;
      },
    };
    const head = {
      appendChild: (node: { textContent: string }) => {
        injectedStyle = node as typeof injectedStyle;
      },
    };
    const clone = {
      querySelectorAll: (selector: string) =>
        selector === 'style, link[rel="stylesheet"]' ? [oldStyle] : [],
      querySelector: (selector: string) => (selector === 'head' ? head : null),
      outerHTML: '<html><head></head><body></body></html>',
    };
    const doc = {
      documentElement: { cloneNode: () => clone },
      styleSheets: [{ cssRules: rules }],
      images: [],
      URL: 'https://example.test/',
      baseURI: 'https://example.test/',
      title: 'Example',
      doctype: { name: 'html' },
      querySelectorAll: (selector: string) =>
        selector === 'body *' ? [usedElement] : selector === '.used' ? [usedElement] : [],
      createTreeWalker: () => ({ nextNode: () => null }),
      createElement: () => ({ textContent: '', remove() {} }),
    };
    globalThis.document = doc as unknown as Document;
    globalThis.NodeFilter = { SHOW_COMMENT: 128 } as typeof NodeFilter;
    globalThis.CSS = { ...globalThis.CSS, supports: () => false };
    globalThis.getComputedStyle = (() => ({
      fontFamily: 'Arial',
      animationName: 'none',
    })) as typeof getComputedStyle;
    globalThis.matchMedia = ((query: string) => ({
      matches: query === 'screen',
    })) as typeof matchMedia;
    globalThis.CSSStyleRule = StyleRule as unknown as typeof CSSStyleRule;
    globalThis.CSSMediaRule = MediaRule as unknown as typeof CSSMediaRule;
    globalThis.CSSFontFaceRule = FontFaceRule as unknown as typeof CSSFontFaceRule;
    globalThis.CSSKeyframesRule = KeyframesRule as unknown as typeof CSSKeyframesRule;
    globalThis.CSSSupportsRule = SupportsRule as unknown as typeof CSSSupportsRule;
    try {
      // Playwright serializes the injected function body, so invoke the same
      // way and catch accidental dependencies on module-scope helpers.
      const injected = new Function(
        `return (${collectFrozenPageInPage.toString()})()`
      ) as () => Promise<FrozenPageSnapshot>;
      const result = await injected();
      expect(result.css).toContain('.used { color: red; }');
      expect(result.css).toContain('@media screen');
      expect(result.css).toContain('@font-face');
      expect(result.css).not.toContain('.unused');
      expect(result.css).not.toContain('@keyframes');
      expect(oldStyle.removed).toBe(true);
      expect(injectedStyle?.textContent).toBe(result.css);
    } finally {
      globalThis.document = previous.document;
      globalThis.NodeFilter = previous.nodeFilter;
      globalThis.CSS = previous.css;
      globalThis.getComputedStyle = previous.style;
      globalThis.matchMedia = previous.matchMedia;
      globalThis.CSSStyleRule = previous.cssStyleRule;
      globalThis.CSSMediaRule = previous.cssMediaRule;
      globalThis.CSSFontFaceRule = previous.cssFontFaceRule;
      globalThis.CSSKeyframesRule = previous.cssKeyframesRule;
      globalThis.CSSSupportsRule = previous.cssSupportsRule;
    }
  });
});
