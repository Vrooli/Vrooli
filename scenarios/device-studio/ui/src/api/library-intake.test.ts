import { webcrypto } from "node:crypto";
import { Buffer } from "node:buffer";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  listIntakeCandidates: vi.fn(),
  uploadFile: vi.fn(),
  getFile: vi.fn(),
  decodeApiError: vi.fn(async (response: Response) => new Error(`HTTP ${response.status}`)),
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: () => ({ listIntakeCandidates: mocks.listIntakeCandidates }),
}));
vi.mock("./client", () => ({
  decodeApiError: mocks.decodeApiError,
  getFile: mocks.getFile,
  PROTO_READ_OPTIONS: { ignoreUnknownFields: true },
  transport: {},
  uploadFile: mocks.uploadFile,
}));

import {
  downloadLocalGeometryCandidate,
  importLocalGeometryCandidate,
  listLocalGeometryCandidates,
  MAX_LOCAL_GEOMETRY_BYTES,
} from "./library-intake";

const candidateJSON = (overrides: Record<string, unknown> = {}) => ({
  id: "candidate-1",
  revision: 1,
  revisionSha256: "revision-hash",
  displayName: "Test device",
  originalFilename: "model.glb",
  originalSha256: "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
  sizeBytes: "3",
  mime: "model/gltf-binary",
  importMethod: "local_upload",
  sourceLabel: "Local file",
  licenseClaim: "unknown",
  attribution: "",
  declaredUses: [],
  parentCandidateId: "",
  parentRevision: 0,
  parentSha256: "",
  reviewStatus: "unreviewed",
  renderStatus: "not_reviewed",
  ...overrides,
});

function byteResponse(bytes: Uint8Array): Response {
  return new Response(Buffer.from(bytes), { status: 200 });
}

describe("library intake API client boundaries", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal("crypto", webcrypto);
  });

  it("returns only the requested candidate when the typed import response preserves identity and unreviewed status", async () => {
    const candidate = candidateJSON();
    mocks.uploadFile.mockResolvedValue(new Response(JSON.stringify({ candidate }), { status: 200 }));
    const file = new File(["glb"], "model.glb", { type: "model/gltf-binary" });

    const result = await importLocalGeometryCandidate(file, {
      candidateId: "candidate-1", displayName: "Test device", sourceLabel: "Local file",
      licenseClaim: "unknown", attribution: "",
    });

    expect(result.id).toBe("candidate-1");
    expect(result.reviewStatus).toBe("unreviewed");
    expect(result.renderStatus).toBe("not_reviewed");
  });

  it.each([
    ["a response for another identity", candidateJSON({ id: "other-candidate" })],
    ["an unexpectedly reviewed response", candidateJSON({ reviewStatus: "reviewed" })],
    ["an unexpectedly renderable response", candidateJSON({ renderStatus: "renderable" })],
  ])("refuses %s from the import service", async (_case, candidate) => {
    mocks.uploadFile.mockResolvedValue(new Response(JSON.stringify({ candidate }), { status: 200 }));
    const file = new File(["glb"], "model.glb", { type: "model/gltf-binary" });

    await expect(importLocalGeometryCandidate(file, {
      candidateId: "candidate-1", displayName: "Test device", sourceLabel: "Local file",
      licenseClaim: "unknown", attribution: "",
    })).rejects.toThrow();
  });

  it("rejects empty and oversized uploads before sending any bytes", async () => {
    const claims = { candidateId: "candidate-1", displayName: "Test device", sourceLabel: "Local file", licenseClaim: "unknown", attribution: "" };
    const empty = new File([], "empty.glb");
    const oversized = new File(["x"], "large.glb");
    Object.defineProperty(oversized, "size", { value: MAX_LOCAL_GEOMETRY_BYTES + 1 });

    await expect(importLocalGeometryCandidate(empty, claims)).rejects.toThrow(/non-empty/);
    await expect(importLocalGeometryCandidate(oversized, claims)).rejects.toThrow(/128 MiB/);
    expect(mocks.uploadFile).not.toHaveBeenCalled();
  });

  it("requires a stable paging cursor and forwards valid cursors to the typed catalog", async () => {
    mocks.listIntakeCandidates.mockResolvedValue({ candidates: [], nextAfterCandidateId: "" });
    await listLocalGeometryCandidates("candidate-1");
    expect(mocks.listIntakeCandidates).toHaveBeenCalledWith({ afterCandidateId: "candidate-1", pageSize: 100 });
    expect(() => listLocalGeometryCandidates("../private" )).toThrow(/stable candidate ID/);
    expect(mocks.listIntakeCandidates).toHaveBeenCalledTimes(1);
  });

  it("returns downloaded bytes only after exact size and SHA-256 verification", async () => {
    const bytes = new TextEncoder().encode("abc");
    mocks.getFile.mockResolvedValue(byteResponse(bytes));
    const candidate = candidateJSON();

    const blob = await downloadLocalGeometryCandidate(candidate as never);

    expect([...new Uint8Array(await blob.arrayBuffer())]).toEqual([...bytes]);
    expect(mocks.getFile).toHaveBeenCalledWith("/library/intake-candidates/candidate-1/revisions/1/content");
  });

  it("rejects content whose size or digest differs from the immutable candidate metadata", async () => {
    const candidate = candidateJSON();
    mocks.getFile.mockResolvedValueOnce(byteResponse(new TextEncoder().encode("ab")));
    await expect(downloadLocalGeometryCandidate(candidate as never)).rejects.toThrow(/size differs/);

    mocks.getFile.mockResolvedValueOnce(byteResponse(new TextEncoder().encode("xyz")));
    await expect(downloadLocalGeometryCandidate(candidate as never)).rejects.toThrow(/digest differs/);
  });
});
