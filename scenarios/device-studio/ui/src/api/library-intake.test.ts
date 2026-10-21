import { webcrypto } from "node:crypto";
import { Buffer } from "node:buffer";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  listIntakeCandidates: vi.fn(),
  uploadFile: vi.fn(),
  getFile: vi.fn(),
  decodeApiError: vi.fn(async (response: Response) => new Error(`HTTP ${response.status}`)),
  fileURL: vi.fn((path: string) => `http://device-studio.test/api/v1${path}`),
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: () => ({ listIntakeCandidates: mocks.listIntakeCandidates }),
}));
vi.mock("./client", () => ({
  decodeApiError: mocks.decodeApiError,
  fileURL: mocks.fileURL,
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
  correctLocalGeometryMaterial,
  getLocalGeometryMaterials,
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

  it("binds material inventory to the exact candidate revision and digest", async () => {
    const candidate = candidateJSON();
    mocks.getFile.mockResolvedValue(new Response(JSON.stringify({ candidate_id: "candidate-1", revision: 1, revision_sha256: "revision-hash", glb_sha256: candidate.originalSha256, materials: [] }), { status: 200 }));
    await getLocalGeometryMaterials(candidate as never);
    expect(mocks.getFile).toHaveBeenCalledWith("/library/intake-candidates/candidate-1/revisions/1/materials?revision_sha256=revision-hash");
    mocks.getFile.mockResolvedValue(new Response(JSON.stringify({ candidate_id: "candidate-1", revision: 1, revision_sha256: "different", glb_sha256: candidate.originalSha256, materials: [] }), { status: 200 }));
    await expect(getLocalGeometryMaterials(candidate as never)).rejects.toThrow(/exact imported candidate revision/);
  });

  it("posts one exact local correction request and refuses a promoted or mismatched child", async () => {
    const request = { parent_candidate_id: "candidate-1", parent_revision: 1, parent_revision_sha256: "revision-hash", child_candidate_id: "child-1", material_index: 0, material_name: "shell", roughness_factor: 0.2 };
    const result = { candidate: { id: "child-1", revision: 1, revision_sha256: "child-revision", original_sha256: "child-glb", review_status: "unreviewed", render_status: "not_reviewed", parent_candidate_id: "candidate-1", parent_revision: 1, parent_sha256: "revision-hash", import_method: "local_material_correction" } };
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify(result), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    const corrected = await correctLocalGeometryMaterial(request);
    expect(corrected.id).toBe("child-1");
    expect(fetchMock).toHaveBeenCalledWith("http://device-studio.test/api/v1/library/intake-candidates/candidate-1/revisions/1/material-corrections", expect.objectContaining({ method: "POST", body: JSON.stringify(request) }));
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({ candidate: { ...result.candidate, parent_sha256: "other" } }), { status: 200 }));
    await expect(correctLocalGeometryMaterial(request)).rejects.toThrow(/expected unreviewed child candidate/);
  });
});
