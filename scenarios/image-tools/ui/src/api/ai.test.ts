import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  client: {
    listAIOperations: vi.fn(),
  },
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: vi.fn(() => mocks.client),
}));

import { listAIOperations, submitAI } from "./ai";
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

describe("api/ai", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  it("submitAI posts a multipart request and parses the submit response", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      makeResponse({
        json: {
          job_id: "job-9",
          estimated_seconds: 12,
          model_id: "rembg",
          tier: "local-cpu",
          warnings: ["slow on CPU"],
        },
      }),
    );

    const out = await submitAI("background_removal", {}, { image: PNG });

    expect(out).toEqual({
      jobId: "job-9",
      estimatedSeconds: 12,
      modelId: "rembg",
      tier: "local-cpu",
      warnings: ["slow on CPU"],
    });

    const [url, init] = vi.mocked(fetch).mock.calls[0] ?? [];
    expect(typeof url === "string" ? url : "").toContain("/ai/background_removal");
    const body = init?.body as FormData;
    expect(body.get("file")).toBeInstanceOf(File);
    expect(typeof body.get("params")).toBe("string");
  });

  it("submitAI serializes typed params (upscale scale) into the params part", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      makeResponse({ json: { job_id: "j", model_id: "m", tier: "local-cpu" } }),
    );

    await submitAI("upscale", { scale: 4 }, { image: PNG });

    const body = vi.mocked(fetch).mock.calls[0]?.[1]?.body as FormData;
    const params = body.get("params");
    const paramsText = typeof params === "string" ? params : "";
    expect(paramsText).toContain("\"scale\"");
    expect(paramsText).toContain("4");
  });

  it("submitAI throws a typed ApiError on a non-2xx response (e.g. model not installed)", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      makeResponse({
        ok: false,
        status: 409,
        json: { code: "failed_precondition", message: "model not installed" },
      }),
    );

    await expect(submitAI("upscale", { scale: 2 }, { image: PNG })).rejects.toBeInstanceOf(ApiError);
  });

  it("listAIOperations delegates to the discovery client", async () => {
    mocks.client.listAIOperations.mockResolvedValueOnce({ operations: [] });

    await listAIOperations();

    expect(mocks.client.listAIOperations).toHaveBeenCalledWith({});
  });
});
