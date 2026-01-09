// Main panel (inline side panel, replaces modal dialog)
export { PreviewSettingsPanel } from './PreviewSettingsPanel';

// Legacy dialog export for backwards compatibility
export { PreviewSettingsDialog } from './PreviewSettingsDialog';

// Components (for advanced usage)
export { PreviewSettingsSidebar } from './PreviewSettingsSidebar';
export { PreviewSettingsHeader } from './PreviewSettingsHeader';

// Hooks
export { usePreviewSettings } from './usePreviewSettings';
export {
  usePreviewSettingsPanel,
  PANEL_MIN_WIDTH,
  PANEL_MAX_WIDTH,
  PANEL_DEFAULT_WIDTH,
} from './usePreviewSettingsPanel';

// Types
export type { SectionId, SectionGroupId, SectionConfig, SectionGroup, BaseSectionProps } from './types';
