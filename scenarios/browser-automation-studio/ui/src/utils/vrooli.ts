/**
 * Utility functions for interacting with Vrooli CLI and scenarios
 */

import { getConfig } from '../config';
import { logger } from './logger';

export interface ScenarioPortInfo {
  port: number;
  status: string;
  url: string;
}

const parseScenarioPortInfo = (value: unknown): ScenarioPortInfo | null => {
  if (!value || typeof value !== 'object') {
    return null;
  }
  const obj = value as Record<string, unknown>;
  if (typeof obj.port !== 'number' || typeof obj.status !== 'string' || typeof obj.url !== 'string') {
    return null;
  }
  return {
    port: obj.port,
    status: obj.status,
    url: obj.url,
  };
};

/**
 * Get the port for a specific scenario using vrooli CLI
 * @param scenarioName - Name of the scenario to get port for
 * @returns Promise<ScenarioPortInfo | null>
 */
export async function getScenarioPort(scenarioName: string): Promise<ScenarioPortInfo | null> {
  try {
    const config = await getConfig();
    const response = await fetch(`${config.API_URL}/scenarios/${encodeURIComponent(scenarioName)}/port`);

    if (!response.ok) {
      logger.error('Failed to get port for scenario', { component: 'VrooliUtils', action: 'getScenarioPort', scenarioName, status: response.statusText });
      return null;
    }

    const data: unknown = await response.json();
    const portInfo = parseScenarioPortInfo(data);
    if (!portInfo) {
      logger.error('Invalid port info response', { component: 'VrooliUtils', action: 'getScenarioPort', scenarioName });
      return null;
    }
    return portInfo;
  } catch (error: unknown) {
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
