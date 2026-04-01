/**
 * useAttachments — Persistent image attachment management for clarification input.
 *
 * Files are held in local state with generated previews. Attachments are
 * persisted to IndexedDB so they survive accidental page refreshes.
 * IndexedDB is used instead of localStorage because image data URLs
 * easily exceed localStorage's ~5MB quota.
 *
 * Follows the same persistence pattern as useCaptureAttachments but uses
 * a separate database so capture and clarification drafts don't collide.
 */

import { useState, useCallback, useEffect, useRef } from "react";
import type { CaptureAttachment } from "./useCaptureAttachments";

export type { CaptureAttachment } from "./useCaptureAttachments";

export interface UseAttachmentsReturn {
  attachments: CaptureAttachment[];
  addFile: (file: File) => void;
  removeFile: (id: string) => void;
  clearAll: () => void;
  getFiles: () => File[];
}

const ALLOWED_TYPES = new Set(["image/jpeg", "image/png", "image/gif", "image/webp"]);
const DB_NAME = "swarm-clarification-attachments";
const STORE_NAME = "attachments";
const PERSIST_DEBOUNCE_MS = 300;

// ---------------------------------------------------------------------------
// IndexedDB helpers
// ---------------------------------------------------------------------------

interface PersistedAttachment {
  id: string;
  fileName: string;
  contentType: string;
  dataUrl: string;
}

function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, 1);
    req.onupgradeneeded = () => {
      const db = req.result;
      if (!db.objectStoreNames.contains(STORE_NAME)) {
        db.createObjectStore(STORE_NAME, { keyPath: "id" });
      }
    };
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  });
}

async function loadFromDB(): Promise<CaptureAttachment[]> {
  try {
    const db = await openDB();
    return new Promise((resolve) => {
      const tx = db.transaction(STORE_NAME, "readonly");
      const store = tx.objectStore(STORE_NAME);
      const req = store.getAll();
      req.onsuccess = () => {
        const records = req.result as PersistedAttachment[];
        resolve(
          records.map((p) => ({
            id: p.id,
            file: dataUrlToFile(p.dataUrl, p.fileName, p.contentType),
            previewUrl: p.dataUrl,
          })),
        );
      };
      req.onerror = () => resolve([]);
    });
  } catch {
    return [];
  }
}

async function saveToDB(attachments: CaptureAttachment[]): Promise<void> {
  try {
    const db = await openDB();
    const tx = db.transaction(STORE_NAME, "readwrite");
    const store = tx.objectStore(STORE_NAME);
    store.clear();
    for (const att of attachments) {
      store.put({
        id: att.id,
        fileName: att.file.name,
        contentType: att.file.type,
        dataUrl: att.previewUrl,
      } satisfies PersistedAttachment);
    }
  } catch {
    // IndexedDB unavailable — silently ignore.
  }
}

async function clearDB(): Promise<void> {
  try {
    const db = await openDB();
    const tx = db.transaction(STORE_NAME, "readwrite");
    tx.objectStore(STORE_NAME).clear();
  } catch {
    // ignore
  }
}

// ---------------------------------------------------------------------------
// Utilities
// ---------------------------------------------------------------------------

let idCounter = 0;

function dataUrlToFile(dataUrl: string, fileName: string, contentType: string): File {
  const base64 = dataUrl.split(",")[1] ?? "";
  const bytes = atob(base64);
  const arr = new Uint8Array(bytes.length);
  for (let i = 0; i < bytes.length; i++) {
    arr[i] = bytes.charCodeAt(i);
  }
  return new File([arr], fileName, { type: contentType });
}

// ---------------------------------------------------------------------------
// Hook
// ---------------------------------------------------------------------------

export function useAttachments(): UseAttachmentsReturn {
  const [attachments, setAttachments] = useState<CaptureAttachment[]>([]);
  const [loaded, setLoaded] = useState(false);
  const persistTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const attachmentsRef = useRef(attachments);
  attachmentsRef.current = attachments;

  // Load persisted attachments from IndexedDB on mount.
  useEffect(() => {
    loadFromDB().then((restored) => {
      if (restored.length > 0) {
        setAttachments(restored);
      }
      setLoaded(true);
    });
  }, []);

  // Debounced persist to IndexedDB on attachment changes (skip before initial load).
  useEffect(() => {
    if (!loaded) return;

    if (persistTimerRef.current !== null) clearTimeout(persistTimerRef.current);
    persistTimerRef.current = setTimeout(() => {
      saveToDB(attachments);
    }, PERSIST_DEBOUNCE_MS);
    return () => {
      if (persistTimerRef.current !== null) clearTimeout(persistTimerRef.current);
    };
  }, [attachments, loaded]);

  // Flush pending persist immediately on page unload.
  useEffect(() => {
    const handleBeforeUnload = () => {
      if (persistTimerRef.current !== null) {
        clearTimeout(persistTimerRef.current);
        persistTimerRef.current = null;
      }
      saveToDB(attachmentsRef.current);
    };
    window.addEventListener("beforeunload", handleBeforeUnload);
    return () => window.removeEventListener("beforeunload", handleBeforeUnload);
  }, []);

  const addFile = useCallback((file: File) => {
    if (!ALLOWED_TYPES.has(file.type)) return;

    const id = `att-${++idCounter}-${Date.now()}`;
    const reader = new FileReader();
    reader.onload = (e) => {
      const previewUrl = e.target?.result as string;
      setAttachments((prev) => [...prev, { id, file, previewUrl }]);
    };
    reader.readAsDataURL(file);
  }, []);

  const removeFile = useCallback((id: string) => {
    setAttachments((prev) => prev.filter((att) => att.id !== id));
  }, []);

  const clearAll = useCallback(() => {
    setAttachments([]);
    clearDB();
  }, []);

  const getFiles = useCallback((): File[] => {
    return attachments.map((att) => att.file);
  }, [attachments]);

  return { attachments, addFile, removeFile, clearAll, getFiles };
}
