import { afterEach, beforeEach, vi } from 'vitest';

// jest-dom matchers are loaded conditionally so the setup file (and
// therefore *every* test) doesn't blow up before `pnpm install` lands the
// new RTL devDependencies. Pure utility / reducer tests that don't import
// from @testing-library/* will run unaffected; integration tests that do
// will fail at their own import site with a clearer error.
//
// The string concat hides the specifier from Vite's static analyzer so
// import-analysis doesn't fail when the package isn't installed yet. Once
// the dep lands, this resolves at runtime exactly like a normal import.
try {
  const moduleId = '@testing-library/' + 'jest-dom/vitest';
  await import(/* @vite-ignore */ moduleId);
} catch {
  /* jest-dom not installed yet; pure tests still run */
}

beforeEach(() => {
  if (typeof window !== 'undefined') {
    window.localStorage.clear();
  }
});

afterEach(() => {
  vi.clearAllMocks();
  vi.restoreAllMocks();
});

if (typeof globalThis.ResizeObserver === 'undefined') {
  globalThis.ResizeObserver = class ResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
}

if (typeof window !== 'undefined' && typeof window.matchMedia !== 'function') {
  window.matchMedia = (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  });
}
