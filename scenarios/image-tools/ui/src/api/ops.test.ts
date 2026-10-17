import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  client: {
    listOperations: vi.fn(),
  },
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: vi.fn(() => mocks.client),
}));

const makeResponse = (init: {
  ok?: boolean;
  contentType?: string;
  body?: Blob | string;
  headers?: Record<string, string>;
  status?: number;
}): Response => {
  const headers = new Headers(init.headers ?? {});
  if (init.contentType) {
    headers.set("Content-Type", init.contentType);
  }
  return {
    ok: init.ok ?? true,
    status: init.status ?? 200,
    headers,
    blob: vi.fn(() => Promise.resolve(init.body instanceof Blob ? init.body : new Blob([]))),
    text: vi.fn(() => Promise.resolve(typeof init.body === "string" ? init.body : "")),
    json: vi.fn(() => Promise.resolve({})),
  } as unknown as Response;
};

describe("api/ops", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
    vi.stubGlobal("URL", {
      ...URL,
      createObjectURL: vi.fn(() => "blob:result"),
    });
  });

  afterEach(() => {
    vi.clearAllMocks();
    vi.unstubAllGlobals();
  });

  it("exports the generated Connect client and forwards listOperations", async () => {
    const { listOperations } = await import("./ops");

    await listOperations();

    expect(mocks.client.listOperations).toHaveBeenCalledWith({});
  });

  it("posts multipart with the operation-keyed params and returns an object URL for image results", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      makeResponse({
        contentType: "image/png",
        body: new Blob(["bytes"]),
        headers: {
          "X-Image-Tools-Width": "256",
          "X-Image-Tools-Height": "128",
          "X-Image-Tools-Format": "png",
          "X-Image-Tools-Job-Id": "job-9",
        },
      }),
    );

    const { runOp } = await import("./ops");
    const file = new File(["x"], "in.png", { type: "image/png" });
    const result = await runOp("resize", file, { width: 256, fit: "fit" });

    expect(result).toEqual({
      kind: "image",
      url: "blob:result",
      width: 256,
      height: 128,
      format: "png",
      jobId: "job-9",
    });

    const [url, requestInit] = vi.mocked(fetch).mock.calls[0] ?? [];
    expect(url as string).toContain("/ops/resize?output=bytes");
    const body = requestInit?.body as FormData;
    expect(body.get("params")).toBe(JSON.stringify({ resize: { width: 256, fit: "fit" } }));
    expect(body.get("file")).toBe(file);
  });

  it("attaches the overlay part when provided", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      makeResponse({ contentType: "image/png", body: new Blob(["bytes"]) }),
    );

    const { runOp } = await import("./ops");
    const file = new File(["x"], "in.png", { type: "image/png" });
    const overlay = new File(["wm"], "mark.png", { type: "image/png" });
    await runOp("overlay", file, { text: "hi" }, { overlay });

    const body = vi.mocked(fetch).mock.calls[0]?.[1]?.body as FormData;
    expect(body.get("overlay")).toBe(overlay);
  });

  it("returns the raw JSON for a metadata read", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      makeResponse({ contentType: "application/json", body: '{"width":10}' }),
    );

    const { runOp } = await import("./ops");
    const file = new File(["x"], "in.png", { type: "image/png" });
    const result = await runOp("metadata", file, {});

    expect(result).toEqual({ kind: "metadata", json: '{"width":10}' });
  });

  it("treats a missing Content-Type as an image result and defaults non-numeric dimensions to 0", async () => {
    // No Content-Type header at all (exercises the `?? ""` fallback so the
    // application/json check is false → the image branch), and a non-numeric
    // width header (exercises toNumber's NaN → 0 guard).
    vi.mocked(fetch).mockResolvedValueOnce(
      makeResponse({
        body: new Blob(["bytes"]),
        headers: { "X-Image-Tools-Width": "not-a-number" },
      }),
    );

    const { runOp } = await import("./ops");
    const file = new File(["x"], "in.png", { type: "image/png" });
    const result = await runOp("resize", file, { width: 10 });

    expect(result).toEqual({
      kind: "image",
      url: "blob:result",
      width: 0,
      height: 0,
      format: "",
      jobId: "",
    });
  });

  it("throws a decoded ApiError on a non-2xx response", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      makeResponse({ ok: false, status: 400, contentType: "application/json" }),
    );

    const { runOp } = await import("./ops");
    const { ApiError } = await import("./client");
    const file = new File(["x"], "in.png", { type: "image/png" });

    await expect(runOp("resize", file, { width: 10 })).rejects.toBeInstanceOf(ApiError);
  });
});
