/**
 * Self-tests for makeNotesMocks. Same shape as `test-utils/mocks/api.test.ts`.
 */
import { describe, expect, it, vi } from "vitest";

import { makeNotesMocks } from "./notes";

describe("makeNotesMocks", () => {
  it("returns a fresh surface on every call", () => {
    const a = makeNotesMocks();
    const b = makeNotesMocks();
    expect(a).not.toBe(b);
    expect(a.listNotes).not.toBe(b.listNotes);
    expect(a.createNote).not.toBe(b.createNote);
    expect(a.getNote).not.toBe(b.getNote);
  });

  it("listNotes default resolves to an empty list", async () => {
    const { listNotes } = makeNotesMocks();
    const r = await listNotes();
    expect(r.notes).toEqual([]);
  });

  it("createNote echoes the user's title and body into the returned Note", async () => {
    const { createNote } = makeNotesMocks();
    const got = await createNote({ title: "from user", body: "hello" });
    expect(got.title).toBe("from user");
    expect(got.body).toBe("hello");
    // Defaulted fields still come from the factory baseline.
    expect(got.id).not.toBe("");
  });

  it("createNote treats body as optional and defaults it to empty string", async () => {
    const { createNote } = makeNotesMocks();
    const got = await createNote({ title: "no body" });
    expect(got.body).toBe("");
  });

  it("getNote echoes the requested id into the returned Note", async () => {
    const { getNote } = makeNotesMocks();
    const got = await getNote("some-id");
    expect(got.id).toBe("some-id");
  });

  it("all surfaces are vi.fns so per-test overrides work", () => {
    const m = makeNotesMocks();
    expect(vi.isMockFunction(m.listNotes)).toBe(true);
    expect(vi.isMockFunction(m.createNote)).toBe(true);
    expect(vi.isMockFunction(m.getNote)).toBe(true);
  });
});
