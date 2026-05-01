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
