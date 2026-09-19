import { createClient, type Client } from "@connectrpc/connect";
import { ApplyService } from "@vrooli/proto-types/architecture-cartographer/v1/apply/apply_pb";

import { transport } from "./client";

export const applyClient: Client<typeof ApplyService> = createClient(ApplyService, transport);
