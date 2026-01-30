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
  if (!Element.prototype.hasPointerCapture) {
    Element.prototype.hasPointerCapture = function () {
      return false;
    };
  }
  if (!Element.prototype.setPointerCapture) {
    Element.prototype.setPointerCapture = function () {};
  }
  if (!Element.prototype.releasePointerCapture) {
    Element.prototype.releasePointerCapture = function () {};
  }
}
