import { createClient, type Client } from "@connectrpc/connect";
import { ConflictsService } from "@vrooli/proto-types/architecture-cartographer/v1/conflicts/conflicts_pb";

import { transport } from "./client";

export const conflictsClient: Client<typeof ConflictsService> = createClient(ConflictsService, transport);
