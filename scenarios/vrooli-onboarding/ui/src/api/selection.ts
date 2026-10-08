import { createClient } from "@connectrpc/connect";
import { create } from "@bufbuild/protobuf";
import { onboardingTransport } from "./base";
import {
  SelectionService,
  type AcceptRecommendationResponse,
  type GetClosureResponse,
  type GetCoreSetResponse,
  type GetRecommendationResponse,
  type GetUnionResponse,
  type ListScenariosResponse,
} from "@vrooli/proto-types/vrooli-onboarding/v1/selection/selection_pb";
import { SelectionSchema } from "@vrooli/proto-types/setup/v1/selection_pb";

const client = createClient(SelectionService, onboardingTransport());

export function fetchScenarios(target = "local"): Promise<ListScenariosResponse> {
  return client.listScenarios({ target }) as unknown as Promise<ListScenariosResponse>;
}

export function fetchCoreSet(seed?: Iterable<string>, target = "local"): Promise<GetCoreSetResponse> {
  return client.getCoreSet({ target, seed: Array.from(seed ?? []).sort() }) as unknown as Promise<GetCoreSetResponse>;
}

export function fetchRecommendation(target = "local"): Promise<GetRecommendationResponse> {
  return client.getRecommendation({ target }) as unknown as Promise<GetRecommendationResponse>;
}

export function acceptRecommendation(
  target = "local",
  profile = "",
  scenarios?: string[],
): Promise<AcceptRecommendationResponse> {
  const selection = scenarios
    ? create(SelectionSchema, {
        schemaVersion: "v1",
        target,
        scenarios: Array.from(new Set(scenarios)).sort(),
        optionalResources: [],
        coreSeed: [],
        trustedBase: [],
        hostTools: [],
        hostSafeguards: [],
        credentialAddresses: [],
        trustPosture: "",
        updateControl: "",
        sessionMode: "",
        operatingMode: {},
        apply: true,
        capacityPosture: "",
        transientHeadroomReserveBytes: 0n,
        resourceCapacity: {},
        fieldPresence: {},
      })
    : undefined;
  return client.acceptRecommendation({ target, profile, selection }) as unknown as Promise<AcceptRecommendationResponse>;
}

export function fetchClosure(target = "local"): Promise<GetClosureResponse> {
  return client.getClosure({ target }) as unknown as Promise<GetClosureResponse>;
}

export function fetchUnion(target = "local"): Promise<GetUnionResponse> {
  return client.getUnion({ target }) as unknown as Promise<GetUnionResponse>;
}
