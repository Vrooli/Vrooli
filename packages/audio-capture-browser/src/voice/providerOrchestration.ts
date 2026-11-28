import type { VoiceCaptureController } from "./voiceCaptureController";
import type { TranscriptionProvider, VoiceBackend } from "./types";

export interface CaptureProviderFactories {
  pcmStream: () => TranscriptionProvider;
  whisper: () => TranscriptionProvider;
}

/** Keep provider choice centralized: streaming is preferred only when the
 * capability snapshot permits it; unavailable backends yield no provider. */
export function createCaptureProvider(
  backend: VoiceBackend,
  streamingReady: boolean,
  factories: CaptureProviderFactories,
): TranscriptionProvider | null {
  if (backend !== "whisper") return null;
  return streamingReady ? factories.pcmStream() : factories.whisper();
}

/** Warm and retain the PCM provider through the lifecycle controller. */
export function ensurePcmStreamProvider(
  controller: Pick<VoiceCaptureController, "ensure">,
  factory: () => TranscriptionProvider,
): TranscriptionProvider {
  return controller.ensure(factory);
}
