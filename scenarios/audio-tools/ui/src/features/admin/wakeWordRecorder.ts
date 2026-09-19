// Wake-word sample recorder seam.
//
// Records a short clip from the microphone via MediaRecorder and returns it
// as a WakeWordSample (raw container bytes + format). MediaRecorder yields
// WebM/Opus in every supported browser, which the audio-tools API decodes
// server-side (the same path the embed uses), so we tag the sample WEBM and
// report the AudioContext capture rate.
//
// Capture stays on the deprecated MediaRecorder/ScriptProcessor pipeline by
// design — AudioWorklet migration is explicitly deferred.

import { AudioFormat } from "@vrooli/proto-types/audio-tools/v1/common/common_pb";
import type { WakeWordSample } from "../../services/wakeWord";

/** Max clip length; a wake word is short, so we cap recording defensively. */
const MAX_CLIP_MS = 3000;

export interface RecordHandle {
  /** Resolves with the captured sample once recording stops. */
  done: Promise<WakeWordSample>;
  /** Stops recording early (otherwise it auto-stops at MAX_CLIP_MS). */
  stop(): void;
}

/**
 * Begin recording a single wake-word sample. The returned handle exposes a
 * `done` promise (the captured sample) and a `stop()` to end early.
 *
 * Injectable so the enrollment page can be unit-tested without a real
 * MediaRecorder (jsdom provides neither getUserMedia nor MediaRecorder).
 */
export async function recordWakeWordSample(): Promise<RecordHandle> {
  const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
  const recorder = new MediaRecorder(stream);
  const chunks: BlobPart[] = [];
  recorder.ondataavailable = (ev) => {
    if (ev.data.size > 0) chunks.push(ev.data);
  };

  const ctx = new AudioContext();
  const sampleRateHz = ctx.sampleRate;
  void ctx.close();

  let resolve!: (s: WakeWordSample) => void;
  let reject!: (e: unknown) => void;
  const done = new Promise<WakeWordSample>((res, rej) => {
    resolve = res;
    reject = rej;
  });

  recorder.onstop = () => {
    stream.getTracks().forEach((track) => track.stop());
    const blob = new Blob(chunks, { type: recorder.mimeType || "audio/webm" });
    blob
      .arrayBuffer()
      .then((buf) => {
        resolve({
          audio: new Uint8Array(buf),
          format: AudioFormat.WEBM,
          sampleRateHz,
        });
      })
      .catch(reject);
  };

  recorder.start();
  const autoStop = setTimeout(() => {
    if (recorder.state !== "inactive") recorder.stop();
  }, MAX_CLIP_MS);

  return {
    done,
    stop() {
      clearTimeout(autoStop);
      if (recorder.state !== "inactive") recorder.stop();
    },
  };
}
