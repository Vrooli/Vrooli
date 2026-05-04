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
 *   - `notesClient.list` resolves to an empty list
 *   - `notesClient.create({ title, body })` echoes the input back as a Note
 *   - `notesClient.get({ id })` echoes the id back as a Note
 *   - `uploadAttachment` resolves to stable attachment metadata
 */
import { vi } from "vitest";

import { makeAttachment, makeCreateNoteResponse, makeListNotesResponse, makeNote } from "./factories";

export interface NotesMockCreateInput {
  title: string;
  body?: string;
}

export interface NotesMocks {
  notesClient: {
    list: ReturnType<typeof vi.fn>;
    create: ReturnType<typeof vi.fn>;
    get: ReturnType<typeof vi.fn>;
  };
  uploadAttachment: ReturnType<typeof vi.fn>;
}

export const makeNotesMocks = (): NotesMocks => ({
  notesClient: {
    list: vi.fn().mockResolvedValue(makeListNotesResponse()),
    create: vi
      .fn()
      .mockImplementation((input: NotesMockCreateInput) =>
        Promise.resolve(makeCreateNoteResponse({ note: makeNote({ title: input.title, body: input.body ?? "" }) })),
      ),
    get: vi
      .fn()
      .mockImplementation((input: { id: string }) => Promise.resolve({ note: makeNote({ id: input.id }) })),
  },
  uploadAttachment: vi.fn().mockResolvedValue(makeAttachment()),
});
