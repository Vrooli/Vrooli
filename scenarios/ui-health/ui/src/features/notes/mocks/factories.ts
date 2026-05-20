/**
 * Test data factories for the notes domain. Co-located with the feature
 * so deleting `features/notes/` takes the factories with it (no central
 * residue).
 *
 * Each `make<Domain>(overrides?)` returns a stable default instance that
 * tests selectively override via `MessageInitShape<Schema>`.
 *
 * # Wire shape lives in proto, not here
 *
 * The Note / ListNotesResponse types are GENERATED proto messages at
 * `packages/proto/gen/typescript/js/ui-health/v1/notes/...`.
 * Factories use `create(<Schema>, overrides)` so:
 *
 *   - the runtime instance includes proto's internal `$typeName` /
 *     reflection state (necessary for `toJson` / `fromJson` round-trips
 *     in tests that exercise the full pipeline);
 *   - field defaults match proto3 semantics (numbers default to 0,
 *     strings to "", maps to {} — never `undefined`);
 *   - adding a field to the proto schema makes it instantly available
 *     in factories without editing this file.
 */
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import {
  CreateNoteResponseSchema,
  ListNotesResponseSchema,
  NoteSchema,
  type CreateNoteResponse,
  type Note,
  type ListNotesResponse,
} from "@vrooli/proto-types/ui-health/v1/notes/notes_pb";
import {
  AttachmentSchema,
  type Attachment,
} from "@vrooli/proto-types/ui-health/v1/notes/attachments_pb";

export type { Attachment, CreateNoteResponse, Note, ListNotesResponse };

export const makeNote = (
  overrides: MessageInitShape<typeof NoteSchema> = {},
): Note =>
  create(NoteSchema, {
    id: "note-1",
    title: "First note",
    body: "Hello, world.",
    createdAt: timestampFromDate(new Date("2026-01-01T00:00:00.000Z")),
    updatedAt: timestampFromDate(new Date("2026-01-01T00:00:00.000Z")),
    attachmentKeys: [],
    ...overrides,
  });

export const makeListNotesResponse = (
  overrides: MessageInitShape<typeof ListNotesResponseSchema> = {},
): ListNotesResponse =>
  create(ListNotesResponseSchema, {
    notes: [],
    ...overrides,
  });

export const makeCreateNoteResponse = (
  overrides: MessageInitShape<typeof CreateNoteResponseSchema> = {},
): CreateNoteResponse =>
  create(CreateNoteResponseSchema, {
    note: makeNote(),
    ...overrides,
  });

export const makeAttachment = (
  overrides: MessageInitShape<typeof AttachmentSchema> = {},
): Attachment =>
  create(AttachmentSchema, {
    key: "notes/note-1/attachment.txt",
    mimeType: "text/plain",
    sizeBytes: 12n,
    noteId: "note-1",
    uploadedAt: timestampFromDate(new Date("2026-01-01T00:01:00.000Z")),
    ...overrides,
  });
