/**
 * useIndexedDBAttachments — Parameterized image attachment hook with IndexedDB persistence.
 *
 * Files are held in local state with generated previews. Attachments are
 * persisted to IndexedDB so they survive accidental page refreshes.
 * IndexedDB is used instead of localStorage because image data URLs
 * easily exceed localStorage's ~5MB quota.
 */

import { useState, useCallback, useEffect, useRef } from "react";

export interface CaptureAttachment {
  id: string;
  file: File;
  previewUrl: string;
}

export interface UseIndexedDBAttachmentsReturn {
  attachments: CaptureAttachment[];
  addFile: (file: File) => void;
  removeFile: (id: string) => void;
  clearAll: () => void;
  getFiles: () => File[];
}

export interface UseIndexedDBAttachmentsOptions {
  dbName: string;
  allowedTypes?: Set<string>;
  persistDebounceMs?: number;
}

const DEFAULT_ALLOWED_TYPES = new Set(["image/jpeg", "image/png", "image/gif", "image/webp"]);
const DEFAULT_PERSIST_DEBOUNCE_MS = 300;
const STORE_NAME = "attachments";

// ---------------------------------------------------------------------------
// IndexedDB helpers
// ---------------------------------------------------------------------------

/** Serializable record stored in IndexedDB. */
interface PersistedAttachment {
  id: string;
  fileName: string;
  contentType: string;
  dataUrl: string;
}

function openDB(dbName: string): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(dbName, 1);
    req.onupgradeneeded = () => {
      const db = req.result;
      if (!db.objectStoreNames.contains(STORE_NAME)) {
        db.createObjectStore(STORE_NAME, { keyPath: "id" });
      }
    };
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error ?? new Error(`Failed to open IndexedDB ${dbName}`));
  });
}

async function loadFromDB(dbName: string): Promise<CaptureAttachment[]> {
  try {
    const db = await openDB(dbName);
    return await new Promise((resolve) => {
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

async function saveToDB(dbName: string, attachments: CaptureAttachment[]): Promise<void> {
  try {
    const db = await openDB(dbName);
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

async function clearDB(dbName: string): Promise<void> {
  try {
    const db = await openDB(dbName);
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

/** Convert a data URL back into a File object. */
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

export function useIndexedDBAttachments(options: UseIndexedDBAttachmentsOptions): UseIndexedDBAttachmentsReturn {
  const { dbName, allowedTypes = DEFAULT_ALLOWED_TYPES, persistDebounceMs = DEFAULT_PERSIST_DEBOUNCE_MS } = options;

  const [attachments, setAttachments] = useState<CaptureAttachment[]>([]);
  const [loaded, setLoaded] = useState(false);
  const persistTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const attachmentsRef = useRef(attachments);
  attachmentsRef.current = attachments;

  // Load persisted attachments from IndexedDB on mount.
  useEffect(() => {
    void loadFromDB(dbName).then((restored) => {
      if (restored.length > 0) {
        setAttachments(restored);
      }
      setLoaded(true);
    });
  }, [dbName]);

  // Debounced persist to IndexedDB on attachment changes (skip before initial load).
  useEffect(() => {
    if (!loaded) return;

    if (persistTimerRef.current !== null) clearTimeout(persistTimerRef.current);
    persistTimerRef.current = setTimeout(() => {
      void saveToDB(dbName, attachments);
    }, persistDebounceMs);
    return () => {
      if (persistTimerRef.current !== null) clearTimeout(persistTimerRef.current);
    };
  }, [attachments, loaded, dbName, persistDebounceMs]);

  // Flush pending persist immediately on page unload so nothing is lost.
  useEffect(() => {
    const handleBeforeUnload = () => {
      if (persistTimerRef.current !== null) {
        clearTimeout(persistTimerRef.current);
        persistTimerRef.current = null;
      }
      // Fire-and-forget — we can't await in beforeunload, but IDB transactions
      // started synchronously before the page tears down usually complete.
      void saveToDB(dbName, attachmentsRef.current);
    };
    window.addEventListener("beforeunload", handleBeforeUnload);
    return () => window.removeEventListener("beforeunload", handleBeforeUnload);
  }, [dbName]);

  const addFile = useCallback((file: File) => {
    if (!allowedTypes.has(file.type)) return;

    const id = `att-${++idCounter}-${Date.now()}`;
    const reader = new FileReader();
    reader.onload = (e) => {
      const previewUrl = e.target?.result as string;
      setAttachments((prev) => [...prev, { id, file, previewUrl }]);
    };
    reader.readAsDataURL(file);
  }, [allowedTypes]);

  const removeFile = useCallback((id: string) => {
    setAttachments((prev) => prev.filter((att) => att.id !== id));
  }, []);

  const clearAll = useCallback(() => {
    setAttachments([]);
    void clearDB(dbName);
  }, [dbName]);

  const getFiles = useCallback((): File[] => {
    return attachments.map((att) => att.file);
  }, [attachments]);

  return { attachments, addFile, removeFile, clearAll, getFiles };
}
