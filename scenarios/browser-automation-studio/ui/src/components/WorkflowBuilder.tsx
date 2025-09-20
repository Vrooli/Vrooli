import { useCallback, useEffect, useRef, useState } from 'react';
import ReactFlow, {
  Node,
  Edge,
  MiniMap,
  Background,
  BackgroundVariant,
  useNodesState,
  useEdgesState,
  addEdge,
  Connection,
  ConnectionMode,
  MarkerType,
  NodeTypes,
  ReactFlowProvider,
} from 'reactflow';
import { useWorkflowStore } from '../stores/workflowStore';
import BrowserActionNode from './nodes/BrowserActionNode';
import NavigateNode from './nodes/NavigateNode';
import ClickNode from './nodes/ClickNode';
import TypeNode from './nodes/TypeNode';
import ScreenshotNode from './nodes/ScreenshotNode';
import WaitNode from './nodes/WaitNode';
import ExtractNode from './nodes/ExtractNode';
import WorkflowCallNode from './nodes/WorkflowCallNode';
import WorkflowToolbar from './WorkflowToolbar';
import CustomConnectionLine from './CustomConnectionLine';
import 'reactflow/dist/style.css';

const nodeTypes: NodeTypes = {
  browserAction: BrowserActionNode,
  navigate: NavigateNode,
  click: ClickNode,
  type: TypeNode,
  screenshot: ScreenshotNode,
  wait: WaitNode,
  extract: ExtractNode,
  workflowCall: WorkflowCallNode,
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

function WorkflowBuilderInner({ projectId }: WorkflowBuilderProps) {
  // TODO: Implement project-specific workflow features
  console.log('Project ID:', projectId);
  
  const { nodes: storeNodes, edges: storeEdges, updateWorkflow } = useWorkflowStore();
  const [nodes, setNodes, onNodesChange] = useNodesState(storeNodes || []);
  const [edges, setEdges, onEdgesChange] = useEdgesState(storeEdges || []);

  useEffect(() => {
    if (storeNodes) {
      setNodes(storeNodes as Node[]);
    }
  }, [storeNodes, setNodes]);

  useEffect(() => {
    if (storeEdges) {
      setEdges(storeEdges as Edge[]);
    }
  }, [storeEdges, setEdges]);
  const connectingNodeId = useRef<string | null>(null);
  
  // Toolbar state
  const [showGrid, setShowGrid] = useState(true);
  const [locked, setLocked] = useState(false);
  
  // Undo/Redo state
  const [history, setHistory] = useState<Array<{nodes: any[], edges: any[]}>>([]);
  const [historyIndex, setHistoryIndex] = useState(-1);

  // Save current state to history
  const saveToHistory = useCallback(() => {
    const currentState = { nodes: [...nodes], edges: [...edges] };
    setHistory(prev => {
      // Remove any future history if we're not at the end
      const newHistory = prev.slice(0, historyIndex + 1);
      // Add new state
      newHistory.push(currentState);
      // Limit history size to prevent memory issues
      if (newHistory.length > 50) {
        newHistory.shift();
        setHistoryIndex(prev => prev - 1);
      }
      return newHistory;
    });
    setHistoryIndex(prev => prev + 1);
  }, [nodes, edges, historyIndex]);

  // Undo function
  const undo = useCallback(() => {
    if (historyIndex > 0) {
      const previousState = history[historyIndex - 1];
      setNodes(previousState.nodes);
      setEdges(previousState.edges);
      setHistoryIndex(prev => prev - 1);
      updateWorkflow({ nodes: previousState.nodes, edges: previousState.edges });
    }
  }, [history, historyIndex, setNodes, setEdges, updateWorkflow]);

  // Redo function
  const redo = useCallback(() => {
    if (historyIndex < history.length - 1) {
      const nextState = history[historyIndex + 1];
      setNodes(nextState.nodes);
      setEdges(nextState.edges);
      setHistoryIndex(prev => prev + 1);
      updateWorkflow({ nodes: nextState.nodes, edges: nextState.edges });
    }
  }, [history, historyIndex, setNodes, setEdges, updateWorkflow]);

  // Duplicate selected nodes
  const duplicateSelected = useCallback(() => {
    const selectedNodes = nodes.filter(node => node.selected);
    if (selectedNodes.length === 0) return;

    saveToHistory();
    
    const newNodes = selectedNodes.map(node => ({
      ...node,
      id: `${node.id}-copy-${Date.now()}`,
      position: {
        x: node.position.x + 50,
        y: node.position.y + 50,
      },
      selected: true,
    }));

    // Deselect original nodes and add new ones
    const updatedNodes = nodes.map(node => ({ ...node, selected: false }));
    const allNodes = [...updatedNodes, ...newNodes];
    
    setNodes(allNodes);
    updateWorkflow({ nodes: allNodes, edges });
  }, [nodes, edges, setNodes, updateWorkflow, saveToHistory]);

  // Delete selected nodes and edges
  const deleteSelected = useCallback(() => {
    const selectedNodeIds = nodes.filter(node => node.selected).map(node => node.id);
    const selectedEdgeIds = edges.filter(edge => edge.selected).map(edge => edge.id);
    
    if (selectedNodeIds.length === 0 && selectedEdgeIds.length === 0) return;

    saveToHistory();
    
    // Remove selected nodes
    const remainingNodes = nodes.filter(node => !node.selected);
    
    // Remove selected edges and edges connected to deleted nodes
    const remainingEdges = edges.filter(edge => 
      !edge.selected && 
      !selectedNodeIds.includes(edge.source) && 
      !selectedNodeIds.includes(edge.target)
    );
    
    setNodes(remainingNodes);
    setEdges(remainingEdges);
    updateWorkflow({ nodes: remainingNodes, edges: remainingEdges });
  }, [nodes, edges, setNodes, setEdges, updateWorkflow, saveToHistory]);

  const onConnect = useCallback(
    (params: Connection) => {
      const newEdges = addEdge(params, edges);
      setEdges(newEdges);
      updateWorkflow({ nodes, edges: newEdges });
    },
    [edges, nodes, setEdges, updateWorkflow]
  );

  // Track when connection starts
  const onConnectStart = useCallback((_: any, { nodeId }: any) => {
    connectingNodeId.current = nodeId;
  }, []);

  // Handle connection end - reset the connecting node id
  const onConnectEnd = useCallback(() => {
    connectingNodeId.current = null;
  }, []);

  const onNodesChangeHandler = useCallback(
    (changes: any) => {
      // Check if this is a significant change that should be saved to history
      const hasSignificantChange = changes.some((change: any) => 
        change.type === 'add' || change.type === 'remove' || 
        (change.type === 'position' && change.positionAbsolute)
      );
      
      if (hasSignificantChange) {
        saveToHistory();
      }
      
      onNodesChange(changes);
      updateWorkflow({ nodes, edges });
    },
    [onNodesChange, nodes, edges, updateWorkflow, saveToHistory]
  );

  const onEdgesChangeHandler = useCallback(
    (changes: any) => {
      // Check if this is a significant change that should be saved to history
      const hasSignificantChange = changes.some((change: any) => 
        change.type === 'add' || change.type === 'remove'
      );
      
      if (hasSignificantChange) {
        saveToHistory();
      }
      
      onEdgesChange(changes);
      updateWorkflow({ nodes, edges });
    },
    [onEdgesChange, nodes, edges, updateWorkflow, saveToHistory]
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
      <WorkflowToolbar 
        showGrid={showGrid} 
        onToggleGrid={() => setShowGrid(!showGrid)}
        locked={locked}
        onToggleLock={() => setLocked(!locked)}
        onUndo={undo}
        onRedo={redo}
        canUndo={historyIndex > 0}
        canRedo={historyIndex < history.length - 1}
        onDuplicate={duplicateSelected}
        onDelete={deleteSelected}
      />
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChangeHandler}
        onEdgesChange={onEdgesChangeHandler}
        onConnect={onConnect}
        onConnectStart={onConnectStart}
        onConnectEnd={onConnectEnd}
        onDrop={onDrop}
        onDragOver={onDragOver}
        nodeTypes={nodeTypes}
        defaultEdgeOptions={defaultEdgeOptions}
        fitView
        className="bg-flow-bg"
        connectOnClick={false}
        connectionMode={ConnectionMode.Loose}
        connectionRadius={50}
        connectionLineComponent={CustomConnectionLine}
        nodesDraggable={!locked}
        nodesConnectable={!locked}
        elementsSelectable={!locked}
        edgesUpdatable={!locked}
      >
        <MiniMap
          nodeStrokeColor={(node) => {
            switch (node.type) {
              case 'navigate': return '#3b82f6';
              case 'click': return '#10b981';
              case 'type': return '#f59e0b';
              case 'screenshot': return '#8b5cf6';
              case 'extract': return '#ec4899';
              case 'wait': return '#6b7280';
              case 'workflowCall': return '#a855f7';
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
              case 'workflowCall': return '#7c3aed';
              default: return '#1a1d29';
            }
          }}
          nodeBorderRadius={8}
        />
        {showGrid && <Background variant={BackgroundVariant.Dots} gap={12} size={1} color="#1a1d29" />}
      </ReactFlow>
    </div>
  );
}

// Wrap with ReactFlowProvider to use hooks
function WorkflowBuilder(props: WorkflowBuilderProps) {
  return (
    <ReactFlowProvider>
      <WorkflowBuilderInner {...props} />
    </ReactFlowProvider>
  );
}

export default WorkflowBuilder;
