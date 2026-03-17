// DOC: docs/internal/SEAMS.md#voice-input-provider-seam
//
// Audio filter chain for speech-band isolation and level monitoring.

/**
 * Build a bandpass filter chain targeting the speech band (80Hz-8kHz).
 * Returns an AnalyserNode (for level monitoring) and a filtered MediaStream
 * suitable for MediaRecorder input.
 */
export function createAudioFilterChain(
  ctx: AudioContext,
  source: MediaStreamAudioSourceNode,
): { analyser: AnalyserNode; filteredStream: MediaStream } {
  const highpass = ctx.createBiquadFilter();
  highpass.type = "highpass";
  highpass.frequency.value = 80;
  highpass.Q.value = 0.707; // Butterworth

  const lowpass = ctx.createBiquadFilter();
  lowpass.type = "lowpass";
  lowpass.frequency.value = 8000;
  lowpass.Q.value = 0.707;

  const destination = ctx.createMediaStreamDestination();
  const analyser = ctx.createAnalyser();
  analyser.fftSize = 128;

  // Chain: source -> highpass -> lowpass -> destination
  //                                     +-> analyser (for level monitoring)
  source.connect(highpass);
  highpass.connect(lowpass);
  lowpass.connect(destination);
  lowpass.connect(analyser);

  return { analyser, filteredStream: destination.stream };
}
