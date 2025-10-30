/**
 * Configuration types for Browser Automation Studio
 */

export interface ConfigResponse {
  apiUrl: string;
  wsUrl?: string;
  apiPort?: string;
  wsPort?: string;
}

export interface Config {
  API_URL: string;
  WS_URL: string;
  API_PORT: string;
  UI_PORT: string;
  WS_PORT: string;
}
