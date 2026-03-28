import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useGraphWebSocket } from "./useGraphWebSocket";
import { useGraphDataStore, graphDataInitialState } from "../stores/graph-data-store";
import type { Node, Edge } from "@xyflow/react";

// Mock WebSocket.
class MockWebSocket {
  static instances: MockWebSocket[] = [];
  url: string;
  onopen: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: (() => void) | null = null;
  readyState = 0; // CONNECTING
  close = vi.fn(() => {
    this.readyState = 3; // CLOSED
  });

  constructor(url: string) {
    this.url = url;
    this.readyState = 0;
    MockWebSocket.instances.push(this);
    // Simulate async open.
    setTimeout(() => {
      if (this.readyState === 0) {
        this.readyState = 1; // OPEN
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

// Mock buildWsUrl and buildApiUrl.
vi.mock("@vrooli/api-base", () => ({
  buildWsUrl: (path: string) => `ws://localhost:8080${path}`,
  buildApiUrl: (path: string) => `http://localhost:8080${path}`,
}));

function resetStore() {
  useGraphDataStore.setState({
    ...graphDataInitialState,
    entityFilters: { ...graphDataInitialState.entityFilters },
  });
}

const makeNode = (id: string): Node => ({
  id,
  type: "test",
  position: { x: 0, y: 0 },
  data: { label: id },
});

const makeEdge = (source: string, target: string): Edge => ({
  id: `${source}->${target}`,
  source,
  target,
});

describe("useGraphWebSocket", () => {
  beforeEach(() => {
    resetStore();
    MockWebSocket.instances = [];
    vi.stubGlobal("WebSocket", MockWebSocket);
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it("does not connect when disabled", () => {
    renderHook(() => useGraphWebSocket({ enabled: false }));
    expect(MockWebSocket.instances).toHaveLength(0);
  });

  it("connects when enabled", () => {
    renderHook(() => useGraphWebSocket({ enabled: true }));
    expect(MockWebSocket.instances).toHaveLength(1);
    expect(MockWebSocket.instances[0].url).toBe("ws://localhost:8080/ws/graph");
  });

  it("disconnects when disabled after being enabled", () => {
    const { rerender } = renderHook(
      ({ enabled }) => useGraphWebSocket({ enabled }),
      { initialProps: { enabled: true } },
    );
    const ws = MockWebSocket.instances[0];
    rerender({ enabled: false });
    expect(ws.close).toHaveBeenCalled();
  });

  it("processes full-sync messages", async () => {
    renderHook(() => useGraphWebSocket({ enabled: true }));
    const ws = MockWebSocket.instances[0];

    await vi.advanceTimersByTimeAsync(0); // Let onopen fire.

    const nodes = [makeNode("a"), makeNode("b")];
    const edges = [makeEdge("a", "b")];

    act(() => {
      ws.simulateMessage({
        type: "full-sync",
        data: { nodes, edges },
        timestamp: Date.now(),
      });
    });

    const state = useGraphDataStore.getState();
    expect(state.nodes).toEqual(nodes);
    expect(state.edges).toEqual(edges);
  });

  it("processes node-update messages", async () => {
    // Pre-populate store.
    useGraphDataStore.setState({
      nodes: [makeNode("a")],
      edges: [],
    });

    const pulseSpy = vi.fn();
    renderHook(() => useGraphWebSocket({ enabled: true, onNodePulse: pulseSpy }));
    const ws = MockWebSocket.instances[0];

    await vi.advanceTimersByTimeAsync(0);

    act(() => {
      ws.simulateMessage({
        type: "node-update",
        data: { id: "a", type: "test", data: { label: "updated-a", status: "running" } },
        timestamp: Date.now(),
      });
    });

    const state = useGraphDataStore.getState();
    expect((state.nodes[0].data as Record<string, unknown>).label).toBe("updated-a");
    expect((state.nodes[0].data as Record<string, unknown>).status).toBe("running");
    expect(pulseSpy).toHaveBeenCalledWith("a");
  });

  it("processes node-add messages", async () => {
    useGraphDataStore.setState({ nodes: [makeNode("a")], edges: [] });

    renderHook(() => useGraphWebSocket({ enabled: true }));
    const ws = MockWebSocket.instances[0];
    await vi.advanceTimersByTimeAsync(0);

    const newNode = makeNode("b");
    act(() => {
      ws.simulateMessage({
        type: "node-add",
        data: newNode,
        timestamp: Date.now(),
      });
    });

    expect(useGraphDataStore.getState().nodes).toHaveLength(2);
  });

  it("processes node-remove messages", async () => {
    useGraphDataStore.setState({ nodes: [makeNode("a"), makeNode("b")], edges: [] });

    renderHook(() => useGraphWebSocket({ enabled: true }));
    const ws = MockWebSocket.instances[0];
    await vi.advanceTimersByTimeAsync(0);

    act(() => {
      ws.simulateMessage({
        type: "node-remove",
        data: { id: "a" },
        timestamp: Date.now(),
      });
    });

    const nodes = useGraphDataStore.getState().nodes;
    expect(nodes).toHaveLength(1);
    expect(nodes[0].id).toBe("b");
  });

  it("processes edge-add messages", async () => {
    useGraphDataStore.setState({ nodes: [], edges: [] });

    renderHook(() => useGraphWebSocket({ enabled: true }));
    const ws = MockWebSocket.instances[0];
    await vi.advanceTimersByTimeAsync(0);

    act(() => {
      ws.simulateMessage({
        type: "edge-add",
        data: makeEdge("a", "b"),
        timestamp: Date.now(),
      });
    });

    expect(useGraphDataStore.getState().edges).toHaveLength(1);
  });

  it("processes edge-remove messages", async () => {
    useGraphDataStore.setState({ nodes: [], edges: [makeEdge("a", "b")] });

    renderHook(() => useGraphWebSocket({ enabled: true }));
    const ws = MockWebSocket.instances[0];
    await vi.advanceTimersByTimeAsync(0);

    act(() => {
      ws.simulateMessage({
        type: "edge-remove",
        data: { id: "a->b" },
        timestamp: Date.now(),
      });
    });

    expect(useGraphDataStore.getState().edges).toHaveLength(0);
  });

  it("ignores heartbeat messages", async () => {
    useGraphDataStore.setState({ nodes: [makeNode("a")], edges: [] });

    renderHook(() => useGraphWebSocket({ enabled: true }));
    const ws = MockWebSocket.instances[0];
    await vi.advanceTimersByTimeAsync(0);

    act(() => {
      ws.simulateMessage({
        type: "heartbeat",
        data: {},
        timestamp: Date.now(),
      });
    });

    // State unchanged.
    expect(useGraphDataStore.getState().nodes).toHaveLength(1);
  });

  it("cleans up on unmount", async () => {
    const { unmount } = renderHook(() => useGraphWebSocket({ enabled: true }));
    const ws = MockWebSocket.instances[0];
    await vi.advanceTimersByTimeAsync(0);

    unmount();
    expect(ws.close).toHaveBeenCalled();
  });
});
