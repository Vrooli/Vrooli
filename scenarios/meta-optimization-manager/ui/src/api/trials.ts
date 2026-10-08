import { createClient, type Client } from "@connectrpc/connect";
import { TrialsService } from "@vrooli/proto-types/meta-optimization-manager/v1/trials/trials_pb";

import { transport } from "./client";

/** Typed Connect client for the TrialsService (the empirical local-model gate). */
export const trialsClient: Client<typeof TrialsService> = createClient(TrialsService, transport);
