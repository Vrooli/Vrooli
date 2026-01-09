// Test environment setup for Vitest + jsdom
import '@testing-library/jest-dom/vitest';
import { cleanup } from '@testing-library/react';
import { afterEach } from 'vitest';

// Setup ResizeObserver mock for ReactFlow components
if (typeof window !== 'undefined') {
  if (!('ResizeObserver' in window)) {
    class ResizeObserverStub {
      observe() {}
      unobserve() {}
      disconnect() {}
    }

    // @ts-expect-error Assigning stub for test runtime only
    window.ResizeObserver = ResizeObserverStub;
  }

  // Enable React 18 concurrent features in tests
  (globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;

  // Mock matchMedia for responsive components
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: (query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: () => {}, // deprecated
      removeListener: () => {}, // deprecated
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => true,
    }),
  });

  // Mock IntersectionObserver for lazy loading components
  if (!('IntersectionObserver' in window)) {
    class IntersectionObserverStub {
      observe() {}
      unobserve() {}
      disconnect() {}
    }

    // @ts-expect-error Assigning stub for test runtime only
    window.IntersectionObserver = IntersectionObserverStub;
  }
}

afterEach(() => {
  cleanup();
});
