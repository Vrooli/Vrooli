import { createClient } from "@connectrpc/connect";
import { CapabilitiesService } from "@vrooli/proto-types/web-console/v1/capabilities/capabilities_pb";

import { transport } from "./client";

// capabilitiesClient is the Connect-Web client for CapabilitiesService.
// UI code imports this directly; the legacy fetch helpers in lib/api.ts
// are shims that delegate here.
export const capabilitiesClient = createClient(CapabilitiesService, transport);
