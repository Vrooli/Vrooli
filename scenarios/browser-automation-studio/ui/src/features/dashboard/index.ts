// Backwards compatibility re-exports
// All components have moved to views/DashboardView or domains/

// Widgets - now in views/DashboardView/widgets/
export {
  RecentWorkflowsWidget,
  RecentExecutionsWidget,
  QuickStartWidget,
  RunningIndicator,
  TemplatesGallery,
} from '@/views/DashboardView/widgets';

// Previews - now in views/DashboardView/previews/
export {
  FeatureShowcase,
  FEATURE_CONFIGS,
  FeaturePreviews,
  TabEmptyState,
} from '@/views/DashboardView/previews';
export type { FeatureConfig } from '@/views/DashboardView/previews';

// Global views - now in views/
export { GlobalWorkflowsView } from '@/views/AllWorkflowsView/GlobalWorkflowsView';
export { GlobalExecutionsView } from '@/views/AllExecutionsView/GlobalExecutionsView';

// Tabs from domains
export { ExecutionsTab } from '@/domains/executions/history/ExecutionsTab';
export { ExportsTab } from '@/domains/exports/ExportsTab';
export { ProjectsTab } from '@/domains/projects/ProjectsTab';

// SchedulesTab - now in views/DashboardView/
export { SchedulesTab } from '@/views/DashboardView/SchedulesSection';

// Re-exports from views/DashboardView for backwards compatibility
export { GlobalSearchModal } from '@/views/DashboardView/GlobalSearchModal';
export { TabNavigation, type DashboardTab } from '@/views/DashboardView/DashboardTabs';
export { HomeTab } from '@/views/DashboardView/HomeSection';
export { RunningExecutionsBadge } from '@/views/DashboardView/RunningExecutionsBadge';
export { WelcomeHero } from '@/views/DashboardView/WelcomeHero';
