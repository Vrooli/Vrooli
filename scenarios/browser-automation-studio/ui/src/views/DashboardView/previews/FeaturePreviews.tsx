import { AIPreview } from './AIPreview';
import { RecordModePreview } from './RecordModePreview';
import { VisualBuilderPreview } from './VisualBuilderPreview';
import { TestMonitorPreview } from './TestMonitorPreview';
import { ExportsPreview } from './ExportsPreview';

export type PreviewRenderer = (isActive: boolean) => JSX.Element;

export const PREVIEW_RENDERERS: PreviewRenderer[] = [
  (isActive) => <AIPreview isActive={isActive} />,
  (isActive) => <RecordModePreview isActive={isActive} />,
  (isActive) => <VisualBuilderPreview isActive={isActive} />,
  (isActive) => <TestMonitorPreview isActive={isActive} />,
  (isActive) => <ExportsPreview isActive={isActive} />,
];

export default PREVIEW_RENDERERS;
