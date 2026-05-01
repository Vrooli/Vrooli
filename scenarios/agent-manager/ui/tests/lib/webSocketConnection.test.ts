import { test } from "vitest";
import assert from "node:assert/strict";
import { nextReconnectDelayMs, shouldReconnectAfterClose } from "../../src/lib/webSocketConnection.js";

test("shouldReconnectAfterClose blocks intentional cleanup closes", () => {
  assert.equal(
    shouldReconnectAfterClose({
      enabled: true,
      intentionalClose: true,
      socketIsCurrent: true,
      reconnectAttempts: 0,
      maxReconnectAttempts: 3,
    }),
    false
  );
});

test("shouldReconnectAfterClose blocks stale socket close callbacks", () => {
  assert.equal(
    shouldReconnectAfterClose({
      enabled: true,
      intentionalClose: false,
      socketIsCurrent: false,
      reconnectAttempts: 0,
      maxReconnectAttempts: 3,
    }),
    false
  );
});

test("shouldReconnectAfterClose allows current unexpected closes within retry limit", () => {
  assert.equal(
    shouldReconnectAfterClose({
      enabled: true,
      intentionalClose: false,
      socketIsCurrent: true,
      reconnectAttempts: 2,
      maxReconnectAttempts: 3,
    }),
    true
  );
});

test("nextReconnectDelayMs applies capped exponential backoff and deterministic jitter", () => {
  assert.equal(nextReconnectDelayMs(1000, 1, 25), 1025);
  assert.equal(nextReconnectDelayMs(1000, 3, 25), 2275);
  assert.equal(nextReconnectDelayMs(20000, 3, 25), 30025);
});
