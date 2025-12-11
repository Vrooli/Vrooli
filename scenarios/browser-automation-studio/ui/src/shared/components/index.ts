/**
 * Shared reusable components - truly app-wide shared components
 *
 * NOTE: Workflow-specific components (ElementPickerModal, BrowserInspectorTab,
 * CustomConnectionLine, VariableSuggestionList) have been moved to
 * @features/workflows/components as they belong to the workflow domain.
 */

// Re-export UI components for convenience
export * from '../ui';

// Shared components
export { default as SubscriptionBadge } from './SubscriptionBadge';
export { default as FeatureGateModal } from './FeatureGateModal';
export type { GatedFeature } from './FeatureGateModal';
