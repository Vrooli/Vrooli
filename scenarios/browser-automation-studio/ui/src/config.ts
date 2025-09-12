// Environment configuration with strict validation
// No fallback values allowed - fail fast if configuration is missing

interface Config {
  API_URL: string;
  WS_URL: string;
  API_PORT: string;
  UI_PORT: string;
  WS_PORT: string;
}

function getRequiredEnvVar(key: string): string {
  const value = import.meta.env[key];
  if (!value) {
    throw new Error(`Required environment variable ${key} is not set. Please ensure the browser-automation-studio is running through the Vrooli lifecycle system.`);
  }
  return value;
}

// Validate and export configuration
export const config: Config = {
  API_URL: getRequiredEnvVar('VITE_API_URL'),
  WS_URL: getRequiredEnvVar('VITE_WS_URL'),
  API_PORT: getRequiredEnvVar('VITE_API_PORT'),
  UI_PORT: getRequiredEnvVar('VITE_UI_PORT'),
  WS_PORT: getRequiredEnvVar('VITE_WS_PORT'),
};

// Export for convenience
export const API_BASE = config.API_URL;
export const WS_BASE = config.WS_URL;