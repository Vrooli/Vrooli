import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import App from "./App";
import { CheckMetadataProvider } from "./shared/contexts/CheckMetadataContext";
import "./shared/theme/tokens.css";
import "./styles.css";

const queryClient = new QueryClient();
const rootElement = document.getElementById("root");

if (window.top !== window.self) {
  initIframeBridgeChild({ appId: "vrooli-autoheal" });
}

if (!rootElement) {
  throw new Error("Root element not found");
}

ReactDOM.createRoot(rootElement).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <CheckMetadataProvider>
        <App />
      </CheckMetadataProvider>
    </QueryClientProvider>
  </React.StrictMode>
);
