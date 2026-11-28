/** Transport-neutral client contract shared by browser and device panes. */
export type RemoteSurfaceKind = "browser" | "device";
export type RemoteSurfaceState =
  | "offline"
  | "blocked"
  | "ready"
  | "connecting"
  | "disconnected"
  | "viewer"
  | "controller"
  | "revoked"
  | "failed";

/** Prevent duplicate owner lifecycle requests when controls are clicked repeatedly. */
export class SingleFlightSurfaceOperation {
  private active?: { name: string; promise: Promise<unknown> };

  run<T>(name: string, operation: () => Promise<T>): Promise<T | undefined> {
    if (this.active?.name === name) return this.active.promise as Promise<T>;
    if (this.active) {
      // Closing is cleanup: let it follow the in-flight owner transition.
      if (name === "close") return this.active.promise.then(
        () => this.run(name, operation),
        () => this.run(name, operation),
      );
      return Promise.resolve(undefined);
    }
    const promise = operation();
    this.active = { name, promise };
    void promise.finally(() => {
      if (this.active?.promise === promise) this.active = undefined;
    }).catch(() => undefined);
    return promise;
  }

  get busy(): boolean { return Boolean(this.active); }
}

export interface ControllerLease {
  id: string;
  epoch: bigint;
  controller: boolean;
}

export interface SurfaceFrameEnvelope {
  surface: RemoteSurfaceKind;
  sequence: bigint;
  capturedAtMs: number;
  width: number;
  height: number;
  byteLength: number;
}

export type BrowserSurfaceInput =
  | { type: "pointer"; phase: "move" | "down" | "up"; button: "left" | "middle" | "right"; x: number; y: number; modifiers?: string[] }
  | { type: "wheel"; x: number; y: number; deltaX: number; deltaY: number }
  | { type: "key"; phase: "down" | "up"; key: string; modifiers: string[] }
  | { type: "text"; text: string };

export interface SurfaceSize { width: number; height: number }
export interface SurfaceBounds { left: number; top: number; width: number; height: number }

/** Map a contain-fitted frame point to the owner's logical viewport. */
export function mapSurfacePoint(
  clientX: number,
  clientY: number,
  bounds: SurfaceBounds,
  frame: SurfaceSize,
  viewport: SurfaceSize,
  clampOutside = false,
): { x: number; y: number } | undefined {
  if (![clientX, clientY, bounds.left, bounds.top, bounds.width, bounds.height, frame.width, frame.height, viewport.width, viewport.height].every(Number.isFinite) ||
      bounds.width <= 0 || bounds.height <= 0 || frame.width <= 0 || frame.height <= 0 || viewport.width <= 0 || viewport.height <= 0) return undefined;
  const aspect = frame.width / frame.height;
  const width = Math.min(bounds.width, bounds.height * aspect);
  const height = Math.min(bounds.height, bounds.width / aspect);
  const left = bounds.left + (bounds.width - width) / 2;
  const top = bounds.top + (bounds.height - height) / 2;
  if (!clampOutside && (clientX < left || clientX > left + width || clientY < top || clientY > top + height)) return undefined;
  const mappedX = clampOutside ? Math.min(left + width, Math.max(left, clientX)) : clientX;
  const mappedY = clampOutside ? Math.min(top + height, Math.max(top, clientY)) : clientY;
  return {
    x: Math.min(viewport.width - 1, Math.max(0, Math.floor(((mappedX - left) / width) * viewport.width))),
    y: Math.min(viewport.height - 1, Math.max(0, Math.floor(((mappedY - top) / height) * viewport.height))),
  };
}

/** Serialize commands and discard queued commands after a lease changes. */
export class OrderedLeaseInput {
  private tail: Promise<void> = Promise.resolve();
  private currentLease = "";
  private nextOrder = 0;
  private pending = 0;
  private readonly releaseReserve = 16;

  /** Keep slow owner transports from turning pointer motion into unbounded memory. */
  constructor(private readonly maxPending = 64) {}

  updateLease(lease?: ControllerLease): void {
    const next = lease?.controller && lease.id && lease.epoch > 0n
      ? `${lease.id}:${lease.epoch}`
      : "";
    if (next !== this.currentLease) {
      this.currentLease = next;
      this.nextOrder = 0;
    }
  }

  enqueue(lease: ControllerLease, send: (order: number) => Promise<void>, release = false): Promise<boolean> {
    if (this.pending >= this.maxPending + (release ? this.releaseReserve : 0)) return Promise.resolve(false);
    const leaseKey = `${lease.id}:${lease.epoch}`;
    const order = ++this.nextOrder;
    let accepted = false;
    this.pending++;
    const operation = this.tail.then(async () => {
      if (!lease.controller || !lease.id || lease.epoch <= 0n || leaseKey !== this.currentLease) return;
      await send(order);
      accepted = true;
    });
    this.tail = operation.catch(() => undefined).finally(() => { this.pending--; });
    return operation.then(() => accepted);
  }
}

export const REMOTE_SURFACE_CLIPBOARD_LIMIT = 64 * 1024;
/** BAS owner text-input stream cap. Clipboard remains separately bounded at 64 KiB. */
export const REMOTE_SURFACE_TEXT_INPUT_LIMIT = 4 * 1024;

export function boundedClipboardText(text: string): string {
  if (new TextEncoder().encode(text).byteLength > REMOTE_SURFACE_CLIPBOARD_LIMIT) {
    throw new Error("Clipboard text exceeds the 64 KiB remote-surface limit.");
  }
  return text;
}

export function boundedRemoteTextInput(text: string): string {
  if (new TextEncoder().encode(text).byteLength > REMOTE_SURFACE_TEXT_INPUT_LIMIT) {
    throw new Error("Browser text input exceeds the 4 KiB remote-surface limit.");
  }
  return text;
}
