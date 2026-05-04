/**
 * Test data factories for the notes domain. Co-located with the feature
 * so deleting `features/notes/` takes the factories with it (no central
 * residue per Pass 3).
 *
 * Each `make<Domain>(overrides?)` returns a stable default instance that
 * tests selectively override via `MessageInitShape<Schema>`.
 *
 * # Wire shape lives in proto, not here
 *
 * The Note / ListNotesResponse types are GENERATED proto messages at
 * `packages/proto/gen/typescript/{{SCENARIO_ID}}/v1/notes/...`.
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
import {
  ListNotesResponseSchema,
  NoteSchema,
  type Note,
  type ListNotesResponse,
} from "@vrooli/proto-types/{{SCENARIO_ID}}/v1/notes/notes_pb";

export type { Note, ListNotesResponse };

export const makeNote = (
  overrides: MessageInitShape<typeof NoteSchema> = {},
): Note =>
  create(NoteSchema, {
    id: "note-1",
    title: "First note",
    body: "Hello, world.",
    createdAt: "2026-01-01T00:00:00.000Z",
    updatedAt: "2026-01-01T00:00:00.000Z",
    ...overrides,
  });

export const makeListNotesResponse = (
  overrides: MessageInitShape<typeof ListNotesResponseSchema> = {},
): ListNotesResponse =>
  create(ListNotesResponseSchema, {
    notes: [],
    ...overrides,
  });
