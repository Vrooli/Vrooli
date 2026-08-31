import { LibraryStringsProvider } from "@vrooli/react-component-library/useLocale/1";
import { i18n } from "./i18n";
import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { installChunkReloadGuard } from "@vrooli/api-base";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
import { API_BASE } from "./api/base";
import { AudioToolsProvider, createAudioToolsClient, registerVoiceTransport } from "./audio-integration";
import App from "./App";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { onProfilerRender } from "./lib/profiler";
import "./i18n";
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

if (import.meta.env.PROD && "serviceWorker" in navigator) {
  window.addEventListener("load", () => {
    navigator.serviceWorker.register("sw.js").catch((error: unknown) => {
      console.warn("Service worker registration failed", error);
    });
  });
}

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

registerVoiceTransport();

const rootEl = document.getElementById("root");
if (!rootEl) {
  throw new Error("Missing #root element in index.html");
}
const appRoot = rootEl;

const queryClient = new QueryClient();

function bootstrap() {
  ReactDOM.createRoot(appRoot).render(
    // vrooli:library-strings-provider start
    <LibraryStringsProvider translate={(key, fallback) => i18n.t(key, { defaultValue: fallback })}>
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
    </LibraryStringsProvider>,
    // vrooli:library-strings-provider end
  );
}

void bootstrap();
