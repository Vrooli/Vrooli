import { afterEach, describe, expect, it, vi } from "vitest";
import { proseApi } from "./prose";

describe("prose API", () => {
  afterEach(() => vi.restoreAllMocks());

  it("sends each operator operation through the typed REST seam", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockImplementation(async () => new Response(JSON.stringify({ ok: true }), { status: 200 }));
    await proseApi.generate("profile", "subject");
    await proseApi.reroll("session");
    await proseApi.createDocument("title", "profile");
    await proseApi.assemble("document");
    await proseApi.registry();
    expect(fetchMock).toHaveBeenCalledTimes(5);
    expect(fetchMock.mock.calls[0]?.[1]).toMatchObject({ method: "POST" });
  });

  it("surfaces structured API failures", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response(JSON.stringify({ code: "bad_request", message: "invalid" }), { status: 400 }));
    await expect(proseApi.registry()).rejects.toThrow("bad_request: invalid");
  });
});
