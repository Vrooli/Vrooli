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

import {
  createCanonicalPcmCapture as createSharedCanonicalPcmCapture,
  createScriptProcessorPcmCapture as createSharedScriptProcessorPcmCapture,
	type AsyncPcmCaptureFactory,
  type PcmCapture,
  type PcmCaptureFactory,
} from "@vrooli/audio-capture-browser";
import { getSharedAudioContext } from "./sharedAudioContext";

export type { AsyncPcmCaptureFactory, PcmCapture, PcmCaptureFactory };

// Some fake-media automation drivers advertise AudioWorklet support but leave
// module loading pending forever. A recorder must still become usable through
// the broadly supported ScriptProcessor graph.
const CANONICAL_CAPTURE_START_TIMEOUT_MS = 1_000;

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
export const createScriptProcessorPcmCapture: PcmCaptureFactory = (stream, onFrame) =>
  createSharedScriptProcessorPcmCapture(getSharedAudioContext(), stream, onFrame);

export const createCanonicalPcmCapture: AsyncPcmCaptureFactory = async (stream, onFrame) => {
  const context = getSharedAudioContext();
  // Acquire the microphone before creating/resuming an AudioContext. Chromium
  // fake-device and some embedded contexts can leave getUserMedia pending when
  // a context is activated first. Once media is available, resume the shared
  // context before attaching either capture graph.
  if (context.state === "suspended") await context.resume();
  // Chromium automation can advertise AudioWorklet while never scheduling its
  // processor. Start the proven ScriptProcessor graph immediately in that
  // environment: a deterministic WAV fixture may be shorter than the
  // worklet-fallback timeout, so waiting would lose the entire turn.
  if (typeof navigator !== "undefined" && navigator.webdriver) {
    return createSharedScriptProcessorPcmCapture(context, stream, onFrame);
  }
  let timedOut = false;
  let timeout: ReturnType<typeof setTimeout> | undefined;
  const preferred = createSharedCanonicalPcmCapture(context, stream, onFrame).then((capture) => {
    // If the preferred graph completes after the fallback is active, tear it
    // down rather than leaving two capture graphs attached to one microphone.
    if (timedOut) capture.stop();
    return capture;
  });

  try {
    return await Promise.race([
      preferred,
      new Promise<never>((_, reject) => {
        timeout = setTimeout(() => reject(new Error("canonical PCM capture startup timed out")), CANONICAL_CAPTURE_START_TIMEOUT_MS);
      }),
    ]);
  } catch {
    timedOut = true;
    return createSharedScriptProcessorPcmCapture(context, stream, onFrame);
  } finally {
    if (timeout) clearTimeout(timeout);
  }
};
