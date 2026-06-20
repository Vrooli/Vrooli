import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
import "./styles.css";

// INTEROP-CRITICAL: Embedded mounts identify themselves before React renders so
// the parent shell can route iframe bridge events to this scenario.
if (window.parent !== window) {
  initIframeBridgeChild({ appId: "structure-health" });
}

// INTEROP-CRITICAL: Spatial navigation is initialized at startup for embedded
// keyboard/gamepad control flows.
initSpatialNav();

const rootEl = document.getElementById("root");
if (!rootEl) {
  throw new Error("Missing #root element in index.html");
}
const appRoot = rootEl;

const queryClient = new QueryClient();

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
