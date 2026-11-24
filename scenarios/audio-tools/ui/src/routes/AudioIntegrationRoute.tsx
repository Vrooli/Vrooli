import type { ReactNode } from "react";

import { API_BASE } from "../api/base";
import { AudioToolsProvider, createAudioToolsClient } from "../audio-integration";

const audioToolsClient = createAudioToolsClient({ baseUrl: API_BASE });

/** Mount the RPC client only on routes that use the browser audio adapter. */
export function AudioIntegrationRoute({ children }: { children: ReactNode }) {
  return <AudioToolsProvider client={audioToolsClient}>{children}</AudioToolsProvider>;
}

export default AudioIntegrationRoute;
