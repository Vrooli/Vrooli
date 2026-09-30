import assert from "node:assert/strict";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, test, vi } from "vitest";
import { useWebSocket } from "../../src/hooks/useWebSocket.js";
import { WebSocketClientMessageType } from "../../src/lib/webSocketProtocol.js";

class ControllableWebSocket {
  static readonly CONNECTING = 0;
  static readonly OPEN = 1;
  static readonly CLOSING = 2;
  static readonly CLOSED = 3;

  readonly sent: string[] = [];
  readyState = ControllableWebSocket.CONNECTING;
  onopen: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  onclose: ((event: CloseEvent) => void) | null = null;

  constructor(readonly url: string) {}

  send(data: string) {
    this.sent.push(data);
  }

  close() {
    this.readyState = ControllableWebSocket.CLOSED;
    this.onclose?.(new CloseEvent("close"));
  }

  open() {
    this.readyState = ControllableWebSocket.OPEN;
    this.onopen?.(new Event("open"));
  }

  receive(data: unknown) {
    this.onmessage?.(
      new MessageEvent("message", {
        data: typeof data === "string" ? data : JSON.stringify(data),
      }),
    );
  }

  remoteClose() {
    this.readyState = ControllableWebSocket.CLOSED;
    this.onclose?.(new CloseEvent("close"));
  }

  fail() {
    this.onerror?.(new Event("error"));
  }
}

function installWebSocketMock() {
  const sockets: ControllableWebSocket[] = [];

  class MockWebSocket extends ControllableWebSocket {
    constructor(url: string) {
      super(url);
      sockets.push(this);
    }
  }

  vi.stubGlobal("WebSocket", MockWebSocket as unknown as typeof WebSocket);

  return { sockets };
}

function sentMessages(socket: ControllableWebSocket) {
  return socket.sent.map((message) => JSON.parse(message) as Record<string, unknown>);
}

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

test("useWebSocket replays queued subscriptions when the socket opens", async () => {
  const { sockets } = installWebSocketMock();
  vi.spyOn(console, "log").mockImplementation(() => undefined);
  const onStatusChange = vi.fn();

  const { result, unmount } = renderHook(() =>
    useWebSocket({
      onStatusChange,
    }),
  );

  await waitFor(() => {
    assert.equal(sockets.length, 1);
  });

  act(() => {
    result.current.subscribe("run-2");
    result.current.subscribe("run-1");
    result.current.subscribeAll();
  });

  assert.deepEqual(sentMessages(sockets[0]), []);

  act(() => {
    sockets[0].open();
  });

  await waitFor(() => {
    assert.equal(result.current.status, "connected");
  });

  assert.deepEqual(sentMessages(sockets[0]).map((message) => message.type), [
    WebSocketClientMessageType.SubscribeAll,
    WebSocketClientMessageType.Subscribe,
    WebSocketClientMessageType.Subscribe,
  ]);
  assert.deepEqual(
    sentMessages(sockets[0]).slice(1).map((message) => {
      const subscription = message.run_subscription as Record<string, unknown>;
      return subscription.run_id;
    }),
    ["run-1", "run-2"],
  );
  assert.deepEqual(onStatusChange.mock.calls.map((call) => call[0]), [
    "connecting",
    "connected",
  ]);
  unmount();
});

test("useWebSocket preserves subscription intent across reconnects", async () => {
  vi.spyOn(Math, "random").mockReturnValue(0);
  vi.spyOn(console, "log").mockImplementation(() => undefined);
  const { sockets } = installWebSocketMock();

  const { result, unmount } = renderHook(() =>
    useWebSocket({
      reconnectInterval: 1,
      maxReconnectAttempts: 1,
    }),
  );

  await waitFor(() => {
    assert.equal(sockets.length, 1);
  });

  act(() => {
    result.current.subscribe("run-1");
    sockets[0].open();
  });

  await waitFor(() => {
    assert.equal(result.current.status, "connected");
  });
  assert.deepEqual(sentMessages(sockets[0]).map((message) => message.type), [
    WebSocketClientMessageType.Subscribe,
  ]);

  act(() => {
    sockets[0].remoteClose();
  });

  await waitFor(() => {
    assert.equal(result.current.status, "disconnected");
  });

  await waitFor(() => {
    assert.equal(sockets.length, 2);
  });

  act(() => {
    sockets[1].open();
  });

  await waitFor(() => {
    assert.equal(result.current.status, "connected");
  });

  assert.deepEqual(sentMessages(sockets[1]).map((message) => message.type), [
    WebSocketClientMessageType.Subscribe,
  ]);
  const subscription = sentMessages(sockets[1])[0].run_subscription as Record<string, unknown>;
  assert.equal(subscription.run_id, "run-1");
  unmount();
});

test("useWebSocket normalizes server messages before invoking onMessage", async () => {
  vi.spyOn(console, "log").mockImplementation(() => undefined);
  vi.spyOn(console, "warn").mockImplementation(() => undefined);
  const { sockets } = installWebSocketMock();
  const onMessage = vi.fn();

  const { unmount } = renderHook(() =>
    useWebSocket({
      onMessage,
    }),
  );

  await waitFor(() => {
    assert.equal(sockets.length, 1);
  });

  act(() => {
    sockets[0].open();
    sockets[0].receive({
      type: "AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_STATUS",
      run_status: {
        run_id: "run-1",
        status: "RUN_STATUS_COMPLETE",
      },
    });
  });

  await waitFor(() => {
    assert.equal(onMessage.mock.calls.length, 1);
  });
  assert.deepEqual(onMessage.mock.calls[0][0], {
    type: "run_status",
    runId: "run-1",
    payload: {
      id: "run-1",
      status: 5,
    },
  });
  unmount();
});

test("useWebSocket stays inert while disabled and reports a disconnected send without constructing a socket", async () => {
  const { sockets } = installWebSocketMock();
  const warning = vi.spyOn(console, "warn").mockImplementation(() => undefined);
  const { result, unmount } = renderHook(() => useWebSocket({ enabled: false }));

  act(() => result.current.send({ type: "operator_ping" }));

  assert.equal(sockets.length, 0);
  assert.equal(result.current.status, "disconnected");
  assert.equal(warning.mock.calls[0]?.[0], "[WebSocket] Cannot send, not connected");
  unmount();
});

test("useWebSocket exposes connection and message failures without delivering malformed evidence", async () => {
  const { sockets } = installWebSocketMock();
  const onMessage = vi.fn();
  const warning = vi.spyOn(console, "warn").mockImplementation(() => undefined);
  const error = vi.spyOn(console, "error").mockImplementation(() => undefined);
  vi.spyOn(console, "log").mockImplementation(() => undefined);
  const { result, unmount } = renderHook(() => useWebSocket({ onMessage }));

  await waitFor(() => assert.equal(sockets.length, 1));
  act(() => {
    sockets[0].open();
    sockets[0].receive("not json");
    sockets[0].receive({ type: "unknown-event" });
    sockets[0].fail();
  });

  await waitFor(() => assert.equal(result.current.status, "error"));
  assert.equal(result.current.error?.message, "WebSocket connection error");
  assert.equal(onMessage.mock.calls.length, 0);
  assert.ok(warning.mock.calls.some((call) => call[0] === "[WebSocket] Ignoring unsupported message"));
  assert.ok(error.mock.calls.some((call) => call[0] === "[WebSocket] Failed to parse message:"));
  unmount();
});

test("useWebSocket applies unsubscribe intent and explicit reconnect without replaying removed subscriptions", async () => {
  vi.spyOn(console, "log").mockImplementation(() => undefined);
  const { sockets } = installWebSocketMock();
  const { result, unmount } = renderHook(() => useWebSocket());
  await waitFor(() => assert.equal(sockets.length, 1));
  act(() => {
    sockets[0].open();
    result.current.subscribe("keep");
    result.current.subscribe("remove");
    result.current.unsubscribe("remove");
    result.current.reconnect();
  });
  await waitFor(() => assert.equal(sockets.length, 2));
  act(() => sockets[1].open());
  await waitFor(() => assert.equal(result.current.status, "connected"));

  const replayed = sentMessages(sockets[1]);
  assert.equal(replayed.length, 1);
  assert.equal((replayed[0].run_subscription as Record<string, unknown>).run_id, "keep");
  unmount();
});

test("useWebSocket can remove the global subscription before reconnecting", async () => {
  vi.spyOn(console, "log").mockImplementation(() => undefined);
  const { sockets } = installWebSocketMock();
  const { result, unmount } = renderHook(() => useWebSocket());
  await waitFor(() => assert.equal(sockets.length, 1));
  act(() => {
    sockets[0].open();
    result.current.subscribeAll();
    result.current.unsubscribeAll();
    result.current.reconnect();
  });
  await waitFor(() => assert.equal(sockets.length, 2));
  act(() => sockets[1].open());
  await waitFor(() => assert.equal(result.current.status, "connected"));
  assert.deepEqual(sentMessages(sockets[1]), []);
  unmount();
});
