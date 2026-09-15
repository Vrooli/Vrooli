// Speaker-enrollment clip recorder seam.
//
// Records one enrollment clip from the microphone via MediaRecorder and returns
// it as raw container bytes + format. MediaRecorder yields WebM/Opus in every
// supported browser, which the audio-tools API decodes server-side (the same
// path the embed uses), so we tag the clip WEBM. Capture stays on the
// MediaRecorder pipeline by design (AudioWorklet migration is deferred), mirror-
// ing features/admin/wakeWordRecorder.ts.

import { AudioFormat } from "@vrooli/proto-types/audio-tools/v1/common/common_pb";

/** A captured enrollment clip ready to POST to the resource. */
export interface EnrollmentClip {
  audio: Uint8Array;
  format: AudioFormat;
}

/** Default cap: enrollment clips should be a few seconds of voiced speech. */
export const MAX_ENROLL_CLIP_MS = 6000;

export interface EnrollRecordHandle {
  /** Resolves with the captured clip once recording stops. */
  done: Promise<EnrollmentClip>;
  /** Stops recording early (otherwise it auto-stops at MAX_ENROLL_CLIP_MS). */
  stop(): void;
}

/**
 * Begin recording one enrollment clip. The returned handle exposes a `done`
 * promise (the captured clip) and `stop()` to end early. jsdom provides neither
 * getUserMedia nor MediaRecorder, so the recorder factory is the seam tests
 * substitute.
 */
export async function recordEnrollmentClip(maxMs = MAX_ENROLL_CLIP_MS): Promise<EnrollRecordHandle> {
  const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
  const recorder = new MediaRecorder(stream);
  const chunks: BlobPart[] = [];
  recorder.ondataavailable = (ev) => {
    if (ev.data.size > 0) chunks.push(ev.data);
  };

  let resolve!: (c: EnrollmentClip) => void;
  let reject!: (e: unknown) => void;
  const done = new Promise<EnrollmentClip>((res, rej) => {
    resolve = res;
    reject = rej;
  });

  recorder.onstop = () => {
    stream.getTracks().forEach((track) => track.stop());
    const blob = new Blob(chunks, { type: recorder.mimeType || "audio/webm" });
    blob
      .arrayBuffer()
      .then((buf) => {
        resolve({ audio: new Uint8Array(buf), format: AudioFormat.WEBM });
      })
      .catch(reject);
  };

  recorder.start();
  const autoStop = setTimeout(() => {
    if (recorder.state !== "inactive") recorder.stop();
  }, maxMs);

  return {
    done,
    stop() {
      clearTimeout(autoStop);
      if (recorder.state !== "inactive") recorder.stop();
    },
  };
}
