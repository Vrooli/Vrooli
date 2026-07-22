// INTEROP-CRITICAL: interop-sensitive configuration below — do not remove without checking host-frame embedding.
import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { installChunkReloadGuard } from "@vrooli/api-base";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
import "./styles.css";

initSpatialNav();

// Code-split routes use lazy(); after a rebuild the old hashed chunks are
// gone, so a tab opened before the deploy would crash on its next
// navigation. This guard reloads once (rate-limited) instead.
installChunkReloadGuard();

// Keep install/offline behavior deterministic without registering a worker in
// development, where Vite's module graph is intentionally mutable.
if (import.meta.env.PROD && "serviceWorker" in navigator) {
  void navigator.serviceWorker.register("/sw.js");
}

const rootEl = document.getElementById("root");
if (!rootEl) {
  throw new Error("Missing #root element in index.html");
}
const appRoot = rootEl;

const queryClient = new QueryClient();

function normalizeRouterBasename(raw: string): string {
  const trimmed = raw.replace(/^\.\//, "/").replace(/\/$/, "");
  return trimmed === "" || trimmed === "." ? "/" : trimmed;
}

async function bootstrap() {
  const [
    { default: App },
    { ErrorBoundary },
    { onProfilerRender },
    { BrowserRouter },
    { ThemeProvider },
  ] = await Promise.all([
    import("./App"),
    import("./components/ErrorBoundary"),
    import("./lib/profiler"),
    import("react-router-dom"),
    import("./theme/ThemeProvider"),
    import("./i18n"),
  ]);

  ReactDOM.createRoot(appRoot).render(
    <React.StrictMode>
      <QueryClientProvider client={queryClient}>
        <ThemeProvider>
          <ErrorBoundary>
            <BrowserRouter
              basename={normalizeRouterBasename(import.meta.env.BASE_URL)}
              future={{ v7_startTransition: true, v7_relativeSplatPath: true }}
            >
              <React.Profiler id="App" onRender={onProfilerRender}>
                <App />
              </React.Profiler>
            </BrowserRouter>
          </ErrorBoundary>
        </ThemeProvider>
      </QueryClientProvider>
    </React.StrictMode>,
  );

  // The runtime validator treats the bridge READY signal as its screenshot
  // gate. Emit it only after React has committed an initial frame; otherwise
  // a fast host can capture the browser's blank document between navigation
  // and the first paint.
  requestAnimationFrame(() => {
    if (window.parent !== window) {
      initIframeBridgeChild({ appId: "ui-health" });
    }
  });
}

void bootstrap();
