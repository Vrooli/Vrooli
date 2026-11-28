/** Provider-neutral messages emitted by Audio Tools' replay-safe stream. */
export interface StreamMessage {
  type: string;
  code?: string;
  text?: string;
  processedSequence?: number;
  providerId?: string;
  modelId?: string;
  providerTier?: "local" | "byok" | "vrooli" | string;
  segmentId?: string;
	segmentIndex?: number;
	startSample?: number;
	endSample?: number;
	speakerPolicy?: "applied" | "degraded" | "rejected" | string;
  score?: number;
  threshold?: number;
  enabled?: boolean;
  profileConfigured?: boolean;
  voiced?: boolean;
  silenceElapsedMs?: number;
  silenceTimeoutMs?: number;
  tickSeq?: number;
  silenceTimedOut?: boolean;
}

/** Convert a generated 64-bit sample cursor without silently rounding it. */
export function safeSampleNumber(value: bigint | number | undefined): number | undefined {
  if (typeof value === "bigint") {
    return value >= 0n && value <= BigInt(Number.MAX_SAFE_INTEGER) ? Number(value) : undefined;
  }
  return Number.isSafeInteger(value) && (value ?? -1) >= 0 ? value : undefined;
}

/** Exact browser-facing message vocabulary stewarded by audio-tools. */
export const STREAM_MESSAGE_TYPES = {
  partial: "partial",
  final: "final",
  segmentFinal: "segment-final",
  segmentRejected: "segment-rejected",
  error: "error",
  done: "done",
  status: "status",
  vadState: "vad-state",
} as const;

export interface StreamMessageHandlers {
  onStatus(code: string, text: string, processedSequence?: bigint, providerIdentity?: { providerId?: string; modelId?: string; providerTier?: string }): void;
  onProviderIdentity?(identity: { providerId?: string; modelId?: string; providerTier?: string }): void;
  onPartial(text: string): void;
	onSegmentFinal(text: string, index: number, speakerPolicy?: string, endSample?: number): void;
  onSegmentAccepted(index: number, score: number, threshold: number): void;
  onSegmentRejected(index: number, score: number, threshold: number): void;
  onSpeakerStatus(enabled: boolean, profileConfigured: boolean): void;
  onVadState(state: { voiced: boolean; silenceElapsedMs: number; silenceTimeoutMs: number; tickSeq: number; silenceTimedOut: boolean }): void;
  onFinal(text: string): void;
  onError(code: string, text: string): void;
}

/**
 * Parses and dispatches the shared v2 stream vocabulary. Segment identities
 * are de-duplicated here so every host gets identical reconnect semantics.
 * Malformed frames are ignored: they must not change capture or journal state.
 */
export function dispatchStreamMessage(raw: unknown, handlers: StreamMessageHandlers, deliveredSegmentIDs: Set<string>): void {
  let message: StreamMessage;
  try {
    message = JSON.parse(String(raw)) as StreamMessage;
  } catch {
    return;
  }
  dispatchStreamEvent(message, handlers, deliveredSegmentIDs);
}

/** Dispatches an already-decoded stream event from Connect or another typed transport. */
export function dispatchStreamEvent(message: StreamMessage, handlers: StreamMessageHandlers, deliveredSegmentIDs: Set<string>): void {
  switch (message.type) {
    case "status":
      handlers.onStatus(
        message.code ?? "stream_status",
        // No fabricated copy. Most status frames are protocol housekeeping —
        // `processed_acknowledgement` arrives per acknowledged wire batch,
        // several times a second — and inventing a sentence for them turned
        // every acknowledgement into a user-facing notice. A status carries
        // human-readable text only when its emitter meant a human to read it.
        message.text ?? "",
        typeof message.processedSequence === "number" && Number.isSafeInteger(message.processedSequence) && message.processedSequence >= 0
          ? BigInt(message.processedSequence)
          : undefined,
        ...(message.providerId || message.modelId || message.providerTier
          ? [{ providerId: message.providerId, modelId: message.modelId, providerTier: message.providerTier }]
          : []),
      );
      if (message.providerId || message.modelId || message.providerTier) {
        handlers.onProviderIdentity?.({ providerId: message.providerId, modelId: message.modelId, providerTier: message.providerTier });
      }
      return;
    case "partial":
      if (message.text) handlers.onPartial(message.text);
      return;
    case "segment-final":
      if (message.providerId || message.modelId || message.providerTier) {
        handlers.onProviderIdentity?.({ providerId: message.providerId, modelId: message.modelId, providerTier: message.providerTier });
      }
      if (message.text === undefined) return;
      if (message.segmentId && deliveredSegmentIDs.has(message.segmentId)) return;
      if (message.segmentId) deliveredSegmentIDs.add(message.segmentId);
      handlers.onSegmentFinal(message.text, message.segmentIndex ?? 0, message.speakerPolicy, message.endSample);
      return;
    case "segment-accepted":
      handlers.onSegmentAccepted(message.segmentIndex ?? 0, message.score ?? 0, message.threshold ?? 0);
      return;
    case "segment-rejected":
      handlers.onSegmentRejected(message.segmentIndex ?? 0, message.score ?? 0, message.threshold ?? 0);
      return;
    case "speaker-status":
      handlers.onSpeakerStatus(Boolean(message.enabled), Boolean(message.profileConfigured));
      return;
    case "vad-state":
      handlers.onVadState({ voiced: Boolean(message.voiced), silenceElapsedMs: message.silenceElapsedMs ?? 0, silenceTimeoutMs: message.silenceTimeoutMs ?? 0, tickSeq: message.tickSeq ?? 0, silenceTimedOut: Boolean(message.silenceTimedOut) });
      return;
    case "final":
      if (message.providerId || message.modelId || message.providerTier) {
        handlers.onProviderIdentity?.({ providerId: message.providerId, modelId: message.modelId, providerTier: message.providerTier });
      }
      handlers.onFinal(message.text?.trim() ?? "");
      return;
    case "error":
      handlers.onError(message.code ?? "stream_error", message.text ?? "Streaming transcription failed");
  }
}
