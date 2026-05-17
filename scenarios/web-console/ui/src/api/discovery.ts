import { createClient } from "@connectrpc/connect";
import { DiscoveryService } from "@vrooli/proto-types/web-console/v1/discovery/discovery_pb";

import { transport } from "./client";

export const discoveryClient = createClient(DiscoveryService, transport);

export interface AudioToolsEndpoint {
  available: boolean;
  baseUrl: string;
  wsBaseUrl: string;
  unavailableReason: string;
}

/**
 * Resolve the audio-tools base URL via the web-console backend.
 * Browsers call this at boot and pass the resolved baseUrl to
 * createAudioToolsClient({ baseUrl }), then mount <AudioToolsProvider
 * client={...} unavailableReason={...}> at app root.
 *
 * Throws only on transport-level failures; an unreachable audio-tools
 * comes back as {available:false, unavailableReason:"<token>"}.
 */
export async function fetchAudioToolsDiscovery(): Promise<AudioToolsEndpoint> {
  const resp = await discoveryClient.getAudioToolsEndpoint({});
  return {
    available: resp.available,
    baseUrl: resp.baseUrl,
    wsBaseUrl: resp.wsBaseUrl,
    unavailableReason: resp.unavailableReason,
  };
}
