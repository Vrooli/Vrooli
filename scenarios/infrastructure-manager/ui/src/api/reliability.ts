import { createClient } from "@connectrpc/connect";
import {
  CoverageService,
  Projection,
  type GetCoverageResponse,
  type ListCellsResponse,
} from "@vrooli/proto-types/infrastructure-manager/v1/coverage/coverage_pb";
import {
  ConditionService,
  type GetConditionResponse,
  type GetTrustDistributionResponse,
} from "@vrooli/proto-types/infrastructure-manager/v1/condition/condition_pb";
import {
  FocusService,
  type GetNextResponse,
  type GetEfficacyResponse,
} from "@vrooli/proto-types/infrastructure-manager/v1/focus/focus_pb";

import { transport } from "./client";

export const coverageClient = createClient(CoverageService, transport);
export const conditionClient = createClient(ConditionService, transport);
export const focusClient = createClient(FocusService, transport);

export function fetchCoverage(): Promise<GetCoverageResponse> {
  return coverageClient.getCoverage({});
}

export function fetchCells(): Promise<ListCellsResponse> {
  return coverageClient.listCells({});
}

export function fetchCondition(): Promise<GetConditionResponse> {
  return conditionClient.getCondition({ projection: Projection.AVAILABILITY });
}

export function fetchTrust(): Promise<GetTrustDistributionResponse> {
  return conditionClient.getTrustDistribution({ projection: Projection.AVAILABILITY });
}

export function fetchFocus(): Promise<GetNextResponse> {
  return focusClient.getNext({ limit: 25 });
}

export function fetchEfficacy(findingId: string): Promise<GetEfficacyResponse> {
  return focusClient.getEfficacy({ findingId });
}
