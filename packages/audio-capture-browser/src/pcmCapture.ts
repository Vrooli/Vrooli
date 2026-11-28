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
  // Keep the compatibility path responsive on browsers that cannot schedule
  // AudioWorklet. At the canonical 16 kHz wire rate, 4096 samples delays the
  // first voiced batch by about 256 ms before the streaming decoder can begin;
  // 1024 bounds that at about 64 ms without changing wire batching, cursors,
  // or journal semantics.
  const processor = context.createScriptProcessor(1024, 1, 1);
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

const WORKLET_PROCESSOR = (processorName: string) => `
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
registerProcessor("${processorName}", VrooliPcmCaptureProcessor);
`;

// AudioWorklet modules belong to an AudioContext. Rebuilding the Blob and
// awaiting addModule on every press makes warm microphone reuse pay the full
// module-load cost again, before the first PCM frame can reach the stream.
// Keep one in-flight/completed load per context. A rejected load is removed so
// a transient browser failure can still take the normal ScriptProcessor path
// on a later attempt.
const workletLoads = new WeakMap<AudioContext, Promise<string>>();
const WORKLET_LOAD_TIMEOUT_MS = 1_000;
let nextWorkletProcessorID = 0;

function loadWorklet(context: AudioContext): Promise<string> {
  const existing = workletLoads.get(context);
  if (existing) return existing;

  // addModule cannot be canceled when the timeout wins. Give each attempt a
  // unique registration name so a late successful load cannot make a retry
  // fail with AudioWorklet's duplicate-processor registration error.
  const processorName = [
    "vrooli-pcm-capture",
    Date.now().toString(36),
    (++nextWorkletProcessorID).toString(36),
    Math.random().toString(36).slice(2),
  ].join("-");
  const moduleURL = URL.createObjectURL(new Blob([WORKLET_PROCESSOR(processorName)], { type: "application/javascript" }));
  let timeout: ReturnType<typeof setTimeout> | undefined;
  let moduleLoad: Promise<void>;
  try {
    moduleLoad = context.audioWorklet!.addModule(moduleURL);
  } catch (error) {
    URL.revokeObjectURL(moduleURL);
    return Promise.reject(error);
  }
  // addModule is allowed to remain pending indefinitely in some browser
  // automation and partially supported runtimes. Bound the shared promise so
  // canonical capture can fall back and a later gesture can retry this context.
  const load = Promise.race([
    moduleLoad.then(() => processorName),
    new Promise<string>((_resolve, reject) => {
      timeout = setTimeout(() => reject(new Error("AudioWorklet module load timed out")), WORKLET_LOAD_TIMEOUT_MS);
    }),
  ]).finally(() => {
    if (timeout) clearTimeout(timeout);
    URL.revokeObjectURL(moduleURL);
  });
  workletLoads.set(context, load);
  void load.catch(() => {
    if (workletLoads.get(context) === load) workletLoads.delete(context);
  });
  return load;
}

/**
 * Start loading the preferred capture module without creating a capture graph.
 * Hosts call this from an explicit voice gesture so module startup can overlap
 * microphone permission/device setup. The promise is shared with
 * createAudioWorkletPcmCapture(), so the later graph creation only waits when
 * the browser has not finished loading the module yet.
 */
export function preloadCanonicalPcmCapture(context: AudioContext): Promise<void> {
  if (typeof navigator !== "undefined" && navigator.webdriver) return Promise.resolve();
  if (!context.audioWorklet || typeof AudioWorkletNode === "undefined") return Promise.resolve();
  return loadWorklet(context).then(() => undefined);
}

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
  const processorName = await loadWorklet(context);
  const source = context.createMediaStreamSource(stream);
  const node = new AudioWorkletNode(context, processorName);
  const silentGain = context.createGain();
  silentGain.gain.value = 0;
  node.port.onmessage = (event: MessageEvent<Float32Array>) => {
    const samples = event.data;
    // The worklet transfers an owned copy of each frame, so the main thread
    // can forward its buffer directly. Copying again here adds one allocation
    // and memory pass for every audio callback without protecting a reused
    // AudioBuffer (the ScriptProcessor fallback still snapshots its input).
    if (samples instanceof Float32Array) onFrame(samples, context.sampleRate);
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
	// Browser automation drivers may expose AudioWorklet but not schedule its
	// processor for fake-media input. Waiting for the watchdog in that case
	// delays the first voiced batch by 750 ms. Automation is already an
	// explicitly non-device qualification lane, so use the universally
	// scheduled compatibility graph immediately; real browsers retain the
	// isolated AudioWorklet path below.
	if (typeof navigator !== "undefined" && navigator.webdriver) {
		return createScriptProcessorPcmCapture(context, stream, onFrame);
	}
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
