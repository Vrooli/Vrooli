import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  client: {
    listAIOperations: vi.fn(),
  },
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: vi.fn(() => mocks.client),
}));

import { fetchAIResult, listAIOperations, submitAI } from "./ai";
import { ApiError } from "./client";

/**
 * Controllable fake `Image`. The most recently constructed instance is captured
 * so a test can drive its `onload`/`onerror` after `src` is assigned (jsdom does
 * not actually decode object URLs), exercising both `imageSize` branches.
 */
class FakeImage {
  static last: FakeImage | null = null;
  onload: (() => void) | null = null;
  onerror: (() => void) | null = null;
  naturalWidth = 0;
  naturalHeight = 0;
  #src = "";
  constructor() {
    FakeImage.last = this;
  }
  set src(value: string) {
    this.#src = value;
  }
  get src(): string {
    return this.#src;
  }
}

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

describe("api/ai.fetchAIResult", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
    vi.stubGlobal("URL", { ...URL, createObjectURL: vi.fn(() => "blob:result") });
    FakeImage.last = null;
    vi.stubGlobal("Image", FakeImage);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  const blobResponse = (blob: Blob): Response =>
    ({
      ok: true,
      status: 200,
      headers: new Headers(),
      json: vi.fn(() => Promise.resolve({})),
      text: vi.fn(() => Promise.resolve("")),
      blob: vi.fn(() => Promise.resolve(blob)),
    }) as unknown as Response;

  // Flush microtasks until fetchAIResult has constructed the (fake) Image whose
  // onload/onerror the test drives, so we never fire the handler before src is set.
  const waitForImage = async (): Promise<FakeImage> => {
    for (let i = 0; i < 20 && !FakeImage.last; i += 1) {
      await Promise.resolve();
    }
    if (!FakeImage.last) {
      throw new Error("fetchAIResult never constructed an Image");
    }
    return FakeImage.last;
  };

  it("materializes the result blob with natural dimensions + a File (ext from ref)", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(blobResponse(new Blob(["x"], { type: "image/png" })));

    const promise = fetchAIResult("out/abc.PNG");
    // Drive the decode: imageSize resolves once the fake Image fires onload.
    const img = await waitForImage();
    img.naturalWidth = 320;
    img.naturalHeight = 240;
    img.onload?.();

    const out = await promise;

    expect(out.url).toBe("blob:result");
    expect(out.width).toBe(320);
    expect(out.height).toBe(240);
    // Extension is taken from the ref (lower-cased), not the mime type.
    expect(out.format).toBe("png");
    expect(out.outputFile).toBeInstanceOf(File);
    expect(out.outputFile.name).toBe("enhanced.png");
    expect(out.outputFile.type).toBe("image/png");
  });

  it("falls back to the mime subtype for the format when the ref has no usable extension", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(blobResponse(new Blob(["x"], { type: "image/webp" })));

    // A ref whose trailing segment is too long / not an extension forces the
    // mime-subtype branch of formatFromRef.
    const promise = fetchAIResult("out/longsegmentwithnoext");
    const img = await waitForImage();
    img.naturalWidth = 10;
    img.naturalHeight = 10;
    img.onload?.();

    const out = await promise;
    expect(out.format).toBe("webp");
    expect(out.outputFile.name).toBe("enhanced.webp");
  });

  it("yields 0x0 dimensions when the decode fails (onerror) and defaults the mime", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(blobResponse(new Blob(["x"], { type: "" })));

    const promise = fetchAIResult("out/broken");
    const img = await waitForImage();
    img.onerror?.();

    const out = await promise;
    expect(out.width).toBe(0);
    expect(out.height).toBe(0);
    // No ext and no mime subtype → formatFromRef defaults to "png".
    expect(out.format).toBe("png");
    expect(out.outputFile.type).toBe("image/png");
  });

  it("throws a typed ApiError when the blob fetch is non-2xx", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      makeResponse({ ok: false, status: 404, json: { code: "not_found", message: "gone" } }),
    );

    await expect(fetchAIResult("out/missing.png")).rejects.toBeInstanceOf(ApiError);
  });
});
