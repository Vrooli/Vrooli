import { createClient, type Client } from "@connectrpc/connect";
import { ManifestService } from "@vrooli/proto-types/architecture-cartographer/v1/manifest/manifest_pb";

import { transport } from "./client";

export const manifestClient: Client<typeof ManifestService> = createClient(ManifestService, transport);
