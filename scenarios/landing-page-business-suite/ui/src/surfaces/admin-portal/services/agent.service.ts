import { triggerAgentCustomization } from '../../../shared/api';

/**
 * Agent customization result from API
 */
export interface AgentResult {
  job_id: string;
  status: string;
  agent_id: string;
}

/**
 * Form state for agent customization
 */
export interface AgentFormState {
  brief: string;
  assets: string;
  preview: boolean;
}

/**
 * Default values for agent form
 */
export const DEFAULT_AGENT_FORM: AgentFormState = {
  brief: '',
  assets: '',
  preview: true,
};

/**
 * Validate the agent brief input
 */
export function validateBrief(brief: string): string | null {
  if (!brief.trim()) {
    return 'Please provide a brief for the agent';
  }
  return null;
}

/**
 * Parse assets string into array of URLs
 */
export function parseAssets(assetsText: string): string[] {
  return assetsText
    .split('\n')
    .map((a) => a.trim())
    .filter((a) => a.length > 0);
}

/**
 * Validate asset URLs (basic URL format validation)
 */
export function validateAssets(assets: string[]): string | null {
  for (const asset of assets) {
    try {
      new URL(asset);
    } catch {
      return `Invalid URL: ${asset}`;
    }
  }
  return null;
}

/**
 * Check if the form is ready for submission
 */
export function isFormValid(form: AgentFormState): boolean {
  return form.brief.trim().length > 0;
}

/**
 * Trigger agent customization via API
 */
export async function submitAgentCustomization(
  scenarioId: string,
  brief: string,
  assets: string[],
  preview: boolean
): Promise<AgentResult> {
  return triggerAgentCustomization(scenarioId, brief, assets, preview);
}

/**
 * Default scenario ID for landing page customization
 */
export const DEFAULT_SCENARIO_ID = 'landing-page';
