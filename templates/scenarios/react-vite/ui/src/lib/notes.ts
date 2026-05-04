import {
  CreateNoteRequestSchema,
  CreateNoteResponseSchema,
  GetNoteResponseSchema,
  ListNotesResponseSchema,
  type Note,
  type ListNotesResponse,
} from "@vrooli/proto-types/{{SCENARIO_ID}}/v1/notes/notes_pb";

import { makeApiError, protoFetch } from "./api";

/**
 * Fetch all notes (newest first). The wire shape is the
 * ListNotesResponse proto in `packages/proto/schemas/{{SCENARIO_ID}}/v1/notes/`.
 *
 * Test code mocks this function via `vi.mock("./lib/notes", ...)`. See
 * `ui/src/lib/notes.test.ts` for the canonical pattern.
 */
export async function listNotes(): Promise<ListNotesResponse> {
  return protoFetch("GET", "/notes", { responseSchema: ListNotesResponseSchema });
}

export interface CreateNoteInput {
  title: string;
  body?: string;
}

/**
 * Create a new note. Server-side validation rejects an empty title with
 * an ApiError carrying code="invalid_request"; surface that to the
 * caller so the UI can highlight the offending field.
 */
export async function createNote(input: CreateNoteInput): Promise<Note> {
  const decoded = await protoFetch("POST", "/notes", {
    requestSchema: CreateNoteRequestSchema,
    request: { title: input.title, body: input.body ?? "" },
    responseSchema: CreateNoteResponseSchema,
  });
  if (!decoded.note) {
    throw makeApiError("internal", "create returned no note");
  }
  return decoded.note;
}

/**
 * Fetch a single note by ID. A missing ID surfaces as an ApiError with
 * code="not_found".
 */
export async function getNote(id: string): Promise<Note> {
  const decoded = await protoFetch("GET", `/notes/${encodeURIComponent(id)}`, {
    responseSchema: GetNoteResponseSchema,
  });
  if (!decoded.note) {
    throw makeApiError("internal", "get returned no note");
  }
  return decoded.note;
}

export type { Note, ListNotesResponse };
