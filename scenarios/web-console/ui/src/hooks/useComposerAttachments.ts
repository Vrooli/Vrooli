import { useCallback, useEffect, useRef, useState } from "react";
import type { ComposerAttachment } from "../components/composer/AttachmentPreviewTray";
import { attachmentKind, isBlockedAttachment } from "../lib/attachments";

export interface UseComposerAttachmentsReturn {
  attachments: ComposerAttachment[];
  /**
   * Stage files locally (creates review previews; nothing uploads yet).
   * Returns the names of files rejected by policy (executables), so the caller
   * can surface why nothing appeared.
   */
  addFiles: (files: File[]) => string[];
  /** Remove one staged attachment and revoke its object URL. */
  removeFile: (id: string) => void;
  /** Remove every staged attachment (after a successful send / discard). */
  clearAll: () => void;
  /** Update a single attachment's lifecycle status (drives the upload spinner). */
  setStatus: (id: string, status: ComposerAttachment["status"]) => void;
}

let idCounter = 0;

/**
 * useComposerAttachments — in-memory staging for composer attachments.
 *
 * Accepts any non-executable file: images, video, audio, PDF, archives, and
 * text/code. Files are held as object-URL previews and are NEVER uploaded or
 * injected until the operator sends. Object URLs are revoked on remove/clear/
 * unmount so nothing leaks.
 */
export function useComposerAttachments(): UseComposerAttachmentsReturn {
  const [attachments, setAttachments] = useState<ComposerAttachment[]>([]);
  const urlsRef = useRef<Set<string>>(new Set());

  const addFiles = useCallback((files: File[]): string[] => {
    const staged: ComposerAttachment[] = [];
    const rejected: string[] = [];
    for (const file of files) {
      if (isBlockedAttachment(file)) {
        rejected.push(file.name);
        continue;
      }
      const previewUrl = URL.createObjectURL(file);
      urlsRef.current.add(previewUrl);
      staged.push({
        id: `catt-${String(++idCounter)}`,
        file,
        previewUrl,
        kind: attachmentKind(file),
        sizeBytes: file.size,
        status: "staged",
      });
    }
    if (staged.length > 0) setAttachments((prev) => [...prev, ...staged]);
    return rejected;
  }, []);

  const removeFile = useCallback((id: string) => {
    setAttachments((prev) => {
      const target = prev.find((a) => a.id === id);
      if (target) {
        try {
          URL.revokeObjectURL(target.previewUrl);
        } catch {
          /* ignore */
        }
        urlsRef.current.delete(target.previewUrl);
      }
      return prev.filter((a) => a.id !== id);
    });
  }, []);

  const clearAll = useCallback(() => {
    for (const url of urlsRef.current) {
      try {
        URL.revokeObjectURL(url);
      } catch {
        /* ignore */
      }
    }
    urlsRef.current.clear();
    setAttachments([]);
  }, []);

  const setStatus = useCallback((id: string, status: ComposerAttachment["status"]) => {
    setAttachments((prev) => prev.map((a) => (a.id === id ? { ...a, status } : a)));
  }, []);

  // Revoke any outstanding object URLs on unmount. `urls` is the stable Set from
  // the ref (created once), captured here so the cleanup uses the same instance.
  useEffect(() => {
    const urls = urlsRef.current;
    return () => {
      for (const url of urls) {
        try {
          URL.revokeObjectURL(url);
        } catch {
          /* ignore */
        }
      }
      urls.clear();
    };
  }, []);

  return { attachments, addFiles, removeFile, clearAll, setStatus };
}
