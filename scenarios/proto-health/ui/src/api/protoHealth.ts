import { createClient } from "@connectrpc/connect";
import {
  ProtoHealthService,
  type DescribeScenarioProtosResponse,
  type ValidateScenarioResponse,
} from "@vrooli/proto-types/proto-health/v1/validation/validation_pb";

import { transport } from "./client";

export const protoHealthClient = createClient(ProtoHealthService, transport);

export async function validateScenario(scenario: string): Promise<ValidateScenarioResponse> {
  return protoHealthClient.validateScenario({ scenario });
}

export async function describeScenarioProtos(
  scenario: string,
): Promise<DescribeScenarioProtosResponse> {
  return protoHealthClient.describeScenarioProtos({ scenario });
}
