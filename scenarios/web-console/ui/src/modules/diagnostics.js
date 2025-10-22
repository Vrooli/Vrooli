import { state, debugFlags } from "./state.js";

let diagnosticsTimer = null;
let lastSnapshot = null;

function sampleDiagnostics() {
  const tabIds = state.tabs.map((tab) => tab.id);
  const domTerminals = document.querySelectorAll(".terminal-screen").length;
  const domNodes = document.querySelectorAll("*").length;
  const resourceEntries =
    typeof performance !== "undefined"
      ? performance.getEntriesByType("resource").length
      : null;
  const paintEntries =
    typeof performance !== "undefined"
      ? performance.getEntriesByType("paint").length
      : null;
  const totalEntries =
    typeof performance !== "undefined" ? performance.getEntries().length : null;
  const heap =
    typeof performance !== "undefined" && performance.memory
      ? {
          usedJSHeapSize: performance.memory.usedJSHeapSize,
          totalJSHeapSize: performance.memory.totalJSHeapSize,
          jsHeapSizeLimit: performance.memory.jsHeapSizeLimit,
        }
      : null;

  return {
    timestamp: Date.now(),
    tabCount: tabIds.length,
    tabIds,
    domTerminals,
    domNodes,
    resourceEntries,
    paintEntries,
    totalEntries,
    heap,
  };
}

function logDiagnostics(snapshot) {
  const messageParts = [
    `[workspace] tabs=${snapshot.tabCount} domTerminals=${snapshot.domTerminals} domNodes=${snapshot.domNodes}`,
  ];
  if (snapshot.resourceEntries !== null) {
    messageParts.push(`resources=${snapshot.resourceEntries}`);
  }
  if (snapshot.totalEntries !== null) {
    messageParts.push(`perfEntries=${snapshot.totalEntries}`);
  }
  if (snapshot.heap && Number.isFinite(snapshot.heap.usedJSHeapSize)) {
    const mb = (snapshot.heap.usedJSHeapSize / (1024 * 1024)).toFixed(2);
    messageParts.push(`heap=${mb}MB`);
  }
  console.debug(messageParts.join(" | "), snapshot.tabIds);
}

function hasMeaningfulChange(prev, next) {
  if (!prev) return true;
  if (prev.tabCount !== next.tabCount) return true;
  if (prev.domTerminals !== next.domTerminals) return true;
  if (prev.domNodes !== next.domNodes) return true;
  if (prev.resourceEntries !== next.resourceEntries) return true;
  if (prev.totalEntries !== next.totalEntries) return true;
  return false;
}

export function initializeDiagnostics() {
  if (typeof window === "undefined") return;
  if (!debugFlags.metrics) return;
  if (diagnosticsTimer !== null) return;

  const logSnapshot = () => {
    const snapshot = sampleDiagnostics();
    if (hasMeaningfulChange(lastSnapshot, snapshot)) {
      logDiagnostics(snapshot);
      lastSnapshot = snapshot;
      window.__WEB_CONSOLE_DIAGNOSTICS__ = snapshot;
    }
  };

  logSnapshot();
  diagnosticsTimer = window.setInterval(logSnapshot, 5000);

  window.addEventListener("beforeunload", () => {
    if (diagnosticsTimer !== null) {
      window.clearInterval(diagnosticsTimer);
      diagnosticsTimer = null;
    }
  });
}
