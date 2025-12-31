/**
 * Sidebar Domain
 *
 * Unified sidebar with Timeline and Auto tabs for the Record page.
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

export { ScreenshotsTab } from './ScreenshotsTab';

export { LogsTab } from './LogsTab';

export { UnifiedSidebar } from './UnifiedSidebar';
export type {
  UnifiedSidebarProps,
  TimelineTabPassthroughProps,
  AutoTabPassthroughProps,
  ScreenshotsTabPassthroughProps,
  LogsTabPassthroughProps,
} from './UnifiedSidebar';
