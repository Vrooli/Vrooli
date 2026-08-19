// DOC: docs/internal/SEAMS.md#voice-capture-lifecycle-controller-seam
//
// Voice Capture Lifecycle Controller
// ==================================
//
// The SINGLE authority for transitioning provider/capture ownership in
// useVoiceCore. Before this seam, provider replacement, disposal, and error
// cleanup were scattered across several branches of useVoiceCore — a provider
// could be replaced directly without
// disposing the old one first, leaking a live mic track. This controller makes
// every such transition go through one idempotent, replay-safe path.
//
// It is deliberately narrow: it owns provider lifecycle + the start-cancellation
// generation token + stale-lease recovery. It does NOT own VAD, the level
// monitor, timers, or React state — those stay in the hook and are reset via the
// injected `onCaptureTeardown` callback so capture-ownership and workflow state
// always tear down together.
//
// The controller wraps the existing `providerRef` rather than replacing it:
// reads stay `providerRef.current` everywhere in the hook; only the sanctioned
// mutations (replace/dispose/shutdown) go through the controller. This keeps the
// seam small and makes "direct `providerRef.current = …` assignment in an error
// path" a reviewable prohibition rather than a scattered pattern.

import {
  getActiveMicLeases,
  releaseAllMicLeases,
  type MicOwner,
  type MicReleaseReason,
} from "./micOwnership";
import { selectStaleLeases, type VoiceStateLite } from "./micLifecyclePolicy";
import type { TranscriptionProvider } from "./types";

/** Minimal mutable ref shape (React's `MutableRefObject`), kept dependency-free. */
export interface ProviderRef {
  current: TranscriptionProvider | null;
}

export interface VoiceCaptureControllerOptions {
  /**
   * Reset hook-local capture state after a provider teardown (clear the
   * no-audio timer, stop the level monitor, reset VAD, drop the cue-session
   * guard, set workflow state). Must be idempotent — the controller may call it
   * from several paths (dispose, shutdown, recovery) and concurrently.
   */
  onCaptureTeardown?: (reason: MicReleaseReason) => void;
}

/** Active-recording owners the controller force-disposes when it finds them
 *  orphaned (a live lease while the workflow is idle). */
const ACTIVE_OWNERS: ReadonlySet<MicOwner> = new Set<MicOwner>([
  "voice-stream",
  "whisper",
]);

export class VoiceCaptureController {
  private readonly providerRef: ProviderRef;
  private readonly onCaptureTeardown?: (reason: MicReleaseReason) => void;
  /** Monotonic token. A provider `start()` captures the value at begin; if it no
   *  longer matches when start resolves, the start was cancelled (stop during
   *  preparing, lifecycle shutdown) and the late-resolving capture must release
   *  immediately instead of entering the recording state. */
  private startGeneration = 0;

  constructor(providerRef: ProviderRef, options: VoiceCaptureControllerOptions = {}) {
    this.providerRef = providerRef;
    this.onCaptureTeardown = options.onCaptureTeardown;
  }

  /** The active provider, or null. Reads elsewhere stay `providerRef.current`. */
  get provider(): TranscriptionProvider | null {
    return this.providerRef.current;
  }

  /** Lazily create a provider when none exists. Returns the active provider. */
  ensure(factory: () => TranscriptionProvider): TranscriptionProvider {
    if (!this.providerRef.current) {
      this.providerRef.current = factory();
    }
    return this.providerRef.current;
  }

  /** Set the active provider explicitly (used right after `ensure`-less lazy
   *  creation in the start path). Disposes any existing provider first so this
   *  can never silently orphan one. */
  set(provider: TranscriptionProvider): TranscriptionProvider {
    return this.replace(provider, "owner-replaced");
  }

  /**
   * Atomically replace the active provider: dispose the previous provider FIRST
   * (releasing its mic lease), then install the next. The core invariant that
   * makes fallback paths safe — no provider replacement can bypass old-provider
   * cleanup. Idempotent when `next` is already current (no-op).
   */
  replace(next: TranscriptionProvider, reason: MicReleaseReason): TranscriptionProvider {
    const prev = this.providerRef.current;
    if (prev === next) return next;
    prev?.dispose();
    this.providerRef.current = next;
    void reason; // reason is for symmetry/logging parity; dispose has no arg
    return next;
  }

  /**
   * Full capture shutdown: cancel any in-flight start, dispose the active
   * provider (releasing its lease), clear the ref, and run hook teardown.
   * Idempotent and safe to call from concurrent paths (error, cancel, unmount,
   * lifecycle hidden). This is the single cleanup primitive.
   */
  shutdown(reason: MicReleaseReason): void {
    this.cancelStarts();
    const prev = this.providerRef.current;
    this.providerRef.current = null;
    prev?.dispose();
    this.onCaptureTeardown?.(reason);
  }

  /** Begin an async provider start; returns the generation token to re-check
   *  after `await provider.start(...)`. */
  beginStart(): number {
    return ++this.startGeneration;
  }

  /** Whether `token` still refers to the most recent start (not cancelled). */
  isCurrentStart(token: number): boolean {
    return token === this.startGeneration;
  }

  /** Invalidate any in-flight start so its late-resolving capture self-releases. */
  cancelStarts(): void {
    this.startGeneration++;
  }

  /**
   * Release live leases the workflow should not be holding (registry-vs-UI
   * mismatch) and log a structured invariant violation. Returns the released
   * snapshots. If a stale ACTIVE-recording lease is found, the dangling provider
   * is also shut down — a live provider lease while the UI is idle means the
   * provider object itself escaped cleanup.
   *
   * `passiveListenerActive` is the listener-ref truth (set synchronously before
   * the passive lease is acquired) so a normal wake-word handoff never trips it.
   */
  recoverStaleLeases(input: {
    voiceState: VoiceStateLite;
    passiveListenerActive: boolean;
    reason?: MicReleaseReason;
  }): Array<{ id: string; owner: MicOwner }> {
    const reason = input.reason ?? "invariant-violation";
    const snapshot = getActiveMicLeases().map((s) => ({ id: s.id, owner: s.owner }));
    const stale = selectStaleLeases({
      leases: snapshot,
      voiceState: input.voiceState,
      passiveListenerActive: input.passiveListenerActive,
    });
    if (stale.length === 0) return [];

    const staleIds = new Set(stale.map((l) => l.id));
    console.warn(
      "[voice] INVARIANT VIOLATION: live mic lease(s) while UI not capturing — self-healing. owners=%s reason=%s",
      stale.map((l) => l.owner).join(","),
      reason,
    );
    releaseAllMicLeases(reason, (l) => staleIds.has(l.id));

    // A stale active-recording lease means the provider object also escaped
    // cleanup; dispose it so it cannot resurrect callbacks on dead tracks.
    if (stale.some((l) => ACTIVE_OWNERS.has(l.owner)) && this.providerRef.current) {
      this.shutdown(reason);
    }
    return stale;
  }
}
