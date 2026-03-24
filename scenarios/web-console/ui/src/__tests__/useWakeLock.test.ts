import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useWakeLock } from "../hooks/useWakeLock";

/** Creates a mock WakeLock API and sentinel for testing. */
function createMockWakeLock() {
  const sentinel: WakeLockSentinel = {
    released: false,
    type: "screen",
    release: vi.fn(() => Promise.resolve()),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(() => true),
    onrelease: null,
  };
  const wakeLock = {
    request: vi.fn(() => Promise.resolve(sentinel)),
  };
  return { wakeLock, sentinel };
}

describe("useWakeLock", () => {
  let original: unknown;

  beforeEach(() => {
    original = navigator.wakeLock;
  });

  afterEach(() => {
    // Restore original state
    if (original === undefined) {
      // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
      delete (navigator as unknown as Record<string, unknown>).wakeLock;
    } else {
      Object.defineProperty(navigator, "wakeLock", {
        value: original,
        writable: true,
        configurable: true,
      });
    }
  });

  function installMock() {
    const mock = createMockWakeLock();
    Object.defineProperty(navigator, "wakeLock", {
      value: mock.wakeLock,
      writable: true,
      configurable: true,
    });
    return mock;
  }

  it("requests wake lock when enabled", async () => {
    const { wakeLock } = installMock();

    renderHook(() => useWakeLock(true));
    // Let the async request settle
    await act(async () => {});

    expect(wakeLock.request).toHaveBeenCalledWith("screen");
  });

  it("does not request when disabled", async () => {
    const { wakeLock } = installMock();

    renderHook(() => useWakeLock(false));
    await act(async () => {});

    expect(wakeLock.request).not.toHaveBeenCalled();
  });

  it("releases sentinel on unmount", async () => {
    const { sentinel } = installMock();

    const { unmount } = renderHook(() => useWakeLock(true));
    await act(async () => {});

    unmount();
    await act(async () => {});

    expect(sentinel.release).toHaveBeenCalled();
  });

  it("re-acquires on visibilitychange → visible", async () => {
    const { wakeLock } = installMock();

    renderHook(() => useWakeLock(true));
    await act(async () => {});

    expect(wakeLock.request).toHaveBeenCalledTimes(1);

    // Simulate tab becoming visible again
    Object.defineProperty(document, "visibilityState", {
      value: "visible",
      writable: true,
      configurable: true,
    });
    await act(async () => {
      document.dispatchEvent(new Event("visibilitychange"));
    });

    expect(wakeLock.request).toHaveBeenCalledTimes(2);
  });

  it("does not re-acquire on visibilitychange when disabled", async () => {
    const { wakeLock } = installMock();

    renderHook(() => useWakeLock(false));
    await act(async () => {});

    Object.defineProperty(document, "visibilityState", {
      value: "visible",
      writable: true,
      configurable: true,
    });
    await act(async () => {
      document.dispatchEvent(new Event("visibilitychange"));
    });

    expect(wakeLock.request).not.toHaveBeenCalled();
  });

  it("handles unsupported browser gracefully", async () => {
    // Ensure wakeLock is not present
    // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
    delete (navigator as unknown as Record<string, unknown>).wakeLock;

    // Should not throw
    const { unmount } = renderHook(() => useWakeLock(true));
    await act(async () => {});
    unmount();
  });

  it("handles rejected request gracefully", async () => {
    const mock = installMock();
    mock.wakeLock.request.mockRejectedValueOnce(new DOMException("Not allowed", "NotAllowedError"));
    const debugSpy = vi.spyOn(console, "debug").mockImplementation(() => {});

    renderHook(() => useWakeLock(true));
    await act(async () => {});

    // Should not throw, just log
    expect(debugSpy).toHaveBeenCalledWith(
      "[useWakeLock] request denied:",
      expect.any(DOMException),
    );
    debugSpy.mockRestore();
  });

  it("releases old sentinel when toggled off then on", async () => {
    const mock = installMock();
    const firstSentinel = { ...createMockWakeLock().sentinel, release: vi.fn(() => Promise.resolve()) };
    const secondSentinel = createMockWakeLock().sentinel;
    mock.wakeLock.request
      .mockResolvedValueOnce(firstSentinel)
      .mockResolvedValueOnce(secondSentinel);

    const { rerender } = renderHook(({ enabled }) => useWakeLock(enabled), {
      initialProps: { enabled: true },
    });
    await act(async () => {});

    // Toggle off
    rerender({ enabled: false });
    await act(async () => {});
    expect(firstSentinel.release).toHaveBeenCalled();

    // Toggle on again
    rerender({ enabled: true });
    await act(async () => {});
    expect(mock.wakeLock.request).toHaveBeenCalledTimes(2);
  });

  it("cleans up visibilitychange listener on unmount", async () => {
    installMock();
    const removeSpy = vi.spyOn(document, "removeEventListener");

    const { unmount } = renderHook(() => useWakeLock(true));
    await act(async () => {});

    unmount();

    expect(removeSpy).toHaveBeenCalledWith("visibilitychange", expect.any(Function));
    removeSpy.mockRestore();
  });
});
