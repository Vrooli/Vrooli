import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { logger } from '../utils/logger';
import { Eye, Code } from 'lucide-react';
import Editor from '@monaco-editor/react';
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
  NodeChange,
  EdgeChange,
  useReactFlow,
} from 'reactflow';
import { useWorkflowStore } from '../stores/workflowStore';
import { normalizeNodes, normalizeEdges } from '../utils/workflowNormalizers';
import BrowserActionNode from './nodes/BrowserActionNode';
import NavigateNode from './nodes/NavigateNode';
import ClickNode from './nodes/ClickNode';
import TypeNode from './nodes/TypeNode';
import ScreenshotNode from './nodes/ScreenshotNode';
import WaitNode from './nodes/WaitNode';
import ExtractNode from './nodes/ExtractNode';
import AssertNode from './nodes/AssertNode';
import WorkflowCallNode from './nodes/WorkflowCallNode';
import WorkflowToolbar from './WorkflowToolbar';
import CustomConnectionLine from './CustomConnectionLine';
import 'reactflow/dist/style.css';
import toast from 'react-hot-toast';
import ResponsiveDialog from './ResponsiveDialog';
import type { ExecutionViewportSettings, ViewportPreset } from '../stores/workflowStore';

const nodeTypes: NodeTypes = {
  browserAction: BrowserActionNode,
  navigate: NavigateNode,
  click: ClickNode,
  type: TypeNode,
  screenshot: ScreenshotNode,
  wait: WaitNode,
  extract: ExtractNode,
  assert: AssertNode,
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

const DEFAULT_DESKTOP_VIEWPORT: ExecutionViewportSettings = { width: 1920, height: 1080, preset: 'desktop' };
const DEFAULT_MOBILE_VIEWPORT: ExecutionViewportSettings = { width: 390, height: 844, preset: 'mobile' };
const MIN_VIEWPORT_DIMENSION = 200;
const MAX_VIEWPORT_DIMENSION = 10000;

const clampViewportDimension = (value: number): number => {
  if (!Number.isFinite(value)) {
    return MIN_VIEWPORT_DIMENSION;
  }
  return Math.min(Math.max(Math.round(value), MIN_VIEWPORT_DIMENSION), MAX_VIEWPORT_DIMENSION);
};

const determineViewportPreset = (width: number, height: number): ViewportPreset => {
  if (!Number.isFinite(width) || !Number.isFinite(height)) {
    return 'custom';
  }
  if (width === DEFAULT_DESKTOP_VIEWPORT.width && height === DEFAULT_DESKTOP_VIEWPORT.height) {
    return 'desktop';
  }
  if (width === DEFAULT_MOBILE_VIEWPORT.width && height === DEFAULT_MOBILE_VIEWPORT.height) {
    return 'mobile';
  }
  return 'custom';
};

const normalizeViewportSetting = (viewport?: ExecutionViewportSettings | null): ExecutionViewportSettings => {
  if (!viewport || !Number.isFinite(viewport.width) || !Number.isFinite(viewport.height)) {
    return { ...DEFAULT_DESKTOP_VIEWPORT };
  }
  const width = clampViewportDimension(viewport.width);
  const height = clampViewportDimension(viewport.height);
  const preset = viewport.preset ?? determineViewportPreset(width, height);
  return { width, height, preset };
};

interface WorkflowBuilderProps {
  projectId?: string;
}

function WorkflowBuilderInner({ projectId }: WorkflowBuilderProps) {
  // Project ID is used for workflow scoping - workflows are associated with projects
  // via the database schema and filtered in the ProjectDetail component
  if (projectId) {
    logger.debug('Project ID loaded', { component: 'WorkflowBuilder', projectId });
  }

  // Use selectors to only subscribe to specific parts of the store
  const storeNodes = useWorkflowStore((state) => state.nodes);
  const storeEdges = useWorkflowStore((state) => state.edges);
  const updateWorkflow = useWorkflowStore((state) => state.updateWorkflow);
  const currentWorkflow = useWorkflowStore((state) => state.currentWorkflow);
  const isDirty = useWorkflowStore((state) => state.isDirty);
  const hasVersionConflict = useWorkflowStore((state) => state.hasVersionConflict);
  const scheduleAutosave = useWorkflowStore((state) => state.scheduleAutosave);
  const cancelAutosave = useWorkflowStore((state) => state.cancelAutosave);

  const [nodes, setNodes, onNodesChange] = useNodesState(storeNodes || []);
  const [edges, setEdges, onEdgesChange] = useEdgesState(storeEdges || []);
  const reactFlowInstance = useReactFlow();

  const graphContainerRef = useRef<HTMLDivElement | null>(null);
  const [graphWidth, setGraphWidth] = useState(0);
  const [isViewportDialogOpen, setViewportDialogOpen] = useState(false);
  const executionViewport = useWorkflowStore((state) => state.currentWorkflow?.executionViewport as ExecutionViewportSettings | undefined);
  const effectiveViewport = useMemo(() => normalizeViewportSetting(executionViewport), [executionViewport]);

  const connectingNodeId = useRef<string | null>(null);
  const isLoadingFromStore = useRef(false);

  // Sync FROM store TO local state (on workflow load)
  useEffect(() => {
    if (storeNodes) {
      isLoadingFromStore.current = true;
      setNodes(storeNodes as Node[]);
      // Reset flag after state update completes
      setTimeout(() => { isLoadingFromStore.current = false; }, 0);
    }
  }, [storeNodes, setNodes]);

  useEffect(() => {
    if (storeEdges) {
      isLoadingFromStore.current = true;
      setEdges(storeEdges as Edge[]);
      setTimeout(() => { isLoadingFromStore.current = false; }, 0);
    }
  }, [storeEdges, setEdges]);


  // Sync FROM local state TO store (on any user change, not from store load)
  useEffect(() => {
    if (!isLoadingFromStore.current) {
      updateWorkflow({ nodes, edges });
    }
  }, [nodes, edges, updateWorkflow]);

  useEffect(() => {
    if (!currentWorkflow) {
      cancelAutosave();
      return;
    }

    if (hasVersionConflict) {
      cancelAutosave();
      return;
    }

    if (isDirty) {
      scheduleAutosave({ source: 'autosave', changeDescription: 'Autosave' });
    } else {
      cancelAutosave();
    }

    return () => {
      cancelAutosave();
    };
  }, [currentWorkflow?.id, isDirty, hasVersionConflict, scheduleAutosave, cancelAutosave]);
  
  // Toolbar state
  const [locked, setLocked] = useState(false);
  
  // Undo/Redo state
  interface WorkflowState {
    nodes: Node[];
    edges: Edge[];
  }
  const [history, setHistory] = useState<WorkflowState[]>([]);
  const [historyIndex, setHistoryIndex] = useState(-1);
  const [viewMode, setViewMode] = useState<'visual' | 'code'>('visual');
  const [codeValue, setCodeValue] = useState('');
  const [codeDirty, setCodeDirty] = useState(false);
  const [codeError, setCodeError] = useState<string | null>(null);

  useEffect(() => {
    if (viewMode !== 'visual') {
      return;
    }

    const element = graphContainerRef.current;
    if (!element) {
      return;
    }

    const updateWidth = () => {
      const rect = element.getBoundingClientRect();
      setGraphWidth(Math.round(rect.width));
    };

    updateWidth();

    if (typeof ResizeObserver !== 'undefined') {
      const observer = new ResizeObserver(() => updateWidth());
      observer.observe(element);
      return () => observer.disconnect();
    }

    window.addEventListener('resize', updateWidth);
    return () => {
      window.removeEventListener('resize', updateWidth);
    };
  }, [viewMode]);

  const buildJsonFromState = useCallback(() => {
    try {
      return JSON.stringify(
        {
          nodes: nodes ?? [],
          edges: edges ?? [],
        },
        null,
        2
      );
    } catch (error) {
      logger.error('Failed to stringify workflow definition', { component: 'WorkflowBuilder', action: 'handleSaveError' }, error);
      return '{\n  "nodes": [],\n  "edges": []\n}';
    }
  }, [nodes, edges]);

  useEffect(() => {
    if (!codeDirty || viewMode !== 'code') {
      setCodeValue(buildJsonFromState());
      if (viewMode !== 'code') {
        setCodeDirty(false);
        setCodeError(null);
      }
    }
  }, [buildJsonFromState, codeDirty, viewMode]);

  // Save current state to history
  const saveToHistory = useCallback(() => {
    const currentState = { nodes: [...nodes], edges: [...edges] };
    setHistory(prev => {
      const newHistory = prev.slice(0, historyIndex + 1);
      newHistory.push(currentState);
      if (newHistory.length > 50) {
        newHistory.shift();
        setHistoryIndex(prevIndex => prevIndex - 1);
      }
      return newHistory;
    });
    setHistoryIndex(prev => prev + 1);
  }, [edges, historyIndex, nodes]);

  const applyCodeChanges = useCallback((options?: { silent?: boolean }) => {
    try {
      const parsed = JSON.parse(codeValue || '{}');
      const parsedNodes = normalizeNodes(parsed?.nodes ?? []);
      const parsedEdges = normalizeEdges(parsed?.edges ?? []);

      saveToHistory();
      setNodes(parsedNodes);
      setEdges(parsedEdges);
      setCodeDirty(false);
      setCodeError(null);
      if (!options?.silent) {
        toast.success('Workflow updated from JSON');
      }
      return true;
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Invalid JSON';
      setCodeError(message);
      toast.error(`Invalid workflow JSON: ${message}`);
      return false;
    }
  }, [codeValue, saveToHistory, setEdges, setNodes, updateWorkflow]);

  const handleViewModeChange = useCallback((mode: 'visual' | 'code') => {
    if (mode === viewMode) {
      return;
    }

    if (mode === 'visual' && viewMode === 'code' && codeDirty) {
      const applied = applyCodeChanges({ silent: true });
      if (!applied) {
        return;
      }
    }

    setViewMode(mode);
    if (mode === 'code') {
      setCodeValue(buildJsonFromState());
      setCodeDirty(false);
      setCodeError(null);
    }
  }, [applyCodeChanges, buildJsonFromState, codeDirty, viewMode]);

  const handleCodeChange = (value: string | undefined) => {
    setCodeValue(value || '');
    setCodeDirty(true);
    setCodeError(null);
  };

  const handleResetCode = () => {
    setCodeValue(buildJsonFromState());
    setCodeDirty(false);
    setCodeError(null);
  };

  // Undo function
  const undo = useCallback(() => {
    if (historyIndex > 0) {
      const previousState = history[historyIndex - 1];
      setNodes(previousState.nodes);
      setEdges(previousState.edges);
      setHistoryIndex(prev => prev - 1);
    }
  }, [history, historyIndex, setNodes, setEdges]);

  // Redo function
  const redo = useCallback(() => {
    if (historyIndex < history.length - 1) {
      const nextState = history[historyIndex + 1];
      setNodes(nextState.nodes);
      setEdges(nextState.edges);
      setHistoryIndex(prev => prev + 1);
    }
  }, [history, historyIndex, setNodes, setEdges]);

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
  }, [nodes, edges, setNodes, saveToHistory]);

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
  }, [nodes, edges, setNodes, setEdges, saveToHistory]);

  const handleViewportSave = useCallback((viewport: ExecutionViewportSettings) => {
    updateWorkflow({ executionViewport: normalizeViewportSetting(viewport) });
  }, [updateWorkflow]);

  const onConnect = useCallback(
    (params: Connection) => {
      const newEdges = addEdge(params, edges);
      setEdges(newEdges);
    },
    [edges, setEdges]
  );

  // Track when connection starts
  const onConnectStart = useCallback((_: React.MouseEvent | React.TouchEvent, { nodeId }: { nodeId: string | null }) => {
    connectingNodeId.current = nodeId;
  }, []);

  // Handle connection end - reset the connecting node id
  const onConnectEnd = useCallback(() => {
    connectingNodeId.current = null;
  }, []);

  const onNodesChangeHandler = useCallback(
    (changes: NodeChange[]) => {
      // Check if this is a significant change that should be saved to history
      const hasSignificantChange = changes.some((change) =>
        change.type === 'add' || change.type === 'remove' ||
        (change.type === 'position' && 'positionAbsolute' in change)
      );

      if (hasSignificantChange) {
        saveToHistory();
      }

      onNodesChange(changes);
    },
    [onNodesChange, saveToHistory]
  );

  const onEdgesChangeHandler = useCallback(
    (changes: EdgeChange[]) => {
      // Check if this is a significant change that should be saved to history
      const hasSignificantChange = changes.some((change) =>
        change.type === 'add' || change.type === 'remove'
      );

      if (hasSignificantChange) {
        saveToHistory();
      }

      onEdgesChange(changes);
    },
    [onEdgesChange, saveToHistory]
  );

  const onDrop = useCallback(
    (event: React.DragEvent) => {
      event.preventDefault();
      const type = event.dataTransfer.getData('nodeType');
      
      if (!type) return;

      const reactFlowBounds = event.currentTarget.getBoundingClientRect();
      const canvasPosition = {
        x: event.clientX - reactFlowBounds.left,
        y: event.clientY - reactFlowBounds.top,
      };

      const position = typeof reactFlowInstance.project === 'function'
        ? reactFlowInstance.project(canvasPosition)
        : canvasPosition;

      const newNode: Node = {
        id: `node-${Date.now()}`,
        type,
        position,
        data: { label: type.charAt(0).toUpperCase() + type.slice(1) },
      };

      setNodes((nds) => nds.concat(newNode));
    },
    [reactFlowInstance, setNodes]
  );

  const onDragOver = useCallback((event: React.DragEvent) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
  }, []);

  return (
    <div ref={graphContainerRef} className="flex-1 relative">
      {viewMode === 'visual' && (
        <WorkflowToolbar 
          locked={locked}
          onToggleLock={() => setLocked(!locked)}
          onUndo={undo}
          onRedo={redo}
          canUndo={historyIndex > 0}
          canRedo={historyIndex < history.length - 1}
          onDuplicate={duplicateSelected}
          onDelete={deleteSelected}
          graphWidth={graphWidth}
          onConfigureViewport={() => setViewportDialogOpen(true)}
          executionViewport={effectiveViewport}
        />
      )}

      <div className="absolute top-4 right-4 z-10 flex items-center gap-2">
        <button
          onClick={() => handleViewModeChange('visual')}
          className={`toolbar-button ${viewMode === 'visual' ? 'active' : ''}`}
          title="Visual Builder"
        >
          <Eye size={18} />
        </button>
        <button
          onClick={() => handleViewModeChange('code')}
          className={`toolbar-button ${viewMode === 'code' ? 'active' : ''}`}
          title="JSON Editor"
        >
          <Code size={18} />
        </button>
      </div>

      {viewMode === 'visual' ? (
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
          style={{ width: '100%', height: '100%' }}
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
          <Background variant={BackgroundVariant.Dots} gap={12} size={1} color="#1a1d29" />
        </ReactFlow>
      ) : (
        <div className="absolute inset-0 flex flex-col bg-[#1e1e1e] border border-gray-800 rounded-lg overflow-hidden">
          <div className="flex-1 overflow-hidden">
            <Editor
              height="100%"
              defaultLanguage="json"
              value={codeValue}
              onChange={handleCodeChange}
              theme="vs-dark"
              options={{
                minimap: { enabled: false },
                fontSize: 13,
                lineNumbers: 'on',
                scrollBeyondLastLine: false,
                automaticLayout: true,
                tabSize: 2,
                insertSpaces: true,
                wordWrap: 'on',
                formatOnPaste: true,
                formatOnType: true,
                renderWhitespace: 'selection',
                scrollbar: {
                  verticalScrollbarSize: 10,
                  horizontalScrollbarSize: 10,
                },
              }}
            />
          </div>
          <div className="flex items-center justify-between border-t border-gray-800 bg-[#252526] px-4 py-3">
            <div className="flex items-center gap-3">
              <div className="text-xs text-gray-400">
                {codeValue.split('\n').length} lines
              </div>
              {codeError && (
                <>
                  <div className="w-px h-4 bg-gray-700" />
                  <div className="text-xs text-red-400">
                    {codeError}
                  </div>
                </>
              )}
            </div>
            <div className="flex items-center gap-2">
              <button
                onClick={handleResetCode}
                className="px-3 py-1.5 rounded-md text-xs bg-gray-700 text-gray-200 hover:bg-gray-600 transition-all disabled:opacity-50"
                disabled={!codeDirty}
              >
                Reset
              </button>
              <button
                onClick={() => applyCodeChanges()}
                className="px-3 py-1.5 rounded-md text-xs bg-purple-600 text-white hover:bg-purple-500 transition-all disabled:opacity-50"
                disabled={!codeDirty}
              >
                Apply Changes
              </button>
            </div>
          </div>
        </div>
      )}

      <ViewportDialog
        isOpen={isViewportDialogOpen}
        onDismiss={() => setViewportDialogOpen(false)}
        onSave={(viewport) => {
          handleViewportSave(viewport);
          setViewportDialogOpen(false);
        }}
        initialValue={effectiveViewport}
      />
    </div>
  );
}

interface ViewportDialogProps {
  isOpen: boolean;
  onDismiss: () => void;
  onSave: (viewport: ExecutionViewportSettings) => void;
  initialValue?: ExecutionViewportSettings;
}

function ViewportDialog({ isOpen, onDismiss, onSave, initialValue }: ViewportDialogProps) {
  const [widthValue, setWidthValue] = useState('');
  const [heightValue, setHeightValue] = useState('');
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!isOpen) {
      return;
    }

    const normalized = normalizeViewportSetting(initialValue);
    setWidthValue(String(normalized.width));
    setHeightValue(String(normalized.height));
    setError(null);
  }, [initialValue, isOpen]);

  const numericWidth = Number.parseInt(widthValue, 10);
  const numericHeight = Number.parseInt(heightValue, 10);
  const activePreset = determineViewportPreset(numericWidth, numericHeight);

  const handlePresetSelect = (presetViewport: ExecutionViewportSettings) => {
    const normalized = normalizeViewportSetting(presetViewport);
    setWidthValue(String(normalized.width));
    setHeightValue(String(normalized.height));
    setError(null);
  };

  const handleWidthChange = (value: string) => {
    setWidthValue(value.replace(/[^0-9]/g, ''));
  };

  const handleHeightChange = (value: string) => {
    setHeightValue(value.replace(/[^0-9]/g, ''));
  };

  const handleSave = () => {
    const parsedWidth = Number.parseInt(widthValue, 10);
    const parsedHeight = Number.parseInt(heightValue, 10);

    if (!Number.isFinite(parsedWidth) || parsedWidth < MIN_VIEWPORT_DIMENSION) {
      setError(`Width must be between ${MIN_VIEWPORT_DIMENSION} and ${MAX_VIEWPORT_DIMENSION} pixels.`);
      return;
    }

    if (!Number.isFinite(parsedHeight) || parsedHeight < MIN_VIEWPORT_DIMENSION) {
      setError(`Height must be between ${MIN_VIEWPORT_DIMENSION} and ${MAX_VIEWPORT_DIMENSION} pixels.`);
      return;
    }

    if (parsedWidth > MAX_VIEWPORT_DIMENSION || parsedHeight > MAX_VIEWPORT_DIMENSION) {
      setError(`Dimensions cannot exceed ${MAX_VIEWPORT_DIMENSION} pixels.`);
      return;
    }

    const width = clampViewportDimension(parsedWidth);
    const height = clampViewportDimension(parsedHeight);
    const preset = determineViewportPreset(width, height);

    onSave({ width, height, preset });
  };

  if (!isOpen) {
    return null;
  }

  const presetButtons: Array<{ id: ViewportPreset; label: string; viewport: ExecutionViewportSettings }> = [
    { id: 'desktop', label: 'Desktop', viewport: DEFAULT_DESKTOP_VIEWPORT },
    { id: 'mobile', label: 'Mobile', viewport: DEFAULT_MOBILE_VIEWPORT },
  ];

  return (
    <ResponsiveDialog
      isOpen={isOpen}
      onDismiss={onDismiss}
      ariaLabel="Configure execution dimensions"
      className="bg-flow-node border border-gray-800 rounded-lg shadow-2xl w-[360px] max-w-[90vw]"
    >
      <div className="px-6 py-4 border-b border-gray-800">
        <h2 className="text-lg font-semibold text-white">Execution dimensions</h2>
        <p className="mt-1 text-sm text-gray-400">Apply these dimensions to workflow runs and preview screenshots.</p>
      </div>

      <div className="px-6 py-5 space-y-5">
        <div>
          <span className="text-xs font-semibold uppercase tracking-wide text-gray-400">Presets</span>
          <div className="mt-2 grid grid-cols-2 gap-2">
            {presetButtons.map(({ id, label, viewport }) => {
              const isActive = activePreset === id;
              return (
                <button
                  type="button"
                  key={id}
                  onClick={() => handlePresetSelect(viewport)}
                  className={`flex flex-col rounded-md border px-3 py-2 text-left text-xs transition-colors ${
                    isActive
                      ? 'border-flow-accent bg-flow-accent/20 text-white'
                      : 'border-gray-700 text-gray-300 hover:border-flow-accent hover:text-white'
                  }`}
                >
                  <span className="font-semibold text-sm">{label}</span>
                  <span className="mt-0.5 text-[11px] text-gray-400">
                    {viewport.width} × {viewport.height}
                  </span>
                </button>
              );
            })}
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <label className="block text-xs font-semibold uppercase tracking-wide text-gray-400">
            Width (px)
            <input
              type="number"
              min={MIN_VIEWPORT_DIMENSION}
              max={MAX_VIEWPORT_DIMENSION}
              value={widthValue}
              onChange={(event) => handleWidthChange(event.target.value)}
              className="mt-1 w-full rounded-md border border-gray-700 bg-flow-bg px-3 py-2 text-sm text-gray-200 focus:border-flow-accent focus:outline-none"
            />
          </label>
          <label className="block text-xs font-semibold uppercase tracking-wide text-gray-400">
            Height (px)
            <input
              type="number"
              min={MIN_VIEWPORT_DIMENSION}
              max={MAX_VIEWPORT_DIMENSION}
              value={heightValue}
              onChange={(event) => handleHeightChange(event.target.value)}
              className="mt-1 w-full rounded-md border border-gray-700 bg-flow-bg px-3 py-2 text-sm text-gray-200 focus:border-flow-accent focus:outline-none"
            />
          </label>
        </div>

        <p className="text-xs text-gray-500">
          Recommended desktop preset works well for most workflows. Use custom dimensions for responsive testing or narrow layouts.
        </p>

        {error && (
          <div className="rounded-md border border-red-500/60 bg-red-500/10 px-3 py-2 text-xs text-red-300">
            {error}
          </div>
        )}
      </div>

      <div className="flex items-center justify-end gap-3 border-t border-gray-800 px-6 py-4">
        <button
          type="button"
          className="rounded-md border border-gray-700 bg-flow-bg px-4 py-2 text-sm font-semibold text-gray-300 hover:border-gray-500 hover:text-white"
          onClick={onDismiss}
        >
          Cancel
        </button>
        <button
          type="button"
          className="rounded-md bg-flow-accent px-4 py-2 text-sm font-semibold text-white hover:bg-blue-500 disabled:opacity-50"
          onClick={handleSave}
        >
          Save
        </button>
      </div>
    </ResponsiveDialog>
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
