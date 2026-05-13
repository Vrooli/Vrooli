import { createClient } from "@connectrpc/connect";
import { SessionsService } from "@vrooli/proto-types/web-console/v1/sessions/sessions_pb";

import { transport } from "./client";

// sessionsClient is the Connect-Web client for SessionsService. UI code
// imports this directly; the legacy fetch helpers in lib/api.ts are
// shims that delegate here and normalize the camelCase proto shape to
// the existing snake_case wire types.
export const sessionsClient = createClient(SessionsService, transport);
