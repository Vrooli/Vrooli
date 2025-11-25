/**
 * UI Test Selectors for Visited Tracker
 *
 * Centralized selectors for UI automation testing.
 * Used by browser-automation-studio workflows in test/playbooks/.
 */

export const SELECTORS = {
  // Campaign Management
  CAMPAIGNS: {
    LIST: '[data-testid="campaigns-list"]',
    CREATE_BUTTON: '[data-testid="create-campaign-button"]',
    CREATE_FORM: '[data-testid="create-campaign-form"]',
    NAME_INPUT: '[data-testid="campaign-name-input"]',
    PATTERN_INPUT: '[data-testid="campaign-pattern-input"]',
    SUBMIT_BUTTON: '[data-testid="campaign-submit-button"]',
    CAMPAIGN_CARD: '[data-testid="campaign-card"]',
    DELETE_BUTTON: '[data-testid="campaign-delete-button"]',
  },

  // File Tracking
  FILES: {
    LIST: '[data-testid="files-list"]',
    FILE_ROW: '[data-testid="file-row"]',
    FILE_PATH: '[data-testid="file-path"]',
    VISIT_COUNT: '[data-testid="visit-count"]',
    STALENESS_SCORE: '[data-testid="staleness-score"]',
    LAST_VISITED: '[data-testid="last-visited"]',
    VISIT_BUTTON: '[data-testid="visit-button"]',
  },

  // Prioritization
  PRIORITY: {
    LEAST_VISITED_TAB: '[data-testid="least-visited-tab"]',
    MOST_STALE_TAB: '[data-testid="most-stale-tab"]',
    REFRESH_BUTTON: '[data-testid="refresh-priority-button"]',
  },

  // Navigation
  NAV: {
    HOME: '[data-testid="nav-home"]',
    CAMPAIGNS: '[data-testid="nav-campaigns"]',
    DOCS: '[data-testid="nav-docs"]',
  },

  // Common
  LOADING: '[data-testid="loading-spinner"]',
  ERROR_MESSAGE: '[data-testid="error-message"]',
  SUCCESS_MESSAGE: '[data-testid="success-message"]',
};
