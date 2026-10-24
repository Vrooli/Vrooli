import { useCallback, useEffect, useRef, useState } from "react";
import { MAX_DATABASE_BYTES, type DatabaseError, type DatabaseInfo, type DatabasePage, type DatabaseQuery, type DatabaseReply, type DatabaseRequest } from "./databaseData";
export function useDatabasePreview(href: string, size: number, attempt: number) {
  const [info, setInfo] = useState<DatabaseInfo | null>(null), [page, setPage] = useState<DatabasePage | null>(null);
  const [error, setError] = useState<DatabaseError | null>(null), [queryError, setQueryError] = useState<DatabaseError | null>(null), [busy, setBusy] = useState(false);
  const instance = useRef<{
    worker: Worker;
    next: number;
    latest: number;
    timer: ReturnType<typeof setTimeout>;
    fail: (error: DatabaseError) => void;
  } | null>(null);
  useEffect(() => {
    setInfo(null);
    setPage(null);
    setError(null);
    setQueryError(null);
    setBusy(false);
    if (size > MAX_DATABASE_BYTES) {
      setError("limit");
      return;
    }
    if (typeof Worker === "undefined" || typeof WebAssembly === "undefined") {
      setError("unavailable");
      return;
    }
    let active = true, worker: Worker;
    try {
      worker = new Worker(new URL("./database.worker.ts", import.meta.url), { type: "module" });
    }
    catch {
      setError("unavailable");
      return;
    }
    const fail = (reason: DatabaseError) => {
      if (!active)
        return;
      active = false;
      worker.terminate();
      if (instance.current)
        clearTimeout(instance.current.timer);
      instance.current = null;
      setError(reason);
      setBusy(false);
      setPage(null);
    };
    const state = {
      worker, next: 0, latest: 0, timer: setTimeout(() => {
        fail("fetch");
      }, 45000), fail
    };
    instance.current = state;
    worker.onerror = () => {
      fail("unavailable");
    };
    worker.onmessage = (event: MessageEvent<DatabaseReply>) => {
      if (!active)
        return;
      const message = event.data;
      if (message.type === "loading") {
        clearTimeout(state.timer);
        state.timer = setTimeout(() => {
          fail("timeout");
        }, 15000);
        return;
      }
      if (message.type === "ready") {
        clearTimeout(state.timer);
        setInfo(message.info);
        return;
      }
      if (message.type === "error" && message.id === undefined) {
        fail(message.error);
        return;
      }
      {
        if (message.id !== state.latest)
          return;
        clearTimeout(state.timer);
        setBusy(false);
        if (message.type === "page") {
          setPage(message.page);
          setQueryError(null);
        }
        else {
          setQueryError(message.error);
        }
      }
    };
    const url = new URL(href, window.location.href);
    url.searchParams.set("representation", "sqlite-snapshot");
    const request: DatabaseRequest = { type: "open", href: url.toString() };
    worker.postMessage(request);
    return () => {
      active = false;
      clearTimeout(state.timer);
      worker.terminate();
      if (instance.current === state)
        instance.current = null;
    };
  }, [href, size, attempt]);
  const browse = useCallback((query: DatabaseQuery) => {
    const state = instance.current;
    if (!state)
      return;
    clearTimeout(state.timer);
    state.latest = ++state.next;
    setBusy(true);
    setQueryError(null);
    state.timer = setTimeout(() => {
      state.fail("timeout");
    }, 5000);
    const message: DatabaseRequest = { type: "page", id: state.latest, query };
    state.worker.postMessage(message);
  }, []);
  return { info, page, error, queryError, busy, browse };
}
