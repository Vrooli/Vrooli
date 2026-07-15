import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { installChunkReloadGuard } from "@vrooli/api-base";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
import { API_BASE } from "./api/base";
import { AudioToolsProvider, createAudioToolsClient } from "./audio-integration";
import "./styles.css";

// INTEROP-CRITICAL: only init the iframe bridge when actually embedded.
// Calling it in the standalone UI sends bridge messages to window === parent
// and pollutes the host page's message bus. The appId is required so the
// parent can dispatch to the correct child.
if (typeof window !== "undefined" && window.parent !== window) {
  initIframeBridgeChild({ appId: "audio-tools" });
}
// INTEROP-CRITICAL: spatial-nav must init in both embedded and standalone
// modes so gamepad navigation works in the dev UI too.
initSpatialNav();

// Code-split routes use lazy(); after a rebuild the old hashed chunks are
// gone, so a tab opened before the deploy would crash on its next
// navigation. This guard reloads once (rate-limited) instead.
installChunkReloadGuard();

// Audio-tools' own UI calls its own API. Build a client targeting this
// scenario's own origin and mount the provider before any audio-integration
// hook fires.
const audioToolsClient = createAudioToolsClient({
  baseUrl: API_BASE,
});

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
            <AudioToolsProvider client={audioToolsClient}>
              <App />
            </AudioToolsProvider>
          </React.Profiler>
        </ErrorBoundary>
      </QueryClientProvider>
    </React.StrictMode>
  );
}

void bootstrap();
