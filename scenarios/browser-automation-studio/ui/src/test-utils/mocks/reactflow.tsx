import { useState, type DragEventHandler, type ReactNode } from 'react';
import type { Edge, Node } from 'reactflow';
import { vi } from 'vitest';
import { selectors } from '@constants/selectors';

type MockReactFlowProps = {
  children?: ReactNode;
  onDrop?: DragEventHandler<HTMLDivElement>;
  onDragOver?: DragEventHandler<HTMLDivElement>;
  nodes?: Node[];
  edges?: Edge[];
};

const mockReactFlowInstance = {
  project: vi.fn((position: { x: number; y: number }) => position),
  zoomIn: vi.fn(),
  zoomOut: vi.fn(),
  fitView: vi.fn(),
  getNodes: vi.fn(() => []),
  getEdges: vi.fn(() => []),
  setNodes: vi.fn(),
  setEdges: vi.fn(),
};

export const reactFlowMocks = {
  useReactFlow: vi.fn(() => mockReactFlowInstance),
  useNodesState: vi.fn((initialNodes: Node[]) => useState(initialNodes)),
  useEdgesState: vi.fn((initialEdges: Edge[]) => useState(initialEdges)),
  addEdge: vi.fn((connection: Partial<Edge>, edges: Edge[]) => [
    ...edges,
    { ...connection, id: `edge-${Date.now()}` } as Edge,
  ]),
  instance: mockReactFlowInstance,
};

function MockReactFlow({
  children,
  onDrop,
  onDragOver,
  nodes,
  edges,
}: MockReactFlowProps) {
  return (
    <div
      data-testid={selectors.workflowBuilder.canvas.reactFlow}
      onDrop={onDrop}
      onDragOver={onDragOver}
      data-nodes-count={nodes?.length ?? 0}
      data-edges-count={edges?.length ?? 0}
    >
      {children}
      {nodes?.map((node: Node) => (
        <div
          key={node.id}
          data-testid={`node-${node.id}`}
          data-node-type={node.type}
          className={node.className}
        >
          {String(node.data?.label ?? node.type ?? '')}
        </div>
      ))}
    </div>
  );
}

export default MockReactFlow;
export const ReactFlow = MockReactFlow;
export const ReactFlowProvider = ({ children }: { children: ReactNode }) => (
  <div>{children}</div>
);
export const MiniMap = () => <div data-testid={selectors.workflowBuilder.canvas.minimap} />;
export const Background = () => <div data-testid={selectors.app.background} />;
export const BackgroundVariant = { Dots: 'dots' };
export const MarkerType = { ArrowClosed: 'arrowclosed' };
export const ConnectionMode = { Loose: 'loose' };
export const useReactFlow = reactFlowMocks.useReactFlow;
export const useNodesState = reactFlowMocks.useNodesState;
export const useEdgesState = reactFlowMocks.useEdgesState;
export const addEdge = reactFlowMocks.addEdge;
