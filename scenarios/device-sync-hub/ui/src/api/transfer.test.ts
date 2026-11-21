import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { seedSession } from "../test-utils";
import { Retention } from "@vrooli/proto-types/device-sync-hub/v1/transfer/transfer_pb";

const { authedFetch } = vi.hoisted(() => ({ authedFetch: vi.fn() }));

vi.mock("./client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("./client")>();
  return { ...actual, authedFetch };
});

import { downloadItem, fetchItemBlob, RETENTION_FORM_VALUE, uploadItem } from "./transfer";

/**
 * Minimal scriptable XHR double. Tests drive the lifecycle by calling the
 * `on*` handlers the production code wires up, exactly as the browser would.
 */
class FakeXhr {
  static last: FakeXhr | null = null;
  method = "";
  url = "";
  responseType = "";
  responseText = "";
  status = 200;
  headers: Record<string, string> = {};
  sentBody: unknown = null;
  upload: { onprogress: ((e: ProgressEvent) => void) | null } = { onprogress: null };
  onload: (() => void) | null = null;
  onerror: (() => void) | null = null;
  onabort: (() => void) | null = null;
  open(method: string, url: string) {
    this.method = method;
    this.url = url;
  }
  setRequestHeader(name: string, value: string) {
    this.headers[name] = value;
  }
  send(body: unknown) {
    this.sentBody = body;
    FakeXhr.last = this;
  }
  abort() {
    this.onabort?.();
  }
}

describe("uploadItem", () => {
  beforeEach(() => {
    seedSession({ deviceToken: "tok-123" });
    vi.stubGlobal("XMLHttpRequest", FakeXhr);
  });
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.clearAllMocks();
    FakeXhr.last = null;
  });

  it("posts the file with the device-token header and resolves the decoded item", async () => {
    const file = new File(["bytes"], "doc.txt", { type: "text/plain" });
    const onProgress = vi.fn();
    const promise = uploadItem(file, { retention: Retention.PINNED, targetDeviceId: "d2", onProgress });

    const xhr = FakeXhr.last!;
    expect(xhr.method).toBe("POST");
    expect(xhr.url).toContain("/transfer/items");
    expect(xhr.headers["X-Device-Token"]).toBe("tok-123");
    const form = xhr.sentBody as FormData;
    expect(form.get("name")).toBe("doc.txt");
    expect(form.get("retention")).toBe(RETENTION_FORM_VALUE[Retention.PINNED]);
    expect(form.get("target_device_id")).toBe("d2");

    // Progress events forward a 0..1 fraction.
    xhr.upload.onprogress?.({ lengthComputable: true, loaded: 5, total: 10 } as ProgressEvent);
    expect(onProgress).toHaveBeenCalledWith(0.5);

    xhr.status = 201;
    xhr.responseText = JSON.stringify({ item: { id: "item-1", name: "doc.txt" } });
    xhr.onload?.();

    const item = await promise;
    expect(item.id).toBe("item-1");
  });

  it("rejects when the success response carries no item", async () => {
    const promise = uploadItem(new File(["x"], "a.txt"));
    const xhr = FakeXhr.last!;
    xhr.status = 200;
    xhr.responseText = JSON.stringify({});
    xhr.onload?.();
    await expect(promise).rejects.toMatchObject({ code: "internal" });
  });

  it("rejects on a malformed success body", async () => {
    const promise = uploadItem(new File(["x"], "a.txt"));
    const xhr = FakeXhr.last!;
    xhr.status = 200;
    xhr.responseText = "not json";
    xhr.onload?.();
    await expect(promise).rejects.toMatchObject({ code: "internal" });
  });

  it("decodes a structured error body on a non-2xx status", async () => {
    const promise = uploadItem(new File(["x"], "a.txt"));
    const xhr = FakeXhr.last!;
    xhr.status = 403;
    xhr.responseText = JSON.stringify({ code: "permission_denied", message: "nope" });
    xhr.onload?.();
    await expect(promise).rejects.toMatchObject({ code: "permission_denied" });
  });

  it("rejects on a network error", async () => {
    const promise = uploadItem(new File(["x"], "a.txt"));
    FakeXhr.last!.onerror?.();
    await expect(promise).rejects.toMatchObject({ code: "unavailable" });
  });

  it("rejects immediately when the signal is already aborted", async () => {
    const controller = new AbortController();
    controller.abort();
    await expect(
      uploadItem(new File(["x"], "a.txt"), { signal: controller.signal }),
    ).rejects.toMatchObject({ code: "canceled" });
  });

  it("aborts the in-flight xhr when the signal fires", async () => {
    const controller = new AbortController();
    const promise = uploadItem(new File(["x"], "a.txt"), { signal: controller.signal });
    controller.abort();
    await expect(promise).rejects.toMatchObject({ code: "canceled" });
  });

  it("falls back to 'held' for the unspecified retention wire value", () => {
    expect(RETENTION_FORM_VALUE[Retention.HELD]).toBe("held");
    void uploadItem(new File(["x"], "a.txt"), { retention: Retention.UNSPECIFIED });
    const form = FakeXhr.last!.sentBody as FormData;
    expect(form.get("retention")).toBe("held");
  });
});

describe("fetchItemBlob", () => {
  afterEach(() => vi.clearAllMocks());

  it("fetches the token-authed content and returns the blob", async () => {
    authedFetch.mockResolvedValue(new Response("data", { status: 200 }));
    const result = await fetchItemBlob("item-1", { thumb: true });
    expect(authedFetch.mock.calls[0]?.[0]).toContain("/transfer/items/item-1/content?thumb=1");
    expect(await result.text()).toBe("data");
  });

  it("throws a decoded api error on a non-ok response", async () => {
    authedFetch.mockResolvedValue(
      new Response(JSON.stringify({ code: "not_found", message: "gone" }), { status: 404 }),
    );
    await expect(fetchItemBlob("missing")).rejects.toMatchObject({ code: "not_found" });
  });
});

describe("downloadItem", () => {
  afterEach(() => vi.restoreAllMocks());

  it("synthesises an anchor click under the original filename then revokes the url", async () => {
    authedFetch.mockResolvedValue(new Response("data", { status: 200 }));
    const createObjectURL = vi.fn().mockReturnValue("blob:fake");
    const revokeObjectURL = vi.fn();
    vi.stubGlobal("URL", { ...URL, createObjectURL, revokeObjectURL });
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});

    await downloadItem("item-1", "my file.txt");

    expect(createObjectURL).toHaveBeenCalled();
    expect(clickSpy).toHaveBeenCalled();
    expect(revokeObjectURL).toHaveBeenCalledWith("blob:fake");
    vi.unstubAllGlobals();
  });
});
