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
    expect(a.notesClient.list).not.toBe(b.notesClient.list);
    expect(a.notesClient.create).not.toBe(b.notesClient.create);
    expect(a.notesClient.get).not.toBe(b.notesClient.get);
    expect(a.uploadAttachment).not.toBe(b.uploadAttachment);
  });

  it("notesClient.list default resolves to an empty list", async () => {
    const { notesClient } = makeNotesMocks();
    const r = await notesClient.list({});
    expect(r.notes).toEqual([]);
  });

  it("notesClient.create echoes the user's title and body into the returned Note", async () => {
    const { notesClient } = makeNotesMocks();
    const got = await notesClient.create({ title: "from user", body: "hello" });
    expect(got.note?.title).toBe("from user");
    expect(got.note?.body).toBe("hello");
    // Defaulted fields still come from the factory baseline.
    expect(got.note?.id).not.toBe("");
  });

  it("notesClient.create treats body as optional and defaults it to empty string", async () => {
    const { notesClient } = makeNotesMocks();
    const got = await notesClient.create({ title: "no body" });
    expect(got.note?.body).toBe("");
  });

  it("notesClient.get echoes the requested id into the returned Note", async () => {
    const { notesClient } = makeNotesMocks();
    const got = await notesClient.get({ id: "some-id" });
    expect(got.note?.id).toBe("some-id");
  });

  it("all surfaces are vi.fns so per-test overrides work", () => {
    const m = makeNotesMocks();
    expect(vi.isMockFunction(m.notesClient.list)).toBe(true);
    expect(vi.isMockFunction(m.notesClient.create)).toBe(true);
    expect(vi.isMockFunction(m.notesClient.get)).toBe(true);
    expect(vi.isMockFunction(m.uploadAttachment)).toBe(true);
  });
});
