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

export const createCanonicalPcmCapture: AsyncPcmCaptureFactory = (stream, onFrame) =>
  createSharedCanonicalPcmCapture(getSharedAudioContext(), stream, onFrame);
