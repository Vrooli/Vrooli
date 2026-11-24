import { beforeEach, describe, expect, it, vi } from "vitest";
import { CommercialUse, Readiness } from "@vrooli/proto-types/audio-tools/v1/models/models_pb";

const { listModelsRpc, resolveModelRpc } = vi.hoisted(() => ({
  listModelsRpc: vi.fn(),
  resolveModelRpc: vi.fn(),
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: () => ({ listModels: listModelsRpc, resolveModel: resolveModelRpc }),
}));

import { isNotFound, listModels, primarySubtag, resolveModel } from "./models";

beforeEach(() => vi.clearAllMocks());

describe("model registry service", () => {
  it("maps records and defaults absent hardware and benchmark fields", async () => {
    listModelsRpc.mockResolvedValue({ models: [{
      id: "whisper", name: "Whisper", providerIds: ["openai"], engineIds: ["whisper"],
      engineDefaults: ["whisper"], capability: "stt", runtime: "local", languages: ["en"],
      sizeMbApprox: 42, params: "small", license: "MIT", commercialUse: CommercialUse.ALLOWED,
      licenseNotes: "", sourceUrl: "https://example.test", sourceRepo: "repo", readiness: Readiness.AVAILABLE,
      roles: ["default"], defaultEligible: true, isDefault: true, allowDefault: true, notes: "verified",
    }] });
    const result = await listModels();
    expect(listModelsRpc).toHaveBeenCalledWith({ capability: "", language: "", commercialOnly: false, onlyReady: false });
    expect(result).toMatchObject({ ok: true, data: [{
      id: "whisper", commercialUse: "allowed", readiness: "available",
      hardware: { cpuCapable: false, gpuRequired: false, minVramGb: 0, minRamGb: 0, osArch: [] },
      benchmarks: { wer: 0, sim: 0, rtf: 0, notes: "" },
    }] });
  });

  it("passes filters through and covers all known and unknown enum values", async () => {
    listModelsRpc.mockResolvedValue({ models: [
      { readiness: Readiness.CANDIDATE, commercialUse: CommercialUse.DISALLOWED },
      { readiness: Readiness.EXPERIMENTAL, commercialUse: CommercialUse.UNKNOWN },
      { readiness: Readiness.WATCH, commercialUse: 99 },
      { readiness: 99, commercialUse: CommercialUse.UNSPECIFIED },
    ] });
    const result = await listModels({ capability: "tts", language: "fr", commercialOnly: true, onlyReady: true });
    expect(listModelsRpc).toHaveBeenCalledWith({ capability: "tts", language: "fr", commercialOnly: true, onlyReady: true });
    expect(result.ok && result.data.map(({ readiness, commercialUse }) => [readiness, commercialUse])).toEqual([
      ["candidate", "disallowed"], ["experimental", "unknown"], ["watch", "unspecified"], ["unspecified", "unspecified"],
    ]);
  });

  it("normalizes RPC failures into result errors", async () => {
    listModelsRpc.mockRejectedValue({ code: "unavailable", message: "offline" });
    const result = await listModels();
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.code).toBe("unavailable");
  });

  it("resolves with optional fields defaulted and reports an absent model as not found", async () => {
    resolveModelRpc.mockResolvedValue({ model: { id: "m", readiness: Readiness.AVAILABLE }, explanation: "best fit" });
    const result = await resolveModel({ capability: "stt" });
    expect(resolveModelRpc).toHaveBeenCalledWith({ capability: "stt", language: "", engineId: "", commercial: false });
    expect(result.ok && result.data).toMatchObject({ model: { id: "m" }, explanation: "best fit" });
    resolveModelRpc.mockResolvedValue({ explanation: "none" });
    const absent = await resolveModel({ capability: "stt", language: "en-US", engineId: "engine", commercial: true });
    expect(resolveModelRpc).toHaveBeenLastCalledWith({ capability: "stt", language: "en-US", engineId: "engine", commercial: true });
    expect(absent.ok).toBe(false);
    if (!absent.ok) expect(absent.error).toMatchObject({ code: "not_found", status: 404 });
  });

  it("normalizes resolve RPC failures", async () => {
    resolveModelRpc.mockRejectedValue({ code: "unavailable", message: "registry offline" });
    const result = await resolveModel({ capability: "tts" });
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.code).toBe("unavailable");
  });

  it("recognizes both not-found encodings and extracts a normalized language subtag", () => {
    expect(isNotFound({ code: "not_found" } as never)).toBe(true);
    expect(isNotFound({ code: "5" } as never)).toBe(true);
    expect(isNotFound({ code: "unavailable" } as never)).toBe(false);
    expect(primarySubtag("  EN-us ")).toBe("en");
    expect(primarySubtag(" ")).toBe("");
  });
});
