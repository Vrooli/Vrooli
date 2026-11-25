import { ARTIFACT_VIEWER_MIN_WIDTH, ARTIFACT_WORKSPACE_MIN_WIDTH, type PreviewSourceContext } from "@vrooli/file-viewer";
import { create } from "zustand";
import { persist } from "zustand/middleware";

export interface FilePreviewRequest {
  sessionId: string;
  path: string;
  source: PreviewSourceContext;
}

export const FILE_PREVIEW_DEFAULT_WIDTH = 440;
export const FILE_PREVIEW_MAX_WIDTH = 900;

export function clampFilePreviewWidth(width: number): number {
  if (!Number.isFinite(width)) return FILE_PREVIEW_DEFAULT_WIDTH;
  return Math.min(FILE_PREVIEW_MAX_WIDTH, Math.max(ARTIFACT_VIEWER_MIN_WIDTH, Math.round(width)));
}

/**
 * A persisted width belongs to the old viewport. On a narrower screen the same
 * pixels would starve the terminal, so cap the dock at the width that still
 * leaves the workspace its own minimum. The stored value is untouched: a
 * desktop load after a phone session gets its chosen width back.
 */
export function fitFilePreviewWidth(width: number, viewportWidth: number): number {
  const available = Math.max(ARTIFACT_VIEWER_MIN_WIDTH, viewportWidth - ARTIFACT_WORKSPACE_MIN_WIDTH);
  return Math.min(clampFilePreviewWidth(width), available);
}

interface FilePreviewState {
  request: FilePreviewRequest | null;
  width: number;
  setRequest: (request: FilePreviewRequest) => void;
  clear: () => void;
  setWidth: (width: number) => void;
}

function isRequest(value: unknown): value is FilePreviewRequest {
  if (!value || typeof value !== "object") return false;
  const candidate = value as Partial<FilePreviewRequest>;
  return typeof candidate.sessionId === "string" && candidate.sessionId.length > 0
    && typeof candidate.path === "string" && candidate.path.length > 0
    && (candidate.source === "message_link" || candidate.source === "inline_code" || candidate.source === "cli");
}

/**
 * The open file preview is part of the workspace, not a transient popover:
 * reloading the console must bring back the same file at the same width.
 * Focus mode is deliberately not persisted — it is a momentary reading state,
 * and restoring it would hide the terminal on the next load.
 */
export const useFilePreviewStore = create<FilePreviewState>()(
  persist(
    (set) => ({
      request: null,
      width: FILE_PREVIEW_DEFAULT_WIDTH,
      setRequest: (request) => { set({ request }); },
      clear: () => { set({ request: null }); },
      setWidth: (width) => { set({ width: clampFilePreviewWidth(width) }); },
    }),
    {
      name: "web-console.file-preview.v1",
      version: 1,
      partialize: (state) => ({ request: state.request, width: state.width }),
      merge: (persisted, current) => {
        const stored = persisted as Partial<Pick<FilePreviewState, "request" | "width">> | undefined;
        return {
          ...current,
          request: isRequest(stored?.request) ? stored.request : null,
          width: clampFilePreviewWidth(stored?.width ?? current.width),
        };
      },
    },
  ),
);
