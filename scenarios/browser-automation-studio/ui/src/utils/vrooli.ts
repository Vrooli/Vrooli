/**
 * Utility functions for interacting with Vrooli CLI and scenarios
 */

import { logger } from './logger';

export interface ScenarioPortInfo {
  port: number;
  status: string;
  url: string;
}

/**
 * Get the port for a specific scenario using vrooli CLI
 * @param scenarioName - Name of the scenario to get port for
 * @returns Promise<ScenarioPortInfo | null>
 */
export async function getScenarioPort(scenarioName: string): Promise<ScenarioPortInfo | null> {
  try {
    // In a real implementation, this would make an API call to a backend endpoint
    // that executes the vrooli CLI command server-side, since browsers can't execute shell commands
    const response = await fetch(`/api/v1/scenarios/${scenarioName}/port`);

    if (!response.ok) {
      logger.error('Failed to get port for scenario', { component: 'VrooliUtils', action: 'getScenarioPort', scenarioName, status: response.statusText });
      return null;
    }

    return await response.json();
  } catch (error) {
    logger.error('Error getting port for scenario', { component: 'VrooliUtils', action: 'getScenarioPort', scenarioName }, error);
    return null;
  }
}

/**
 * Open a scenario in a new tab by getting its port and constructing the URL
 * @param scenarioName - Name of the scenario to open
 */
export async function openScenario(scenarioName: string): Promise<void> {
  const portInfo = await getScenarioPort(scenarioName);
  
  if (!portInfo) {
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