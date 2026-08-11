/** @vrooliComponentSource services.undo-manager */
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { useAnnounce } from "../../../../hooks/useAnnounce/versions/1.0.0/useAnnounce";

export type UndoStatus =
  | "available"
  | "submitting"
  | "success"
  | "error"
  | "expired";

export interface UndoInput {
  id?: string;
  title: string;
  detail?: string;
  expiresMs?: number;
  undo: () => void | Promise<void>;
  successMessage?: string;
  successDetail?: string;
}

export interface UndoSeed extends UndoInput {
  status?: UndoStatus;
  error?: string;
}

export interface UndoRecord extends Omit<UndoInput, "id"> {
  id: string;
  status: UndoStatus;
  error?: string;
  createdAt: number;
  expiresAt: number;
}

export interface UndoManagerHandle {
  records: UndoRecord[];
  push: (input: UndoInput) => string;
  undo: (id: string) => Promise<void>;
  retry: (id: string) => Promise<void>;
  dismiss: (id: string) => void;
}

interface UndoManagerOptions {
  children?: ReactNode;
  initialRecords?: UndoSeed[];
  maxVisible?: number;
  defaultExpiresMs?: number;
}

const UndoManagerContext = createContext<UndoManagerHandle | null>(null);
let sequence = 0;

const makeId = () => `undo-${Date.now()}-${sequence++}`;
const normalizeExpiry = (value: number | undefined, fallback: number) =>
  Math.min(Math.max(value ?? fallback, 2000), 30000);

function createRecord(input: UndoSeed, fallbackExpiry: number): UndoRecord {
  const createdAt = Date.now();
  const expiresMs = normalizeExpiry(input.expiresMs, fallbackExpiry);
  return {
    ...input,
    id: input.id ?? makeId(),
    status: input.status ?? "available",
    createdAt,
    expiresAt: createdAt + expiresMs,
    expiresMs,
  };
}

export function UndoManagerProvider({
  children,
  initialRecords = [],
  maxVisible = 1,
  defaultExpiresMs = 8000,
}: UndoManagerOptions) {
  const announce = useAnnounce();
  const [records, setRecords] = useState<UndoRecord[]>(() =>
    initialRecords
      .slice(-Math.max(1, maxVisible))
      .map((record) => createRecord(record, defaultExpiresMs)),
  );
  const recordsRef = useRef(records);
  recordsRef.current = records;

  const dismiss = useCallback((id: string) => {
    setRecords((current) => current.filter((record) => record.id !== id));
  }, []);

  const performUndo = useCallback(
    async (id: string) => {
      const record = recordsRef.current.find(
        (candidate) => candidate.id === id,
      );
      if (
        !record ||
        (record.status !== "available" && record.status !== "error")
      )
        return;
      setRecords((current) =>
        current.map((candidate) =>
          candidate.id === id
            ? { ...candidate, status: "submitting", error: undefined }
            : candidate,
        ),
      );
      try {
        await record.undo();
        setRecords((current) =>
          current.map((candidate) =>
            candidate.id === id
              ? { ...candidate, status: "success" }
              : candidate,
          ),
        );
        announce(record.successMessage ?? `${record.title} restored.`);
      } catch (error) {
        const message =
          error instanceof Error
            ? error.message
            : "The change could not be restored.";
        setRecords((current) =>
          current.map((candidate) =>
            candidate.id === id
              ? { ...candidate, status: "error", error: message }
              : candidate,
          ),
        );
        announce(`Undo failed: ${message}`, { priority: "assertive" });
      }
    },
    [announce],
  );

  const push = useCallback(
    (input: UndoInput) => {
      const record = createRecord(input, defaultExpiresMs);
      setRecords((current) =>
        [...current, record].slice(-Math.max(1, maxVisible)),
      );
      announce(`${record.title}. Undo is available.`);
      return record.id;
    },
    [announce, defaultExpiresMs, maxVisible],
  );

  useEffect(() => {
    const timers = records
      .filter((record) => record.status === "available")
      .map((record) => {
        const remaining = Math.max(record.expiresAt - Date.now(), 0);
        return window.setTimeout(
          () =>
            setRecords((current) =>
              current.map((candidate) =>
                candidate.id === record.id && candidate.status === "available"
                  ? { ...candidate, status: "expired" }
                  : candidate,
              ),
            ),
          remaining,
        );
      });
    return () => timers.forEach((timer) => window.clearTimeout(timer));
  }, [records]);

  const handle = useMemo<UndoManagerHandle>(
    () => ({ records, push, undo: performUndo, retry: performUndo, dismiss }),
    [dismiss, performUndo, push, records],
  );
  return (
    <UndoManagerContext.Provider value={handle}>
      {children}
    </UndoManagerContext.Provider>
  );
}

export function useUndoManager(): UndoManagerHandle {
  const manager = useContext(UndoManagerContext);
  if (!manager)
    throw new Error("useUndoManager must be used within UndoManagerProvider");
  return manager;
}
