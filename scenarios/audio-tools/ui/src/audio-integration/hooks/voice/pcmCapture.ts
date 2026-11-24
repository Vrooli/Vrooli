// DOC: docs/internal/SEAMS.md#voice-input-provider-seam
//
// PCM capture seam for the streaming voice path. The browser taps the mic
// MediaStream and delivers raw Float32 PCM frames at the AudioContext's
// native rate. PcmVoiceStreamProvider owns the policy (downsample, encode,
// send/buffer/drop); the shared browser package owns graph creation and its
// AudioWorklet-to-ScriptProcessor recovery policy.
//
// The factory is injectable so PcmVoiceStreamProvider can be unit-tested by
// pushing synthetic frames without a real AudioContext (which jsdom does
// not provide). Production wires createScriptProcessorPcmCapture.
// HOST DIFFERENCE: this adapter owns the Audio Tools shared AudioContext
// binding; capture behavior stays in @vrooli/audio-capture-browser.

import {
  createCanonicalPcmCapture as createSharedCanonicalPcmCapture,
  createScriptProcessorPcmCapture as createSharedScriptProcessorPcmCapture,
	type AsyncPcmCaptureFactory,
  type PcmCapture,
  type PcmCaptureFactory,
} from "@vrooli/audio-capture-browser";
import { getSharedAudioContext } from "./sharedAudioContext";

export type { AsyncPcmCaptureFactory, PcmCapture, PcmCaptureFactory };

/**
 * Explicit compatibility capture using a ScriptProcessorNode on the shared
 * AudioContext. ScriptProcessor is deprecated but universally supported;
 * the codebase already uses it for passive wake-word capture
 * (audioUtils.createPassiveCapturePipeline) for the same reason.
 * The shared implementation uses a 1024-sample buffer for this compatibility
 * path so fake-media and older browsers do not add a large first-batch delay.
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
  return createSharedCanonicalPcmCapture(context, stream, onFrame);
};
