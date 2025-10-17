// Basic test environment wiring for Vitest + jsdom.
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

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  (globalThis as any).IS_REACT_ACT_ENVIRONMENT = true;
}
