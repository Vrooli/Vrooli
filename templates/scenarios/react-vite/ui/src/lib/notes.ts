import { create, fromJson, toJsonString, type JsonValue } from "@bufbuild/protobuf";
import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

import {
  CreateNoteRequestSchema,
  CreateNoteResponseSchema,
  GetNoteResponseSchema,
  ListNotesResponseSchema,
  type Note,
  type ListNotesResponse,
} from "@vrooli/proto-types/{{SCENARIO_ID}}/v1/notes/notes_pb";
import {
  ErrorEnvelopeSchema,
  type ErrorEnvelope,
} from "@vrooli/proto-types/{{SCENARIO_ID}}/v1/errors/errors_pb";

const API_BASE = resolveApiBase({ appendSuffix: true });

/**
 * Typed error thrown when the API returns a non-2xx response. The
 * server-side error envelope (proto: ErrorEnvelope) round-trips through
 * here so callers see a structured `code` + `message` rather than a
 * raw HTTP status.
 *
 * Extends Error so it remains throwable; the `code` field is the
 * load-bearing signal for callers branching on failure mode (e.g.,
 * "not_found" vs "invalid_request").
 */
export class ApiError extends Error {
  readonly code: string;
  readonly status: number;

  constructor(envelope: ErrorEnvelope, status: number) {
    super(`${envelope.code}: ${envelope.message}`);
    this.name = "ApiError";
    this.code = envelope.code;
    this.status = status;
  }
}

/**
 * Build an ApiError from a non-2xx response. Returns the error rather
 * than throwing it so callers express the throw at the call site
 * (`throw await decodeApiError(res)`) — a `Promise<never>` returner
 * reads as if the call site is doing nothing, when in fact every
 * caller relies on the thrown error to short-circuit decoding.
 */
async function decodeApiError(res: Response): Promise<ApiError> {
  let envelope: ErrorEnvelope;
  try {
    const json = (await res.json()) as JsonValue;
    envelope = fromJson(ErrorEnvelopeSchema, json, { ignoreUnknownFields: true });
  } catch {
    envelope = create(ErrorEnvelopeSchema, {
      code: "internal",
      message: `unexpected ${res.status} response (no envelope)`,
    });
  }
  return new ApiError(envelope, res.status);
}

/**
 * Fetch all notes (newest first). The wire shape is the
 * ListNotesResponse proto in `packages/proto/schemas/{{SCENARIO_ID}}/v1/notes/`.
 *
 * Test code mocks this function via `vi.mock("./lib/notes", ...)`. See
 * `ui/src/lib/notes.test.ts` for the canonical pattern.
 */
export async function listNotes(): Promise<ListNotesResponse> {
  const url = buildApiUrl("/notes", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    throw await decodeApiError(res);
  }
  const json = (await res.json()) as JsonValue;
  return fromJson(ListNotesResponseSchema, json, { ignoreUnknownFields: true });
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
  const url = buildApiUrl("/notes", { baseUrl: API_BASE });
  const reqMsg = create(CreateNoteRequestSchema, {
    title: input.title,
    body: input.body ?? "",
  });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: toJsonString(CreateNoteRequestSchema, reqMsg),
  });
  if (!res.ok) {
    throw await decodeApiError(res);
  }
  const json = (await res.json()) as JsonValue;
  const decoded = fromJson(CreateNoteResponseSchema, json, { ignoreUnknownFields: true });
  if (!decoded.note) {
    throw new ApiError(
      create(ErrorEnvelopeSchema, { code: "internal", message: "create returned no note" }),
      500,
    );
  }
  return decoded.note;
}

/**
 * Fetch a single note by ID. A missing ID surfaces as an ApiError with
 * code="not_found".
 */
export async function getNote(id: string): Promise<Note> {
  const url = buildApiUrl(`/notes/${encodeURIComponent(id)}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    throw await decodeApiError(res);
  }
  const json = (await res.json()) as JsonValue;
  const decoded = fromJson(GetNoteResponseSchema, json, { ignoreUnknownFields: true });
  if (!decoded.note) {
    throw new ApiError(
      create(ErrorEnvelopeSchema, { code: "internal", message: "get returned no note" }),
      500,
    );
  }
  return decoded.note;
}

export type { Note, ListNotesResponse, ErrorEnvelope };
