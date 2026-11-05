import { createClient, type Client } from "@connectrpc/connect";
import { FocusService } from "@vrooli/proto-types/meta-optimization-manager/v1/focus/focus_pb";

import { transport } from "./client";

/** Typed Connect client for the FocusService (gaps registry + prioritization). */
export const focusClient: Client<typeof FocusService> = createClient(FocusService, transport);
