import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { DEFAULT_SYNC_CHANNEL_NAME } from "./sync";
import type { StudioSyncEvent } from "./sync";
import {
  STUDIO_SYNC_COALESCE_MS,
  useStudioSync,
  type StudioSyncHandlers,
} from "./useStudioSync";

function makeHandlers(overrides: Partial<StudioSyncHandlers> = {}): StudioSyncHandlers {
  return {
    activeDesignId: "design-a",
    activeRevision: 4,
    dirty: false,
    saving: false,
    onReloadDesign: vi.fn(),
    onConflict: vi.fn(),
    onDesignsChanged: vi.fn(),
    onAssetsChanged: vi.fn(),
    ...overrides,
  };
}

/** Deliver a message the way another tab's channel would. */
function emit(event: unknown): void {
  act(() => {
    window.dispatchEvent(new CustomEvent(DEFAULT_SYNC_CHANNEL_NAME, { detail: event }));
  });
}

function advance(ms: number = STUDIO_SYNC_COALESCE_MS): void {
  act(() => {
    vi.advanceTimersByTime(ms);
  });
}

beforeEach(() => {
  vi.useFakeTimers();
  // Pin the fallback transport: BroadcastChannel delivery is a task, and the
  // behaviour under test is the hook's, not the browser's scheduling.
  vi.stubGlobal("BroadcastChannel", undefined);
});

afterEach(() => {
  Reflect.deleteProperty(document, "visibilityState");
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

describe("useStudioSync reconciliation", () => {
  it("reloads an idle tab when another tab saved a newer revision", () => {
    const handlers = makeHandlers();
    renderHook(() => useStudioSync(handlers));

    emit({ type: "designs-changed", designId: "design-a", revision: 5, source: "tab-b" });
    expect(handlers.onReloadDesign).not.toHaveBeenCalled();
    advance();

    expect(handlers.onReloadDesign).toHaveBeenCalledTimes(1);
    expect(handlers.onConflict).not.toHaveBeenCalled();
    expect(handlers.onDesignsChanged).toHaveBeenCalledTimes(1);
  });

  it("raises the conflict banner instead of overwriting unsaved edits", () => {
    const handlers = makeHandlers({ dirty: true });
    renderHook(() => useStudioSync(handlers));

    emit({ type: "designs-changed", designId: "design-a", revision: 5, source: "tab-b" });
    advance();

    expect(handlers.onConflict).toHaveBeenCalledTimes(1);
    expect(handlers.onReloadDesign).not.toHaveBeenCalled();
  });

  it("ignores a targeted event while our own save is in flight", () => {
    const handlers = makeHandlers({ saving: true });
    renderHook(() => useStudioSync(handlers));

    emit({ type: "designs-changed", designId: "design-a", revision: 5, source: "tab-b" });
    advance();

    expect(handlers.onReloadDesign).not.toHaveBeenCalled();
    expect(handlers.onConflict).not.toHaveBeenCalled();
    // Refetch would race the save's own revision.
    expect(handlers.onDesignsChanged).not.toHaveBeenCalled();
  });

  it("treats a coarse designs-changed event as a list refresh only", () => {
    const handlers = makeHandlers();
    renderHook(() => useStudioSync(handlers));

    emit({ type: "designs-changed", source: "tab-b" });
    advance();

    expect(handlers.onDesignsChanged).toHaveBeenCalledTimes(1);
    expect(handlers.onReloadDesign).not.toHaveBeenCalled();
    expect(handlers.onConflict).not.toHaveBeenCalled();
  });

  it("ignores a targeted event for a design that is not active", () => {
    const handlers = makeHandlers();
    renderHook(() => useStudioSync(handlers));

    emit({ type: "designs-changed", designId: "design-b", revision: 9, source: "tab-b" });
    advance();

    expect(handlers.onDesignsChanged).toHaveBeenCalledTimes(1);
    expect(handlers.onReloadDesign).not.toHaveBeenCalled();
    expect(handlers.onConflict).not.toHaveBeenCalled();
  });
});

describe("useStudioSync asset signals", () => {
  it("refetches assets for the active design", () => {
    const handlers = makeHandlers();
    renderHook(() => useStudioSync(handlers));

    emit({ type: "assets-changed", designId: "design-a", source: "tab-b" });
    advance();

    expect(handlers.onAssetsChanged).toHaveBeenCalledTimes(1);
    expect(handlers.onAssetsChanged).toHaveBeenCalledWith("design-a");
  });

  it("does not fire for a design whose assets are not loaded", () => {
    const handlers = makeHandlers();
    renderHook(() => useStudioSync(handlers));

    emit({ type: "assets-changed", designId: "design-b", source: "tab-b" });
    advance();

    expect(handlers.onAssetsChanged).not.toHaveBeenCalled();
  });
});

describe("useStudioSync coalescing", () => {
  it("collapses a burst into one callback per kind per window", () => {
    const handlers = makeHandlers();
    renderHook(() => useStudioSync(handlers));

    emit({ type: "designs-changed", source: "tab-b" });
    emit({ type: "designs-changed", source: "tab-c" });
    emit({ type: "assets-changed", designId: "design-a", source: "tab-b" });
    emit({ type: "assets-changed", designId: "design-a", source: "tab-c" });

    advance(STUDIO_SYNC_COALESCE_MS - 1);
    expect(handlers.onDesignsChanged).not.toHaveBeenCalled();
    expect(handlers.onAssetsChanged).not.toHaveBeenCalled();

    advance(1);
    expect(handlers.onDesignsChanged).toHaveBeenCalledTimes(1);
    expect(handlers.onAssetsChanged).toHaveBeenCalledTimes(1);
  });

  it("reconciles a burst against the highest revision seen", () => {
    const handlers = makeHandlers();
    renderHook(() => useStudioSync(handlers));

    emit({ type: "designs-changed", designId: "design-a", revision: 5, source: "tab-b" });
    emit({ type: "designs-changed", designId: "design-a", revision: 8, source: "tab-c" });
    emit({ type: "designs-changed", designId: "design-a", revision: 6, source: "tab-d" });
    advance();

    expect(handlers.onReloadDesign).toHaveBeenCalledTimes(1);
  });

  it("opens a fresh window after a flush", () => {
    const handlers = makeHandlers();
    renderHook(() => useStudioSync(handlers));

    emit({ type: "designs-changed", source: "tab-b" });
    advance();
    emit({ type: "designs-changed", source: "tab-b" });
    advance();

    expect(handlers.onDesignsChanged).toHaveBeenCalledTimes(2);
  });
});

describe("useStudioSync focus refetch", () => {
  it("refetches the list on focus, debounced", () => {
    const handlers = makeHandlers();
    renderHook(() => useStudioSync(handlers));

    act(() => {
      window.dispatchEvent(new Event("focus"));
    });
    expect(handlers.onDesignsChanged).not.toHaveBeenCalled();

    advance();
    expect(handlers.onDesignsChanged).toHaveBeenCalledTimes(1);
  });

  it("refetches when the document becomes visible", () => {
    Object.defineProperty(document, "visibilityState", {
      configurable: true,
      get: () => "visible",
    });
    const handlers = makeHandlers();
    renderHook(() => useStudioSync(handlers));

    act(() => {
      document.dispatchEvent(new Event("visibilitychange"));
    });
    advance();

    expect(handlers.onDesignsChanged).toHaveBeenCalledTimes(1);
  });

  it("does not refetch while the document stays hidden", () => {
    Object.defineProperty(document, "visibilityState", {
      configurable: true,
      get: () => "hidden",
    });
    const handlers = makeHandlers();
    renderHook(() => useStudioSync(handlers));

    act(() => {
      document.dispatchEvent(new Event("visibilitychange"));
    });
    advance();

    expect(handlers.onDesignsChanged).not.toHaveBeenCalled();
  });

  it("skips the focus refetch while a save is in flight", () => {
    const handlers = makeHandlers({ saving: true });
    renderHook(() => useStudioSync(handlers));

    act(() => {
      window.dispatchEvent(new Event("focus"));
    });
    advance();

    expect(handlers.onDesignsChanged).not.toHaveBeenCalled();
  });
});

describe("useStudioSync teardown", () => {
  it("cancels a pending burst on unmount", () => {
    const handlers = makeHandlers();
    const { unmount } = renderHook(() => useStudioSync(handlers));

    emit({ type: "designs-changed", designId: "design-a", revision: 5, source: "tab-b" });
    unmount();
    advance();

    expect(handlers.onDesignsChanged).not.toHaveBeenCalled();
    expect(handlers.onReloadDesign).not.toHaveBeenCalled();
    expect(handlers.onConflict).not.toHaveBeenCalled();
  });

  it("cancels a pending focus refetch on unmount", () => {
    const handlers = makeHandlers();
    const { unmount } = renderHook(() => useStudioSync(handlers));

    act(() => {
      window.dispatchEvent(new Event("focus"));
    });
    unmount();
    advance();

    expect(handlers.onDesignsChanged).not.toHaveBeenCalled();
  });

  it("detaches its channel so later events cannot reach it", () => {
    const handlers = makeHandlers();
    const { unmount } = renderHook(() => useStudioSync(handlers));

    unmount();
    emit({ type: "designs-changed", source: "tab-b" });
    advance();

    expect(handlers.onDesignsChanged).not.toHaveBeenCalled();
  });
});

describe("useStudioSync robustness", () => {
  it("drops malformed events without throwing or calling back", () => {
    const handlers = makeHandlers();
    renderHook(() => useStudioSync(handlers));

    expect(() => {
      emit({ type: "designs-changed" });
      emit({ type: "designs-changed", source: "" });
      emit({ type: "assets-changed", source: "tab-b" });
      emit({ type: "assets-changed", designId: 7, source: "tab-b" });
      emit({ type: "unexpected", source: "tab-b" });
      emit(null);
      emit("not an object");
    }).not.toThrow();

    advance();
    expect(handlers.onDesignsChanged).not.toHaveBeenCalled();
    expect(handlers.onAssetsChanged).not.toHaveBeenCalled();
    expect(handlers.onReloadDesign).not.toHaveBeenCalled();
    expect(handlers.onConflict).not.toHaveBeenCalled();
  });

  it("reads the latest handlers without resubscribing", () => {
    const first = makeHandlers();
    const second = makeHandlers({ activeRevision: 10 });
    const { rerender } = renderHook(
      ({ handlers }: { handlers: StudioSyncHandlers }) => useStudioSync(handlers),
      { initialProps: { handlers: first } },
    );

    rerender({ handlers: second });
    // Newer than the *second* handler's revision (10), so only it reloads.
    emit({ type: "designs-changed", designId: "design-a", revision: 11, source: "tab-b" });
    advance();

    expect(first.onReloadDesign).not.toHaveBeenCalled();
    expect(second.onReloadDesign).toHaveBeenCalledTimes(1);
  });

  it("ignores its own broadcasts after they leave the channel", () => {
    const raw = vi.fn();
    window.addEventListener(DEFAULT_SYNC_CHANNEL_NAME, raw);
    const handlers = makeHandlers();
    const { result } = renderHook(() => useStudioSync(handlers));

    act(() => {
      result.current.notifyDesignsChanged({ designId: "design-a", revision: 6 });
    });

    expect(raw).toHaveBeenCalledTimes(1);
    const event = raw.mock.calls[0]?.[0] as CustomEvent<StudioSyncEvent> | undefined;
    expect(event?.detail).toMatchObject({
      type: "designs-changed",
      designId: "design-a",
      revision: 6,
    });

    advance();
    expect(handlers.onReloadDesign).not.toHaveBeenCalled();
    expect(handlers.onDesignsChanged).not.toHaveBeenCalled();
    window.removeEventListener(DEFAULT_SYNC_CHANNEL_NAME, raw);
  });

  it("broadcasts coarse design, asset and template signals", () => {
    const raw = vi.fn();
    window.addEventListener(DEFAULT_SYNC_CHANNEL_NAME, raw);
    const { result } = renderHook(() => useStudioSync(makeHandlers()));

    act(() => {
      result.current.notifyDesignsChanged();
      result.current.notifyAssetsChanged("design-a");
      result.current.notifyTemplateChanged();
    });

    expect(raw).toHaveBeenCalledTimes(3);
    const details = raw.mock.calls.map(
      (call) => (call[0] as CustomEvent<StudioSyncEvent>).detail,
    );
    expect(details).toEqual([
      { type: "designs-changed", source: expect.any(String) },
      { type: "assets-changed", designId: "design-a", source: expect.any(String) },
      { type: "template-changed", source: expect.any(String) },
    ]);
    window.removeEventListener(DEFAULT_SYNC_CHANNEL_NAME, raw);
  });
});
