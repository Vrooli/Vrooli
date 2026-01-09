export const markAppReady = () => {
  try {
    document.documentElement.dataset.appReady = "true";
  } catch {
    // no-op: best-effort marker
  }
};

export const ensureReadyMarker = () => {
  try {
    if (document.querySelector('[data-testid="app-ready"]')) {
      return;
    }
    const marker = document.createElement("div");
    marker.setAttribute("data-testid", "app-ready");
    marker.setAttribute("aria-hidden", "true");
    marker.style.position = "absolute";
    marker.style.width = "1px";
    marker.style.height = "1px";
    marker.style.overflow = "hidden";
    marker.style.clip = "rect(0 0 0 0)";
    document.body.appendChild(marker);
  } catch {
    // no-op
  }
};
