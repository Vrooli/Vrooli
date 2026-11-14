// @ts-check
/**
 * Gesture transport only. The host composites the iframe; never scale, measure,
 * or change scrolling geometry inside the authored document during a pinch.
 * @param {Window} previewWindow
 * @returns {() => void}
 */
export function installHtmlPreviewPinch(previewWindow) {
  const document = previewWindow.document;
  const root = document.documentElement;
  const originalTouchAction = root.style.touchAction;
  let panEnabled = false;
  let pinching = false;
  let panning = false;
  /** @param {Record<string, unknown>} message */
  const send = (message) => { previewWindow.parent.postMessage({ type: "wc-html-gesture", ...message }, "*"); };
  /** @param {TouchEvent} event */
  const pair = (event) => {
    if (event.touches.length !== 2) return null;
    const a = event.touches[0];
    const b = event.touches[1];
    if (!a || !b) return null;
    const distance = Math.hypot(b.screenX - a.screenX, b.screenY - a.screenY);
    return distance > 0 ? { distance, x: (a.screenX + b.screenX) / 2, y: (a.screenY + b.screenY) / 2, localX: (a.clientX + b.clientX) / 2, localY: (a.clientY + b.clientY) / 2 } : null;
  };
  /** @param {TouchEvent} event */
  const onStart = (event) => {
    const touches = pair(event);
    pinching = !!touches;
    panning = !touches && panEnabled && event.touches.length === 1;
    if (touches) {
      if (event.cancelable) event.preventDefault();
      send({ kind: "pinch-start", ...touches });
    } else if (panning && event.touches[0]) {
      send({ kind: "pan-start", x: event.touches[0].screenX, y: event.touches[0].screenY });
    }
  };
  /** @param {TouchEvent} event */
  const onMove = (event) => {
    const touches = pair(event);
    const finger = event.touches[0];
    if (pinching && touches) {
      if (event.cancelable) event.preventDefault();
      send({ kind: "pinch-move", distance: touches.distance, x: touches.x, y: touches.y });
    } else if (panning && event.touches.length === 1 && finger) {
      if (event.cancelable) event.preventDefault();
      send({ kind: "pan-move", x: finger.screenX, y: finger.screenY });
    }
  };
  /** @param {TouchEvent} event */
  const onEnd = (event) => {
    pinching = false;
    const finger = event.touches.length === 1 ? event.touches[0] : undefined;
    panning = panEnabled && !!finger;
    send({ kind: "end" });
    if (panning && finger) send({ kind: "pan-start", x: finger.screenX, y: finger.screenY });
  };
  const onCancel = () => { pinching = false; panning = false; send({ kind: "cancel" }); };
  /** @param {WheelEvent} event */
  const onWheel = (event) => {
    if (!event.ctrlKey) return;
    if (event.cancelable) event.preventDefault();
    send({ kind: "wheel", localX: event.clientX, localY: event.clientY, delta: event.deltaY * (event.deltaMode === 1 ? 16 : event.deltaMode === 2 ? previewWindow.innerHeight : 1) });
  };
  /** @param {MessageEvent} event */
  const onMessage = (event) => {
    if (event.source !== previewWindow.parent || !event.data || event.data.type !== "wc-html-inspection") return;
    if (typeof event.data.pan === "boolean") {
      panEnabled = event.data.pan;
      root.style.touchAction = panEnabled ? "none" : "pan-x pan-y";
    }
    if (typeof event.data.scrollX === "number" && Number.isFinite(event.data.scrollX) && typeof event.data.scrollY === "number" && Number.isFinite(event.data.scrollY)) {
      previewWindow.scrollBy({ left: event.data.scrollX, top: event.data.scrollY, behavior: "instant" });
    }
  };
  root.style.touchAction = "pan-x pan-y";
  document.addEventListener("touchstart", onStart, { passive: false, capture: true });
  document.addEventListener("touchmove", onMove, { passive: false, capture: true });
  document.addEventListener("touchend", onEnd, true);
  document.addEventListener("touchcancel", onCancel, true);
  document.addEventListener("wheel", onWheel, { passive: false, capture: true });
  previewWindow.addEventListener("message", onMessage);
  send({ kind: "ready" });
  return () => {
    document.removeEventListener("touchstart", onStart, true);
    document.removeEventListener("touchmove", onMove, true);
    document.removeEventListener("touchend", onEnd, true);
    document.removeEventListener("touchcancel", onCancel, true);
    document.removeEventListener("wheel", onWheel, true);
    previewWindow.removeEventListener("message", onMessage);
    root.style.touchAction = originalTouchAction;
  };
}
