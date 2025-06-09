import { vi } from 'vitest';

// Essential mocks only - no fancy stuff
vi.mock('../utils/display/device.js', () => ({}));
vi.mock('../utils/display/chatTools.js', () => ({}));
vi.mock('@uiw/react-codemirror', () => ({ default: () => null }));
vi.mock('../utils/pubsub.js', () => ({
    PubSub: {
        get: () => ({
            subscribe: vi.fn(() => vi.fn()),
            publish: vi.fn(),
            hasSubscribers: () => false,
        }),
    },
}));

// Minimal browser mocks
Object.defineProperty(window, 'matchMedia', {
    value: () => ({ matches: false, addListener: () => {}, removeListener: () => {} }),
});

global.IntersectionObserver = vi.fn(() => ({
    observe: vi.fn(),
    unobserve: vi.fn(),
    disconnect: vi.fn(),
}));

global.ResizeObserver = vi.fn(() => ({
    observe: vi.fn(),
    unobserve: vi.fn(), 
    disconnect: vi.fn(),
}));