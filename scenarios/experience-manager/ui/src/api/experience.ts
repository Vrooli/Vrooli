import { createClient } from "@connectrpc/connect";
import {
  ContractService,
  StudioSessionService,
  type SpecDocument,
} from "@vrooli/proto-types/experience-manager/v1/contract/contract_pb";

import { API_BASE, decodeApiError, transport } from "./client";

export const contractClient = createClient(ContractService, transport);
export const studioClient = createClient(StudioSessionService, transport);

export interface ExperienceStateSpec {
  id: string;
  description?: string;
}

export interface ExperienceClaimSpec {
  id: string;
  type: string;
  tier: string;
  statement?: string;
  elements?: string[];
  states?: string[];
}

export interface ExperiencePageSpec {
  page: {
    id: string;
    title: string;
    routes?: string[];
    purpose?: string;
  };
  priorities?: Array<{ statement: string; notes?: string }>;
  states?: ExperienceStateSpec[];
  elements?: Array<{ id: string; role?: string; name?: string; description?: string }>;
  claims?: ExperienceClaimSpec[];
}

export interface ScenarioSpecPage {
  document: SpecDocument;
  spec: ExperiencePageSpec;
}

export interface EvidenceQuery {
  scenario: string;
  page: string;
  claim?: string;
  limit?: number;
}

export interface ReconciliationEvidenceRow {
  id: string;
  scenario: string;
  page: string;
  route: string;
  state: string;
  claim: string;
  claimType: string;
  verdict: string;
  captureRef: string;
  axNodeJson: string;
  message: string;
  checkedAt: string;
}

export async function fetchFleet() {
  return contractClient.listFleet({});
}

export async function fetchScenarioSpec(scenario: string): Promise<ScenarioSpecPage[]> {
  const list = await studioClient.listSpec({ scenario });
  const pages = await Promise.all(
    list.pages.map(async (document) => {
      const detail = await studioClient.showSpec({ scenario, page: document.id });
      return {
        document,
        spec: JSON.parse(detail.json) as ExperiencePageSpec,
      };
    }),
  );
  return pages;
}

export async function fetchEvidence({ scenario, page, claim = "", limit = 50 }: EvidenceQuery): Promise<ReconciliationEvidenceRow[]> {
  const response = await fetch(
    `${API_BASE.replace(/\/$/, "")}/vrooli.experience_manager.v1.contract.StudioSessionService/ListEvidence`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ scenario, page, claim, limit }),
      cache: "no-store",
    },
  );
  if (!response.ok) {
    throw await decodeApiError(response);
  }
  const body = (await response.json()) as { evidence?: ReconciliationEvidenceRow[] };
  return body.evidence ?? [];
}

export async function recaptureScenario(scenario: string) {
  return contractClient.validateScenario({ scenario, includeExecution: true });
}
