import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import { AudioToolsProvider, createAudioToolsClient } from "@audio-tools/embed";
import App from "./App";
import { fetchAudioToolsDiscovery } from "./api/discovery";
import "./i18n";
import "./styles.css";

const queryClient = new QueryClient();

// INTEROP-CRITICAL: iframe bridge must be initialized before React mount
// so parent scenarios receive the ready signal and can coordinate routing.
if (window.top !== window.self) {
  initIframeBridgeChild({ appId: "web-console" });
}

// AUDIO-TOOLS DISCOVERY: resolve the audio-tools base URL via the
// web-console backend BEFORE React mounts so the @audio-tools/embed
// lazy singleton (which reads window.__AUDIO_TOOLS_URL__) wires up
// against the right host. A discovery failure leaves the global unset
// and the AudioToolsProvider is mounted with an explicit empty-base
// client so consumer hooks render typed errors instead of crashing.
async function bootstrap(): Promise<{ baseUrl: string; unavailableReason: string }> {
  try {
    const ep = await fetchAudioToolsDiscovery();
    if (ep.available && ep.baseUrl) {
      window.__AUDIO_TOOLS_URL__ = ep.baseUrl;
      return { baseUrl: ep.baseUrl, unavailableReason: "" };
    }
    return { baseUrl: "", unavailableReason: ep.unavailableReason || "discovery_failed" };
  } catch (err) {
    return { baseUrl: "", unavailableReason: "discovery_failed" };
  }
}

void bootstrap().then(({ baseUrl, unavailableReason }) => {
  const rootEl = document.getElementById("root");
  if (!rootEl) throw new Error("Missing #root element in index.html");

  // Stash the unavailable reason for AudioUnavailableBanner consumers.
  window.__AUDIO_TOOLS_UNAVAILABLE_REASON__ = unavailableReason;

  // Construct an explicit client (rather than relying solely on the
  // window-global lazy singleton) so SSR / test paths can override via
  // <AudioToolsProvider client={...}>. baseUrl falls back to
  // "http://localhost:0" (the embed's sentinel) when discovery failed —
  // calls fail their HTTP request rather than crashing at construction.
  const audioToolsClient = createAudioToolsClient({ baseUrl: baseUrl || undefined });

  ReactDOM.createRoot(rootEl).render(
    <React.StrictMode>
      <QueryClientProvider client={queryClient}>
        <AudioToolsProvider client={audioToolsClient}>
          <App />
        </AudioToolsProvider>
      </QueryClientProvider>
    </React.StrictMode>,
  );
});
