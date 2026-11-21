/**
 * Recording workspace route composition root.
 *
 * Keep the established lazy page export stable while delegating recording and
 * execution ownership to the focused session hooks and workspace surface.
 */
import { RecordModeWorkspace, type RecordModePageProps } from './RecordModeWorkspace';

export type { RecordModePageProps, WorkflowTypeParam } from './RecordModeWorkspace';

export function RecordModePage(props: RecordModePageProps) {
  return <RecordModeWorkspace {...props} />;
}
