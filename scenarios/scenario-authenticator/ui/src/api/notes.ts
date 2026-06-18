import { fromJson, type JsonValue } from "@bufbuild/protobuf";
import { createClient } from "@connectrpc/connect";
import {
  NotesService,
  type Note,
  type ListNotesResponse,
} from "@vrooli/proto-types/scenario-authenticator/v1/notes/notes_pb";
import {
  UploadAttachmentResponseSchema,
  type Attachment,
} from "@vrooli/proto-types/scenario-authenticator/v1/notes/attachments_pb";
import { TimeWindowToken } from "@vrooli/proto-types/measures/v1/measures_pb";

import {
  PROTO_READ_OPTIONS,
  decodeApiError,
  makeApiError,
  transport,
  uploadFile,
} from "./client";

export const notesClient = createClient(NotesService, transport);

/**
 * countNotesInWindow answers the `notes count` measure: how many notes were
 * created in a canonical time window. It builds the shared TimeWindow proto
 * (the `time_window` measure param) and returns the scalar count. This is the
 * UI half of the same measure search-hub can answer from natural language —
 * the reference for surfacing a measure result in a scenario UI.
 */
export async function countNotesInWindow(
  token: TimeWindowToken = TimeWindowToken.THIS_WEEK,
): Promise<number> {
  const resp = await notesClient.countNotes({
    window: { window: { case: "token", value: token } },
  });
  return Number(resp.count);
}

export async function uploadAttachment(noteId: string, file: File): Promise<Attachment> {
  const formData = new FormData();
  formData.set("file", file, file.name);

  const res = await uploadFile(`/notes/${encodeURIComponent(noteId)}/attachments`, formData);
  if (!res.ok) {
    throw await decodeApiError(res);
  }
  const decoded = fromJson(
    UploadAttachmentResponseSchema,
    (await res.json()) as JsonValue,
    PROTO_READ_OPTIONS,
  );
  if (!decoded.attachment) {
    throw makeApiError("internal", "upload returned no attachment");
  }
  return decoded.attachment;
}

export { TimeWindowToken };
export type { Note, ListNotesResponse };
