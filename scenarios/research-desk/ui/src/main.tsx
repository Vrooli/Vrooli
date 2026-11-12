import { SpatialNavProvider } from "@vrooli/iframe-bridge/react";
import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
import { initAppNotifications } from "@vrooli/react-component-library/AppNotificationSettings/1";
import "./styles.css";

// INTEROP-CRITICAL: Embedded mounts identify themselves before React renders so
// the parent shell can route iframe bridge events to this scenario.
if (window.parent !== window) {
  initIframeBridgeChild({ appId: "research-desk" });
}

// INTEROP-CRITICAL: Spatial navigation is initialized at startup for embedded
// keyboard/gamepad control flows.
const spatialNav = initSpatialNav();
if (import.meta.hot) import.meta.hot.dispose(() => spatialNav.dispose());

// Re-confirms this installed app with Notification Hub and opens the screen a
// notification tap asked for. Does nothing until the scenario sets
// "appDelivery": true in .vrooli/notifications.json and the operator turns
// notifications on in Settings.
initAppNotifications();

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
  const [{ default: App }, { ErrorBoundary }, { i18n }, { LibraryStringsProvider }] = await Promise.all([
    import("./App"),
    import("./components/ErrorBoundary"),
    import("./i18n"),
    import("@vrooli/react-component-library/useLocale/1"),
  ]);

  ReactDOM.createRoot(appRoot).render(
    // vrooli:library-strings-provider start
    // Library components' own labels (menu, close, loading) go through the desk's translator.
    <LibraryStringsProvider translate={(key, fallback) => i18n.t(key, { defaultValue: fallback })}>
    <React.StrictMode>
      <SpatialNavProvider controller={spatialNav}>
        <QueryClientProvider client={queryClient}>
          {/* ErrorBoundary nests INSIDE QueryClientProvider (and after the
              ./i18n side-effect init above) so the localised fallback can
              call useTranslation. A render-time crash inside QueryClient
              itself would escape this boundary, but that failure mode is
              covered by react-query's own tests, not application logic. */}
          <ErrorBoundary>
            <App />
          </ErrorBoundary>
        </QueryClientProvider>
      </SpatialNavProvider>
    </React.StrictMode>
    </LibraryStringsProvider>
    // vrooli:library-strings-provider end
  );
}

void bootstrap();
