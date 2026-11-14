import sqlite3InitModule from "@sqlite.org/sqlite-wasm";
import { DatabaseEngine } from "./databaseEngine";
import { MAX_DATABASE_BYTES, type DatabaseError, type DatabaseRequest, type DatabaseReply } from "./databaseData";
let engine: DatabaseEngine | undefined;
function reply(value: DatabaseReply) {
  self.postMessage(value);
}
function errorCode(error: unknown): DatabaseError {
  const message = error instanceof Error ? error.message : "";
  if (message === "limit" || /too (big|many)|too long|out of memory|allocation/i.test(message))
    return "limit";
  if (message === "timeout" || /interrupted/i.test(message))
    return "timeout";
  if (message === "stale" || message === "busy" || message === "fetch" || message === "unavailable")
    return message;
  return "invalid";
}
self.onmessage = async (event: MessageEvent<DatabaseRequest>) => {
  const request = event.data;
  try {
    if (request.type === "page") {
      if (!engine)
        throw new Error("invalid");
      reply({ type: "page", id: request.id, page: engine.page(request.query) });
      return;
    }
    engine?.close();
    engine = undefined;
    const response = await fetch(request.href, { credentials: "same-origin", cache: "no-store" });
    if (!response.ok || !response.body) {
      await response.body?.cancel();
      throw new Error(response.status === 413 ? "limit" : response.status === 409 || response.status === 404 || response.status === 410 ? "stale" : response.status === 503 ? "busy" : response.status === 504 ? "timeout" : response.status === 422 ? "invalid" : "fetch");
    }
    const reader = response.body.getReader(), chunks: Uint8Array[] = [];
    let size = 0;
    try {
      for (;;) {
        const { value, done } = await reader.read();
        if (done)
          break;
        size += value.byteLength;
        if (size > MAX_DATABASE_BYTES) {
          await reader.cancel();
          throw new Error("limit");
        }
        chunks.push(value);
      }
    }
    finally {
      reader.releaseLock();
    }
    const bytes = new Uint8Array(size);
    let offset = 0;
    for (const chunk of chunks) {
      bytes.set(chunk, offset);
      offset += chunk.length;
    }
    reply({ type: "loading" });
    const sqlite = await sqlite3InitModule();
    engine = new DatabaseEngine(sqlite, bytes.buffer);
    reply({ type: "ready", info: { tables: engine.tables, bytes: size, snapshotTime: response.headers.get("X-Preview-Snapshot-Time") ?? new Date().toISOString() } });
  }
  catch (error) {
    reply({ type: "error", error: errorCode(error), ...(request.type === "page" ? { id: request.id } : {}) });
  }
};
