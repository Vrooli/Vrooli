import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError } from "./client";
import { createNote, getNote, listNotes } from "./notes";

describe("api/notes", () => {
  let fetchSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchSpy = vi.fn();
    vi.stubGlobal("fetch", fetchSpy);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  describe("listNotes", () => {
    it("returns ListNotesResponse decoded from the wire body", async () => {
      fetchSpy.mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            notes: [
              {
                id: "a",
                title: "first",
                body: "hello",
                created_at: "2026-01-01T00:00:00Z",
                updated_at: "2026-01-01T00:00:00Z",
              },
            ],
          }),
          { status: 200 },
        ),
      );

      const got = await listNotes();

      expect(got.notes).toHaveLength(1);
      expect(got.notes[0]?.title).toBe("first");
      expect(got.notes[0]?.createdAt).toBe("2026-01-01T00:00:00Z");
    });

    it("returns an empty notes array when the wire body has none", async () => {
      fetchSpy.mockResolvedValueOnce(new Response('{"notes":[]}', { status: 200 }));
      const got = await listNotes();
      expect(got.notes).toEqual([]);
    });
  });

  describe("createNote", () => {
    it("POSTs the proto-encoded request body and returns the created Note", async () => {
      fetchSpy.mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            note: {
              id: "uuid",
              title: "first",
              body: "hello",
              created_at: "2026-01-01T00:00:00Z",
              updated_at: "2026-01-01T00:00:00Z",
            },
          }),
          { status: 201 },
        ),
      );

      const got = await createNote({ title: "first", body: "hello" });

      expect(got.id).toBe("uuid");
      expect(got.title).toBe("first");

      const [url, init] = fetchSpy.mock.calls[0] as [string, RequestInit];
      expect(url).toMatch(/\/notes$/);
      expect(init.method).toBe("POST");
      expect(JSON.parse(init.body as string)).toEqual({ title: "first", body: "hello" });
    });

    it("surfaces server-side validation failures as ApiError(invalid_request)", async () => {
      fetchSpy.mockResolvedValueOnce(
        new Response(JSON.stringify({ code: "invalid_request", message: "title required" }), {
          status: 400,
        }),
      );

      const err = await createNote({ title: "" }).catch((e: unknown) => e);
      expect(err).toBeInstanceOf(ApiError);
      expect((err as ApiError).code).toBe("invalid_request");
      expect((err as ApiError).message).toContain("title required");
    });
  });

  describe("getNote", () => {
    it("returns the decoded Note on the happy path", async () => {
      fetchSpy.mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            note: {
              id: "abc",
              title: "found",
              body: "",
              created_at: "2026-01-01T00:00:00Z",
              updated_at: "2026-01-01T00:00:00Z",
            },
          }),
          { status: 200 },
        ),
      );

      const got = await getNote("abc");
      expect(got.id).toBe("abc");
      expect(got.title).toBe("found");
    });

    it("surfaces 404s as ApiError(not_found)", async () => {
      fetchSpy.mockResolvedValueOnce(
        new Response(JSON.stringify({ code: "not_found", message: "note missing not found" }), {
          status: 404,
        }),
      );

      const err = await getNote("missing").catch((e: unknown) => e);
      expect(err).toBeInstanceOf(ApiError);
      expect((err as ApiError).code).toBe("not_found");
      expect((err as ApiError).status).toBe(404);
    });

    it("URL-encodes the id so route-confused inputs don't silently misroute", async () => {
      fetchSpy.mockResolvedValueOnce(
        new Response(
          '{"note":{"id":"x/y","title":"t","body":"","created_at":"t","updated_at":"t"}}',
          { status: 200 },
        ),
      );

      await getNote("x/y");
      const [url] = fetchSpy.mock.calls[0] as [string];
      expect(url).toContain("x%2Fy");
      expect(url).not.toContain("x/y");
    });
  });
});

