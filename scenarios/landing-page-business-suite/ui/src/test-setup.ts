import '@testing-library/jest-dom/vitest';
import { vi } from 'vitest';

// Global mock for useLandingVariant hook - prevents "must be used within provider" errors
// Individual tests can override this mock if they need different behavior
vi.mock('./app/providers/useLandingVariant', () => ({
  useLandingVariant: () => ({
    variant: { slug: 'test-variant', name: 'Test Variant' },
    config: { sections: [], downloads: [], fallback: false, branding: { coming_soon_enabled: false } },
    loading: false,
    error: null,
    resolution: 'api_select',
    statusNote: null,
    lastUpdated: Date.now(),
    refresh: vi.fn().mockResolvedValue(undefined),
  }),
}));

// Global mock for useToast hook - prevents "must be used within ToastProvider" errors
vi.mock('./shared/ui/useToast', () => ({
  useToast: () => ({
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
    warning: vi.fn(),
  }),
}));

// Polyfill for jsdom missing pointer capture APIs (required for @radix-ui/react-select)
if (typeof Element !== 'undefined') {
  // jsdom versions differ from the DOM lib typings; inspect as an unknown test
  // environment boundary before installing only the missing Radix APIs.
  const elementPrototype = Element.prototype as unknown as Record<string, unknown>;
  if (typeof elementPrototype.hasPointerCapture !== 'function') {
    Object.defineProperty(Element.prototype, 'hasPointerCapture', {
      value: () => false,
      configurable: true,
    });
  }
  if (typeof elementPrototype.setPointerCapture !== 'function') {
    Object.defineProperty(Element.prototype, 'setPointerCapture', {
      value: () => undefined,
      configurable: true,
    });
  }
  if (typeof elementPrototype.releasePointerCapture !== 'function') {
    Object.defineProperty(Element.prototype, 'releasePointerCapture', {
      value: () => undefined,
      configurable: true,
    });
  }
  if (typeof elementPrototype.scrollIntoView !== 'function') {
    Object.defineProperty(Element.prototype, 'scrollIntoView', {
      value: () => undefined,
      configurable: true,
    });
  }
}
