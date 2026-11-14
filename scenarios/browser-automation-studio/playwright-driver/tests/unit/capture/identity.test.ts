import { collectComponentIdentityInPage } from '../../../src/capture/identity';

describe('collectComponentIdentityInPage', () => {
  it('retains exact RCL asset, version, stamp, and source-slot metadata', () => {
    const previousDocument = globalThis.document;
    const previousCSS = globalThis.CSS;
    const element = {
      tagName: 'BUTTON',
      getAttribute(name: string) {
        return (
          (
            {
              id: 'main-action',
              'data-rcl-asset': 'controls.button',
              'data-rcl-version': '2.1.0',
              'data-rcl-stamp': 'vite',
              'data-rcl-source-slot': 'primitives.button',
            } as Record<string, string>
          )[name] ?? null
        );
      },
    };
    globalThis.document = { querySelectorAll: () => [element] } as unknown as Document;
    globalThis.CSS = { ...globalThis.CSS, escape: (value: string) => value };
    try {
      expect(collectComponentIdentityInPage()).toEqual({
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
      });
    } finally {
      globalThis.document = previousDocument;
      globalThis.CSS = previousCSS;
    }
  });
});
