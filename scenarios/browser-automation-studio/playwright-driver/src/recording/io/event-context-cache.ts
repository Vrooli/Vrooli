import { createHash } from 'node:crypto';
import type { RawBrowserEvent } from '../types';

const EVENT_CONTEXT_TTL_MS = 5 * 60 * 1000;
const PRUNE_INTERVAL_MS = 1000;
const MAX_EVENT_CONTEXTS = 10_000;

type EventContext = {
  bodyDigest: string;
  framePath: string[];
  driverPageId?: string;
  expiresAt: number;
};
type PendingContext = { bodyDigest: string; promise: Promise<EventContext> };

/**
 * Keep the driver-derived attribution for an event stable across browser retries.
 * The page's body and UUID are its idempotency key; frame/page attribution is
 * resolved by Playwright and can change while a page is being rerendered.
 */
export class RecordingEventContextCache {
  private readonly contexts = new Map<string, EventContext>();
  private readonly pending = new Map<string, PendingContext>();
  private nextPruneAt = 0;

  constructor(private readonly now: () => number = Date.now) {}

  async resolve(
    event: RawBrowserEvent,
    deriveContext: () => Promise<{ framePath: string[]; driverPageId?: string }> | { framePath: string[]; driverPageId?: string }
  ): Promise<{ framePath: string[]; driverPageId?: string }> {
    if (!event.id) throw new Error('Recording event identity is required');

    const now = this.now();
    // Expiry work must not grow with the cache size for every browser event.
    // A one-second sweep keeps the cache bounded while making the common path O(1).
    if (now >= this.nextPruneAt) {
      this.prune(now);
      this.nextPruneAt = now + PRUNE_INTERVAL_MS;
    }
    const bodyDigest = createHash('sha256').update(JSON.stringify(event)).digest('hex');
    const existing = this.contexts.get(event.id);
    if (existing) {
      if (existing.bodyDigest !== bodyDigest) {
        throw new Error('Recording event identity conflicts with its original browser payload');
      }
      existing.expiresAt = now + EVENT_CONTEXT_TTL_MS;
      return { framePath: [...existing.framePath], driverPageId: existing.driverPageId };
    }

    const pending = this.pending.get(event.id);
    if (pending) {
      if (pending.bodyDigest !== bodyDigest) {
        throw new Error('Recording event identity conflicts with its original browser payload');
      }
      const context = await pending.promise;
      context.expiresAt = this.now() + EVENT_CONTEXT_TTL_MS;
      return { framePath: [...context.framePath], driverPageId: context.driverPageId };
    }

    if (this.contexts.size + this.pending.size >= MAX_EVENT_CONTEXTS) {
      throw new Error('Recording event retry context capacity exhausted');
    }

    // Derive live Playwright metadata only on the first attempt. A retry may
    // arrive from a frame that was detached or replaced after the original
    // observation failed to commit; that must not prevent replaying its pinned
    // attribution.
    const promise = (async (): Promise<EventContext> => {
      const { framePath, driverPageId } = await deriveContext();
      return {
        bodyDigest,
        framePath: [...framePath],
        driverPageId,
        expiresAt: now + EVENT_CONTEXT_TTL_MS,
      };
    })();
    const entry: PendingContext = { bodyDigest, promise };
    this.pending.set(event.id, entry);
    try {
      const context = await promise;
      this.contexts.set(event.id, context);
      return { framePath: [...context.framePath], driverPageId: context.driverPageId };
    } finally {
      if (this.pending.get(event.id) === entry) this.pending.delete(event.id);
    }
  }

  private prune(now: number): void {
    for (const [id, context] of this.contexts) {
      if (context.expiresAt <= now) this.contexts.delete(id);
    }
  }
}
