import assert from "node:assert/strict";
import { Code, ConnectError } from "@connectrpc/connect";
import { test } from "vitest";
import { startErrorText } from "./startErrorText.js";

test("a refused caller is told who may act, in words that name the action", () => {
  assert.equal(startErrorText(new ConnectError("no", Code.PermissionDenied)), "Only a verified operator can start a usage mode.");
  assert.equal(startErrorText(new ConnectError("no", Code.PermissionDenied), "start Learn mode"), "Only a verified operator can start Learn mode.");
});

test("any other failure shows the server's own message, and a plain error shows its text", () => {
  assert.equal(startErrorText(new ConnectError("runner claude has no usage tiers", Code.InvalidArgument)), "runner claude has no usage tiers");
  assert.equal(startErrorText(new Error("network down")), "network down");
  assert.equal(startErrorText("odd"), "The request could not be completed.");
});
