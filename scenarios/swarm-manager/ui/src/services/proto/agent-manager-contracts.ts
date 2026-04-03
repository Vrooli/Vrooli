import { AgentManagerStatusResponseSchema } from "@vrooli/proto-types/swarm-manager/v1/api/agent_manager_pb";
import { createProtoSchema } from "./shared";

export const agentManagerStatusResponseSchema = createProtoSchema(
  AgentManagerStatusResponseSchema,
  "agent-manager status"
);
