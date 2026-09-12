/**
 * Utility functions for interacting with Vrooli CLI and scenarios
 */

import { logger } from './logger';
import { scenariosClient } from '../api/scenarios';

export interface ScenarioPortInfo {
  port: number;
  status: string;
  url: string;
}

/**
 * Get the port for a specific scenario via the ScenariosService Connect-RPC.
 * @param scenarioName - Name of the scenario to get port for
 * @returns Promise<ScenarioPortInfo | null>
 */
export async function getScenarioPort(scenarioName: string): Promise<ScenarioPortInfo | null> {
  try {
    const resp = await scenariosClient.getPort({ name: scenarioName });
    return {
      port: resp.port,
      status: resp.status,
      url: resp.url,
    };
  } catch (error: unknown) {
    logger.error(
      'Error getting port for scenario',
      { component: 'VrooliUtils', action: 'getScenarioPort', scenarioName },
      error,
    );
    return null;
  }
}

/**
 * Open a scenario in a new tab by getting its port and constructing the URL
 * @param scenarioName - Name of the scenario to open
 */
export async function openScenario(scenarioName: string): Promise<void> {
  const portInfo = await getScenarioPort(scenarioName);

  if (!portInfo || !portInfo.url) {
    alert(`Failed to get port information for ${scenarioName}. Make sure the scenario is running.`);
    return;
  }

  // Open in new tab
  window.open(portInfo.url, '_blank');
}

/**
 * Open the calendar scenario in a new tab
 */
export async function openCalendar(): Promise<void> {
  await openScenario('calendar');
}
