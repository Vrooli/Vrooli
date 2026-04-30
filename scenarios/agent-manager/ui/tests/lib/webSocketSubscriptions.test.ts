import test from "node:test";
import assert from "node:assert/strict";
import { WebSocketClientMessageType } from "../../src/lib/webSocketProtocol.js";
import { createWebSocketSubscriptionManager } from "../../src/lib/webSocketSubscriptions.js";

function decodeType(message: Record<string, unknown>) {
  return message.type;
}

function decodeRunId(message: Record<string, unknown>) {
  const subscription = message.run_subscription;
  return typeof subscription === "object" && subscription !== null && "run_id" in subscription
    ? subscription.run_id
    : "";
}

test("subscription calls update desired state even while the socket is closed", () => {
  const sent: Record<string, unknown>[] = [];
  const manager = createWebSocketSubscriptionManager({
    isOpen: () => false,
    send: (message) => sent.push(message),
  });

  manager.subscribe("run-2");
  manager.subscribe("run-1");
  manager.subscribeAll();

  assert.deepEqual(sent, []);
  assert.deepEqual(manager.snapshot(), {
    desiredRunSubscriptions: ["run-1", "run-2"],
    desiredAllEvents: true,
  });
});

test("replayDesired sends all active subscription intent in deterministic order", () => {
  const sent: Record<string, unknown>[] = [];
  const manager = createWebSocketSubscriptionManager({
    isOpen: () => true,
    send: (message) => sent.push(message),
  });

  manager.subscribe("run-2");
  manager.subscribe("run-1");
  manager.subscribeAll();
  sent.length = 0;

  manager.replayDesired();

  assert.deepEqual(sent.map(decodeType), [
    WebSocketClientMessageType.SubscribeAll,
    WebSocketClientMessageType.Subscribe,
    WebSocketClientMessageType.Subscribe,
  ]);
  assert.deepEqual(sent.slice(1).map(decodeRunId), ["run-1", "run-2"]);
});

test("unsubscribe updates desired state and sends immediately when connected", () => {
  const sent: Record<string, unknown>[] = [];
  const manager = createWebSocketSubscriptionManager({
    isOpen: () => true,
    send: (message) => sent.push(message),
  });

  manager.subscribe("run-1");
  manager.unsubscribe("run-1");
  manager.subscribeAll();
  manager.unsubscribeAll();

  assert.deepEqual(manager.snapshot(), {
    desiredRunSubscriptions: [],
    desiredAllEvents: false,
  });
  assert.deepEqual(sent.map(decodeType), [
    WebSocketClientMessageType.Subscribe,
    WebSocketClientMessageType.Unsubscribe,
    WebSocketClientMessageType.SubscribeAll,
    WebSocketClientMessageType.UnsubscribeAll,
  ]);
});

test("repeated subscription calls do not send duplicate live messages", () => {
  const sent: Record<string, unknown>[] = [];
  const manager = createWebSocketSubscriptionManager({
    isOpen: () => true,
    send: (message) => sent.push(message),
  });

  manager.subscribe("run-1");
  manager.subscribe("run-1");
  manager.subscribeAll();
  manager.subscribeAll();

  assert.deepEqual(sent.map(decodeType), [
    WebSocketClientMessageType.Subscribe,
    WebSocketClientMessageType.SubscribeAll,
  ]);
});
