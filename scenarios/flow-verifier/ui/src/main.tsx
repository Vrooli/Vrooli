import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
import "./styles.css";

initIframeBridgeChild();
initSpatialNav();

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
// match any URL, which silently renders an empty tree (and trips Lighthouse
// with NO_FCP). Strip the relative prefix and trailing slash so the router
// gets a basename it can actually match against.
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
