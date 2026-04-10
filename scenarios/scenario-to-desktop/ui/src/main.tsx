import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import App from "./App";
import "./styles.css";

const queryClient = new QueryClient();

if (window.top !== window.self) {
  initIframeBridgeChild({ appId: "scenario-to-desktop" });
}

const rootElement = document.getElementById("root");
if (!rootElement) {
  throw new Error("Root element not found. Check that index.html contains a <div id=\"root\"></div>.");
}
ReactDOM.createRoot(rootElement).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </React.StrictMode>
);
