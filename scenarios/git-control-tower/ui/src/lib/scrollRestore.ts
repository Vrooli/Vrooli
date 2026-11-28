/**
 * Restore a scroll container after its content has had time to measure.
 *
 * Virtualized lists update their total height after the commit that swaps the
 * content. Setting `scrollTop` on the first frame is clamped to the old, shorter
 * height, so the position is silently lost. Retry across a few frames until the
 * requested offset fits, then report the position the browser actually kept.
 */
export function restoreScrollPosition(
  element: HTMLElement,
  target: number,
  onResult: (actual: number) => void,
  timeoutMs = 900,
): () => void {
  if (!Number.isFinite(target) || target <= 0) {
    element.scrollTop = 0;
    onResult(0);
    return () => {};
  }
  const started = typeof performance !== 'undefined' ? performance.now() : Date.now();
  let frame = 0;
  let settled = false;
  const cleanup = () => {
    cancelAnimationFrame(frame);
    element.removeEventListener('wheel', cancel);
    element.removeEventListener('touchstart', cancel);
    element.removeEventListener('pointerdown', cancel);
  };
  const finish = (actual: number) => {
    if (settled) return;
    settled = true;
    cleanup();
    onResult(actual);
  };
  function cancel() {
    if (settled) return;
    settled = true;
    cleanup();
  }
  const attempt = () => {
    if (settled) return;
    element.scrollTop = target;
    const actual = element.scrollTop;
    const now = typeof performance !== 'undefined' ? performance.now() : Date.now();
    if (Math.abs(actual - target) < 2 || now - started >= timeoutMs) {
      finish(actual);
      return;
    }
    frame = requestAnimationFrame(attempt);
  };
  // A deliberate wheel/touch/pointer input wins over a late restore.
  element.addEventListener('wheel', cancel, { passive: true });
  element.addEventListener('touchstart', cancel, { passive: true });
  element.addEventListener('pointerdown', cancel);
  frame = requestAnimationFrame(attempt);
  return cancel;
}
