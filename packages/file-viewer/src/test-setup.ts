import "@testing-library/jest-dom/vitest";
import { cleanup, configure } from "@testing-library/react";
import { afterEach, vi } from "vitest";

// Shared by the jsdom suites and the few node-environment ones (the Vite plugin).
if (typeof window !== "undefined") {
  // Renderers load lazily, so the first test to open a kind pays for its modules.
  configure({ asyncUtilTimeout: 8000 });

  // jsdom does not implement matchMedia, which the shared component library's
  // useMediaQuery hook calls through useSyncExternalStore. It answers "does not
  // match" to everything, which for the `(min-width: ...)` queries the library
  // asks means the small-viewport branch.
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    writable: true,
    value: vi.fn((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(() => false),
    })),
  });

  Object.defineProperty(HTMLCanvasElement.prototype, "getContext", {
    configurable: true,
    value: vi.fn(() => ({})),
  });

  // jsdom doesn't implement scrollTo or scrollIntoView.
  Element.prototype.scrollTo = vi.fn() as unknown as typeof Element.prototype.scrollTo;
  Element.prototype.scrollIntoView = vi.fn();
  HTMLMediaElement.prototype.play = vi.fn().mockResolvedValue(undefined);
  HTMLMediaElement.prototype.pause = vi.fn();
  HTMLMediaElement.prototype.load = vi.fn();
}

afterEach(() => {
  cleanup();
});
