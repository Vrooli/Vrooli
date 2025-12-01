/**
 * Pages Module Index
 * Central export point for all page-specific modules
 */

// DashboardPage - Dashboard view (COMPLETE)
export { DashboardPage, dashboardPage } from './DashboardPage.js';

// SuitesPage - Test suites management view (COMPLETE)
export { SuitesPage, suitesPage } from './SuitesPage.js';

// ExecutionsPage - Test executions history view (COMPLETE)
export { ExecutionsPage, executionsPage } from './ExecutionsPage.js';

// CoveragePage - Code coverage analysis view (COMPLETE)
export { CoveragePage, coveragePage } from './CoveragePage.js';

// VaultPage - Test vault management view (COMPLETE)
export { VaultPage, vaultPage } from './VaultPage.js';

// ReportsPage - Analytics and reports view (COMPLETE)
export { ReportsPage, reportsPage } from './ReportsPage.js';

// SettingsPage - Application settings view (COMPLETE)
export { SettingsPage, settingsPage } from './SettingsPage.js';

// Export default object with all page singletons
export default {
    dashboardPage,
    suitesPage,
    executionsPage,
    coveragePage,
    vaultPage,
    reportsPage,
    settingsPage
};
