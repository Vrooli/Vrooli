// Connect-RPC discovery client. Resolves the audio-tools base URL via
// the swarm-manager backend so the browser never composes scenario
// URLs on its own (see feedback_scenario_url_resolution.md).
//
// Used at boot from main.tsx; the result feeds AudioToolsProvider.

import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { DiscoveryService } from "@vrooli/proto-types/swarm-manager/v1/discovery/discovery_pb";

const transport = createConnectTransport({
  baseUrl: typeof window !== "undefined" ? window.location.origin : "",
});

export const discoveryClient = createClient(DiscoveryService, transport);

export interface AudioToolsEndpoint {
  available: boolean;
  baseUrl: string;
  wsBaseUrl: string;
  unavailableReason: string;
}

export async function fetchAudioToolsDiscovery(): Promise<AudioToolsEndpoint> {
  const resp = await discoveryClient.getAudioToolsEndpoint({});
  return {
    available: resp.available,
    baseUrl: resp.baseUrl,
    wsBaseUrl: resp.wsBaseUrl,
    unavailableReason: resp.unavailableReason,
  };
}
