export function validateObservation(observation, maxFrameBytes) {
  const baseline = observation?.baseline;
  const slow = observation?.slowReader;
  // Allow at most one frame of finite-window count uncertainty; the independent
  // 9,000-unique-frame minimum remains exact.
  const boundaryFps = baseline ? 1000 / baseline.durationMs : 0;
  if (
    !baseline ||
    baseline.durationMs < 300_000 ||
    baseline.renderedFrames < 9_000 ||
    baseline.uniqueFixtureFrames < 9_000 ||
    baseline.renderedFps + boundaryFps < 30 ||
    baseline.renderedFrames / (baseline.durationMs / 1000) + boundaryFps < 30 ||
    baseline.p95FrameAgeMs > 100 ||
    baseline.maxFrameBytes <= 0 ||
    baseline.maxFrameBytes > maxFrameBytes ||
    baseline.p95DecodeMs < 0 ||
    baseline.p95DecodeMs > 100 ||
    baseline.maxDecodeMs < baseline.p95DecodeMs ||
    baseline.maxDecodeMs > 250
  ) {
    throw new Error('five-minute live motion observation is incomplete or out of band');
  }

  if (
    !slow ||
    slow.durationMs < 15_000 ||
    slow.stallCount < 2 ||
    slow.receivedFrames <= slow.renderedFrames ||
    slow.decodedFrames < slow.renderedFrames ||
    slow.renderedFrames <= 0 ||
    slow.maxConcurrentDecodes !== 1 ||
    !slow.maxApiQueueBytes ||
    slow.maxApiQueueBytes > maxFrameBytes ||
    slow.samples.some((sample) => sample.frameAgeMs > 1000 || sample.frameBytes > maxFrameBytes)
  ) {
    throw new Error('live slow-reader observation is incomplete or exceeded a bound');
  }
}
