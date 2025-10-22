import { debugFlags } from "./state.js";

export function debugLog(tab, message, extra) {
  if (!tab || typeof window === "undefined") return;
  const runtimeDebug = window.__WEB_CONSOLE_DEBUG__ || {};
  const enabled = Boolean(
    runtimeDebug.inputTelemetry || debugFlags.inputTelemetry,
  );
  if (!enabled) return;
  const payload = extra ? { ...extra } : undefined;
  const prefix = `[web-console:${tab.id || "unknown"}]`;
  if (payload) {
    console.debug(prefix, message, payload);
  } else {
    console.debug(prefix, message);
  }
}
