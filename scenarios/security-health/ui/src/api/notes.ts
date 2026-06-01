import { fromJson, type JsonValue } from "@bufbuild/protobuf";
import { createClient } from "@connectrpc/connect";
import {
  NotesService,
  type Note,
  type ListNotesResponse,
} from "@vrooli/proto-types/security-health/v1/notes/notes_pb";
import {
  UploadAttachmentResponseSchema,
  type Attachment,
} from "@vrooli/proto-types/security-health/v1/notes/attachments_pb";

import {
  PROTO_READ_OPTIONS,
  decodeApiError,
  makeApiError,
  transport,
  uploadFile,
} from "./client";

export const notesClient = createClient(NotesService, transport);

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

export type { Note, ListNotesResponse };
