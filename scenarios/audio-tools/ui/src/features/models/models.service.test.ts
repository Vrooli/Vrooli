import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  CommercialUse,
  Readiness,
} from "@vrooli/proto-types/audio-tools/v1/models/models_pb";

const rpc = vi.hoisted(() => ({
  listModels: vi.fn(),
  resolveModel: vi.fn(),
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: () => rpc,
}));

import {
  isNotFound,
  listModels,
  primarySubtag,
  resolveModel,
} from "../../services/models";
import { makeApiError } from "../../api/client";
import { makeModelRecord } from "./mocks/factories";

beforeEach(() => {
  vi.clearAllMocks();
});

describe("primarySubtag", () => {
  it.each([
    ["en-US", "en"],
    ["EN", "en"],
    [" pt-BR ", "pt"],
    ["", ""],
  ])("normalises %s to %s", (input, expected) => {
    expect(primarySubtag(input)).toBe(expected);
  });
});

describe("listModels", () => {
  it("forwards every filter and maps enum facts to string unions", async () => {
    rpc.listModels.mockResolvedValue({ models: [makeModelRecord()] });

    const result = await listModels({
      capability: "stt",
      language: "en",
      commercialOnly: true,
      onlyReady: true,
    });

    expect(rpc.listModels.mock.calls[0]?.[0]).toEqual({
      capability: "stt",
      language: "en",
      commercialOnly: true,
      onlyReady: true,
    });
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.data).toHaveLength(1);
      expect(result.data[0]?.readiness).toBe("available");
      expect(result.data[0]?.commercialUse).toBe("allowed");
      expect(result.data[0]?.hardware.minRamGb).toBe(4);
      expect(result.data[0]?.benchmarks.wer).toBe(0.05);
    }
  });

  it("defaults unspecified filters to empty strings and false flags", async () => {
    rpc.listModels.mockResolvedValue({ models: [] });

    await listModels();

    expect(rpc.listModels.mock.calls[0]?.[0]).toEqual({
      capability: "",
      language: "",
      commercialOnly: false,
      onlyReady: false,
    });
  });

  it("maps unknown readiness and commercial states", async () => {
    rpc.listModels.mockResolvedValue({
      models: [
        makeModelRecord({
          readiness: Readiness.UNSPECIFIED,
          commercialUse: CommercialUse.UNKNOWN,
        }),
      ],
    });

    const result = await listModels();
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.data[0]?.readiness).toBe("unspecified");
      expect(result.data[0]?.commercialUse).toBe("unknown");
    }
  });

  it("returns an error Result when the RPC rejects", async () => {
    rpc.listModels.mockRejectedValue(
      Object.assign(new Error("registry down"), { code: "unavailable" }),
    );

    const result = await listModels();
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.error.code).toBe("unavailable");
  });
});

describe("resolveModel", () => {
  it("returns the resolved model and explanation", async () => {
    rpc.resolveModel.mockResolvedValue({
      model: makeModelRecord(),
      explanation: "selected: readiness=available",
    });

    const result = await resolveModel({
      capability: "stt",
      language: "en",
      commercial: true,
    });

    expect(rpc.resolveModel.mock.calls[0]?.[0]).toEqual({
      capability: "stt",
      language: "en",
      commercial: true,
    });
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.data.model.id).toBe("whisper-large-v3");
      expect(result.data.explanation).toBe(
        "selected: readiness=available",
      );
    }
  });

  it("reports NOT_FOUND when the response carries no model", async () => {
    rpc.resolveModel.mockResolvedValue({ explanation: "nothing eligible" });

    const result = await resolveModel({ capability: "speaker" });
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.error.code).toBe("not_found");
      expect(isNotFound(result.error)).toBe(true);
    }
  });
});

describe("isNotFound", () => {
  it("accepts the REST and Connect code shapes", () => {
    expect(isNotFound(makeApiError("not_found", "x"))).toBe(true);
    expect(isNotFound(makeApiError("5", "x"))).toBe(true);
  });

  it("rejects unrelated codes", () => {
    expect(isNotFound(makeApiError("unavailable", "x"))).toBe(false);
  });
});
