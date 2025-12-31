/**
 * Sidebar Domain
 *
 * Unified sidebar with Timeline, Auto, and Artifacts tabs for the Record page.
 */

// Types
export * from './types';

// Hooks
export { useUnifiedSidebar } from './useUnifiedSidebar';
export type { UseUnifiedSidebarOptions, UseUnifiedSidebarReturn } from './useUnifiedSidebar';

export { useAISettings, estimateNavigationCost, formatCost } from './useAISettings';
export type { UseAISettingsOptions, UseAISettingsReturn } from './useAISettings';

// Components
export { AISettingsModal } from './AISettingsModal';
export type { AISettingsModalProps } from './AISettingsModal';

export { AIMessageBubble } from './AIMessageBubble';
export type { AIMessageBubbleProps } from './AIMessageBubble';

export { TimelineTab } from './TimelineTab';
export type { TimelineTabProps } from './TimelineTab';

export { AutoTab } from './AutoTab';
export type { AutoTabProps } from './AutoTab';

export { ArtifactsTab } from './ArtifactsTab';
export type { ArtifactsTabProps, ConsoleLogEntry, NetworkEventEntry, DomSnapshotEntry } from './ArtifactsTab';

export { UnifiedSidebar } from './UnifiedSidebar';
export type {
  UnifiedSidebarProps,
  TimelineTabPassthroughProps,
  AutoTabPassthroughProps,
  ArtifactsTabPassthroughProps,
} from './UnifiedSidebar';
