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
    expect(a.notesClient.listNotes).not.toBe(b.notesClient.listNotes);
    expect(a.notesClient.createNote).not.toBe(b.notesClient.createNote);
    expect(a.notesClient.getNote).not.toBe(b.notesClient.getNote);
    expect(a.uploadAttachment).not.toBe(b.uploadAttachment);
  });

  it("notesClient.listNotes default resolves to an empty list", async () => {
    const { notesClient } = makeNotesMocks();
    const r = await notesClient.listNotes({});
    expect(r.notes).toEqual([]);
  });

  it("notesClient.createNote echoes the user's title and body into the returned Note", async () => {
    const { notesClient } = makeNotesMocks();
    const got = await notesClient.createNote({ title: "from user", body: "hello" });
    expect(got.note?.title).toBe("from user");
    expect(got.note?.body).toBe("hello");
    // Defaulted fields still come from the factory baseline.
    expect(got.note?.id).not.toBe("");
  });

  it("notesClient.createNote treats body as optional and defaults it to empty string", async () => {
    const { notesClient } = makeNotesMocks();
    const got = await notesClient.createNote({ title: "no body" });
    expect(got.note?.body).toBe("");
  });

  it("notesClient.getNote echoes the requested id into the returned Note", async () => {
    const { notesClient } = makeNotesMocks();
    const got = await notesClient.getNote({ id: "some-id" });
    expect(got.note?.id).toBe("some-id");
  });

  it("all surfaces are vi.fns so per-test overrides work", () => {
    const m = makeNotesMocks();
    expect(vi.isMockFunction(m.notesClient.listNotes)).toBe(true);
    expect(vi.isMockFunction(m.notesClient.createNote)).toBe(true);
    expect(vi.isMockFunction(m.notesClient.getNote)).toBe(true);
    expect(vi.isMockFunction(m.uploadAttachment)).toBe(true);
  });
});
