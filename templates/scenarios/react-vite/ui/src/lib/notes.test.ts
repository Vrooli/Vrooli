/**
 * Unit tests for the lib/notes seam — the canonical CRUD client wrapper.
 *
 * Same stub-fetch pattern as lib/api.test.ts. What these tests pin:
 *
 *   - happy paths decode the proto-typed wire body into runtime types
 *     (Note, ListNotesResponse) via fromJson + the generated schema
 *   - non-2xx responses throw an ApiError carrying the proto-typed
 *     ErrorEnvelope code/message — UI surfaces the typed `code` to
 *     decide between "validation failed" and "not found" states
 *   - createNote serialises the proto request via toJsonString so the
 *     wire body always matches the schema (no hand-rolled JSON drift)
 *   - getNote URL-encodes the ID so notes with slashes / spaces don't
 *     silently route to the wrong endpoint
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError, createNote, getNote, listNotes } from "./notes";

describe("lib/notes", () => {
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
      const payload = {
        notes: [
          {
            id: "a",
            title: "first",
            body: "hello",
            created_at: "2026-01-01T00:00:00Z",
            updated_at: "2026-01-01T00:00:00Z",
          },
        ],
      };
      fetchSpy.mockResolvedValueOnce(new Response(JSON.stringify(payload), { status: 200 }));

      const got = await listNotes();

      expect(got.notes).toHaveLength(1);
      expect(got.notes[0]?.title).toBe("first");
    });

    it("returns an empty notes array when the wire body has none", async () => {
      fetchSpy.mockResolvedValueOnce(new Response('{"notes":[]}', { status: 200 }));
      const got = await listNotes();
      expect(got.notes).toEqual([]);
    });

    it("throws ApiError with the typed envelope on a non-2xx response", async () => {
      const envelope = { code: "internal", message: "store down" };
      fetchSpy.mockResolvedValueOnce(
        new Response(JSON.stringify(envelope), { status: 500 }),
      );

      await expect(listNotes()).rejects.toMatchObject({
        name: "ApiError",
        code: "internal",
        status: 500,
      });
    });
  });

  describe("createNote", () => {
    it("POSTs the proto-encoded request body and returns the created Note", async () => {
      const payload = {
        note: {
          id: "uuid",
          title: "first",
          body: "hello",
          created_at: "2026-01-01T00:00:00Z",
          updated_at: "2026-01-01T00:00:00Z",
        },
      };
      fetchSpy.mockResolvedValueOnce(new Response(JSON.stringify(payload), { status: 201 }));

      const got = await createNote({ title: "first", body: "hello" });

      expect(got.id).toBe("uuid");
      expect(got.title).toBe("first");

      const [url, init] = fetchSpy.mock.calls[0] as [string, RequestInit];
      expect(url).toMatch(/\/notes$/);
      expect(init.method).toBe("POST");
      // toJsonString returns a string; assert via type guard so the
      // lint rule against Object's default stringification stays clean.
      expect(typeof init.body).toBe("string");
      expect(JSON.parse(init.body as string)).toEqual({ title: "first", body: "hello" });
    });

    it("surfaces server-side validation failures as ApiError(invalid_request)", async () => {
      const envelope = { code: "invalid_request", message: "title required" };
      fetchSpy.mockResolvedValueOnce(
        new Response(JSON.stringify(envelope), { status: 400 }),
      );

      const err = await createNote({ title: "" }).catch((e: unknown) => e);
      expect(err).toBeInstanceOf(ApiError);
      expect((err as ApiError).code).toBe("invalid_request");
      expect((err as ApiError).message).toContain("title required");
    });
  });

  describe("getNote", () => {
    it("returns the decoded Note on the happy path", async () => {
      const payload = {
        note: {
          id: "abc",
          title: "found",
          body: "",
          created_at: "2026-01-01T00:00:00Z",
          updated_at: "2026-01-01T00:00:00Z",
        },
      };
      fetchSpy.mockResolvedValueOnce(new Response(JSON.stringify(payload), { status: 200 }));

      const got = await getNote("abc");
      expect(got.id).toBe("abc");
      expect(got.title).toBe("found");
    });

    it("surfaces 404s as ApiError(not_found)", async () => {
      const envelope = { code: "not_found", message: "note \"missing\" not found" };
      fetchSpy.mockResolvedValueOnce(
        new Response(JSON.stringify(envelope), { status: 404 }),
      );

      const err = await getNote("missing").catch((e: unknown) => e);
      expect(err).toBeInstanceOf(ApiError);
      expect((err as ApiError).code).toBe("not_found");
      expect((err as ApiError).status).toBe(404);
    });

    it("URL-encodes the id so route-confused inputs don't silently misroute", async () => {
      fetchSpy.mockResolvedValueOnce(
        new Response('{"note":{"id":"x/y","title":"t","body":"","created_at":"t","updated_at":"t"}}', { status: 200 }),
      );

      await getNote("x/y");
      const [url] = fetchSpy.mock.calls[0] as [string];
      expect(url).toContain("x%2Fy");
      expect(url).not.toContain("x/y");
    });
  });
});
