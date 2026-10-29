import assert from "node:assert/strict";
import { test } from "vitest";
import { ConnectError, Code } from "@connectrpc/connect";
import { presentError, type ErrorKind } from "./errorPresentation";

// Each row is a failure as a transport actually reports it, and what an operator
// must be told instead of the raw string.
const cases: Array<[unknown, ErrorKind, RegExp, boolean]> = [
  [new ConnectError("HTTP 401", Code.Unauthenticated), "auth", /sign in again/i, true],
  ["Request failed: 401", "auth", /sign in again/i, true],
  [new ConnectError("denied", Code.PermissionDenied), "forbidden", /grant access/i, false],
  ["Request failed: 503", "unreachable", /scenario is running/i, true],
  [new TypeError("Failed to fetch"), "unreachable", /not responding/i, true],
  [new ConnectError("", Code.Unavailable), "unreachable", /not responding/i, true],
  [new ConnectError("unsupported content type text/html", Code.Unknown), "misrouted", /web page instead of the API/i, true],
  ["Request failed: 404", "not_found", /no longer exists/i, false],
  [new ConnectError("deadline", Code.DeadlineExceeded), "timeout", /too long/i, true],
  [new ConnectError("agent_manager.v1.measures.MeasuresService.UsageBurners is not implemented", Code.Unimplemented), "unimplemented", /does not serve this yet/i, false],
];

test("presentError turns transport failures into a cause and an action, keeping the raw text as detail", () => {
  for (const [error, kind, message, retryable] of cases) {
    const view = presentError(error);
    assert.equal(view.kind, kind, String(error));
    assert.match(view.message, message);
    assert.equal(view.retryable, retryable);
    assert.ok(!view.title.includes("["), "the title never carries a transport code");
    assert.equal(view.detail, error instanceof Error ? error.message : error);
  }
});

test("presentError shows a server's own sentence for rejected requests and short unknown failures", () => {
  const rejected = presentError(new ConnectError("scope path must stay inside the project root", Code.InvalidArgument));
  assert.equal(rejected.kind, "rejected");
  assert.equal(rejected.message, "scope path must stay inside the project root");
  assert.equal(rejected.retryable, false);

  assert.equal(presentError(new Error("Failed to cancel workflow")).message, "Failed to cancel workflow");
  const opaque = presentError("Request failed: 500");
  assert.equal(opaque.kind, "unknown");
  assert.equal(opaque.message, "The request could not be completed.");
  assert.equal(presentError({ unexpected: true }).message, "The request could not be completed.");
  assert.equal(presentError("x".repeat(400)).message, "The request could not be completed.", "a dump is never body copy");
});
