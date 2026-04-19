import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { act, renderHook } from "@testing-library/react";
import { useFullscreen } from "./useFullscreen";

describe("useFullscreen", () => {
  // Capture the original descriptors so we can restore them after each test.
  const originalRequestDesc = Object.getOwnPropertyDescriptor(
    HTMLElement.prototype,
    "requestFullscreen",
  );
  const originalExitDesc = Object.getOwnPropertyDescriptor(
    Document.prototype,
    "exitFullscreen",
  );
  let fullscreenElement: Element | null = null;
  let requestSpy: ReturnType<typeof vi.fn>;
  let exitSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fullscreenElement = null;
    Object.defineProperty(document, "fullscreenElement", {
      configurable: true,
      get: () => fullscreenElement,
    });

    requestSpy = vi.fn((element: HTMLElement) => {
      fullscreenElement = element;
      document.dispatchEvent(new Event("fullscreenchange"));
      return Promise.resolve();
    });
    exitSpy = vi.fn(() => {
      fullscreenElement = null;
      document.dispatchEvent(new Event("fullscreenchange"));
      return Promise.resolve();
    });

    Object.defineProperty(HTMLElement.prototype, "requestFullscreen", {
      configurable: true,
      writable: true,
      value: function (this: HTMLElement) {
        return requestSpy(this);
      },
    });
    Object.defineProperty(Document.prototype, "exitFullscreen", {
      configurable: true,
      writable: true,
      value: function (this: Document) {
        return exitSpy();
      },
    });
  });

  afterEach(() => {
    if (originalRequestDesc) {
      Object.defineProperty(HTMLElement.prototype, "requestFullscreen", originalRequestDesc);
    } else {
      // jsdom does not define requestFullscreen by default; delete our stub.
      delete (HTMLElement.prototype as unknown as Record<string, unknown>)["requestFullscreen"];
    }
    if (originalExitDesc) {
      Object.defineProperty(Document.prototype, "exitFullscreen", originalExitDesc);
    } else {
      delete (Document.prototype as unknown as Record<string, unknown>)["exitFullscreen"];
    }
    vi.clearAllMocks();
  });

  it("starts not fullscreen and does not auto-invoke", () => {
    const { result } = renderHook(() => useFullscreen());
    expect(result.current.isFullscreen).toBe(false);
    expect(requestSpy).not.toHaveBeenCalled();
  });

  it("enters and exits fullscreen via imperative methods", async () => {
    const { result } = renderHook(() => useFullscreen());

    await act(async () => {
      await result.current.enter();
    });
    expect(result.current.isFullscreen).toBe(true);
    expect(requestSpy).toHaveBeenCalledTimes(1);

    await act(async () => {
      await result.current.exit();
    });
    expect(result.current.isFullscreen).toBe(false);
    expect(exitSpy).toHaveBeenCalledTimes(1);
  });

  it("toggle enters when not fullscreen and exits when fullscreen", async () => {
    const { result } = renderHook(() => useFullscreen());

    await act(async () => {
      await result.current.toggle();
    });
    expect(result.current.isFullscreen).toBe(true);

    await act(async () => {
      await result.current.toggle();
    });
    expect(result.current.isFullscreen).toBe(false);
  });
});
