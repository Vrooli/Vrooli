/**
 * Cross-tab freshness for the design studio.
 *
 * Two tabs editing the same studio used to discover each other only when a
 * save came back `409`. That safety net stays (the API owns concurrency), but
 * it cannot announce a *new* design or a newly uploaded asset: nothing saves,
 * so nothing conflicts. This module is the missing announcement channel plus
 * the rule that decides what a tab is allowed to do when it hears one.
 *
 * The transport is deliberately small. `BroadcastChannel` is the real path;
 * a `window` `CustomEvent` fallback keeps the same contract in jsdom and in
 * browsers that cannot construct a channel (private-mode SecurityError). Both
 * paths stamp every outgoing message with this module instance's tab id and
 * both drop a message whose stamped source is our own, so a tab never reacts
 * to itself.
 *
 * `reconcileDesign` is the safety contract, not a hint. It is ordered so the
 * cheap and dangerous checks win: a non-active design is never touched, an
 * in-flight save is allowed to settle (it will broadcast), unsaved local edits
 * become an explicit conflict instead of being overwritten, and only a
 * strictly newer remote revision triggers a reload.
 */

export type StudioSyncEvent =
  | {
      type: "designs-changed";
      source: string;
      /**
       * Optional targeted change. When a save broadcasts the design it wrote
       * and the revision it produced, a receiving tab can reconcile exactly
       * that document instead of refetching blindly. Omitted means "the list
       * moved, not necessarily the open document" (create, delete, rename).
       */
      designId?: string;
      revision?: number;
    }
  | { type: "assets-changed"; designId: string; source: string }
  | { type: "template-changed"; source: string };

export interface SyncChannel {
  post(event: StudioSyncEvent): void;
  subscribe(fn: (event: StudioSyncEvent) => void): () => void;
  close(): void;
}

export type SyncDecision = "reload" | "conflict" | "ignore";

/** Default channel/event name. Tests can isolate a channel by passing another. */
export const DEFAULT_SYNC_CHANNEL_NAME = "rcl.studio.sync";

/**
 * Stable for the lifetime of this module instance, which is one tab: bundlers
 * evaluate the module once per window. `crypto.randomUUID` is preferred so two
 * tabs that load in the same millisecond cannot collide.
 */
const TAB_ID: string = (() => {
  const cryptoApi: Crypto | undefined = typeof globalThis.crypto === "undefined"
    ? undefined
    : globalThis.crypto;
  if (cryptoApi && typeof cryptoApi.randomUUID === "function") {
    return `tab-${cryptoApi.randomUUID()}`;
  }
  return `tab-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
})();

export function currentTabId(): string {
  return TAB_ID;
}

/**
 * Accepts a value only when it is a well-formed event. Receipt is untrusted:
 * another tab, a stale bundle, or anything else on the same channel could
 * send a number or a half-written object. Malformed input is dropped, never
 * thrown, so one bad message cannot break the studio's listener.
 */
export function isStudioSyncEvent(value: unknown): value is StudioSyncEvent {
  if (typeof value !== "object" || value === null) return false;
  const record = value as Record<string, unknown>;
  const source = record["source"];
  if (typeof source !== "string" || source.length === 0) return false;
  switch (record["type"]) {
    case "designs-changed": {
      const designId = record["designId"];
      const revision = record["revision"];
      // Untargeted list signal (create / delete / rename).
      if (designId === undefined && revision === undefined) return true;
      // A targeted signal must carry both halves of the reconciliation key.
      return (
        typeof designId === "string" &&
        designId.length > 0 &&
        typeof revision === "number" &&
        Number.isFinite(revision)
      );
    }
    case "assets-changed": {
      const designId = record["designId"];
      return typeof designId === "string" && designId.length > 0;
    }
    case "template-changed":
      return true;
    default:
      return false;
  }
}

/**
 * The decision a receiving tab is allowed to make about one design.
 *
 * Order matters and is deliberate:
 *  1. not the active design      → ignore (nothing on screen to update)
 *  2. a save is in flight        → ignore (it settles and broadcasts its own)
 *  3. unsaved local edits        → conflict (never silently discard them)
 *  4. strictly newer remote rev  → reload
 *  5. same or older revision     → ignore (someone replayed or reordered)
 */
export function reconcileDesign(args: {
  activeId: string;
  activeRevision: number;
  remoteId: string;
  remoteRevision: number;
  dirty: boolean;
  saving: boolean;
}): SyncDecision {
  if (args.remoteId !== args.activeId) return "ignore";
  if (args.saving) return "ignore";
  if (args.dirty) return "conflict";
  if (args.remoteRevision > args.activeRevision) return "reload";
  return "ignore";
}

function createInertChannel(): SyncChannel {
  return {
    post: () => undefined,
    subscribe: () => () => undefined,
    close: () => undefined,
  };
}

function createBroadcastSyncChannel(name: string): SyncChannel {
  const channel = new BroadcastChannel(name);
  const listeners = new Set<(event: StudioSyncEvent) => void>();
  let closed = false;

  const onMessage = (event: MessageEvent): void => {
    if (closed) return;
    const payload: unknown = event.data;
    if (!isStudioSyncEvent(payload)) return;
    if (payload.source === TAB_ID) return;
    for (const listener of listeners) listener(payload);
  };

  channel.addEventListener("message", onMessage);

  return {
    post: (event) => {
      if (closed) return;
      // The transport owns attribution; a caller cannot forge another tab.
      channel.postMessage({ ...event, source: TAB_ID });
    },
    subscribe: (fn) => {
      if (closed) return () => undefined;
      listeners.add(fn);
      return () => {
        listeners.delete(fn);
      };
    },
    close: () => {
      if (closed) return;
      closed = true;
      listeners.clear();
      channel.removeEventListener("message", onMessage);
      channel.close();
    },
  };
}

function createWindowSyncChannel(name: string): SyncChannel {
  const listeners = new Set<(event: StudioSyncEvent) => void>();
  let closed = false;

  const onCustomEvent = (event: Event): void => {
    if (closed) return;
    if (!(event instanceof CustomEvent)) return;
    const payload: unknown = event.detail;
    if (!isStudioSyncEvent(payload)) return;
    if (payload.source === TAB_ID) return;
    for (const listener of listeners) listener(payload);
  };

  window.addEventListener(name, onCustomEvent);

  return {
    post: (event) => {
      if (closed) return;
      window.dispatchEvent(new CustomEvent(name, { detail: { ...event, source: TAB_ID } }));
    },
    subscribe: (fn) => {
      if (closed) return () => undefined;
      listeners.add(fn);
      return () => {
        listeners.delete(fn);
      };
    },
    close: () => {
      if (closed) return;
      closed = true;
      listeners.clear();
      window.removeEventListener(name, onCustomEvent);
    },
  };
}

export function createSyncChannel(name: string = DEFAULT_SYNC_CHANNEL_NAME): SyncChannel {
  if (typeof window === "undefined") return createInertChannel();
  if (typeof BroadcastChannel === "function") {
    try {
      return createBroadcastSyncChannel(name);
    } catch {
      // Private mode / policy can refuse the constructor. The fallback is
      // same-tab only for `post`, but it is a working listener surface and a
      // reload still repairs state, so degrading beats crashing the studio.
    }
  }
  return createWindowSyncChannel(name);
}
