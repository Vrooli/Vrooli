import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
import "./styles.css";

// ╔══════════════════════════════════════════════════════════════╗
// ║  INTEROP-CRITICAL: Iframe bridge initialization              ║
// ║                                                              ║
// ║  Must run BEFORE React mount so storage shimming and the     ║
// ║  bridge message channel are in place before components load. ║
// ║  The window.parent guard makes this a no-op outside an       ║
// ║  iframe (localhost / tunnel contexts).                       ║
// ╚══════════════════════════════════════════════════════════════╝
declare global {
  interface Window {
    __dtvBridgeInitialized?: boolean;
  }
}

if (
  typeof window !== "undefined" &&
  window.parent !== window &&
  !window.__dtvBridgeInitialized
) {
  let parentOrigin: string | undefined;
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin;
    }
  } catch {
    // Fall back to default origin when parsing fails.
  }
  initIframeBridgeChild({ parentOrigin, appId: "development-toolchain-validator" });
  window.__dtvBridgeInitialized = true;
}
initSpatialNav();

const rootEl = document.getElementById("root");
if (!rootEl) {
  throw new Error("Missing #root element in index.html");
}
const appRoot = rootEl;

const queryClient = new QueryClient();

async function bootstrap() {
  const [{ default: App }, { ErrorBoundary }, { onProfilerRender }, prefs] = await Promise.all([
    import("./App"),
    import("./shared/ui/composites/ErrorBoundary"),
    import("./lib/profiler"),
    import("./shared/stores/preferencesStore"),
    import("./i18n"),
  ]);
  // Mirror persisted preferences onto <html> immediately so design tokens
  // resolve correctly on first paint.
  prefs.applyPreferencesToDocument(prefs.usePreferencesStore.getState());

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
