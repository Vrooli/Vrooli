import { createClient } from "@connectrpc/connect";
import { CapabilitiesService } from "@vrooli/proto-types/content-desk/v1/capabilities/capabilities_pb";
import { transport } from "./client";

export const capabilitiesClient = createClient(CapabilitiesService, transport);
