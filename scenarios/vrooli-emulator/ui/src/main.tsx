import { SpatialNavProvider } from "@vrooli/iframe-bridge/react";
import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
import App from "./App";
import "./styles.css";

const queryClient = new QueryClient();

if (window.top !== window.self) {
  initIframeBridgeChild();
}

const spatialNav = initSpatialNav();
if (import.meta.hot) import.meta.hot.dispose(() => spatialNav.dispose());

const rootElement = document.getElementById("root");
if (!rootElement) {
  throw new Error("#root element is missing from index.html");
}

ReactDOM.createRoot(rootElement).render(
  <React.StrictMode>
    <SpatialNavProvider controller={spatialNav}>
      <QueryClientProvider client={queryClient}>
        <App />
      </QueryClientProvider>
    </SpatialNavProvider>
  </React.StrictMode>,
);
