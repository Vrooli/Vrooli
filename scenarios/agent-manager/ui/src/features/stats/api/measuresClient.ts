import { createClient, type Client } from "@connectrpc/connect";
import { createScenarioConnectTransport } from "@vrooli/api-base";
import { MeasuresService } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { resolveAgentManagerApiBase } from "../../../lib/api";

// The statistics UI has a single typed transport for durable analytics. Keeping
// this client separate from the legacy REST client makes the product cutover
// explicit and prevents a second handwritten wire contract from appearing.
export const measuresClient: Client<typeof MeasuresService> = createClient(
  MeasuresService,
  createScenarioConnectTransport({ baseUrl: resolveAgentManagerApiBase() }),
);
