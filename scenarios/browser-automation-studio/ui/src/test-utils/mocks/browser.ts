export function installBrowserApiShims() {
  if (typeof window === 'undefined') {
    return;
  }

  if (!('ResizeObserver' in window)) {
    class ResizeObserverStub {
      observe() {}
      unobserve() {}
      disconnect() {}
    }

    // @ts-expect-error Assigning stub for test runtime only.
    window.ResizeObserver = ResizeObserverStub;
  }

  (globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;

  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: (query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: () => {},
      removeListener: () => {},
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => true,
    }),
  });

  if (!('IntersectionObserver' in window)) {
    class IntersectionObserverStub {
      observe() {}
      unobserve() {}
      disconnect() {}
    }

    // @ts-expect-error Assigning stub for test runtime only.
    window.IntersectionObserver = IntersectionObserverStub;
  }
}
