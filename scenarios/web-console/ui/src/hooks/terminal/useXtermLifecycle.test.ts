import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
// provider-free-exception: this test mounts a DOM-only xterm lifecycle harness;
// it does not render application components or use application providers.
import { render } from "@testing-library/react";
import { createElement, type MutableRefObject, type ReactNode } from "react";
import { useXtermLifecycle, waitForTerminalFont } from "./useXtermLifecycle";

const fit = vi.fn();
const textarea = {
  inputMode: "",
  addEventListener: vi.fn(),
  removeEventListener: vi.fn(),
};
const terminal = {
  open: vi.fn(), dispose: vi.fn(), loadAddon: vi.fn(), options: {}, cols: 80, rows: 24,
  onTitleChange: vi.fn(() => ({ dispose: vi.fn() })), textarea,
  buffer: { active: { baseY: 0, viewportY: 0 } }, scrollLines: vi.fn(), element: null,
};
vi.mock("@xterm/xterm", () => ({ Terminal: vi.fn(() => terminal) }));
vi.mock("@xterm/addon-fit", () => ({ FitAddon: vi.fn(() => ({ fit })) }));
vi.mock("@xterm/addon-web-links", () => ({ WebLinksAddon: vi.fn(() => ({})) }));

describe("useXtermLifecycle", () => {
  beforeEach(() => {
    terminal.options = {};
  });

  it("waits for the bundled face before fitting", async () => {
    const load = vi.fn(() => Promise.resolve([]));
    Object.defineProperty(document, "fonts", { configurable: true, value: { load } });
    await waitForTerminalFont(14);
    expect(load).toHaveBeenCalledWith(expect.stringContaining("14px"));
  });

  it("constructs one terminal and waits for the initial fit", async () => {
    const { result } = renderHook(() => useXtermLifecycle({
      sessionId: "s", paneFontSize: 14, paneTheme: {}, wheelScrollSensitivity: 1,
      sendResize: vi.fn(), getServerSize: () => null, isFollower: () => false,
      renamePaneById: vi.fn(), syncPaneUpdate: vi.fn(),
    }));
    const host = document.createElement("div");
    Object.defineProperty(host, "clientWidth", { value: 800 });
    Object.defineProperty(host, "clientHeight", { value: 600 });
    act(() => {
      (result.current.containerRef as MutableRefObject<HTMLDivElement | null>).current = host;
      (result.current.terminalHostRef as MutableRefObject<HTMLDivElement | null>).current = document.createElement("div");
    });
    expect(result.current).toBeDefined();
  });

  it("applies every effective font increment to the rendered terminal", async () => {
    const base = {
      sessionId: "font-session", paneTheme: {}, wheelScrollSensitivity: 1,
      sendResize: vi.fn(), getServerSize: () => null, isFollower: () => false,
      renamePaneById: vi.fn(), syncPaneUpdate: vi.fn(),
    };
    let latest: ReturnType<typeof useXtermLifecycle> | undefined;
    vi.stubGlobal("ResizeObserver", class {
      observe() {}
      disconnect() {}
    });
    function Harness({ size }: { size: number }): ReactNode {
      latest = useXtermLifecycle({ ...base, paneFontSize: size });
      return createElement("div", { ref: latest.containerRef }, createElement("div", { ref: latest.terminalHostRef }));
    }
    const view = render(createElement(Harness, { size: 14 }));

    await act(async () => { await Promise.resolve(); });
    expect((latest?.terminal?.options as { fontSize?: number }).fontSize).toBe(14);
    for (const size of [15, 16, 17, 18, 19]) {
      await act(async () => {
        view.rerender(createElement(Harness, { size }));
        await Promise.resolve();
      });
      expect((latest?.terminal?.options as { fontSize?: number }).fontSize).toBe(size);
    }
    view.unmount();
    vi.unstubAllGlobals();
  });

  it("coalesces a burst of resize notifications into one state update per frame", async () => {
    let resize: (() => void) | undefined;
    let frame: FrameRequestCallback | undefined;
    let width = 800;
    vi.stubGlobal("ResizeObserver", class {
      constructor(callback: () => void) { resize = callback; }
      observe() {}
      disconnect() {}
    });
    vi.stubGlobal("requestAnimationFrame", (callback: FrameRequestCallback) => {
      frame = callback;
      return 1;
    });
    vi.stubGlobal("cancelAnimationFrame", () => {});

    let renders = 0;
    let latest: ReturnType<typeof useXtermLifecycle> | undefined;
    const sendResize = vi.fn();
    const getServerSize = () => null;
    const isFollower = () => false;
    const renamePaneById = vi.fn();
    const syncPaneUpdate = vi.fn();
    function Harness(): ReactNode {
      renders += 1;
      latest = useXtermLifecycle({
        sessionId: "resize-session", paneFontSize: 14, paneTheme: {}, wheelScrollSensitivity: 1,
        sendResize, getServerSize, isFollower, renamePaneById, syncPaneUpdate,
      });
      return createElement("div", { ref: latest.containerRef });
    }
    const view = render(createElement(Harness));
    await act(async () => { await Promise.resolve(); });
    const host = latest?.containerRef.current;
    if (!host) throw new Error("resize harness did not attach its container");
    Object.defineProperty(host, "clientWidth", { configurable: true, get: () => width });
    Object.defineProperty(host, "clientHeight", { configurable: true, value: 600 });
    const before = renders;
    width = 810;
    act(() => {
      resize?.();
      resize?.();
      frame?.(0);
    });
    expect(renders - before).toBeLessThanOrEqual(1);
    view.unmount();
    vi.unstubAllGlobals();
  });

  it("keeps touch input from opening the native keyboard after xterm blur", () => {
    textarea.addEventListener.mockClear();
    textarea.removeEventListener.mockClear();
    vi.stubGlobal("ResizeObserver", class {
      observe() {}
      disconnect() {}
    });
    Object.defineProperty(navigator, "maxTouchPoints", { configurable: true, value: 1 });
    const host = document.createElement("div");
    Object.defineProperty(host, "clientWidth", { value: 800 });
    Object.defineProperty(host, "clientHeight", { value: 600 });
    const lifecycleOptions = {
      sessionId: "touch-session", paneFontSize: 14, paneTheme: {}, wheelScrollSensitivity: 1,
      sendResize: vi.fn(), getServerSize: () => null, isFollower: () => false,
      renamePaneById: vi.fn(), syncPaneUpdate: vi.fn(),
    };
    const { unmount } = renderHook(() => {
      const lifecycle = useXtermLifecycle(lifecycleOptions);
      (lifecycle.containerRef as MutableRefObject<HTMLDivElement | null>).current = host;
      (lifecycle.terminalHostRef as MutableRefObject<HTMLDivElement | null>).current = document.createElement("div");
      return lifecycle;
    });

    const blur = textarea.addEventListener.mock.calls.find(([event]) => event === "blur")?.[1] as (() => void) | undefined;
    expect(blur).toBeDefined();
    textarea.inputMode = "text";
    blur?.();
    expect(textarea.inputMode).toBe("none");

    unmount();
    expect(textarea.removeEventListener).toHaveBeenCalledWith("blur", blur);
    Reflect.deleteProperty(navigator, "maxTouchPoints");
    vi.unstubAllGlobals();
  });
});
