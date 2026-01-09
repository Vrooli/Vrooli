/**
 * Constants for the issue report dialog and related components
 */

// Maximum lines/events to include in reports
export const REPORT_APP_LOGS_MAX_LINES = 200;
export const REPORT_CONSOLE_LOGS_MAX_LINES = 150;
export const REPORT_NETWORK_MAX_EVENTS = 150;

// Maximum lengths for payload field values to prevent oversized reports
export const MAX_CONSOLE_TEXT_LENGTH = 2000;
export const MAX_NETWORK_URL_LENGTH = 2048;
export const MAX_NETWORK_ERROR_LENGTH = 1500;
export const MAX_NETWORK_REQUEST_ID_LENGTH = 128;

// Bridge failure labels for user-facing error messages
export const BRIDGE_FAILURE_LABELS: Record<string, string> = {
  HELLO: 'Preview never received the HELLO handshake from the iframe.',
  READY: 'Iframe bridge did not send the READY signal.',
  'SPA hooks': 'Single-page navigation hook did not respond to a test navigation.',
  'BACK/FWD': 'History navigation commands were not acknowledged by the iframe.',
  CHECK_FAILED: 'Runtime bridge check could not complete. Try refreshing the preview.',
  NO_IFRAME: 'Preview iframe was unavailable when the runtime check executed.',
  CAP_LOGS: 'Preview bridge did not advertise console log capture capability. Restart the scenario to refresh the UI bundle.',
  CAP_NETWORK: 'Preview bridge did not advertise network request capture capability. Restart the scenario to refresh the UI bundle.',
};

// Detailed information about bridge capability failures
export const BRIDGE_CAPABILITY_DETAILS: Record<string, { title: string; recommendation: string }> = {
  CAP_LOGS: {
    title: 'Missing console log capture capability',
    recommendation: 'Inspect the scenario UI bootstrap and ensure the iframe bridge is initialized with console log capture enabled.',
  },
  CAP_NETWORK: {
    title: 'Missing network request capture capability',
    recommendation: 'Inspect the scenario UI bootstrap and ensure the iframe bridge is initialized with network capture enabled.',
  },
};
