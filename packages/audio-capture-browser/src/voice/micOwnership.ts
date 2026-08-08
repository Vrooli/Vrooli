// DOC: docs/internal/SEAMS.md#mic-ownership-seam
//
// Microphone Ownership Registry
// =============================
//
// Single registry for EVERY browser `getUserMedia` audio stream opened by
// web-console UI code. Each stream is acquired (or registered) under one named
// owner and handed back as a *lease*. Releasing a lease stops all of its tracks
// exactly once — `MediaStreamTrack.stop()` is the only reliable application
// signal that capture should end; nulling a reference or unmounting React does
// not stop the OS microphone (this is the iOS-PWA "mic stuck in Dynamic Island"
// failure mode this module exists to prevent).
//
// Why a registry and not per-owner ad-hoc cleanup:
//   1. Every live mic stream has exactly one observable owner.
//   2. Emergency page-lifecycle cleanup (tab hidden / pagehide / freeze) can
//      release ALL leases without each owner installing its own handler.
//   3. Each owner registers an `onRelease` callback so a lease released by the
//      registry (emergency cleanup, OS track-end) drives the owner's own state
//      back to a consistent "not capturing" value.
//
// Lease release is idempotent: double cleanup is harmless. Logs carry the owner
// id, reason, and live-track count — never audio content, transcripts, or
// device labels.
//
// DOC: docs/internal/VOICE-LATENCY.md#visibility-based-mic-lifecycle

/** Every code path in web-console UI that can open a browser mic stream. */
export type MicOwner =
  | "voice-stream"
  | "whisper"
  | "web-speech"
  | "passive-wake-word"
  | "wake-word-enrollment"
  | "wake-word-test"
  | "speaker-enrollment"
  | "mic-permission-probe";

/** Stable, low-noise reason codes attached to every lease release. */
export type MicReleaseReason =
  | "manual-stop"
  | "provider-error"
  | "setup-error"
  | "toggle-off"
  | "owner-replaced"
  | "unmount"
  | "hidden"
  | "pagehide"
  | "freeze"
  | "ended"
  // A live lease was found while the UI was idle/off (the "mic indicator on but
  // app looks idle" violation); the registry-driven recovery path released it.
  | "invariant-violation"
  // User-triggered "release microphone" recovery from the mic control.
  | "recovery"
  | "test-reset";

/** Scope a page-lifecycle event releases. The backstop never sees "none". */
export type LifecycleReleaseScope = "all" | "non-active";

export type MicLeaseMetadata = Record<string, string | number | boolean | undefined>;

/** Active-recording owners keep the mic open by user intent; passive / prewarm /
 *  settings owners must release on tab-hidden. */
export function isActiveRecordingOwner(owner: MicOwner): boolean {
  return owner === "voice-stream" || owner === "whisper" || owner === "web-speech";
}

export interface MicLease {
  readonly id: string;
  readonly owner: MicOwner;
  readonly stream: MediaStream;
  readonly acquiredAt: number;
  readonly metadata?: MicLeaseMetadata;
  /** True once the lease has been released (tracks stopped). */
  readonly released: boolean;
  /** Stop all tracks and run cleanup exactly once. Idempotent. */
  release(reason: MicReleaseReason): void;
}

/** Read-only snapshot for tests / debug surfaces. Never exposes the raw stream. */
export interface MicLeaseSnapshot {
  id: string;
  owner: MicOwner;
  acquiredAt: number;
  trackCount: number;
  liveTrackCount: number;
  metadata?: MicLeaseMetadata;
}

export type MicLeasePredicate = (lease: Pick<MicLease, "id" | "owner" | "metadata">) => boolean;

export interface AcquireMicOptions {
  metadata?: MicLeaseMetadata;
  /**
   * Fired exactly once when this lease is released, regardless of who released
   * it (the owner, emergency lifecycle cleanup, or an OS `ended` event). Lets
   * the owner reset its own state to a consistent "not capturing" value. Never
   * allowed to throw out of `release()`.
   */
  onRelease?: (reason: MicReleaseReason) => void;
}

let _idSeq = 0;
const _leases = new Set<LeaseImpl>();

/** Subscribers notified (with a fresh metadata-only snapshot) on every lease
 *  acquire/release, so the UI can derive live-mic honesty without polling. */
export type MicLeaseListener = (snapshots: MicLeaseSnapshot[]) => void;
const _listeners = new Set<MicLeaseListener>();

function notifyLeaseListeners(): void {
  if (_listeners.size === 0) return;
  const snapshot = getActiveMicLeases();
  for (const listener of [..._listeners]) {
    try {
      listener(snapshot);
    } catch (err) {
      console.warn("[mic] lease listener threw:", err);
    }
  }
}

/**
 * Subscribe to lease acquire/release. The listener receives a metadata-only
 * snapshot (never the raw stream) on every change and once is not called for
 * the current state — call `getActiveMicLeases()` for the initial read.
 * Returns an unsubscribe function.
 */
export function subscribeMicLeases(listener: MicLeaseListener): () => void {
  _listeners.add(listener);
  return () => {
    _listeners.delete(listener);
  };
}

function safeTrackStop(track: MediaStreamTrack): boolean {
  const wasLive = track.readyState === "live";
  if (typeof track.stop === "function") {
    try { track.stop(); } catch { /* already stopped */ }
  }
  return wasLive;
}

class LeaseImpl implements MicLease {
  readonly id: string;
  readonly owner: MicOwner;
  readonly stream: MediaStream;
  readonly acquiredAt: number;
  readonly metadata?: MicLeaseMetadata;
  private readonly onRelease?: (reason: MicReleaseReason) => void;
  private trackCleanups: Array<() => void> = [];
  private _released = false;

  constructor(owner: MicOwner, stream: MediaStream, options?: AcquireMicOptions) {
    this.id = `mic-${owner}-${++_idSeq}`;
    this.owner = owner;
    this.stream = stream;
    this.acquiredAt = Date.now();
    this.metadata = options?.metadata;
    this.onRelease = options?.onRelease;

    for (const track of stream.getTracks()) {
      if (typeof track.addEventListener !== "function") continue;
      const onEnded = () => {
        // OS/browser revoked the device (sleep/wake, another app seized it,
        // permission revoked). Release the whole lease so the owner resets.
        console.warn("[mic] track ended owner=%s id=%s", this.owner, this.id);
        this.release("ended");
      };
      const onMute = () => {
        console.warn("[mic] track muted owner=%s id=%s (readyState=%s)", this.owner, this.id, track.readyState);
      };
      const onUnmute = () => {
        console.info("[mic] track unmuted owner=%s id=%s", this.owner, this.id);
      };
      track.addEventListener("ended", onEnded, { once: true });
      track.addEventListener("mute", onMute);
      track.addEventListener("unmute", onUnmute);
      this.trackCleanups.push(() => {
        if (typeof track.removeEventListener !== "function") return;
        try {
          track.removeEventListener("ended", onEnded);
          track.removeEventListener("mute", onMute);
          track.removeEventListener("unmute", onUnmute);
        } catch { /* listener never attached */ }
      });
    }
  }

  get released(): boolean {
    return this._released;
  }

  release(reason: MicReleaseReason): void {
    if (this._released) return;
    this._released = true;

    for (const cleanup of this.trackCleanups) cleanup();
    this.trackCleanups = [];

    let liveCount = 0;
    for (const track of this.stream.getTracks()) {
      if (safeTrackStop(track)) liveCount++;
    }
    _leases.delete(this);
    console.info("[mic] release owner=%s id=%s reason=%s liveTracks=%d", this.owner, this.id, reason, liveCount);
    notifyLeaseListeners();

    if (this.onRelease) {
      try {
        this.onRelease(reason);
      } catch (err) {
        console.warn("[mic] onRelease threw owner=%s id=%s:", this.owner, this.id, err);
      }
    }
  }
}

/**
 * Register a stream that was acquired elsewhere (an unavoidable external/provider
 * path) so it participates in lease ownership and emergency cleanup.
 */
export function registerMicStream(owner: MicOwner, stream: MediaStream, options?: AcquireMicOptions): MicLease {
  const lease = new LeaseImpl(owner, stream, options);
  _leases.add(lease);
  console.info("[mic] acquire owner=%s id=%s tracks=%d", owner, lease.id, stream.getTracks().length);
  notifyLeaseListeners();
  return lease;
}

/**
 * Acquire a fresh mic stream through `getUserMedia` and register it under
 * `owner`. Rejects (propagating the browser error) if access is denied.
 */
export async function acquireMicStream(
  owner: MicOwner,
  constraints: MediaStreamConstraints,
  options?: AcquireMicOptions,
): Promise<MicLease> {
  const stream = await navigator.mediaDevices.getUserMedia(constraints);
  return registerMicStream(owner, stream, options);
}

/** Release a single lease. Idempotent; safe to call on an already-released lease. */
export function releaseMicLease(lease: MicLease | null | undefined, reason: MicReleaseReason): void {
  lease?.release(reason);
}

/**
 * Release every active lease (optionally filtered). Used by page-lifecycle
 * emergency cleanup. Iterates a snapshot so `onRelease` side effects cannot
 * mutate the set mid-loop.
 */
export function releaseAllMicLeases(reason: MicReleaseReason, predicate?: MicLeasePredicate): void {
  for (const lease of [..._leases]) {
    if (predicate && !predicate(lease)) continue;
    lease.release(reason);
  }
}

/** Metadata snapshot of every active lease. Never exposes the raw stream. */
export function getActiveMicLeases(): MicLeaseSnapshot[] {
  return [..._leases].map((lease) => {
    const tracks = lease.stream.getTracks();
    return {
      id: lease.id,
      owner: lease.owner,
      acquiredAt: lease.acquiredAt,
      trackCount: tracks.length,
      liveTrackCount: tracks.filter((t) => t.readyState === "live").length,
      metadata: lease.metadata,
    };
  });
}

let _lifecycleRefcount = 0;
let _uninstallLifecycle: (() => void) | null = null;

/** Default backstop policy: hidden releases non-active leases (the active
 *  recording is stopped by its owner for UI consistency); pagehide/freeze
 *  release all. Callers running as a standalone/PWA inject a resolver that
 *  upgrades `hidden` to `all` — see `micLifecyclePolicy.decideMicLifecycle`. */
const DEFAULT_LIFECYCLE_SCOPE: (event: "hidden" | "pagehide" | "freeze") => LifecycleReleaseScope =
  (event) => (event === "hidden" ? "non-active" : "all");

/**
 * Install page-lifecycle emergency cleanup (idempotent + ref-counted, so it can
 * be installed from multiple mounts without stacking listeners):
 *
 *   - `visibilitychange` → hidden: release leases per `resolveScope("hidden")`.
 *     The default releases every NON-active-recording lease; a standalone/PWA
 *     caller releases ALL (iOS keeps the OS mic indicator on otherwise).
 *   - `pagehide` / `freeze`: release ALL leases. The page is going away —
 *     privacy/hardware release wins over preserving a partial recording. MDN
 *     notes mobile `pagehide` is not fully reliable, so `visibilitychange` is
 *     the primary session-end signal and `pagehide`/`freeze` are complementary.
 *
 * `resolveScope` is read on every event (not cached) so display-mode can change
 * at runtime. Returns an uninstall function.
 */
export function installMicLifecycleCleanup(
  resolveScope: (event: "hidden" | "pagehide" | "freeze") => LifecycleReleaseScope = DEFAULT_LIFECYCLE_SCOPE,
): () => void {
  const release = (event: "hidden" | "pagehide" | "freeze") => {
    const scope = resolveScope(event);
    if (scope === "all") {
      releaseAllMicLeases(event);
    } else {
      releaseAllMicLeases(event, (l) => !isActiveRecordingOwner(l.owner));
    }
  };
  _lifecycleRefcount++;
  if (_lifecycleRefcount === 1) {
    const onVisibility = () => {
      if (typeof document !== "undefined" && document.visibilityState === "hidden") {
        release("hidden");
      }
    };
    const onPageHide = () => release("pagehide");
    const onFreeze = () => release("freeze");

    document.addEventListener("visibilitychange", onVisibility);
    window.addEventListener("pagehide", onPageHide);
    // Chrome page-lifecycle "freeze" — best-effort, not present everywhere.
    document.addEventListener("freeze", onFreeze);

    _uninstallLifecycle = () => {
      document.removeEventListener("visibilitychange", onVisibility);
      window.removeEventListener("pagehide", onPageHide);
      document.removeEventListener("freeze", onFreeze);
    };
  }

  let uninstalled = false;
  return () => {
    if (uninstalled) return;
    uninstalled = true;
    _lifecycleRefcount = Math.max(0, _lifecycleRefcount - 1);
    if (_lifecycleRefcount === 0 && _uninstallLifecycle) {
      _uninstallLifecycle();
      _uninstallLifecycle = null;
    }
  };
}

/** Test-only: release all leases silently and reset the lifecycle installer. */
export function _resetMicOwnershipForTesting(): void {
  for (const lease of [..._leases]) lease.release("test-reset");
  _leases.clear();
  _listeners.clear();
  if (_uninstallLifecycle) {
    _uninstallLifecycle();
    _uninstallLifecycle = null;
  }
  _lifecycleRefcount = 0;
}
