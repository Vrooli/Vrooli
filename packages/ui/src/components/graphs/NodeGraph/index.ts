import { NodeType } from '@local/shared';

export * from './NodeContextMenu/NodeContextMenu';
export * from './NodeGraphColumn/NodeGraphColumn';
export * from './NodeGraphContainer/NodeGraphContainer';
export * from './NodeGraphEdge/NodeGraphEdge';
export * from './nodes';

export const NodeWidth = {
    [NodeType.Combine]: 100,
    [NodeType.Decision]: 100,
    [NodeType.End]: 100,
    [NodeType.Loop]: 100,
    [NodeType.Redirect]: 100,
    [NodeType.RoutineList]: 350,
    [NodeType.Start]: 100,
}