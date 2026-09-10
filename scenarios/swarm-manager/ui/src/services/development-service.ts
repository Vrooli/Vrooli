import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { DevelopmentService } from "@vrooli/proto-types/swarm-manager/v1/api/development_pb";
import { TransitionService } from "@vrooli/proto-types/swarm-manager/v1/api/transition_pb";
import { API_BASE } from "../lib/api-client";

const transport = createConnectTransport({ baseUrl: API_BASE });
export const developmentService = {
  ...createClient(DevelopmentService, transport),
  previewDevelopment: createClient(TransitionService, transport).previewDevelopment,
};
export type DevelopmentClient = typeof developmentService;
