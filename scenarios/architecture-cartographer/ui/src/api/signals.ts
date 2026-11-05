import { createClient, type Client } from "@connectrpc/connect";
import { SignalsService } from "@vrooli/proto-types/architecture-cartographer/v1/signals/signals_pb";

import { transport } from "./client";

export const signalsClient: Client<typeof SignalsService> = createClient(SignalsService, transport);
