import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError } from "./client";

const mocks = vi.hoisted(() => ({
  client: {
    list: vi.fn(),
    create: vi.fn(),
    get: vi.fn(),
  },
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: vi.fn(() => mocks.client),
}));

describe("api/notes", () => {
  let fetchSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchSpy = vi.fn();
    vi.stubGlobal("fetch", fetchSpy);
  });

  afterEach(() => {
    vi.clearAllMocks();
    vi.unstubAllGlobals();
  });

  it("exports the generated Connect client", async () => {
    const { notesClient } = await import("./notes");

    await notesClient.list({});

    expect(mocks.client.list).toHaveBeenCalledWith({});
  });

  describe("uploadAttachment", () => {
    it("posts FormData to the multipart REST endpoint and returns attachment metadata", async () => {
      const { uploadAttachment } = await import("./notes");
      const file = new File(["hello"], "hello.txt", { type: "text/plain" });
      fetchSpy.mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            attachment: {
              key: "notes/note-1/hello.txt",
              mime_type: "text/plain",
              size_bytes: "5",
              note_id: "note-1",
              uploaded_at: "2026-01-01T00:00:00Z",
            },
          }),
          { status: 200 },
        ),
      );

      const got = await uploadAttachment("note-1", file);

      expect(got.key).toBe("notes/note-1/hello.txt");
      expect(got.sizeBytes).toBe(5n);
      const [url, init] = fetchSpy.mock.calls[0] as [string, RequestInit];
      expect(url).toMatch(/\/notes\/note-1\/attachments$/);
      expect(init.body).toBeInstanceOf(FormData);
    });

    it("URL-encodes the note id so route-confused inputs do not misroute", async () => {
      const { uploadAttachment } = await import("./notes");
      fetchSpy.mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            attachment: {
              key: "notes/x-y/file.txt",
              mime_type: "text/plain",
              size_bytes: "1",
              note_id: "x/y",
              uploaded_at: "2026-01-01T00:00:00Z",
            },
          }),
          { status: 200 },
        ),
      );

      await uploadAttachment("x/y", new File(["x"], "x.txt"));

      const [url] = fetchSpy.mock.calls[0] as [string];
      expect(url).toContain("x%2Fy");
      expect(url).not.toContain("x/y");
    });

    it("surfaces multipart failures as ApiError", async () => {
      const { uploadAttachment } = await import("./notes");
      fetchSpy.mockResolvedValueOnce(
        new Response(JSON.stringify({ code: "not_found", message: "note missing" }), {
          status: 404,
        }),
      );

      const err = await uploadAttachment("missing", new File(["x"], "x.txt")).catch(
        (e: unknown) => e,
      );

      expect(err).toBeInstanceOf(ApiError);
      expect((err as ApiError).code).toBe("not_found");
    });
  });
});
