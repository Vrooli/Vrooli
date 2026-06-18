import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  client: {
    listAnalysisOperations: vi.fn(),
  },
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: vi.fn(() => mocks.client),
}));

import { analyze, listAnalysisOperations } from "./analysis";
import { ApiError } from "./client";

const makeResponse = (init: { ok?: boolean; status?: number; json?: unknown }): Response =>
  ({
    ok: init.ok ?? true,
    status: init.status ?? 200,
    headers: new Headers(),
    json: vi.fn(() => Promise.resolve(init.json ?? {})),
    text: vi.fn(() => Promise.resolve("")),
    blob: vi.fn(() => Promise.resolve(new Blob([]))),
  }) as unknown as Response;

const PNG = new File(["bytes"], "in.png", { type: "image/png" });

describe("api/analysis", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  it("analyze('probe') posts multipart and normalizes the ProbeResult (int64 → number)", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      makeResponse({
        json: {
          job_id: "j1",
          probe: {
            width: 640,
            height: 480,
            format: "png",
            color_model: "rgba",
            has_alpha: true,
            megapixels: 0.31,
            size_bytes: "12345",
            dominant_colors: [{ hex: "#abcdef", fraction: 0.5 }],
          },
        },
      }),
    );

    const out = await analyze("probe", PNG);

    expect(out.kind).toBe("probe");
    if (out.kind === "probe") {
      expect(out.width).toBe(640);
      expect(out.height).toBe(480);
      expect(out.sizeBytes).toBe(12_345);
      expect(out.dominantColors).toHaveLength(1);
    }

    const [url, init] = vi.mocked(fetch).mock.calls[0] ?? [];
    expect(typeof url === "string" ? url : "").toContain("/analysis/probe");
    const body = init?.body as FormData;
    expect(body.get("file")).toBeInstanceOf(File);
  });

  it("analyze('ocr') normalizes blocks with pixel boxes", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      makeResponse({
        json: {
          job_id: "j2",
          ocr: {
            full_text: "hi",
            language: "eng",
            blocks: [{ text: "hi", confidence: 0.9, box: { x: 1, y: 2, width: 3, height: 4 } }],
          },
        },
      }),
    );

    const out = await analyze("ocr", PNG);

    expect(out.kind).toBe("ocr");
    if (out.kind === "ocr") {
      expect(out.fullText).toBe("hi");
      expect(out.blocks[0]?.box).toEqual({ x: 1, y: 2, width: 3, height: 4 });
    }
  });

  it("analyze('nsfw_classify') normalizes the verdict and categories", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      makeResponse({
        json: {
          job_id: "j3",
          nsfw: {
            nsfw: true,
            score: 0.8,
            label: "nsfw",
            threshold: 0.5,
            categories: [{ label: "nsfw", score: 0.8 }],
          },
        },
      }),
    );

    const out = await analyze("nsfw_classify", PNG);

    expect(out.kind).toBe("nsfw");
    if (out.kind === "nsfw") {
      expect(out.flagged).toBe(true);
      expect(out.score).toBeCloseTo(0.8);
      expect(out.categories).toHaveLength(1);
    }
  });

  it("analyze('ocr') yields a null box when a block carries no bounding box", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      makeResponse({
        json: {
          job_id: "j2b",
          ocr: {
            full_text: "hi",
            language: "eng",
            blocks: [{ text: "hi", confidence: 0.9 }],
          },
        },
      }),
    );

    const out = await analyze("ocr", PNG);

    expect(out.kind).toBe("ocr");
    if (out.kind === "ocr") {
      expect(out.blocks[0]?.box).toBeNull();
    }
  });

  it("analyze throws an internal ApiError when the response carries no result oneof", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(makeResponse({ json: { job_id: "j-empty" } }));

    await expect(analyze("ocr", PNG)).rejects.toMatchObject({
      name: "ApiError",
      code: "internal",
    });
  });

  it("analyze throws a typed ApiError on a non-2xx response (e.g. model not installed)", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      makeResponse({
        ok: false,
        status: 409,
        json: { code: "failed_precondition", message: "model not installed" },
      }),
    );

    await expect(analyze("ocr", PNG)).rejects.toBeInstanceOf(ApiError);
  });

  it("listAnalysisOperations delegates to the discovery client", async () => {
    mocks.client.listAnalysisOperations.mockResolvedValueOnce({ operations: [] });

    await listAnalysisOperations();

    expect(mocks.client.listAnalysisOperations).toHaveBeenCalledWith({});
  });
});
