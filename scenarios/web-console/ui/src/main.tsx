import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import { AudioToolsProvider, createAudioToolsClient } from "./audio-integration";
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
// web-console backend BEFORE React mounts so the AudioToolsProvider wires
// the client against the right host. A discovery failure mounts the
// provider with a sentinel base URL ("http://localhost:0") and an
// unavailableReason prop so consumer hooks render typed errors and
// AudioUnavailableBanner can surface the reason.
async function bootstrap(): Promise<{ baseUrl: string; unavailableReason: string }> {
  try {
    const ep = await fetchAudioToolsDiscovery();
    if (ep.available && ep.baseUrl) {
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

  // Construct an explicit client. baseUrl falls back to the
  // "http://localhost:0" sentinel when discovery failed — calls fail
  // their HTTP request rather than crashing at module construction.
  const audioToolsClient = createAudioToolsClient({
    baseUrl: baseUrl || "http://localhost:0",
  });

  ReactDOM.createRoot(rootEl).render(
    <React.StrictMode>
      <QueryClientProvider client={queryClient}>
        <AudioToolsProvider client={audioToolsClient} unavailableReason={unavailableReason || undefined}>
          <App />
        </AudioToolsProvider>
      </QueryClientProvider>
    </React.StrictMode>,
  );
});
