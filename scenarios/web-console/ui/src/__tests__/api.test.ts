import { describe, it, expect, vi, beforeEach } from "vitest";
import { apiBaseMock } from "../test-utils";

// Mock api-base before importing api module
vi.mock("@vrooli/api-base", () => apiBaseMock());

// [REQ:P0-002a] PTY Session Backend - API client
// [REQ:P0-004a] api-base HTTP Integration
describe("api module", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  // Session CRUD + APIError-shape tests previously lived here but exercised
  // a fake REST surface (mocked globalThis.fetch). The real implementation
  // is Connect-RPC; behavior is covered by component/hook tests against
  // ../api/sessions. The error envelope is now ConnectError, not APIError.

  // [REQ:P0-004b] api-base WebSocket Integration
  it("buildSessionWsUrl constructs correct WebSocket URL", async () => {
    const { buildSessionWsUrl } = await import("../api/sessions");
    const url = buildSessionWsUrl("session-abc");

    expect(url).toContain("/sessions/session-abc/ws");
    expect(url).toMatch(/^ws/);
  });

  // resolveFileReference / getFileReferenceContent are covered through the
  // ConversationService in src/api/conversation.ts; their REST-era tests
  // were removed when the lib/api shims were retired.
});

describe("toErrorInfo", () => {
  it("extracts fields from APIError", async () => {
    const { APIError, toErrorInfo } = await import("../lib/errors");
    const err = new APIError(429, {
      error: "Limit reached",
      code: "session_limit_reached",
      category: "resource_limit",
      recovery: "Close a session",
      retry: true,
    });
    const info = toErrorInfo(err);
    expect(info.message).toBe("Limit reached");
    expect(info.recovery).toBe("Close a session");
    expect(info.retry).toBe(true);
  });

  it("handles plain Error (no recovery/retry)", async () => {
    const { toErrorInfo } = await import("../lib/errors");
    const info = toErrorInfo(new Error("network failed"));
    expect(info.message).toBe("network failed");
    expect(info.recovery).toBeUndefined();
    expect(info.retry).toBeUndefined();
  });

  it("handles non-Error values", async () => {
    const { toErrorInfo } = await import("../lib/errors");
    const info = toErrorInfo("string error");
    expect(info.message).toBe("Unknown error");
  });

  it("omits empty recovery/retry from APIError", async () => {
    const { APIError, toErrorInfo } = await import("../lib/errors");
    const err = new APIError(400, {
      error: "Bad input",
      code: "invalid_body",
    });
    const info = toErrorInfo(err);
    expect(info.message).toBe("Bad input");
    expect(info.recovery).toBeUndefined();
    expect(info.retry).toBeUndefined();
  });
});
