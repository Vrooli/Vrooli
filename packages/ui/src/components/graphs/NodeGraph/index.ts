import { NodeType } from 'graphql/generated/globalTypes';

export * from './AddAfterLinkDialog/AddAfterLinkDialog';
export * from './AddBeforeLinkDialog/AddBeforeLinkDialog';
export * from './edges';
export * from './EndNodeDialog/EndNodeDialog';
export * from './NodeContextMenu/NodeContextMenu';
export * from './NodeColumn/NodeColumn';
export * from './NodeGraph/NodeGraph';
export * from './nodes';

export const NodeWidth = {
    [NodeType.End]: 100,
    [NodeType.Redirect]: 100,
    [NodeType.RoutineList]: 350,
    [NodeType.Start]: 100,
}