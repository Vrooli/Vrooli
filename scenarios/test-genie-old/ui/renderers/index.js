/**
 * Renderers Module Index
 * Central export point for all rendering modules
 */

// TableRenderer - Unified table rendering
export { TableRenderer, tableRenderer } from './TableRenderer.js';

// ChartRenderer - Custom chart rendering
export { ChartRenderer, chartRenderer } from './ChartRenderer.js';

// CardRenderer - Card and metric rendering
export { CardRenderer, cardRenderer } from './CardRenderer.js';

// DetailPanelRenderer - Detail panel and complex component rendering
export { DetailPanelRenderer, detailPanelRenderer } from './DetailPanelRenderer.js';

// Export default object with all renderer singletons
export default {
    tableRenderer,
    chartRenderer,
    cardRenderer,
    detailPanelRenderer
};
