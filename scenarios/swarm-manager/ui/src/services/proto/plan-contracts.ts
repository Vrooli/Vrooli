import { PlanBoardResponseSchema } from "@vrooli/proto-types/swarm-manager/v1/api/plan_pb";
import { createProtoSchema } from "./shared";

export const planBoardResponseSchema = createProtoSchema(
  PlanBoardResponseSchema,
  "plan",
);
