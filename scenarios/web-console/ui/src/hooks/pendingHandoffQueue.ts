import type { GateResult } from "../components/terminal/inputGate";

const STORAGE_KEY = "wc-pending-handoffs";

type QueueStorage = Pick<Storage, "getItem" | "setItem" | "removeItem">;

function sessionStorageOrNull(): QueueStorage | null {
  try {
    return typeof window === "undefined" ? null : window.sessionStorage;
  } catch {
    return null;
  }
}

/**
 * Fallback for handoffs whose session exists before its terminal mounts.
 * FIFO order is per session. An item leaves the queue only after the terminal
 * accepts it (`sent` or `queued`); a rejection keeps that item and everything
 * behind it available for the next reconciliation attempt. The pending intents
 * live in this tab's sessionStorage, so a reload can recover them without
 * sharing their text with another tab or keeping it after the tab is closed.
 */
export class PendingHandoffQueue {
  private readonly bySession = new Map<string, string[]>();

  constructor(private readonly storage: QueueStorage | null = sessionStorageOrNull()) {
    this.restore();
  }

  enqueue(sessionId: string, text: string): void {
    if (!text) return;
    const queue = this.bySession.get(sessionId) ?? [];
    queue.push(text);
    this.bySession.set(sessionId, queue);
    this.persist();
  }

  flush(sessionId: string, deliver: (text: string) => GateResult): number {
    const queue = this.bySession.get(sessionId);
    if (!queue?.length) return 0;

    let accepted = 0;
    while (queue.length > 0) {
      const next = queue[0];
      if (next === undefined) break;
      const result = deliver(next);
      if (result.status === "rejected") break;
      queue.shift();
      accepted += 1;
    }

    if (queue.length === 0) this.bySession.delete(sessionId);
    this.persist();
    return accepted;
  }

  removeSession(sessionId: string): void {
    this.bySession.delete(sessionId);
    this.persist();
  }

  getPending(sessionId: string): readonly string[] {
    return this.bySession.get(sessionId) ?? [];
  }

  private restore(): void {
    if (!this.storage) return;
    try {
      const raw = this.storage.getItem(STORAGE_KEY);
      if (!raw) return;
      const restored: unknown = JSON.parse(raw);
      if (!restored || typeof restored !== "object" || Array.isArray(restored)) return;
      for (const [sessionId, value] of Object.entries(restored)) {
        if (!Array.isArray(value) || !value.every((text) => typeof text === "string" && text.length > 0)) continue;
        this.bySession.set(sessionId, [...value]);
      }
    } catch {
      // A corrupt or unavailable session store must not stop the workspace
      // from running. The in-memory queue still protects this tab's retries.
    }
  }

  private persist(): void {
    if (!this.storage) return;
    try {
      if (this.bySession.size === 0) {
        this.storage.removeItem(STORAGE_KEY);
      } else {
        this.storage.setItem(STORAGE_KEY, JSON.stringify(Object.fromEntries(this.bySession)));
      }
    } catch {
      // Keep the in-memory fallback usable if browser storage is unavailable.
    }
  }
}
