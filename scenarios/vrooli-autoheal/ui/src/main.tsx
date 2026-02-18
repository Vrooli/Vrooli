import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import App from "./App";
import { CheckMetadataProvider } from "./contexts/CheckMetadataContext";
import "./styles.css";

const queryClient = new QueryClient();
const rootElement = document.getElementById("root");

if (window.top !== window.self) {
  initIframeBridgeChild();
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
