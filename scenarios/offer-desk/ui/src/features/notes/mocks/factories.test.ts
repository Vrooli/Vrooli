/**
 * Self-tests for the notes-domain proto-typed test factories. Co-located
 * with the feature so deleting `features/notes/` takes them along.
 *
 * Same contract as the central `test-utils/factories.test.ts`:
 *
 *   - sane defaults make the most common test path `makeX()` no-args
 *   - overrides merge field-level (no all-or-nothing replacement)
 *   - the returned instance round-trips through proto's
 *     `toJson` / `fromJson` byte-identically — i.e., it includes the
 *     internal `$typeName`/reflection state proto runtime needs
 */
import { fromJson, toJson } from "@bufbuild/protobuf";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { describe, expect, it } from "vitest";

import {
  ListNotesResponseSchema,
  NoteSchema,
} from "@vrooli/proto-types/offer-desk/v1/notes/notes_pb";
import { AttachmentSchema } from "@vrooli/proto-types/offer-desk/v1/notes/attachments_pb";

import { makeAttachment, makeListNotesResponse, makeNote } from "./factories";

describe("makeNote", () => {
  it("returns a note with non-empty id/title and RFC3339 timestamps", () => {
    const n = makeNote();
    expect(n.id).not.toBe("");
    expect(n.title).not.toBe("");
    expect(n.createdAt).toBeDefined();
    expect(n.updatedAt).toBeDefined();
    if (!n.createdAt || !n.updatedAt) {
      throw new Error("factory did not populate note timestamps");
    }
    expect(Number.isNaN(timestampDate(n.createdAt).getTime())).toBe(false);
    expect(Number.isNaN(timestampDate(n.updatedAt).getTime())).toBe(false);
  });

  it("merges overrides without dropping defaults", () => {
    const n = makeNote({ id: "custom-1", title: "Custom" });
    expect(n.id).toBe("custom-1");
    expect(n.title).toBe("Custom");
    expect(n.createdAt).toBeDefined();
  });

  it("round-trips through NoteSchema JSON encode + decode", () => {
    const original = makeNote({ id: "rt-1", title: "round trip" });
    const decoded = fromJson(NoteSchema, toJson(NoteSchema, original));
    expect(decoded.id).toBe("rt-1");
    expect(decoded.title).toBe("round trip");
    expect(decoded.createdAt).toEqual(original.createdAt);
  });
});

describe("makeListNotesResponse", () => {
  it("defaults to an empty notes array (proto3: maps/arrays default to [], not undefined)", () => {
    const r = makeListNotesResponse();
    expect(r.notes).toEqual([]);
  });

  it("accepts an overrides.notes array of factory-built Notes", () => {
    const r = makeListNotesResponse({
      notes: [makeNote({ id: "a" }), makeNote({ id: "b" })],
    });
    expect(r.notes).toHaveLength(2);
    expect(r.notes[0]?.id).toBe("a");
    expect(r.notes[1]?.id).toBe("b");
  });

  it("round-trips through ListNotesResponseSchema with embedded notes", () => {
    const original = makeListNotesResponse({
      notes: [makeNote({ id: "rt-list-1", title: "embedded" })],
    });
    const decoded = fromJson(
      ListNotesResponseSchema,
      toJson(ListNotesResponseSchema, original),
    );
    expect(decoded.notes).toHaveLength(1);
    expect(decoded.notes[0]?.id).toBe("rt-list-1");
    expect(decoded.notes[0]?.title).toBe("embedded");
  });
});

describe("makeAttachment", () => {
  it("returns attachment metadata with stable defaults", () => {
    const attachment = makeAttachment();
    expect(attachment.key).not.toBe("");
    expect(attachment.noteId).toBe("note-1");
    expect(attachment.sizeBytes).toBeGreaterThan(0n);
    expect(attachment.uploadedAt).toBeDefined();
  });

  it("round-trips through AttachmentSchema JSON encode + decode", () => {
    const original = makeAttachment({ key: "custom.bin", sizeBytes: 99n });
    const decoded = fromJson(AttachmentSchema, toJson(AttachmentSchema, original));
    expect(decoded.key).toBe("custom.bin");
    expect(decoded.sizeBytes).toBe(99n);
  });
});
