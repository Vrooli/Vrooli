/**
 * Self-tests for the workspace-domain test factories. Co-located with the
 * feature so deleting `features/workspace/` takes them along.
 */
import { fromJson, toJson } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";

import { ListOperationsResponseSchema } from "@vrooli/proto-types/image-tools/v1/ops/ops_pb";

import {
  makeListOperationsResponse,
  makeOperationInfo,
  makeRunOpImageResult,
  makeRunOpMetadataResult,
} from "./factories";

describe("makeOperationInfo", () => {
  it("returns an operation with a non-empty name and category", () => {
    const op = makeOperationInfo();
    expect(op.name).not.toBe("");
    expect(op.category).not.toBe("");
  });

  it("merges overrides without dropping defaults", () => {
    const op = makeOperationInfo({ name: "crop" });
    expect(op.name).toBe("crop");
    expect(op.summary).not.toBe("");
  });
});

describe("makeListOperationsResponse", () => {
  it("includes decodable/encodable formats and a non-empty operation list", () => {
    const r = makeListOperationsResponse();
    expect(r.operations.length).toBeGreaterThan(0);
    expect(r.decodableFormats).toContain("png");
    expect(r.encodableFormats).toContain("png");
  });

  it("round-trips through ListOperationsResponseSchema JSON encode + decode", () => {
    const original = makeListOperationsResponse();
    const decoded = fromJson(ListOperationsResponseSchema, toJson(ListOperationsResponseSchema, original));
    expect(decoded.operations.map((o) => o.name)).toEqual(original.operations.map((o) => o.name));
  });

  it("accepts operation overrides", () => {
    const r = makeListOperationsResponse({ operations: [makeOperationInfo({ name: "rotate" })] });
    expect(r.operations.map((o) => o.name)).toEqual(["rotate"]);
  });
});

describe("run-op result factories", () => {
  it("makeRunOpImageResult defaults to an image kind with metadata", () => {
    const r = makeRunOpImageResult();
    expect(r.kind).toBe("image");
    expect(r.url).not.toBe("");
    expect(r.width).toBeGreaterThan(0);
  });

  it("makeRunOpMetadataResult defaults to a metadata kind with JSON", () => {
    const r = makeRunOpMetadataResult();
    expect(r.kind).toBe("metadata");
    expect(() => JSON.parse(r.json)).not.toThrow();
  });
});
