import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
import "./styles.css";

initIframeBridgeChild();
initSpatialNav();

// Audio-tools' own UI calls its own API. Setting the global before any
// @audio-tools/embed module loads ensures the embed's lazy singleton
// targets this scenario's own origin rather than the http://localhost:0
// sentinel. Embed consumers may still override with <AudioToolsProvider>.
declare global {
  interface Window {
    __AUDIO_TOOLS_URL__?: string;
  }
}
if (typeof window !== "undefined" && !window.__AUDIO_TOOLS_URL__) {
  window.__AUDIO_TOOLS_URL__ = window.location.origin;
}

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
