/**
 * Mock builders for `api/notes` — the UI ↔ API notes-CRUD boundary.
 * Co-located with the notes feature; deleting `features/notes/` takes
 * these mocks with it.
 *
 * See `test-utils/mocks/api.ts` for the full builder/hoisting rationale;
 * the same pattern applies. Canonical usage:
 *
 *   import { makeNotesMocks } from "./mocks/notes";
 *
 *   vi.mock("../../api/notes", async (importOriginal) => {
 *     const actual = await importOriginal<typeof import("../../api/notes")>();
 *     return { ...actual, ...makeNotesMocks() };
 *   });
 *
 * The `...actual` spread keeps the re-exported proto types intact —
 * only the network-touching functions are substituted. `ApiError`
 * itself lives in `api/client`; tests that need it import from there.
 *
 * Default behaviors:
 *
 *   - `listNotes` resolves to an empty list
 *   - `createNote({ title, body })` echoes the input back as a Note
 *     (so tests can assert "the title the user typed reaches the
 *     server" without arranging a per-test mockResolvedValue)
 *   - `getNote(id)` echoes the id back as a Note
 */
import { vi } from "vitest";

import { makeListNotesResponse, makeNote } from "./factories";

export interface NotesMockCreateInput {
  title: string;
  body?: string;
}

export interface NotesMocks {
  listNotes: ReturnType<typeof vi.fn>;
  createNote: ReturnType<typeof vi.fn>;
  getNote: ReturnType<typeof vi.fn>;
}

export const makeNotesMocks = (): NotesMocks => ({
  listNotes: vi.fn().mockResolvedValue(makeListNotesResponse()),
  createNote: vi
    .fn()
    .mockImplementation((input: NotesMockCreateInput) =>
      Promise.resolve(makeNote({ title: input.title, body: input.body ?? "" })),
    ),
  getNote: vi
    .fn()
    .mockImplementation((id: string) => Promise.resolve(makeNote({ id }))),
});
