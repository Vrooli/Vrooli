/** A live canonical capture graph. stop is safe to call more than once. */
export interface PcmCapture {
  stop(): void;
}

export type PcmCaptureFactory = (
  stream: MediaStream,
  onFrame: (samples: Float32Array, sampleRate: number) => void,
) => PcmCapture;

export type AsyncPcmCaptureFactory = (
  stream: MediaStream,
  onFrame: (samples: Float32Array, sampleRate: number) => void,
) => PcmCapture | Promise<PcmCapture>;

/**
 * Build the browser-independent part of a ScriptProcessor PCM graph. The host
 * supplies its shared AudioContext policy; the shared core owns frame-copy and
 * teardown semantics so consumers cannot drift on capture ordering.
 */
export function createScriptProcessorPcmCapture(
  context: AudioContext,
  stream: MediaStream,
  onFrame: (samples: Float32Array, sampleRate: number) => void,
): PcmCapture {
  const source = context.createMediaStreamSource(stream);
  // eslint-disable-next-line @typescript-eslint/no-deprecated -- broad browser fallback while AudioWorklet is introduced behind this seam.
  const processor = context.createScriptProcessor(4096, 1, 1);
  // eslint-disable-next-line @typescript-eslint/no-deprecated -- matches ScriptProcessor fallback above.
  processor.onaudioprocess = (event: AudioProcessingEvent) => {
    // eslint-disable-next-line @typescript-eslint/no-deprecated -- ScriptProcessor input API.
    onFrame(new Float32Array(event.inputBuffer.getChannelData(0)), context.sampleRate);
  };
  const silentGain = context.createGain();
  silentGain.gain.value = 0;
  source.connect(processor);
  processor.connect(silentGain);
  silentGain.connect(context.destination);
  let stopped = false;
  return {
    stop() {
      if (stopped) return;
      stopped = true;
      // eslint-disable-next-line @typescript-eslint/no-deprecated -- detaching the handler created above.
      processor.onaudioprocess = null;
      try {
        source.disconnect();
        processor.disconnect();
        silentGain.disconnect();
      } catch {
        // Browser teardown may have already disconnected the graph.
      }
    },
  };
}

const WORKLET_PROCESSOR = `
class VrooliPcmCaptureProcessor extends AudioWorkletProcessor {
  process(inputs) {
    const input = inputs[0] && inputs[0][0];
    if (input) {
      const copy = input.slice();
      this.port.postMessage(copy, [copy.buffer]);
    }
    return true;
  }
}
registerProcessor("vrooli-pcm-capture", VrooliPcmCaptureProcessor);
`;

/**
 * Preferred capture path. It isolates audio callbacks from the UI thread on
 * modern browsers, then callers can use createScriptProcessorPcmCapture as a
 * tested compatibility fallback when AudioWorklet is unavailable or blocked.
 */
export async function createAudioWorkletPcmCapture(
  context: AudioContext,
  stream: MediaStream,
  onFrame: (samples: Float32Array, sampleRate: number) => void,
): Promise<PcmCapture> {
  if (!context.audioWorklet || typeof AudioWorkletNode === "undefined") {
    throw new Error("AudioWorklet capture is unavailable");
  }
  const moduleURL = URL.createObjectURL(new Blob([WORKLET_PROCESSOR], { type: "application/javascript" }));
  try {
    await context.audioWorklet.addModule(moduleURL);
  } finally {
    URL.revokeObjectURL(moduleURL);
  }
  const source = context.createMediaStreamSource(stream);
  const node = new AudioWorkletNode(context, "vrooli-pcm-capture");
  const silentGain = context.createGain();
  silentGain.gain.value = 0;
  node.port.onmessage = (event: MessageEvent<Float32Array>) => {
    const samples = event.data;
    if (samples instanceof Float32Array) onFrame(new Float32Array(samples), context.sampleRate);
  };
  source.connect(node);
  node.connect(silentGain);
  silentGain.connect(context.destination);
  let stopped = false;
  return {
    stop() {
      if (stopped) return;
      stopped = true;
      node.port.onmessage = null;
      try {
        source.disconnect();
        node.disconnect();
        silentGain.disconnect();
      } catch {
        // Browser graph may already be torn down.
      }
    },
  };
}

export async function createCanonicalPcmCapture(
  context: AudioContext,
  stream: MediaStream,
  onFrame: (samples: Float32Array, sampleRate: number) => void,
): Promise<PcmCapture> {
  let capture: PcmCapture;
  let stopped = false;
  let receivedFrame = false;
  let fallbackTimer: ReturnType<typeof setTimeout> | undefined;
  const deliverFrame = (samples: Float32Array, sampleRate: number) => {
    receivedFrame = true;
    if (fallbackTimer) clearTimeout(fallbackTimer);
    onFrame(samples, sampleRate);
  };

  try {
    capture = await createAudioWorkletPcmCapture(context, stream, deliverFrame);
  } catch {
    return createScriptProcessorPcmCapture(context, stream, onFrame);
  }

  // Some automated Chromium drivers initialise an AudioWorklet graph but
  // never schedule its processor. Keep the preferred path when it delivers a
  // frame, while recovering capture deterministically when it stays silent.
  fallbackTimer = setTimeout(() => {
    if (stopped || receivedFrame) return;
    capture.stop();
    capture = createScriptProcessorPcmCapture(context, stream, deliverFrame);
  }, 750);

  return {
    stop() {
      if (stopped) return;
      stopped = true;
      if (fallbackTimer) clearTimeout(fallbackTimer);
      capture.stop();
    },
  };
}
