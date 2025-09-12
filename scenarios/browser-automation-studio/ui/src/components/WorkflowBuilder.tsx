import { useCallback } from 'react';
import ReactFlow, {
  Node,
  // Edge,
  Controls,
  MiniMap,
  Background,
  BackgroundVariant,
  useNodesState,
  useEdgesState,
  addEdge,
  Connection,
  MarkerType,
  NodeTypes,
} from 'reactflow';
import { useWorkflowStore } from '../stores/workflowStore';
import BrowserActionNode from './nodes/BrowserActionNode';
import NavigateNode from './nodes/NavigateNode';
import ClickNode from './nodes/ClickNode';
import TypeNode from './nodes/TypeNode';
import ScreenshotNode from './nodes/ScreenshotNode';
import WaitNode from './nodes/WaitNode';
import ExtractNode from './nodes/ExtractNode';
import WorkflowToolbar from './WorkflowToolbar';
import 'reactflow/dist/style.css';

const nodeTypes: NodeTypes = {
  browserAction: BrowserActionNode,
  navigate: NavigateNode,
  click: ClickNode,
  type: TypeNode,
  screenshot: ScreenshotNode,
  wait: WaitNode,
  extract: ExtractNode,
};

const defaultEdgeOptions = {
  animated: true,
  type: 'smoothstep',
  markerEnd: {
    type: MarkerType.ArrowClosed,
    width: 20,
    height: 20,
    color: '#4a5568',
  },
};

interface WorkflowBuilderProps {
  projectId?: string;
}

function WorkflowBuilder({ projectId }: WorkflowBuilderProps) {
  // TODO: Implement project-specific workflow features
  console.log('Project ID:', projectId);
  
  const { nodes: storeNodes, edges: storeEdges, updateWorkflow } = useWorkflowStore();
  const [nodes, setNodes, onNodesChange] = useNodesState(storeNodes || []);
  const [edges, setEdges, onEdgesChange] = useEdgesState(storeEdges || []);

  const onConnect = useCallback(
    (params: Connection) => {
      const newEdges = addEdge(params, edges);
      setEdges(newEdges);
      updateWorkflow({ nodes, edges: newEdges });
    },
    [edges, nodes, setEdges, updateWorkflow]
  );

  const onNodesChangeHandler = useCallback(
    (changes: any) => {
      onNodesChange(changes);
      updateWorkflow({ nodes, edges });
    },
    [onNodesChange, nodes, edges, updateWorkflow]
  );

  const onEdgesChangeHandler = useCallback(
    (changes: any) => {
      onEdgesChange(changes);
      updateWorkflow({ nodes, edges });
    },
    [onEdgesChange, nodes, edges, updateWorkflow]
  );

  const onDrop = useCallback(
    (event: React.DragEvent) => {
      event.preventDefault();
      const type = event.dataTransfer.getData('nodeType');
      
      if (!type) return;

      const reactFlowBounds = event.currentTarget.getBoundingClientRect();
      const position = {
        x: event.clientX - reactFlowBounds.left,
        y: event.clientY - reactFlowBounds.top,
      };

      const newNode: Node = {
        id: `node-${Date.now()}`,
        type,
        position,
        data: { label: type.charAt(0).toUpperCase() + type.slice(1) },
      };

      setNodes((nds) => nds.concat(newNode));
      updateWorkflow({ nodes: [...nodes, newNode], edges });
    },
    [nodes, edges, setNodes, updateWorkflow]
  );

  const onDragOver = useCallback((event: React.DragEvent) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
  }, []);

  return (
    <div className="flex-1 relative">
      <WorkflowToolbar />
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChangeHandler}
        onEdgesChange={onEdgesChangeHandler}
        onConnect={onConnect}
        onDrop={onDrop}
        onDragOver={onDragOver}
        nodeTypes={nodeTypes}
        defaultEdgeOptions={defaultEdgeOptions}
        fitView
        className="bg-flow-bg"
      >
        <Controls className="react-flow__controls" />
        <MiniMap
          nodeStrokeColor={(node) => {
            switch (node.type) {
              case 'navigate': return '#3b82f6';
              case 'click': return '#10b981';
              case 'type': return '#f59e0b';
              case 'screenshot': return '#8b5cf6';
              case 'extract': return '#ec4899';
              case 'wait': return '#6b7280';
              default: return '#4a5568';
            }
          }}
          nodeColor={(node) => {
            switch (node.type) {
              case 'navigate': return '#1e40af';
              case 'click': return '#065f46';
              case 'type': return '#92400e';
              case 'screenshot': return '#5b21b6';
              case 'extract': return '#9f1239';
              case 'wait': return '#374151';
              default: return '#1a1d29';
            }
          }}
          nodeBorderRadius={8}
        />
        <Background variant={BackgroundVariant.Dots} gap={12} size={1} color="#1a1d29" />
      </ReactFlow>
    </div>
  );
}

export default WorkflowBuilder;