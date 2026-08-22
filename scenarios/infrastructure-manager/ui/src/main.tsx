import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
import "./styles.css";

// INTEROP-CRITICAL: Embedded mounts identify themselves before React renders so
// the parent shell can route iframe bridge events to this scenario.
if (window.parent !== window) {
  initIframeBridgeChild({ appId: "infrastructure-manager" });
}

// INTEROP-CRITICAL: Spatial navigation is initialized at startup for embedded
// keyboard/gamepad control flows.
initSpatialNav();

if (import.meta.env.PROD && "serviceWorker" in navigator) {
  window.addEventListener("load", () => {
    navigator.serviceWorker.register("sw.js").catch((error: unknown) => {
      console.warn("Service worker registration failed", error);
    });
  });
}

const rootEl = document.getElementById("root");
if (!rootEl) {
  throw new Error("Missing #root element in index.html");
}
const appRoot = rootEl;

/**
 * Reads do not retry.
 *
 * This scenario's whole doctrine is that a source which did not answer IS the
 * reading: every surface renders an unreachable source as instrument state,
 * with its reason, rather than as a plant fault. Retrying three times with
 * backoff replaces that fact with a spinner for several seconds, which is both
 * a worse answer for the operator and long enough to leave a required
 * experience surface unsettled.
 *
 * `refetchOnWindowFocus` is off for the same reason: a board that silently
 * re-reads when the window regains focus can change its figures without the
 * reader noticing that anything was re-measured.
 */
const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: false, refetchOnWindowFocus: false },
  },
});

async function bootstrap() {
  const [{ default: App }, { ErrorBoundary }, { onProfilerRender }] = await Promise.all([
    import("./App"),
    import("./components/ErrorBoundary"),
    import("./lib/profiler"),
    import("./i18n"),
  ]);

  ReactDOM.createRoot(appRoot).render(
    <React.StrictMode>
      <QueryClientProvider client={queryClient}>
        {/* ErrorBoundary nests INSIDE QueryClientProvider (and after the
            ./i18n side-effect init above) so the localised fallback can
            call useTranslation. A render-time crash inside QueryClient
            itself would escape this boundary, but that failure mode is
            covered by react-query's own tests, not application logic. */}
        <ErrorBoundary>
          {/* Top-level Profiler boundary. Inert in regular prod (react-dom strips
              the profiling hook); emits user_timing entries via onProfilerRender
              when the perf-build channel is active. See lib/profiler.ts. Add
              inner <Profiler> boundaries around heavy subtrees as needed; do
              not remove this one. */}
          <React.Profiler id="App" onRender={onProfilerRender}>
            <App />
          </React.Profiler>
        </ErrorBoundary>
      </QueryClientProvider>
    </React.StrictMode>
  );
}

void bootstrap();
