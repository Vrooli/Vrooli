import { useCallback, useEffect, useRef, useState } from "react";
import type { ComposerAttachment } from "../components/composer/AttachmentPreviewTray";

const DEFAULT_ALLOWED_TYPES = new Set(["image/jpeg", "image/png", "image/gif", "image/webp"]);

export interface UseComposerAttachmentsReturn {
  attachments: ComposerAttachment[];
  /** Stage image files locally (creates review thumbnails; nothing uploads yet). */
  addFiles: (files: File[]) => void;
  /** Remove one staged attachment and revoke its object URL. */
  removeFile: (id: string) => void;
  /** Remove every staged attachment (after a successful send / discard). */
  clearAll: () => void;
  /** Update a single attachment's lifecycle status (drives the upload spinner). */
  setStatus: (id: string, status: ComposerAttachment["status"]) => void;
}

let idCounter = 0;

/**
 * useComposerAttachments — in-memory staging for composer image attachments.
 *
 * Modeled on swarm-manager's attachment composer, but web-console targets a raw
 * terminal: files are held as object-URL thumbnails and are NEVER uploaded or
 * injected until the operator sends. Object URLs are revoked on remove/clear/
 * unmount so nothing leaks.
 */
export function useComposerAttachments(
  allowedTypes: Set<string> = DEFAULT_ALLOWED_TYPES,
): UseComposerAttachmentsReturn {
  const [attachments, setAttachments] = useState<ComposerAttachment[]>([]);
  const urlsRef = useRef<Set<string>>(new Set());

  const addFiles = useCallback(
    (files: File[]) => {
      const staged: ComposerAttachment[] = [];
      for (const file of files) {
        if (!allowedTypes.has(file.type)) continue;
        const previewUrl = URL.createObjectURL(file);
        urlsRef.current.add(previewUrl);
        staged.push({ id: `catt-${++idCounter}`, file, previewUrl, status: "staged" });
      }
      if (staged.length > 0) setAttachments((prev) => [...prev, ...staged]);
    },
    [allowedTypes],
  );

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
