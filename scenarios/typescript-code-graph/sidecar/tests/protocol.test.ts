import { describe, expect, it } from "vitest";

import { parseMessage } from "../src/framing.js";
import { isRequest } from "../src/protocol.js";
import type {
  ErrorResponse,
  ExtractRequest,
  HandshakeResponse,
  RewriteApplyRequest,
} from "../src/protocol.js";

describe("protocol round-trip", () => {
  it("handshake request parses + classifies", () => {
    const req = {
      type: "handshake",
      request_id: "r1",
      protocol_version: 1,
    };
    const line = JSON.stringify(req);
    const parsed = parseMessage(line);
    expect(isRequest(parsed)).toBe(true);
  });

  it("heartbeat request round-trips", () => {
    const req = { type: "heartbeat", request_id: "hb1" };
    expect(isRequest(parseMessage(JSON.stringify(req)))).toBe(true);
  });

  it("extract request round-trips with snake_case keys", () => {
    const req: ExtractRequest = {
      type: "extract",
      request_id: "ex1",
      scenario_path: "/tmp/x",
    };
    const parsed = parseMessage(JSON.stringify(req)) as ExtractRequest;
    expect(parsed.scenario_path).toBe("/tmp/x");
    expect(isRequest(parsed)).toBe(true);
  });

  it("rewrite_apply request round-trips with file_move + import_rewrite", () => {
    const req: RewriteApplyRequest = {
      type: "rewrite_apply",
      request_id: "rw1",
      scenario_path: "/tmp/x",
      operations: [
        { file_move: { from_path: "a.ts", to_path: "b.ts" } },
        { import_rewrite: { old_path: "./a", new_path: "./b" } },
      ],
    };
    const parsed = parseMessage(JSON.stringify(req)) as RewriteApplyRequest;
    expect(parsed.operations).toHaveLength(2);
    expect(parsed.operations[0]!.file_move?.from_path).toBe("a.ts");
    expect(parsed.operations[1]!.import_rewrite?.new_path).toBe("./b");
  });

  it("cancel + shutdown classify as requests", () => {
    expect(isRequest(parseMessage('{"type":"cancel","request_id":"r"}'))).toBe(true);
    expect(isRequest(parseMessage('{"type":"shutdown"}'))).toBe(true);
  });

  it("unknown type is not a request", () => {
    expect(isRequest(parseMessage('{"type":"bogus"}'))).toBe(false);
  });

  it("handshake response shape preserves all fields", () => {
    const resp: HandshakeResponse = {
      type: "handshake",
      request_id: "r1",
      protocol_version: 1,
      sidecar_version: "0.1.0",
      ts_morph_version: "28.0.0",
    };
    const parsed = JSON.parse(JSON.stringify(resp)) as HandshakeResponse;
    expect(parsed.protocol_version).toBe(1);
    expect(parsed.ts_morph_version).toBe("28.0.0");
  });

  it("error response uses error kind enum", () => {
    const err: ErrorResponse = {
      type: "error",
      request_id: "r1",
      kind: "no_tsconfig_found",
      message: "nope",
    };
    const parsed = JSON.parse(JSON.stringify(err)) as ErrorResponse;
    expect(parsed.kind).toBe("no_tsconfig_found");
  });
});
