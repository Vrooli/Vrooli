import { readFileSync } from 'node:fs';
import path from 'node:path';
import {
  collectRclStyleTokensInPage,
  collectStyleCensusInPage,
} from '../../../src/capture/style-census';

type FakeElement = {
  tagName: string;
  className: string;
  getAttribute: (name: string) => string | null;
  getBoundingClientRect: () => { width: number; height: number };
};

function setup(count = 2): () => void {
  const previousDocument = globalThis.document;
  const previousStyle = globalThis.getComputedStyle;
  const elements: FakeElement[] = Array.from({ length: count }, (_, index) => ({
    tagName: index ? 'BUTTON' : 'DIV',
    className: 'sample',
    getAttribute: (name) => (name === 'id' ? `node-${index}` : null),
    getBoundingClientRect: () => ({ width: 20, height: 10 }),
  }));
  globalThis.document = {
    querySelectorAll: () => elements,
  } as unknown as Document;
  globalThis.getComputedStyle = (() => ({
    display: 'block',
    visibility: 'visible',
    opacity: '1',
    color: 'rgb(10, 20, 30)',
    backgroundColor: 'rgb(255, 255, 255)',
    borderTopColor: 'rgb(10, 20, 30)',
    outlineColor: 'rgba(0, 0, 0, 0)',
    fontFamily: 'Inter, sans-serif',
    fontSize: '24px',
    fontWeight: '400',
    lineHeight: '24px',
    letterSpacing: 'normal',
    marginTop: '0px',
    marginRight: '0px',
    marginBottom: '0px',
    marginLeft: '0px',
    paddingTop: '8px',
    paddingRight: '8px',
    paddingBottom: '8px',
    paddingLeft: '8px',
    gap: 'normal',
    rowGap: 'normal',
    columnGap: 'normal',
    width: '20px',
    minWidth: '0px',
    maxWidth: 'none',
    height: '10px',
    minHeight: '0px',
    maxHeight: 'none',
    borderTopWidth: '0px',
    outlineWidth: '0px',
    outlineOffset: '0px',
    borderTopLeftRadius: '4px',
    borderTopRightRadius: '4px',
    borderBottomRightRadius: '4px',
    borderBottomLeftRadius: '4px',
    boxShadow: 'none',
    zIndex: 'auto',
    transitionDuration: '0s',
    transitionTimingFunction: 'ease',
  })) as typeof getComputedStyle;
  return () => {
    globalThis.document = previousDocument;
    globalThis.getComputedStyle = previousStyle;
  };
}

describe('injected style census', () => {
  it('clusters repeated values deterministically and preserves counts and sources', () => {
    const restore = setup();
    try {
      const result = collectStyleCensusInPage();
      const color = result.clusters.find(
        (cluster) =>
          cluster.category === 'color' &&
          cluster.property === 'foreground' &&
          cluster.value === 'rgb(10, 20, 30)'
      );
      expect(color?.count).toBe(2);
      expect(color?.sources).toHaveLength(2);
      const fontSize = result.clusters.find(
        (cluster) => cluster.category === 'typography' && cluster.property === 'font-size'
      );
      const lineHeight = result.clusters.find(
        (cluster) => cluster.category === 'typography' && cluster.property === 'line-height'
      );
      expect(fontSize?.value).toBe(lineHeight?.value);
      expect(fontSize?.property).not.toBe(lineHeight?.property);
      expect(fontSize?.id).not.toBe(lineHeight?.id);
      const repeated = collectStyleCensusInPage();
      expect(result.clusters.map((cluster) => cluster.id)).toEqual(
        repeated.clusters.map((cluster) => cluster.id)
      );
      expect(color?.id).toBe(
        repeated.clusters.find(
          (cluster) =>
            cluster.category === 'color' &&
            cluster.property === 'foreground' &&
            cluster.value === 'rgb(10, 20, 30)'
        )?.id
      );
    } finally {
      restore();
    }
  });

  it('reports exact and nearest comparable token distances', () => {
    const restore = setup(1);
    try {
      const exact = collectStyleCensusInPage({
        tokens: [
          { name: 'ink', category: 'color', property: 'foreground', value: 'rgb(10, 20, 30)' },
        ],
      }).clusters.find(
        (cluster) =>
          cluster.category === 'color' &&
          cluster.property === 'foreground' &&
          cluster.value === 'rgb(10, 20, 30)'
      );
      expect(exact?.token).toEqual({
        name: 'ink',
        property: 'foreground',
        value: 'rgb(10, 20, 30)',
        distance: 0,
      });
      const nearest = collectStyleCensusInPage({
        tokens: [{ name: 'near', category: 'color', property: 'foreground', value: '#0b141e' }],
      }).clusters.find(
        (cluster) =>
          cluster.category === 'color' &&
          cluster.property === 'foreground' &&
          cluster.value === 'rgb(10, 20, 30)'
      );
      expect(nearest?.token).toEqual({
        name: 'near',
        property: 'foreground',
        value: '#0b141e',
        distance: 1,
      });
    } finally {
      restore();
    }
  });

  it('does not invent token matches when no catalog is supplied', () => {
    const restore = setup(1);
    try {
      const result = collectStyleCensusInPage();
      expect(
        result.clusters.every(
          (cluster) =>
            cluster.token === null && cluster.tokenMissingReason === 'catalog-not-provided'
        )
      ).toBe(true);
    } finally {
      restore();
    }
  });

  it('reads published RCL token families projected by the app without labeling arbitrary variables', () => {
    const previousDocument = globalThis.document;
    const previousStyle = globalThis.getComputedStyle;
    const values: Record<string, string> = {
      '--color-foreground': '#111827',
      '--color-background': '#ffffff',
      '--space-md': '24px',
      '--radius-panel': '8px',
      '--app-background': '#ffffff',
      '--panel-padding': '16px',
      '--control-height-md': '40px',
      '--text-title-tracking': '-0.01em',
      '--opacity-muted': '0.64',
      '--layer-overlay': '300',
      '--focus-ring-width': '2px',
      '--focus-ring-color': 'var(--color-focus)',
      '--color-danger-foreground-inverse': 'var(--color-primary-foreground)',
      '--color-danger-border': '#dc2626',
      '--color-info': '#0284c7',
      '--elev-floating': '0 4px 12px rgb(0 0 0 / 12%)',
      '--layout-panel': '320px',
    };
    const names = Object.keys(values);
    globalThis.document = {
      documentElement: {},
      styleSheets: [],
    } as unknown as Document;
    globalThis.getComputedStyle = (() => ({
      length: names.length,
      item: (index: number) => names[index],
      getPropertyValue: (name: string) => values[name] ?? '',
    })) as unknown as typeof getComputedStyle;
    try {
      expect(collectRclStyleTokensInPage()).toEqual(
        expect.arrayContaining([
          {
            name: '--color-foreground',
            category: 'color',
            property: 'foreground',
            value: '#111827',
          },
          {
            name: '--color-background',
            category: 'color',
            property: 'background',
            value: '#ffffff',
          },
          {
            name: '--app-background',
            category: 'color',
            property: 'background',
            value: '#ffffff',
          },
          { name: '--space-md', category: 'spacing', property: 'padding-top', value: '24px' },
          { name: '--panel-padding', category: 'spacing', property: 'padding-top', value: '16px' },
          { name: '--radius-panel', category: 'radius', property: 'radius-top-left', value: '8px' },
          { name: '--control-height-md', category: 'layout', property: 'height', value: '40px' },
          {
            name: '--text-title-tracking',
            category: 'typography',
            property: 'letter-spacing',
            value: '-0.01em',
          },
          { name: '--opacity-muted', category: 'effects', property: 'opacity', value: '0.64' },
          { name: '--layer-overlay', category: 'effects', property: 'z-index', value: '300' },
          {
            name: '--focus-ring-width',
            category: 'layout',
            property: 'outline-width',
            value: '2px',
          },
          {
            name: '--focus-ring-color',
            category: 'color',
            property: 'outline',
            value: 'var(--color-focus)',
          },
          {
            name: '--color-danger-foreground-inverse',
            category: 'color',
            property: 'foreground',
            value: 'var(--color-primary-foreground)',
          },
          {
            name: '--color-danger-border',
            category: 'color',
            property: 'border-top',
            value: '#dc2626',
          },
          {
            name: '--color-info',
            category: 'color',
            property: 'background',
            value: '#0284c7',
          },
          {
            name: '--elev-floating',
            category: 'shadow',
            property: 'box-shadow',
            value: '0 4px 12px rgb(0 0 0 / 12%)',
          },
        ])
      );
      expect(collectRclStyleTokensInPage().some((token) => token.name === '--layout-panel')).toBe(
        false
      );
    } finally {
      globalThis.document = previousDocument;
      globalThis.getComputedStyle = previousStyle;
    }
  });

  it('resolves projected token aliases through computed CSS properties', () => {
    const previousDocument = globalThis.document;
    const previousStyle = globalThis.getComputedStyle;
    const values: Record<string, string> = { '--app-background': 'var(--color-background)' };
    const names = Object.keys(values);
    const applied = new Map<string, string>();
    const probe = {
      style: {
        cssText: '',
        setProperty: (property: string, value: string) => applied.set(property, value),
        removeProperty: (property: string) => applied.delete(property),
      },
      remove: () => undefined,
    };
    globalThis.document = {
      documentElement: {},
      styleSheets: [],
      body: { appendChild: () => undefined },
      createElement: () => probe,
    } as unknown as Document;
    globalThis.getComputedStyle = ((element: Element) => {
      if (element === probe) {
        return {
          getPropertyValue: (property: string) =>
            applied.get(property) === 'var(--app-background)' && property === 'background-color'
              ? 'rgb(255, 255, 255)'
              : '',
        };
      }
      return {
        length: names.length,
        item: (index: number) => names[index],
        getPropertyValue: (name: string) => values[name] ?? '',
      };
    }) as unknown as typeof getComputedStyle;
    try {
      expect(collectRclStyleTokensInPage()).toContainEqual({
        name: '--app-background',
        category: 'color',
        property: 'background',
        value: 'rgb(255, 255, 255)',
      });
    } finally {
      globalThis.document = previousDocument;
      globalThis.getComputedStyle = previousStyle;
    }
  });

  it('classifies every published visual token family and leaves only breakpoint metadata out', () => {
    const source = path.resolve(
      process.cwd(),
      '../../react-component-library/ui/src/design-tokens.css'
    );
    const names = Array.from(
      readFileSync(source, 'utf8').matchAll(/^\s*(--[\w-]+):/gm),
      (match) => match[1]
    ).filter((name): name is string => Boolean(name));
    const previousDocument = globalThis.document;
    const previousStyle = globalThis.getComputedStyle;
    globalThis.document = { documentElement: {}, styleSheets: [] } as unknown as Document;
    globalThis.getComputedStyle = (() => ({
      length: names.length,
      item: (index: number) => names[index],
      getPropertyValue: () => '16px',
    })) as unknown as typeof getComputedStyle;
    try {
      const classified = new Set(collectRclStyleTokensInPage().map((token) => token.name));
      expect(names.filter((name) => !classified.has(name))).toEqual(['--breakpoint-md']);
    } finally {
      globalThis.document = previousDocument;
      globalThis.getComputedStyle = previousStyle;
    }
  });
});
