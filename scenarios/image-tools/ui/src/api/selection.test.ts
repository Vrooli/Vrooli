import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  client: {
    listRegionClasses: vi.fn(),
    suggestEdits: vi.fn(),
  },
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: vi.fn(() => mocks.client),
}));

import { listRegionClasses, segment, SegmentMode, suggestEdits } from "./selection";
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

describe("api/selection", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  it("segment posts multipart and parses the SegmentResult", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      makeResponse({
        json: {
          job_id: "j1",
          mask_ref: "mask/x.png",
          box: { x: 0.3, y: 0.3, width: 0.4, height: 0.4 },
          region_class: "object",
          confidence: 0.7,
          area_fraction: 0.16,
          tier: "builtin-cpu",
          suggested_edits: [
            { id: "remove", label: "Remove", operation: "object_removal", requires_mask: true },
          ],
        },
      }),
    );

    const out = await segment({ image: PNG, mode: SegmentMode.POINT, points: [{ x: 0.5, y: 0.5 }] });

    expect(out.maskRef).toBe("mask/x.png");
    expect(out.regionClass).toBe("object");
    expect(out.suggestedEdits).toHaveLength(1);
    // The multipart body carried the file + protojson params.
    const body = vi.mocked(fetch).mock.calls[0]?.[1]?.body as FormData;
    expect(body.get("file")).toBeInstanceOf(File);
    expect(body.get("params")).toContain("SEGMENT_MODE_POINT");
  });

  it("segment surfaces a non-2xx as a typed ApiError", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      makeResponse({ ok: false, status: 422, json: { code: "invalid_request", message: "bad seed" } }),
    );
    await expect(segment({ image: PNG, mode: SegmentMode.POINT })).rejects.toBeInstanceOf(ApiError);
  });

  it("listRegionClasses delegates to the Connect client", async () => {
    mocks.client.listRegionClasses.mockResolvedValueOnce({ classes: [{ name: "sky", summary: "", edits: [] }] });
    const out = await listRegionClasses();
    expect(out.classes[0]?.name).toBe("sky");
  });

  it("suggestEdits returns the resolved class + edits", async () => {
    mocks.client.suggestEdits.mockResolvedValueOnce({ regionClass: "object", edits: [{ id: "remove" }] });
    const out = await suggestEdits("bogus");
    expect(out.regionClass).toBe("object");
    expect(out.edits).toHaveLength(1);
  });
});
