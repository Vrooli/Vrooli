import { SpatialNavProvider } from "@vrooli/iframe-bridge/react";
import { LibraryStringsProvider } from "@vrooli/react-component-library/useLocale/1";
import { BaseStyles } from "@vrooli/react-component-library/BaseStyles/1";
import { i18n } from "./i18n";
import React from "react";
import ReactDOM from "react-dom/client";
import { MutationCache, QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Code, ConnectError } from "@connectrpc/connect";
import { writeAccessQueryKey } from "./features/access/useWriteAccess";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
import "./styles.css";

// INTEROP-CRITICAL: Embedded mounts identify themselves before React renders so
// the parent shell can route iframe bridge events to this scenario.
if (window.parent !== window) {
  initIframeBridgeChild({ appId: "money-ledger" });
}

// INTEROP-CRITICAL: Spatial navigation is initialized at startup for embedded
// keyboard/gamepad control flows.
const spatialNav = initSpatialNav();
if (import.meta.hot) import.meta.hot.dispose(() => spatialNav.dispose());

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

// A write refused for identity means this page's idea of who it is has gone
// stale (a session expired, or was signed out elsewhere): re-ask, so the
// access notice and the disabled forms catch up at once.
const queryClient: QueryClient = new QueryClient({
  mutationCache: new MutationCache({
    onError: (error) => {
      if (error instanceof ConnectError && (error.code === Code.Unauthenticated || error.code === Code.PermissionDenied)) {
        void queryClient.invalidateQueries({ queryKey: writeAccessQueryKey });
      }
    },
  }),
});

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
  <SpatialNavProvider controller={spatialNav}>
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
  </SpatialNavProvider>
</React.StrictMode>
    </LibraryStringsProvider>,
    // vrooli:library-strings-provider end
  );
}

void bootstrap();
