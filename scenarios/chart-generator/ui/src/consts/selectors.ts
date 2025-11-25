/**
 * Test selectors for UI automation via browser-automation-studio
 *
 * Usage:
 *   - BAS playbooks reference these selectors for reliable element targeting
 *   - Keep this file in sync with UI components
 *   - Use data-testid attributes in components for stable selection
 */

export const SELECTORS = {
  // Main tabs
  TAB_CHARTS: '[data-testid="tab-charts"]',
  TAB_STYLES: '[data-testid="tab-styles"]',
  TAB_DATA: '[data-testid="tab-data"]',

  // Chart type selector
  CHART_TYPE_BAR: '[data-testid="chart-type-bar"]',
  CHART_TYPE_LINE: '[data-testid="chart-type-line"]',
  CHART_TYPE_PIE: '[data-testid="chart-type-pie"]',
  CHART_TYPE_SCATTER: '[data-testid="chart-type-scatter"]',
  CHART_TYPE_AREA: '[data-testid="chart-type-area"]',
  CHART_TYPE_GANTT: '[data-testid="chart-type-gantt"]',
  CHART_TYPE_HEATMAP: '[data-testid="chart-type-heatmap"]',
  CHART_TYPE_TREEMAP: '[data-testid="chart-type-treemap"]',
  CHART_TYPE_CANDLESTICK: '[data-testid="chart-type-candlestick"]',

  // Style selector
  STYLE_PROFESSIONAL: '[data-testid="style-professional"]',
  STYLE_LIGHT: '[data-testid="style-light"]',
  STYLE_DARK: '[data-testid="style-dark"]',
  STYLE_CORPORATE: '[data-testid="style-corporate"]',
  STYLE_MINIMAL: '[data-testid="style-minimal"]',

  // Data panel
  DATA_INPUT_TEXTAREA: '[data-testid="data-input"]',
  DATA_VALIDATE_BUTTON: '[data-testid="data-validate"]',
  DATA_ERROR_MESSAGE: '[data-testid="data-error"]',
  SAMPLE_DATA_BUTTON: '[data-testid="sample-data"]',

  // Chart preview
  CHART_PREVIEW_CONTAINER: '[data-testid="chart-preview"]',
  CHART_SVG: '[data-testid="chart-svg"]',

  // Export controls
  EXPORT_PNG_BUTTON: '[data-testid="export-png"]',
  EXPORT_SVG_BUTTON: '[data-testid="export-svg"]',

  // API connectivity status
  API_STATUS_INDICATOR: '[data-testid="api-status"]',
} as const;
