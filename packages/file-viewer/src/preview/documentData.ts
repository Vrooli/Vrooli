export const MAX_DOCUMENT_BYTES = 15 * 1024 * 1024;
export const MAX_DOCUMENT_HTML = 2 * 1024 * 1024;
export const MAX_DOCUMENT_IMAGES = 8 * 1024 * 1024;
export interface DocumentData { html: string; warnings: boolean; omittedImages: number }
export type DocumentError = "invalid" | "encrypted" | "limit" | "unavailable" | "fetch";
export type DocumentResult = { data: DocumentData; error?: never } | { data?: never; error: DocumentError };
