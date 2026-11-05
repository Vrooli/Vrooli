import { createClient } from "@connectrpc/connect";
import { onboardingTransport } from "./base";
import {
  OperatorInputsService,
  type Answer,
  type ListOperatorInputsResponse,
  type ResolveOperatorInputsResponse,
} from "@vrooli/proto-types/vrooli-onboarding/v1/operatorinputs/operatorinputs_pb";

const transport = onboardingTransport();
export const operatorInputsClient = createClient(OperatorInputsService, transport);

export function fetchOperatorInputs(target = "local"): Promise<ListOperatorInputsResponse> {
  return operatorInputsClient.listOperatorInputs({ target }) as unknown as Promise<ListOperatorInputsResponse>;
}

export function resolveOperatorInputs(
  answers: Answer[],
  target = "local",
): Promise<ResolveOperatorInputsResponse> {
  return operatorInputsClient.resolveOperatorInputs({ target, answers }) as unknown as Promise<ResolveOperatorInputsResponse>;
}
