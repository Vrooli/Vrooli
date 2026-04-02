import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useIndexedDBAttachments } from "./useIndexedDBAttachments";
import type { UseIndexedDBAttachmentsOptions } from "./useIndexedDBAttachments";

// ---------------------------------------------------------------------------
// Fake IndexedDB
// ---------------------------------------------------------------------------

/** Minimal in-memory IndexedDB mock scoped to these tests. */
function createFakeIndexedDB() {
  const stores = new Map<string, Map<string, unknown>>();

  function getStore(dbName: string, storeName: string): Map<string, unknown> {
    const key = `${dbName}/${storeName}`;
    if (!stores.has(key)) stores.set(key, new Map());
    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion -- guaranteed by the line above
    return stores.get(key)!;
  }

  const fakeOpen = vi.fn((dbName: string, _version?: number) => {
    const storeName = "attachments";
    const store = getStore(dbName, storeName);

    const objectStoreNames = { contains: () => true };
    const fakeDB = {
      objectStoreNames,
      createObjectStore: vi.fn(),
      transaction: (_name: string, _mode?: string) => ({
        objectStore: () => ({
          getAll: () => {
            const req = {
              result: Array.from(store.values()),
              onsuccess: null as (() => void) | null,
              onerror: null as (() => void) | null,
            };
            // Fire onsuccess asynchronously so the caller can attach the handler.
            queueMicrotask(() => req.onsuccess?.());
            return req;
          },
          put: (val: { id: string }) => { store.set(val.id, val); },
          clear: () => { store.clear(); },
        }),
      }),
    };

    const request = {
      result: fakeDB,
      onsuccess: null as (() => void) | null,
      onerror: null as (() => void) | null,
      onupgradeneeded: null as (() => void) | null,
    };
    queueMicrotask(() => request.onsuccess?.());
    return request;
  });

  return {
    open: fakeOpen,
    clear() { stores.clear(); },
  };
}

// ---------------------------------------------------------------------------
// FileReader mock — synchronously resolve with a deterministic data URL
// ---------------------------------------------------------------------------

class FakeFileReader {
  result: string | null = null;
  onload: ((e: { target: { result: string } }) => void) | null = null;

  readAsDataURL(file: File) {
    const url = `data:${file.type};base64,dGVzdA==`; // "test" in base64
    this.result = url;
    queueMicrotask(() => this.onload?.({ target: { result: url } }));
  }
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

const defaultOpts: UseIndexedDBAttachmentsOptions = { dbName: "test-db", persistDebounceMs: 0 };

describe("useIndexedDBAttachments", () => {
  let fakeIDB: ReturnType<typeof createFakeIndexedDB>;

  beforeEach(() => {
    fakeIDB = createFakeIndexedDB();
    vi.stubGlobal("indexedDB", fakeIDB);
    vi.stubGlobal("FileReader", FakeFileReader);
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
    fakeIDB.clear();
  });

  it("starts with an empty attachments array", () => {
    const { result } = renderHook(() => useIndexedDBAttachments(defaultOpts));
    expect(result.current.attachments).toEqual([]);
  });

  it("adds a file and produces a preview URL", async () => {
    const { result } = renderHook(() => useIndexedDBAttachments(defaultOpts));

    const file = new File(["hello"], "photo.png", { type: "image/png" });
    await act(async () => {
      result.current.addFile(file);
      // Let the microtask (FileReader.onload) flush
      await Promise.resolve();
    });

    expect(result.current.attachments).toHaveLength(1);
    const first = result.current.attachments[0];
    expect(first).toBeDefined();
    expect(first?.file.name).toBe("photo.png");
    expect(first?.previewUrl).toContain("data:image/png");
  });

  it("rejects files whose type is not in allowedTypes", async () => {
    const { result } = renderHook(() => useIndexedDBAttachments(defaultOpts));

    const file = new File(["data"], "doc.pdf", { type: "application/pdf" });
    await act(async () => {
      result.current.addFile(file);
      await Promise.resolve();
    });

    expect(result.current.attachments).toHaveLength(0);
  });

  it("accepts files when custom allowedTypes is provided", async () => {
    const opts: UseIndexedDBAttachmentsOptions = {
      dbName: "test-db",
      allowedTypes: new Set(["application/pdf"]),
      persistDebounceMs: 0,
    };
    const { result } = renderHook(() => useIndexedDBAttachments(opts));

    const file = new File(["data"], "doc.pdf", { type: "application/pdf" });
    await act(async () => {
      result.current.addFile(file);
      await Promise.resolve();
    });

    expect(result.current.attachments).toHaveLength(1);
  });

  it("removes a file by id", async () => {
    const { result } = renderHook(() => useIndexedDBAttachments(defaultOpts));

    const file = new File(["hello"], "photo.png", { type: "image/png" });
    await act(async () => {
      result.current.addFile(file);
      await Promise.resolve();
    });

    const att = result.current.attachments[0];
    expect(att).toBeDefined();
    const id = att?.id ?? "";
    act(() => {
      result.current.removeFile(id);
    });

    expect(result.current.attachments).toHaveLength(0);
  });

  it("clears all files", async () => {
    const { result } = renderHook(() => useIndexedDBAttachments(defaultOpts));

    await act(async () => {
      result.current.addFile(new File(["a"], "a.png", { type: "image/png" }));
      await Promise.resolve();
    });
    await act(async () => {
      result.current.addFile(new File(["b"], "b.png", { type: "image/png" }));
      await Promise.resolve();
    });

    expect(result.current.attachments).toHaveLength(2);

    act(() => {
      result.current.clearAll();
    });

    expect(result.current.attachments).toHaveLength(0);
  });

  it("getFiles returns the underlying File objects", async () => {
    const { result } = renderHook(() => useIndexedDBAttachments(defaultOpts));

    await act(async () => {
      result.current.addFile(new File(["x"], "x.png", { type: "image/png" }));
      await Promise.resolve();
    });

    const files = result.current.getFiles();
    expect(files).toHaveLength(1);
    const f = files[0];
    expect(f).toBeInstanceOf(File);
    expect(f?.name).toBe("x.png");
  });
});
