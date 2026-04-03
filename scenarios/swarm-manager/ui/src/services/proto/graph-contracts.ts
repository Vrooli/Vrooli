import { GraphResponseSchema } from "@vrooli/proto-types/swarm-manager/v1/api/graph_pb";
import { createProtoSchema } from "./shared";

export const graphResponseSchema = createProtoSchema(
  GraphResponseSchema,
  "graph"
);
