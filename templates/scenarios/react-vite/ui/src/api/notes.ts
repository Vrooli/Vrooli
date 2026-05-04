import {
  CreateNoteRequestSchema,
  CreateNoteResponseSchema,
  GetNoteResponseSchema,
  ListNotesResponseSchema,
  type Note,
  type ListNotesResponse,
} from "@vrooli/proto-types/{{SCENARIO_ID}}/v1/notes/notes_pb";

import { makeApiError, protoFetch } from "./client";

export async function listNotes(): Promise<ListNotesResponse> {
  return protoFetch("GET", "/notes", { responseSchema: ListNotesResponseSchema });
}

export interface CreateNoteInput {
  title: string;
  body?: string;
}

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

