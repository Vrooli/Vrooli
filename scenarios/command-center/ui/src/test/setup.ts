/**
 * Vitest test setup file.
 * Configures testing-library matchers and global test utilities.
 */
import "@testing-library/jest-dom";

// jsdom does not implement canvas. Returning null keeps axe's color-contrast
// rule deterministic without pretending that unit tests provide scene pixels.
Object.defineProperty(HTMLCanvasElement.prototype, "getContext", {
  configurable: true,
  value: () => null,
});

// jsdom does not implement matchMedia; the capability probe reads reduced-motion through it.
Object.defineProperty(window, "matchMedia", {
  configurable: true,
  value: (query: string) => ({ matches: false, media: query, onchange: null, addEventListener: () => undefined, removeEventListener: () => undefined, addListener: () => undefined, removeListener: () => undefined, dispatchEvent: () => false }),
});
