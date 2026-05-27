// DOC: docs/internal/SEAMS.md#voice-input-provider-seam
//
// PCM capture seam for the streaming voice path. The browser taps the mic
// MediaStream and delivers raw Float32 PCM frames at the AudioContext's
// native rate. VoiceStreamProvider owns the policy (downsample, encode,
// send/buffer/drop); this module owns only the Web Audio wiring.
//
// The factory is injectable so VoiceStreamProvider can be unit-tested by
// pushing synthetic frames without a real AudioContext (which jsdom does
// not provide). Production wires createScriptProcessorPcmCapture.

import { getSharedAudioContext } from "./sharedAudioContext";

/** A live PCM capture. Calling stop() tears down the Web Audio graph. */
export interface PcmCapture {
  stop(): void;
}

/**
 * Factory that begins capturing PCM from `stream`, invoking `onFrame` for
 * each delivered frame with the raw Float32 samples and the capture sample
 * rate. The samples passed to onFrame are owned by the callee (a copy);
 * the factory must not retain or mutate them after the call.
 */
export type PcmCaptureFactory = (
  stream: MediaStream,
  onFrame: (samples: Float32Array, sampleRate: number) => void,
) => PcmCapture;

/**
 * Production PCM capture using a ScriptProcessorNode on the shared
 * AudioContext. ScriptProcessor is deprecated but universally supported;
 * the codebase already uses it for passive wake-word capture
 * (audioUtils.createPassiveCapturePipeline) for the same reason. A
 * 4096-sample buffer is ~85 ms at 48 kHz — low callback overhead.
 *
 * The node must be connected to the context destination (through a silent
 * gain) or Chrome stops rendering the subgraph as a power-saving measure.
 */
export const createScriptProcessorPcmCapture: PcmCaptureFactory = (stream, onFrame) => {
  const ctx = getSharedAudioContext();
  const source = ctx.createMediaStreamSource(stream);
  // eslint-disable-next-line @typescript-eslint/no-deprecated -- ScriptProcessor kept for broad browser support; AudioWorklet migration tracked in PROBLEMS.md
  const processor = ctx.createScriptProcessor(4096, 1, 1);
  // eslint-disable-next-line @typescript-eslint/no-deprecated -- onaudioprocess matches the deprecated ScriptProcessor above
  processor.onaudioprocess = (e: AudioProcessingEvent) => {
    // eslint-disable-next-line @typescript-eslint/no-deprecated -- inputBuffer matches the deprecated ScriptProcessor above
    const input = e.inputBuffer.getChannelData(0);
    // Copy: the AudioBuffer is reused across callbacks, so the consumer
    // must receive a stable snapshot.
    onFrame(new Float32Array(input), ctx.sampleRate);
  };

  const silentGain = ctx.createGain();
  silentGain.gain.value = 0;
  source.connect(processor);
  processor.connect(silentGain);
  silentGain.connect(ctx.destination);

  return {
    stop() {
      // eslint-disable-next-line @typescript-eslint/no-deprecated -- detaching the deprecated ScriptProcessor handler installed above
      processor.onaudioprocess = null;
      try {
        source.disconnect();
        processor.disconnect();
        silentGain.disconnect();
      } catch {
        // Nodes may already be disconnected if the context was torn down.
      }
    },
  };
};
