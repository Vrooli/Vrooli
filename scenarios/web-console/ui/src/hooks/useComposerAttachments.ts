import { useCallback, useEffect, useRef, useState } from "react";
import type { ComposerAttachment } from "../components/composer/AttachmentPreviewTray";
import { attachmentKind, attachmentKindForName, isBlockedAttachment } from "../lib/attachments";
import { basename } from "../lib/paths";

/** A file or folder picked on the host, attached by its absolute path. */
export interface HostAttachmentPick {
  path: string;
  isDirectory: boolean;
  sizeBytes?: number;
}

export interface UseComposerAttachmentsReturn {
  attachments: ComposerAttachment[];
  /**
   * Stage files locally (creates review previews; nothing uploads yet).
   * Returns the names of files rejected by policy (executables), so the caller
   * can surface why nothing appeared.
   */
  addFiles: (files: File[]) => string[];
  /**
   * Stage host files/folders by path. Nothing is read or uploaded: the path is
   * what gets sent. A path already staged is not added twice.
   */
  addHostPaths: (picks: HostAttachmentPick[]) => void;
  /** Remove one staged attachment and revoke its object URL. */
  removeFile: (id: string) => void;
  /** Remove every staged attachment (after a successful send / discard). */
  clearAll: () => void;
  /** Update a single attachment's lifecycle status (drives the upload spinner). */
  setStatus: (id: string, status: ComposerAttachment["status"]) => void;
}

let idCounter = 0;

/**
 * Resolve staged attachments to terminal paths, in attachment order. Uploads
 * are uploaded one at a time (showing a spinner); host attachments already are
 * their absolute path and never touch the network. On failure every spinner is
 * cleared and the error rethrown so the caller keeps the whole batch to retry.
 */
export async function resolveStagedAttachmentPaths(
  staged: readonly ComposerAttachment[],
  upload: (file: File) => Promise<string>,
  setStatus: (id: string, status: ComposerAttachment["status"]) => void,
): Promise<string[]> {
  const paths: string[] = [];
  try {
    for (const att of staged) {
      if (att.source === "host") {
        paths.push(att.path);
        continue;
      }
      setStatus(att.id, "uploading");
      paths.push(await upload(att.file));
      setStatus(att.id, "staged");
    }
  } catch (err) {
    for (const att of staged) setStatus(att.id, "staged");
    throw err;
  }
  return paths;
}

/**
 * useComposerAttachments — in-memory staging for composer attachments.
 *
 * Accepts any non-executable file: images, video, audio, PDF, archives, and
 * text/code. Files are held as object-URL previews and are NEVER uploaded or
 * injected until the operator sends. Host paths are staged as references and
 * never upload at all. Object URLs are revoked on remove/clear/
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
        source: "upload",
        name: file.name,
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

  const addHostPaths = useCallback((picks: HostAttachmentPick[]) => {
    setAttachments((prev) => {
      const present = new Set(prev.flatMap((a) => (a.source === "host" ? [a.path] : [])));
      const staged: ComposerAttachment[] = [];
      for (const pick of picks) {
        if (present.has(pick.path)) continue;
        present.add(pick.path);
        const name = basename(pick.path) || pick.path;
        staged.push({
          id: `catt-${String(++idCounter)}`,
          source: "host",
          name,
          path: pick.path,
          isDirectory: pick.isDirectory,
          kind: pick.isDirectory ? "file" : attachmentKindForName(name),
          sizeBytes: pick.sizeBytes ?? 0,
          status: "staged",
        });
      }
      return staged.length > 0 ? [...prev, ...staged] : prev;
    });
  }, []);

  const removeFile = useCallback((id: string) => {
    setAttachments((prev) => {
      const target = prev.find((a) => a.id === id);
      if (target?.source === "upload") {
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

  return { attachments, addFiles, addHostPaths, removeFile, clearAll, setStatus };
}
