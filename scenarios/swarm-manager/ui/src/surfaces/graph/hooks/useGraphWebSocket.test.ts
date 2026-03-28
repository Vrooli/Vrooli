import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useGraphWebSocket } from "./useGraphWebSocket";
import { cloneGraphDataInitialState, useGraphDataStore } from "../stores/graph-data-store";

class MockWebSocket {
  static instances: MockWebSocket[] = [];
  url: string;
  onopen: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: (() => void) | null = null;
  readyState = 0;
  close = vi.fn(() => {
    this.readyState = 3;
  });

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
    setTimeout(() => {
      if (this.readyState === 0) {
        this.readyState = 1;
        this.onopen?.();
      }
    }, 0);
  }

  simulateMessage(data: unknown) {
    this.onmessage?.({ data: JSON.stringify(data) });
  }

  simulateClose() {
    this.readyState = 3;
    this.onclose?.();
  }
}

vi.mock("@vrooli/api-base", async () => {
  const actual = await vi.importActual<typeof import("@vrooli/api-base")>("@vrooli/api-base");
  return {
    ...actual,
    buildWsUrl: (path: string) => `ws://localhost:8080${path}`,
  };
});

function resetStore(fetchGraphSpy = vi.fn().mockResolvedValue(undefined)) {
  useGraphDataStore.setState({
    ...cloneGraphDataInitialState(),
    fetchGraph: fetchGraphSpy,
  });
  return fetchGraphSpy;
}

function getFirstSocket(): MockWebSocket {
  const socket = MockWebSocket.instances[0];
  if (!socket) {
    throw new Error("Expected a WebSocket instance");
  }
  return socket;
}

describe("useGraphWebSocket", () => {
  beforeEach(() => {
    MockWebSocket.instances = [];
    vi.stubGlobal("WebSocket", MockWebSocket);
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it("does not connect when disabled", () => {
    resetStore();
    renderHook(() => useGraphWebSocket({ enabled: false, lens: "topology" }));
    expect(MockWebSocket.instances).toHaveLength(0);
  });

  it("connects when enabled", () => {
    resetStore();
    renderHook(() => useGraphWebSocket({ enabled: true, lens: "topology" }));
    expect(MockWebSocket.instances).toHaveLength(1);
    expect(getFirstSocket().url).toBe("ws://localhost:8080/ws/graph");
  });

  it("disconnects when disabled after being enabled", () => {
    resetStore();
    const { rerender } = renderHook(({ enabled }) => useGraphWebSocket({ enabled, lens: "topology" }), {
      initialProps: { enabled: true },
    });
    const ws = getFirstSocket();

    rerender({ enabled: false });
    expect(ws.close).toHaveBeenCalled();
  });

  it("refreshes only when the current lens is invalidated", async () => {
    const fetchGraphSpy = resetStore();
    renderHook(() => useGraphWebSocket({ enabled: true, lens: "flow" }));
    const ws = getFirstSocket();

    await vi.advanceTimersByTimeAsync(0);

    act(() => {
      ws.simulateMessage({
        type: "invalidate",
        data: { lenses: ["topology"] },
        timestamp: Date.now(),
      });
    });

    await vi.advanceTimersByTimeAsync(200);
    expect(fetchGraphSpy).not.toHaveBeenCalled();

    act(() => {
      ws.simulateMessage({
        type: "invalidate",
        data: { lenses: ["flow", "operations"] },
        timestamp: Date.now(),
      });
    });

    await vi.advanceTimersByTimeAsync(150);
    expect(fetchGraphSpy).toHaveBeenCalledWith("flow", { silent: true, force: true });
  });

  it("ignores heartbeat messages", async () => {
    const fetchGraphSpy = resetStore();
    renderHook(() => useGraphWebSocket({ enabled: true, lens: "topology" }));
    const ws = getFirstSocket();

    await vi.advanceTimersByTimeAsync(0);

    act(() => {
      ws.simulateMessage({
        type: "heartbeat",
        data: {},
        timestamp: Date.now(),
      });
    });

    await vi.advanceTimersByTimeAsync(250);
    expect(fetchGraphSpy).not.toHaveBeenCalled();
  });

  it("pulses updated nodes without forcing a refresh on node events alone", async () => {
    const fetchGraphSpy = resetStore();
    const pulseSpy = vi.fn();
    renderHook(() => useGraphWebSocket({ enabled: true, lens: "operations", onNodePulse: pulseSpy }));
    const ws = getFirstSocket();

    await vi.advanceTimersByTimeAsync(0);

    act(() => {
      ws.simulateMessage({
        type: "node-update",
        data: { id: "run/123" },
        timestamp: Date.now(),
      });
    });

    expect(pulseSpy).toHaveBeenCalledWith("run/123");
    await vi.advanceTimersByTimeAsync(250);
    expect(fetchGraphSpy).not.toHaveBeenCalled();
  });

  it("refreshes the current lens before reconnecting after close", async () => {
    const fetchGraphSpy = resetStore();
    renderHook(() => useGraphWebSocket({ enabled: true, lens: "operations" }));
    const ws = getFirstSocket();

    await vi.advanceTimersByTimeAsync(0);

    act(() => {
      ws.simulateClose();
    });

    expect(fetchGraphSpy).not.toHaveBeenCalled();
    await vi.advanceTimersByTimeAsync(1000);
    expect(fetchGraphSpy).toHaveBeenCalledWith("operations", { silent: true, force: true });
    expect(MockWebSocket.instances).toHaveLength(2);
  });

  it("cleans up on unmount", async () => {
    resetStore();
    const { unmount } = renderHook(() => useGraphWebSocket({ enabled: true, lens: "topology" }));
    const ws = getFirstSocket();

    await vi.advanceTimersByTimeAsync(0);
    unmount();

    expect(ws.close).toHaveBeenCalled();
  });
});
