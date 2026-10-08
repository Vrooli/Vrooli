import { vi } from "vitest";

/**
 * Tell `matchMedia` what viewport a test is running at.
 *
 * jsdom does not implement `matchMedia`, and the shared component library reads
 * it through `useBreakpoint` to choose between its anchored and sheet
 * presentations. The default stub answers `matches: false` to everything, which
 * for a `(min-width: …)` query means *below* the breakpoint — so a test that
 * says nothing is exercising the small-viewport branch, not the large one. A
 * test that means to assert desktop behaviour has to say so.
 *
 * @param width Viewport inline size in CSS pixels.
 */
export function setViewportWidth(width: number) {
  const parseMinWidth = (query: string) => {
    const rem = /\(min-width:\s*([\d.]+)rem\)/.exec(query);
    if (rem) return Number(rem[1]) * 16;
    const px = /\(min-width:\s*([\d.]+)px\)/.exec(query);
    return px ? Number(px[1]) : null;
  };
  const parseMaxWidth = (query: string) => {
    const rem = /\(max-width:\s*([\d.]+)rem\)/.exec(query);
    if (rem) return Number(rem[1]) * 16;
    const px = /\(max-width:\s*([\d.]+)px\)/.exec(query);
    return px ? Number(px[1]) : null;
  };
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    writable: true,
    value: vi.fn((query: string) => {
      const min = parseMinWidth(query);
      const max = parseMaxWidth(query);
      let matches = false;
      if (min !== null) matches = width >= min;
      else if (max !== null) matches = width <= max;
      return {
        matches,
        media: query,
        onchange: null,
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        addListener: vi.fn(),
        removeListener: vi.fn(),
        dispatchEvent: vi.fn(() => false),
      };
    }),
  });
}

/** A viewport at or above the library's medium breakpoint (48rem). */
export const setDesktopViewport = () => {
  setViewportWidth(1280);
};

/** A viewport below the library's medium breakpoint. */
export const setMobileViewport = () => {
  setViewportWidth(390);
};
