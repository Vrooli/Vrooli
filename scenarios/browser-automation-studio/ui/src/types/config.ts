/**
 * Configuration types for Vrooli Ascension
 */

export interface Config {
  API_URL: string;
  WS_URL: string;
  API_PORT: string;
  UI_PORT: string;
  WS_PORT: string;
  /** Playwright driver port for direct WebSocket frame streaming (driver port + 1 = frame server) */
  PLAYWRIGHT_DRIVER_PORT?: number;
}
