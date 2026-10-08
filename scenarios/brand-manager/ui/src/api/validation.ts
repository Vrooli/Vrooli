import { createClient } from "@connectrpc/connect";
import { ScenarioValidationService } from "@vrooli/proto-types/scenario-validation/v1/validation_pb";

import { transport } from "./client";

export const validationClient = createClient(ScenarioValidationService, transport);

// This mirrors the common.v1.PhasePresentation display fields exactly without
// deriving any maturity semantics. Domain-specific branding detail remains on
// its existing richer surfaces.
export interface SharedPhasePresentation {
  contractVersion: string;
  provider: string;
  phase: string;
  currentLevel: string;
  currentLevelLabel: string;
  nextAction: string;
  northStar: string;
  documentationTopics: string[];
  capabilities: Array<{
    id: string;
    label: string;
    currentLevel: string;
    currentLevelLabel: string;
    nextLevel: string;
    nextUnlock: string;
  }>;
}

export async function fetchBrandingValidation() {
  return validationClient.validateScenario({ scenario: "brand-manager" });
}

export function sharedPresentationFromResponse(response: unknown): SharedPhasePresentation | undefined {
  return (response as { assessment?: { presentation?: SharedPhasePresentation } } | undefined)?.assessment?.presentation;
}
