import { useCallback, useEffect, useSyncExternalStore } from "react";

import {
  deleteSnippet,
  listSnippets,
  promoteSnippet,
  touchSnippet,
  upsertSnippet,
  type SnippetDTO,
  type UpsertSnippetInput,
} from "../api/snippets";

export type SnippetsStatus = "loading" | "ready" | "request-error";

interface CacheSnapshot {
  snippets: SnippetDTO[];
  status: SnippetsStatus;
}

const listeners = new Set<() => void>();
let snapshot: CacheSnapshot = { snippets: [], status: "loading" };
let loaded = false;
let loadPromise: Promise<void> | null = null;

function publish(next: CacheSnapshot): void {
  snapshot = next;
  listeners.forEach((listener) => { listener(); });
}

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

function getSnapshot(): CacheSnapshot {
  return snapshot;
}

export function sortSnippets(items: readonly SnippetDTO[]): SnippetDTO[] {
  return [...items].sort((a, b) => {
    if (a.pinned !== b.pinned) return a.pinned ? -1 : 1;
    if (a.last_used_at !== b.last_used_at) return b.last_used_at.localeCompare(a.last_used_at);
    if (a.use_count !== b.use_count) return b.use_count - a.use_count;
    return a.id.localeCompare(b.id);
  });
}

async function load(force = false): Promise<void> {
  if (loadPromise) return loadPromise;
  if (loaded && !force) return;
  publish({ ...snapshot, status: "loading" });
  loadPromise = listSnippets()
    .then((snippets) => {
      loaded = true;
      publish({ snippets: sortSnippets(snippets), status: "ready" });
    })
    .catch(() => {
      publish({ ...snapshot, status: "request-error" });
    })
    .finally(() => {
      loadPromise = null;
    });
  return loadPromise;
}

function replaceSnippet(items: readonly SnippetDTO[], next: SnippetDTO): SnippetDTO[] {
  const found = items.some((item) => item.id === next.id);
  return sortSnippets(found ? items.map((item) => item.id === next.id ? next : item) : [...items, next]);
}

export interface UseSnippetsResult extends CacheSnapshot {
  reload: () => Promise<void>;
  save: (input: UpsertSnippetInput) => Promise<SnippetDTO>;
  remove: (id: string) => Promise<boolean>;
  touch: (id: string) => Promise<void>;
  promote: (id: string) => Promise<string>;
}

export function useSnippets(): UseSnippetsResult {
  const current = useSyncExternalStore(subscribe, getSnapshot, getSnapshot);

  useEffect(() => {
    void load();
  }, []);

  const reload = useCallback(() => load(true), []);
  const save = useCallback(async (input: UpsertSnippetInput) => {
    const saved = await upsertSnippet(input);
    publish({ snippets: replaceSnippet(snapshot.snippets, saved), status: "ready" });
    return saved;
  }, []);
  const remove = useCallback(async (id: string) => {
    const deleted = await deleteSnippet(id);
    if (deleted) {
      publish({ snippets: snapshot.snippets.filter((item) => item.id !== id), status: "ready" });
    }
    return deleted;
  }, []);
  const touch = useCallback(async (id: string) => {
    const previous = snapshot.snippets.find((item) => item.id === id);
    if (!previous) return;
    const optimistic: SnippetDTO = {
      ...previous,
      use_count: previous.use_count + 1,
      last_used_at: new Date().toISOString(),
    };
    publish({ snippets: replaceSnippet(snapshot.snippets, optimistic), status: "ready" });
    try {
      const touched = await touchSnippet(id);
      publish({ snippets: replaceSnippet(snapshot.snippets, touched), status: "ready" });
    } catch {
      publish({ snippets: replaceSnippet(snapshot.snippets, previous), status: "ready" });
    }
  }, []);

  const promote = useCallback((id: string) => promoteSnippet(id), []);

  return { ...current, reload, save, remove, touch, promote };
}

/** Clears the shared module cache between isolated tests. */
export function resetSnippetsCacheForTests(): void {
  loaded = false;
  loadPromise = null;
  snapshot = { snippets: [], status: "loading" };
  listeners.clear();
}
