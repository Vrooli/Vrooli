import { Retention } from "../../api/transfer";

/**
 * One item staged in the Send panel before it's pushed. Files carry the
 * `File` handle (and a local object URL for an image preview); text snippets
 * carry their body. Each has its own retention + optional target so a batch can
 * mix policies. `id` is a client-only staging id (not a server id).
 */
export interface StagedFile {
  kind: "file";
  id: string;
  file: File;
  previewUrl: string | null;
  retention: Retention;
  targetDeviceId: string;
}

export interface StagedText {
  kind: "text";
  id: string;
  text: string;
  retention: Retention;
  targetDeviceId: string;
}

export type StagedItem = StagedFile | StagedText;

let counter = 0;
export function nextStagingId(prefix: string): string {
  counter += 1;
  return `${prefix}-${counter}`;
}

/** Build a staged file entry, deriving an image preview URL when applicable. */
export function stageFile(file: File): StagedFile {
  const previewUrl = file.type.startsWith("image/") ? URL.createObjectURL(file) : null;
  return {
    kind: "file",
    id: nextStagingId("file"),
    file,
    previewUrl,
    retention: Retention.HELD,
    targetDeviceId: "",
  };
}

export function stageText(text: string): StagedText {
  return {
    kind: "text",
    id: nextStagingId("text"),
    text,
    retention: Retention.HELD,
    targetDeviceId: "",
  };
}

export { Retention };
