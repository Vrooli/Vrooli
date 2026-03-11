import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import App from "./App";
import "./styles.css";

const queryClient = new QueryClient();

// INTEROP-CRITICAL: iframe-bridge initialization must happen before render
// to enable Vrooli orchestration when running in iframe context
if (window.top !== window.self) {
  initIframeBridgeChild({ appId: 'reference-react-vite' });
}

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </React.StrictMode>
);
