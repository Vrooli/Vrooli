import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
import "./styles.css";

// INTEROP-CRITICAL: Embedded mounts identify themselves before React renders so
// the parent shell can route iframe bridge events to this scenario.
if (window.top !== window.self) {
  initIframeBridgeChild({ appId: "react-component-library" });
}

// INTEROP-CRITICAL: Spatial navigation is initialized at startup for embedded
// keyboard/gamepad control flows.
initSpatialNav();

if ("serviceWorker" in navigator) {
  window.addEventListener("load", () => {
    const base = import.meta.env.BASE_URL.replace(/\/$/, "");
    const swPath = base === "." ? "./sw.js" : `${base}/sw.js`;
    void navigator.serviceWorker.register(swPath);
  });
}

const rootEl = document.getElementById("root");
if (!rootEl) {
  throw new Error("Missing #root element in index.html");
}
const appRoot = rootEl;

const queryClient = new QueryClient();

// Vite's `import.meta.env.BASE_URL` is built from the `base` config option
// (set to `./` so emitted assets resolve relative to wherever the SPA is
// mounted — tunnels, proxies, sub-paths). react-router-dom's `basename`
// expects an *absolute* path; passing `./` makes BrowserRouter unable to
// match any URL, which silently renders an empty tree.
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
    import("./components/theme/ThemeProvider"),
    import("./i18n"),
  ]);

  ReactDOM.createRoot(appRoot).render(
    <React.StrictMode>
      <QueryClientProvider client={queryClient}>
        <ThemeProvider>
          <ErrorBoundary>
            <BrowserRouter basename={normalizeRouterBasename(import.meta.env.BASE_URL)}>
              <React.Profiler id="App" onRender={onProfilerRender}>
                <App />
              </React.Profiler>
            </BrowserRouter>
          </ErrorBoundary>
        </ThemeProvider>
      </QueryClientProvider>
    </React.StrictMode>
  );
}

void bootstrap();
