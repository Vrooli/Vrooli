import { AIPreview } from './previews/AIPreview';
import { RecordModePreview } from './previews/RecordModePreview';
import { VisualBuilderPreview } from './previews/VisualBuilderPreview';
import { TestMonitorPreview } from './previews/TestMonitorPreview';
import { ExportsPreview } from './previews/ExportsPreview';

export type PreviewRenderer = (isActive: boolean) => JSX.Element;

export const PREVIEW_RENDERERS: PreviewRenderer[] = [
  (isActive) => <AIPreview isActive={isActive} />,
  (isActive) => <RecordModePreview isActive={isActive} />,
  (isActive) => <VisualBuilderPreview isActive={isActive} />,
  (isActive) => <TestMonitorPreview isActive={isActive} />,
  (isActive) => <ExportsPreview isActive={isActive} />,
];

export default PREVIEW_RENDERERS;
