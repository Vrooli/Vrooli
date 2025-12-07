import { apiCall } from './common';

export function triggerAgentCustomization(scenarioId: string, brief: string, assets: string[], preview: boolean = true) {
  return apiCall<{ job_id: string; status: string; agent_id: string }>('/customize', {
    method: 'POST',
    body: JSON.stringify({ scenario_id: scenarioId, brief, assets, preview }),
  });
}
