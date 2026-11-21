import { createClient } from "@connectrpc/connect";
import { RehearsalService, type Rehearsal } from "@vrooli/proto-types/client-rehearsal/v1/rehearsals/rehearsals_pb";
import { transport } from "./client";

export const rehearsalClient = createClient(RehearsalService, transport);
export type { Rehearsal };
