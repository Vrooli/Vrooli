import { LibraryStringsProvider } from "@vrooli/react-component-library/useLocale/1";
import { BaseStyles } from "@vrooli/react-component-library/BaseStyles/1";
import { i18n } from "./i18n";
import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { installChunkReloadGuard } from "@vrooli/api-base";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
import "./styles.css";

// Code-split routes use lazy(); after a rebuild the old hashed chunks are
// gone, so a tab opened before the deploy would crash on its next navigation.
// Recover once, rate-limited, by loading the current index/chunk manifest.
installChunkReloadGuard();

// INTEROP-CRITICAL: Embedded mounts identify themselves before React renders so
// the parent shell can route iframe bridge events to this scenario.
if (window.parent !== window) {
  initIframeBridgeChild({ appId: "device-control" });
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

const queryClient = new QueryClient();

async function bootstrap() {
  const [{ default: App }, { ErrorBoundary }, { onProfilerRender }] = await Promise.all([
    import("./App"),
    import("./components/ErrorBoundary"),
    import("./lib/profiler"),
    import("./i18n"),
  ]);

  ReactDOM.createRoot(appRoot).render(
    // vrooli:library-strings-provider start
    <LibraryStringsProvider translate={(key, fallback) => i18n.t(key, { defaultValue: fallback })}>
      <BaseStyles />
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
    </LibraryStringsProvider>,
    // vrooli:library-strings-provider end
  );
}

void bootstrap();
