import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  client: {
    listDiffModes: vi.fn(),
  },
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: vi.fn(() => mocks.client),
}));

import { compare, DiffMode, listDiffModes } from "./diff";
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

const BASE = new File(["base"], "base.png", { type: "image/png" });
const COMPARE = new File(["compare"], "compare.png", { type: "image/png" });

describe("api/diff", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  it("compare posts multipart and parses the DiffResult (int64 as bigint)", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      makeResponse({
        json: {
          job_id: "j1",
          verdict: "different",
          dimensions_match: true,
          base_width: 100,
          base_height: 80,
          compare_width: 100,
          compare_height: 80,
          changed_pixels: "1234",
          total_pixels: "8000",
          changed_fraction: 0.154,
          mae: 12.5,
          rmse: 20.1,
          psnr: 28.3,
          phash_distance: 7,
          phash_similarity: 0.89,
          ssim: 0.74,
          heatmap_ref: "out/heat.png",
          warnings: ["images differ in size"],
        },
      }),
    );

    const out = await compare({ base: BASE, compare: COMPARE, mode: DiffMode.PERCEPTUAL, tolerance: 0.1 });

    expect(out.verdict).toBe("different");
    expect(out.changedPixels).toBe(1234n);
    expect(out.totalPixels).toBe(8000n);
    expect(out.heatmapRef).toBe("out/heat.png");
    expect(out.warnings).toHaveLength(1);
    // The multipart body carried both files + protojson params.
    const body = vi.mocked(fetch).mock.calls[0]?.[1]?.body as FormData;
    expect(body.get("base")).toBeInstanceOf(File);
    expect(body.get("compare")).toBeInstanceOf(File);
    expect(body.get("params")).toContain("DIFF_MODE_PERCEPTUAL");
  });

  it("compare defaults includeHeatmap to true", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(makeResponse({ json: { verdict: "identical" } }));
    await compare({ base: BASE, compare: COMPARE });
    const body = vi.mocked(fetch).mock.calls[0]?.[1]?.body as FormData;
    expect(body.get("params")).toContain("\"includeHeatmap\":true");
  });

  it("compare can disable the heat-map for a metrics-only run", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(makeResponse({ json: { verdict: "similar" } }));
    await compare({ base: BASE, compare: COMPARE, includeHeatmap: false, highlightHex: "#ff00ff" });
    const body = vi.mocked(fetch).mock.calls[0]?.[1]?.body as FormData;
    const params = body.get("params") as string;
    expect(params).not.toContain("\"includeHeatmap\":true");
    expect(params).toContain("#ff00ff");
  });

  it("compare surfaces a non-2xx as a typed ApiError", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      makeResponse({ ok: false, status: 422, json: { code: "invalid_request", message: "bad image" } }),
    );
    await expect(compare({ base: BASE, compare: COMPARE })).rejects.toBeInstanceOf(ApiError);
  });

  it("listDiffModes delegates to the Connect client", async () => {
    mocks.client.listDiffModes.mockResolvedValueOnce({
      modes: [
        { name: "pixel", summary: "exact" },
        { name: "perceptual", summary: "same-picture" },
      ],
    });
    const out = await listDiffModes();
    expect(out.modes).toHaveLength(2);
    expect(out.modes[0]?.name).toBe("pixel");
  });
});
