import { createClient, type Client } from "@connectrpc/connect";
import { ConvergenceService } from "@vrooli/proto-types/meta-optimization-manager/v1/convergence/convergence_pb";

import { transport } from "./client";

/** Typed Connect client for the ConvergenceService (template & reference fitness). */
export const convergenceClient: Client<typeof ConvergenceService> = createClient(
  ConvergenceService,
  transport,
);
