/**
 * Reference-image upload rules shared by both studio surfaces.
 *
 * The server enforces the same type and size bound. Checking here turns a slow
 * 8 MiB round trip into an immediate message, and keeps the reason typed so the
 * page can render copy instead of parsing an error string.
 */

export const MAX_ASSET_BYTES = 8 * 1024 * 1024;

export class AssetImportError extends Error {
  reason: "type" | "size";

  constructor(reason: "type" | "size") {
    super(reason);
    this.name = "AssetImportError";
    this.reason = reason;
  }
}

/** Validate a file before reading it. Throws AssetImportError on a bad type or
 * an oversized upload; returns normally otherwise. */
export function assertImageUpload(file: File): void {
  if (!file.type.startsWith("image/")) throw new AssetImportError("type");
  if (file.size > MAX_ASSET_BYTES) throw new AssetImportError("size");
}

/**
 * The fields both upload stores share. A workspace asset and a legacy design
 * asset are structurally the same view; the pane only needs these.
 */
export interface ReferenceUpload {
  id: string;
  name: string;
  bytes: number;
  fidelity: "reference" | "captured";
  url: string;
}
