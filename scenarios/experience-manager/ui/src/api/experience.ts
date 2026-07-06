import { createClient } from "@connectrpc/connect";
import type { MessageInitShape } from "@bufbuild/protobuf";
import {
  ContractService,
  StudioSessionService,
  type BindingSuggestion,
  type ExperienceFinding,
  type FileDiff,
  PageFormSchema,
  type RenderedVariant,
  type RenderSpecResponse,
  type SpecDocument,
  SpecVariantSchema,
  type ValidateScenarioResponse,
} from "@vrooli/proto-types/experience-manager/v1/contract/contract_pb";
import {
  ScenarioValidationService,
  type FixResponse,
} from "@vrooli/proto-types/scenario-validation/v1/validation_pb";

import { API_BASE, decodeApiError, transport } from "./client";

export const contractClient = createClient(ContractService, transport);
export const studioClient = createClient(StudioSessionService, transport);
export const scenarioValidationClient = createClient(ScenarioValidationService, transport);

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
    prd_refs?: string[];
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

export interface StudioPageDraft {
  id: string;
  title: string;
  purpose: string;
  routes: string[];
  prdRefs: string[];
  status: string;
  priorities: Array<{ statement: string; notes: string }>;
  states: Array<{ id: string; description: string }>;
  elements: Array<{ id: string; role: string; name: string; description: string }>;
  claims: Array<{
    id: string;
    type: string;
    statement: string;
    tier: string;
    elements: string[];
    states: string[];
    viewports: string[];
    locales: string[];
    rationale: string;
  }>;
  bindings: Array<{ elementId: string; testid: string; selector: string; note: string }>;
  sketchRegions: Array<{ id: string; elements: string[] }>;
}

export interface StudioVariantDraft {
  id: string;
  title: string;
  page: StudioPageDraft;
}

export interface StudioApplyResult {
  applied: boolean;
  diffs: FileDiff[];
  validation?: ValidateScenarioResponse;
}

export interface StudioVariantPreview {
  html: string;
  degradedReason: string;
  variants: RenderedVariant[];
}

export interface FindingsFixResult {
  preview: FixResponse;
  validation?: ValidateScenarioResponse;
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

export async function fetchFindings(scenario: string): Promise<ExperienceFinding[]> {
  const response = await contractClient.validateScenario({ scenario, includeExecution: true });
  return response.report?.findings ?? [];
}

export async function previewFindingsFixes(scenario: string): Promise<FixResponse> {
  return scenarioValidationClient.previewFix({ scenario, ruleIds: [] });
}

export async function applyFindingsFixes(scenario: string, ruleIds: string[]): Promise<FindingsFixResult> {
  const applied = await scenarioValidationClient.applyFix({ scenario, ruleIds });
  const validation = await contractClient.validateScenario({ scenario, includeExecution: true });
  return { preview: applied, validation };
}

export async function renderStudioSpec(scenario: string, page: string): Promise<RenderSpecResponse> {
  return studioClient.renderSpec({ scenario, page, mode: "wireframe" });
}

export async function suggestStudioBindings(scenario: string, page: string): Promise<BindingSuggestion[]> {
  const response = await studioClient.suggestBindings({ scenario, page, limit: 5 });
  return response.suggestions;
}

export async function compareStudioVariants(
  scenario: string,
  page: string,
  variants: StudioVariantDraft[],
): Promise<StudioVariantPreview> {
  const response = await studioClient.compareVariants({
    scenario,
    page,
    mode: "wireframe",
    variants,
  });
  return {
    html: response.html,
    degradedReason: response.degradedReason,
    variants: response.variants,
  };
}

export async function applyStudioDraft(scenario: string, page: StudioPageDraft): Promise<StudioApplyResult> {
  const started = await studioClient.startAuthoringSession({ scenario });
  const sessionID = started.session?.id ?? "";
  if (!sessionID) {
    throw new Error("studio session did not return an id");
  }
  await studioClient.submitPage({ sessionId: sessionID, page: page as MessageInitShape<typeof PageFormSchema> });
  const preview = await studioClient.previewSession({ sessionId: sessionID });
  const hasErrors = preview.validation?.report?.findings.some((finding) => finding.severity === "error") ?? false;
  if (hasErrors) {
    return {
      applied: false,
      diffs: preview.diffs,
      validation: preview.validation,
    };
  }
  const applied = await studioClient.applySession({ sessionId: sessionID });
  return {
    applied: true,
    diffs: applied.diffs,
    validation: applied.validation,
  };
}

export async function promoteStudioVariant(
  scenario: string,
  page: string,
  variant: StudioVariantDraft,
): Promise<StudioApplyResult> {
  const promoted = await studioClient.promoteVariant({
    scenario,
    page,
    variant: variant as MessageInitShape<typeof SpecVariantSchema>,
  });
  return {
    applied: true,
    diffs: promoted.diffs,
    validation: promoted.validation,
  };
}
