/**
 * DashboardView module exports
 *
 * Main dashboard view with tabs for Home, Projects, Executions, Schedules, and Exports.
 */

export { default as DashboardView } from './DashboardView';
export { default as DashboardViewWrapper } from './DashboardViewWrapper';
export { TabNavigation, type DashboardTab } from './DashboardTabs';
export { HomeTab } from './HomeSection';
export { WelcomeHero } from './WelcomeHero';
export { GlobalSearchModal } from './GlobalSearchModal';
export { RunningExecutionsBadge } from './RunningExecutionsBadge';
export { SchedulesTab } from './SchedulesSection';

// Widgets
export * from './widgets';

// Previews
export * from './previews';
