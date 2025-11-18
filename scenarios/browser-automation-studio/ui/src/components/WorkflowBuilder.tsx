import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { logger } from "../utils/logger";
import { Eye, Code } from "lucide-react";
import Editor from "@monaco-editor/react";
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
} from "reactflow";
import { useWorkflowStore } from "../stores/workflowStore";
import { normalizeNodes, normalizeEdges } from "../utils/workflowNormalizers";
import BrowserActionNode from "./nodes/BrowserActionNode";
import NavigateNode from "./nodes/NavigateNode";
import ClickNode from "./nodes/ClickNode";
import TypeNode from "./nodes/TypeNode";
import ShortcutNode from "./nodes/ShortcutNode";
import ScreenshotNode from "./nodes/ScreenshotNode";
import WaitNode from "./nodes/WaitNode";
import ExtractNode from "./nodes/ExtractNode";
import AssertNode from "./nodes/AssertNode";
import WorkflowCallNode from "./nodes/WorkflowCallNode";
import ScriptNode from "./nodes/ScriptNode";
import KeyboardNode from "./nodes/KeyboardNode";
import HoverNode from "./nodes/HoverNode";
import DragDropNode from "./nodes/DragDropNode";
import FocusNode from "./nodes/FocusNode";
import BlurNode from "./nodes/BlurNode";
import ScrollNode from "./nodes/ScrollNode";
import RotateNode from "./nodes/RotateNode";
import GestureNode from "./nodes/GestureNode";
import SelectNode from "./nodes/SelectNode";
import SetVariableNode from "./nodes/SetVariableNode";
import UseVariableNode from "./nodes/UseVariableNode";
import UploadFileNode from "./nodes/UploadFileNode";
import TabSwitchNode from "./nodes/TabSwitchNode";
import FrameSwitchNode from "./nodes/FrameSwitchNode";
import SetCookieNode from "./nodes/SetCookieNode";
import GetCookieNode from "./nodes/GetCookieNode";
import ClearCookieNode from "./nodes/ClearCookieNode";
import SetStorageNode from "./nodes/SetStorageNode";
import GetStorageNode from "./nodes/GetStorageNode";
import ClearStorageNode from "./nodes/ClearStorageNode";
import NetworkMockNode from "./nodes/NetworkMockNode";
import ConditionalNode from "./nodes/ConditionalNode";
import LoopNode from "./nodes/LoopNode";
import WorkflowToolbar from "./WorkflowToolbar";
import CustomConnectionLine from "./CustomConnectionLine";
import "reactflow/dist/style.css";
import toast from "react-hot-toast";
import ResponsiveDialog from "./ResponsiveDialog";
import type {
  ExecutionViewportSettings,
  ViewportPreset,
} from "../stores/workflowStore";
import { selectors } from "../consts/selectors";
import { validateWorkflowDefinition } from "../utils/workflowValidation";
import type {
  WorkflowDefinition,
  WorkflowValidationResult,
} from "../types/workflow";

const nodeTypes: NodeTypes = {
  browserAction: BrowserActionNode,
  navigate: NavigateNode,
  click: ClickNode,
  hover: HoverNode,
  dragDrop: DragDropNode,
  focus: FocusNode,
  blur: BlurNode,
  scroll: ScrollNode,
  select: SelectNode,
  uploadFile: UploadFileNode,
  rotate: RotateNode,
  gesture: GestureNode,
  tabSwitch: TabSwitchNode,
  frameSwitch: FrameSwitchNode,
  conditional: ConditionalNode,
  setVariable: SetVariableNode,
  setCookie: SetCookieNode,
  getCookie: GetCookieNode,
  clearCookie: ClearCookieNode,
  setStorage: SetStorageNode,
  getStorage: GetStorageNode,
  clearStorage: ClearStorageNode,
  networkMock: NetworkMockNode,
  type: TypeNode,
  shortcut: ShortcutNode,
  keyboard: KeyboardNode,
  evaluate: ScriptNode,
  screenshot: ScreenshotNode,
  wait: WaitNode,
  extract: ExtractNode,
  assert: AssertNode,
  useVariable: UseVariableNode,
  workflowCall: WorkflowCallNode,
  loop: LoopNode,
};

const defaultEdgeOptions = {
  animated: true,
  type: "smoothstep",
  markerEnd: {
    type: MarkerType.ArrowClosed,
    width: 20,
    height: 20,
    color: "#4a5568",
  },
};

const DEFAULT_DESKTOP_VIEWPORT: ExecutionViewportSettings = {
  width: 1920,
  height: 1080,
  preset: "desktop",
};
const DEFAULT_MOBILE_VIEWPORT: ExecutionViewportSettings = {
  width: 390,
  height: 844,
  preset: "mobile",
};
const MIN_VIEWPORT_DIMENSION = 200;
const MAX_VIEWPORT_DIMENSION = 10000;

const clampViewportDimension = (value: number): number => {
  if (!Number.isFinite(value)) {
    return MIN_VIEWPORT_DIMENSION;
  }
  return Math.min(
    Math.max(Math.round(value), MIN_VIEWPORT_DIMENSION),
    MAX_VIEWPORT_DIMENSION,
  );
};

const determineViewportPreset = (
  width: number,
  height: number,
): ViewportPreset => {
  if (!Number.isFinite(width) || !Number.isFinite(height)) {
    return "custom";
  }
  if (
    width === DEFAULT_DESKTOP_VIEWPORT.width &&
    height === DEFAULT_DESKTOP_VIEWPORT.height
  ) {
    return "desktop";
  }
  if (
    width === DEFAULT_MOBILE_VIEWPORT.width &&
    height === DEFAULT_MOBILE_VIEWPORT.height
  ) {
    return "mobile";
  }
  return "custom";
};

const normalizeViewportSetting = (
  viewport?: ExecutionViewportSettings | null,
): ExecutionViewportSettings => {
  if (
    !viewport ||
    !Number.isFinite(viewport.width) ||
    !Number.isFinite(viewport.height)
  ) {
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
    logger.debug("Project ID loaded", {
      component: "WorkflowBuilder",
      projectId,
    });
  }

  // Use selectors to only subscribe to specific parts of the store
  const storeNodes = useWorkflowStore((state) => state.nodes);
  const storeEdges = useWorkflowStore((state) => state.edges);
  const updateWorkflow = useWorkflowStore((state) => state.updateWorkflow);
  const currentWorkflow = useWorkflowStore((state) => state.currentWorkflow);
  const isDirty = useWorkflowStore((state) => state.isDirty);
  const hasVersionConflict = useWorkflowStore(
    (state) => state.hasVersionConflict,
  );
  const scheduleAutosave = useWorkflowStore((state) => state.scheduleAutosave);
  const cancelAutosave = useWorkflowStore((state) => state.cancelAutosave);

  const [nodes, setNodes, onNodesChange] = useNodesState(storeNodes || []);
  const [edges, setEdges, onEdgesChange] = useEdgesState(storeEdges || []);
  const reactFlowInstance = useReactFlow();

  const graphContainerRef = useRef<HTMLDivElement | null>(null);
  const [graphWidth, setGraphWidth] = useState(0);
  const [isViewportDialogOpen, setViewportDialogOpen] = useState(false);
  const executionViewport = useWorkflowStore(
    (state) =>
      state.currentWorkflow?.executionViewport as
        | ExecutionViewportSettings
        | undefined,
  );
  const effectiveViewport = useMemo(
    () => normalizeViewportSetting(executionViewport),
    [executionViewport],
  );

  const connectingNodeId = useRef<string | null>(null);
  const isLoadingFromStore = useRef(false);

  // Sync FROM store TO local state (on workflow load)
  useEffect(() => {
    if (storeNodes) {
      isLoadingFromStore.current = true;
      setNodes(storeNodes as Node[]);
      // Reset flag after state update completes
      setTimeout(() => {
        isLoadingFromStore.current = false;
      }, 0);
    }
  }, [storeNodes, setNodes]);

  useEffect(() => {
    if (storeEdges) {
      isLoadingFromStore.current = true;
      setEdges(storeEdges as Edge[]);
      setTimeout(() => {
        isLoadingFromStore.current = false;
      }, 0);
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
      scheduleAutosave({ source: "autosave", changeDescription: "Autosave" });
    } else {
      cancelAutosave();
    }

    return () => {
      cancelAutosave();
    };
  }, [
    currentWorkflow?.id,
    isDirty,
    hasVersionConflict,
    scheduleAutosave,
    cancelAutosave,
  ]);

  // Toolbar state
  const [locked, setLocked] = useState(false);

  // Undo/Redo state
  interface WorkflowState {
    nodes: Node[];
    edges: Edge[];
  }
  const [history, setHistory] = useState<WorkflowState[]>([]);
  const [historyIndex, setHistoryIndex] = useState(-1);
  const [viewMode, setViewMode] = useState<"visual" | "code">("visual");
  const [codeValue, setCodeValue] = useState("");
  const [codeDirty, setCodeDirty] = useState(false);
  const [codeError, setCodeError] = useState<string | null>(null);
  const [validationResult, setValidationResult] =
    useState<WorkflowValidationResult | null>(null);
  const [isValidatingCode, setIsValidatingCode] = useState(false);

  useEffect(() => {
    if (viewMode !== "visual") {
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

    if (typeof ResizeObserver !== "undefined") {
      const observer = new ResizeObserver(() => updateWidth());
      observer.observe(element);
      return () => observer.disconnect();
    }

    window.addEventListener("resize", updateWidth);
    return () => {
      window.removeEventListener("resize", updateWidth);
    };
  }, [viewMode]);

  const buildJsonFromState = useCallback(() => {
    try {
      const baseDefinition = (currentWorkflow?.flow_definition ??
        currentWorkflow?.flowDefinition) as WorkflowDefinition | undefined;
      const definition: WorkflowDefinition = {
        nodes: nodes ?? [],
        edges: edges ?? [],
      };

      if (
        baseDefinition?.metadata &&
        Object.keys(baseDefinition.metadata).length > 0
      ) {
        definition.metadata = baseDefinition.metadata;
      }

      const mergedSettings: Record<string, unknown> = {
        ...(baseDefinition?.settings ?? {}),
      };
      if (effectiveViewport) {
        mergedSettings.executionViewport = effectiveViewport;
      }
      if (Object.keys(mergedSettings).length > 0) {
        definition.settings = mergedSettings;
      }

      return JSON.stringify(definition, null, 2);
    } catch (error) {
      logger.error(
        "Failed to stringify workflow definition",
        { component: "WorkflowBuilder", action: "handleSaveError" },
        error,
      );
      return '{\n  "nodes": [],\n  "edges": []\n}';
    }
  }, [currentWorkflow, nodes, edges, effectiveViewport]);

  useEffect(() => {
    if (!codeDirty || viewMode !== "code") {
      setCodeValue(buildJsonFromState());
      if (viewMode !== "code") {
        setCodeDirty(false);
        setCodeError(null);
      }
    }
  }, [buildJsonFromState, codeDirty, viewMode]);

  // Save current state to history
  const saveToHistory = useCallback(() => {
    const currentState = { nodes: [...nodes], edges: [...edges] };
    setHistory((prev) => {
      const newHistory = prev.slice(0, historyIndex + 1);
      newHistory.push(currentState);
      if (newHistory.length > 50) {
        newHistory.shift();
        setHistoryIndex((prevIndex) => prevIndex - 1);
      }
      return newHistory;
    });
    setHistoryIndex((prev) => prev + 1);
  }, [edges, historyIndex, nodes]);

  const applyCodeChanges = useCallback(
    async (options?: { silent?: boolean }) => {
      let parsedNodes: Node[] = [];
      let parsedEdges: Edge[] = [];

      let parsedDefinition: WorkflowDefinition = { nodes: [], edges: [] };
      try {
        parsedDefinition = JSON.parse(codeValue || "{}") as WorkflowDefinition;
        parsedNodes = normalizeNodes(parsedDefinition?.nodes ?? []);
        parsedEdges = normalizeEdges(parsedDefinition?.edges ?? []);
      } catch (error) {
        const message = error instanceof Error ? error.message : "Invalid JSON";
        setCodeError(message);
        toast.error(`Invalid workflow JSON: ${message}`);
        return false;
      }

      setIsValidatingCode(true);
      try {
        const normalizedDefinition: WorkflowDefinition = {
          ...parsedDefinition,
          nodes: parsedNodes,
          edges: parsedEdges,
        };
        const validation =
          await validateWorkflowDefinition(normalizedDefinition);
        setValidationResult(validation);
        if (!validation.valid) {
          const firstError =
            validation.errors[0]?.message ??
            "Workflow failed schema validation";
          setCodeError(firstError);
          toast.error(`Workflow validation failed: ${firstError}`);
          return false;
        }
        if (validation.warnings.length > 0 && !options?.silent) {
          const warningMessage =
            validation.warnings[0]?.message ??
            "Workflow validated with warnings";
          toast(`⚠️ ${warningMessage}`);
        }
      } catch (error) {
        const message =
          error instanceof Error ? error.message : "Workflow validation failed";
        setCodeError(message);
        toast.error(message);
        return false;
      } finally {
        setIsValidatingCode(false);
      }

      saveToHistory();
      setNodes(parsedNodes);
      setEdges(parsedEdges);
      setCodeDirty(false);
      setCodeError(null);
      if (!options?.silent) {
        toast.success("Workflow updated from JSON");
      }
      return true;
    },
    [codeValue, saveToHistory, setEdges, setNodes],
  );

  const handleViewModeChange = useCallback(
    async (mode: "visual" | "code") => {
      if (mode === viewMode) {
        return;
      }

      if (mode === "visual" && viewMode === "code" && codeDirty) {
        const applied = await applyCodeChanges({ silent: true });
        if (!applied) {
          return;
        }
      }

      setViewMode(mode);
      if (mode === "code") {
        setCodeValue(buildJsonFromState());
        setCodeDirty(false);
        setCodeError(null);
      }
    },
    [applyCodeChanges, buildJsonFromState, codeDirty, viewMode],
  );

  const handleCodeChange = (value: string | undefined) => {
    setCodeValue(value || "");
    setCodeDirty(true);
    setCodeError(null);
    setValidationResult(null);
  };

  const handleResetCode = () => {
    setCodeValue(buildJsonFromState());
    setCodeDirty(false);
    setCodeError(null);
    setValidationResult(null);
  };

  // Undo function
  const undo = useCallback(() => {
    if (historyIndex > 0) {
      const previousState = history[historyIndex - 1];
      setNodes(previousState.nodes);
      setEdges(previousState.edges);
      setHistoryIndex((prev) => prev - 1);
    }
  }, [history, historyIndex, setNodes, setEdges]);

  // Redo function
  const redo = useCallback(() => {
    if (historyIndex < history.length - 1) {
      const nextState = history[historyIndex + 1];
      setNodes(nextState.nodes);
      setEdges(nextState.edges);
      setHistoryIndex((prev) => prev + 1);
    }
  }, [history, historyIndex, setNodes, setEdges]);

  // Duplicate selected nodes
  const duplicateSelected = useCallback(() => {
    const selectedNodes = nodes.filter((node) => node.selected);
    if (selectedNodes.length === 0) return;

    saveToHistory();

    const newNodes = selectedNodes.map((node) => ({
      ...node,
      id: `${node.id}-copy-${Date.now()}`,
      position: {
        x: node.position.x + 50,
        y: node.position.y + 50,
      },
      selected: true,
    }));

    // Deselect original nodes and add new ones
    const updatedNodes = nodes.map((node) => ({ ...node, selected: false }));
    const allNodes = [...updatedNodes, ...newNodes];

    setNodes(allNodes);
  }, [nodes, edges, setNodes, saveToHistory]);

  // Delete selected nodes and edges
  const deleteSelected = useCallback(() => {
    const selectedNodeIds = nodes
      .filter((node) => node.selected)
      .map((node) => node.id);
    const selectedEdgeIds = edges
      .filter((edge) => edge.selected)
      .map((edge) => edge.id);

    if (selectedNodeIds.length === 0 && selectedEdgeIds.length === 0) return;

    saveToHistory();

    // Remove selected nodes
    const remainingNodes = nodes.filter((node) => !node.selected);

    // Remove selected edges and edges connected to deleted nodes
    const remainingEdges = edges.filter(
      (edge) =>
        !edge.selected &&
        !selectedNodeIds.includes(edge.source) &&
        !selectedNodeIds.includes(edge.target),
    );

    setNodes(remainingNodes);
    setEdges(remainingEdges);
  }, [nodes, edges, setNodes, setEdges, saveToHistory]);

  const handleViewportSave = useCallback(
    (viewport: ExecutionViewportSettings) => {
      updateWorkflow({ executionViewport: normalizeViewportSetting(viewport) });
    },
    [updateWorkflow],
  );

  const deriveConditionMetadata = useCallback((handleId?: string | null) => {
    if (!handleId) {
      return null;
    }
    if (handleId === "ifTrue") {
      return { condition: "if_true", label: "IF TRUE", stroke: "#4ade80" };
    }
    if (handleId === "ifFalse") {
      return { condition: "if_false", label: "IF FALSE", stroke: "#f87171" };
    }
    if (handleId === "loopBody") {
      return { condition: "loop_body", label: "LOOP BODY", stroke: "#38bdf8" };
    }
    if (handleId === "loopAfter") {
      return { condition: "loop_next", label: "AFTER LOOP", stroke: "#7c3aed" };
    }
    return null;
  }, []);

  const deriveTargetConditionMetadata = useCallback(
    (handleId?: string | null) => {
      if (!handleId) {
        return null;
      }
      if (handleId === "loopContinue") {
        return {
          condition: "loop_continue",
          label: "CONTINUE",
          stroke: "#22c55e",
        };
      }
      if (handleId === "loopBreak") {
        return { condition: "loop_break", label: "BREAK", stroke: "#f43f5e" };
      }
      return null;
    },
    [],
  );

  const enhanceConnection = useCallback(
    (connection: Connection): Edge => {
      const meta = deriveConditionMetadata(
        connection.sourceHandle ?? undefined,
      );
      const nextEdge: Edge = {
        ...connection,
        data: (connection as any).data
          ? { ...(connection as any).data }
          : undefined,
      } as Edge;
      if (meta) {
        nextEdge.data = { ...(nextEdge.data ?? {}), condition: meta.condition };
        nextEdge.label = meta.label;
        nextEdge.style = { ...(nextEdge.style ?? {}), stroke: meta.stroke };
      }
      const targetMeta = deriveTargetConditionMetadata(
        connection.targetHandle ?? undefined,
      );
      if (targetMeta) {
        nextEdge.data = {
          ...(nextEdge.data ?? {}),
          condition: targetMeta.condition,
        };
        nextEdge.label = targetMeta.label;
        nextEdge.style = {
          ...(nextEdge.style ?? {}),
          stroke: targetMeta.stroke,
        };
      }
      return nextEdge;
    },
    [deriveConditionMetadata, deriveTargetConditionMetadata],
  );

  const onConnect = useCallback(
    (params: Connection) => {
      setEdges((current) => addEdge(enhanceConnection(params), current));
    },
    [enhanceConnection, setEdges],
  );

  // Track when connection starts
  const onConnectStart = useCallback(
    (
      _: React.MouseEvent | React.TouchEvent,
      { nodeId }: { nodeId: string | null },
    ) => {
      connectingNodeId.current = nodeId;
    },
    [],
  );

  // Handle connection end - reset the connecting node id
  const onConnectEnd = useCallback(() => {
    connectingNodeId.current = null;
  }, []);

  const onNodesChangeHandler = useCallback(
    (changes: NodeChange[]) => {
      // Check if this is a significant change that should be saved to history
      const hasSignificantChange = changes.some(
        (change) =>
          change.type === "add" ||
          change.type === "remove" ||
          (change.type === "position" && "positionAbsolute" in change),
      );

      if (hasSignificantChange) {
        saveToHistory();
      }

      onNodesChange(changes);
    },
    [onNodesChange, saveToHistory],
  );

  const onEdgesChangeHandler = useCallback(
    (changes: EdgeChange[]) => {
      // Check if this is a significant change that should be saved to history
      const hasSignificantChange = changes.some(
        (change) => change.type === "add" || change.type === "remove",
      );

      if (hasSignificantChange) {
        saveToHistory();
      }

      onEdgesChange(changes);
    },
    [onEdgesChange, saveToHistory],
  );

  const onDrop = useCallback(
    (event: React.DragEvent) => {
      event.preventDefault();
      const type = event.dataTransfer.getData("nodeType");

      if (!type) return;

      const reactFlowBounds = event.currentTarget.getBoundingClientRect();
      const canvasPosition = {
        x: event.clientX - reactFlowBounds.left,
        y: event.clientY - reactFlowBounds.top,
      };

      const position =
        typeof reactFlowInstance.project === "function"
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
    [reactFlowInstance, setNodes],
  );

  const onDragOver = useCallback((event: React.DragEvent) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = "move";
  }, []);

  return (
    <div
      ref={graphContainerRef}
      className="flex-1 relative"
      data-testid={selectors.workflowBuilder.canvas.root}
    >
      {viewMode === "visual" && (
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

      <div
        className="absolute top-4 right-4 z-10 flex items-center gap-2"
        data-testid={selectors.workflowBuilder.viewModeToggle}
      >
        <button
          onClick={() => handleViewModeChange("visual")}
          className={`toolbar-button ${viewMode === "visual" ? "active" : ""}`}
          title="Visual Builder"
          data-testid={selectors.workflowBuilder.visualModeButton}
        >
          <Eye size={18} />
        </button>
        <button
          onClick={() => handleViewModeChange("code")}
          className={`toolbar-button ${viewMode === "code" ? "active" : ""}`}
          title="JSON Editor"
          data-testid={selectors.workflowBuilder.code.modeButton}
        >
          <Code size={18} />
        </button>
      </div>

      {viewMode === "visual" ? (
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
          style={{ width: "100%", height: "100%" }}
          connectOnClick={false}
          connectionMode={ConnectionMode.Loose}
          connectionRadius={50}
          connectionLineComponent={CustomConnectionLine}
          nodesDraggable={!locked}
          nodesConnectable={!locked}
          elementsSelectable={!locked}
          edgesUpdatable={!locked}
          data-testid={selectors.workflowBuilder.canvas.root}
        >
          <MiniMap
            nodeStrokeColor={(node) => {
              switch (node.type) {
                case "navigate":
                  return "#3b82f6";
                case "click":
                  return "#10b981";
                case "type":
                  return "#f59e0b";
                case "shortcut":
                  return "#6366f1";
                case "screenshot":
                  return "#8b5cf6";
                case "extract":
                  return "#ec4899";
                case "wait":
                  return "#6b7280";
                case "workflowCall":
                  return "#a855f7";
                default:
                  return "#4a5568";
              }
            }}
            nodeColor={(node) => {
              switch (node.type) {
                case "navigate":
                  return "#1e40af";
                case "click":
                  return "#065f46";
                case "type":
                  return "#92400e";
                case "shortcut":
                  return "#312e81";
                case "screenshot":
                  return "#5b21b6";
                case "extract":
                  return "#9f1239";
                case "wait":
                  return "#374151";
                case "workflowCall":
                  return "#7c3aed";
                default:
                  return "#1a1d29";
              }
            }}
            nodeBorderRadius={8}
          />
          <Background
            variant={BackgroundVariant.Dots}
            gap={12}
            size={1}
            color="#1a1d29"
          />
        </ReactFlow>
      ) : (
        <div
          className="absolute inset-0 flex flex-col bg-[#1e1e1e] border border-gray-800 rounded-lg overflow-hidden"
          data-testid={selectors.workflowBuilder.code.view}
        >
          <div
            className="flex-1 overflow-hidden"
            data-testid={selectors.workflowBuilder.code.editorContainer}
          >
            <Editor
              height="100%"
              defaultLanguage="json"
              value={codeValue}
              onChange={handleCodeChange}
              theme="vs-dark"
              data-testid={selectors.workflowBuilder.code.editor}
              options={{
                minimap: { enabled: false },
                fontSize: 13,
                lineNumbers: "on",
                scrollBeyondLastLine: false,
                automaticLayout: true,
                tabSize: 2,
                insertSpaces: true,
                wordWrap: "on",
                formatOnPaste: true,
                formatOnType: true,
                renderWhitespace: "selection",
                scrollbar: {
                  verticalScrollbarSize: 10,
                  horizontalScrollbarSize: 10,
                },
              }}
            />
          </div>
          <div
            className="flex items-center justify-between border-t border-gray-800 bg-[#252526] px-4 py-3"
            data-testid={selectors.workflowBuilder.code.toolbar}
          >
            <div className="flex items-center gap-3">
              <div
                className="text-xs text-gray-400"
                data-testid={selectors.workflowBuilder.code.lineCount}
              >
                {codeValue.split("\n").length} lines
              </div>
              {codeError && (
                <>
                  <div className="w-px h-4 bg-gray-700" />
                  <div
                    className="text-xs text-red-400"
                    data-testid={selectors.workflowBuilder.code.error}
                  >
                    {codeError}
                  </div>
                </>
              )}
              {validationResult && !codeError && (
                <>
                  <div className="w-px h-4 bg-gray-700" />
                  <div
                    className={`text-xs ${validationResult.valid ? "text-emerald-400" : "text-red-400"}`}
                    data-testid={selectors.workflowBuilder.code.validation}
                  >
                    {validationResult.valid
                      ? `Validated · ${validationResult.stats.node_count} nodes`
                      : "Validation failed"}
                  </div>
                </>
              )}
            </div>
            <div className="flex items-center gap-2">
              <button
                onClick={handleResetCode}
                className="px-3 py-1.5 rounded-md text-xs bg-gray-700 text-gray-200 hover:bg-gray-600 transition-all disabled:opacity-50"
                disabled={!codeDirty}
                data-testid={selectors.workflowBuilder.code.resetButton}
              >
                Reset
              </button>
              <button
                onClick={() => {
                  void applyCodeChanges();
                }}
                className="px-3 py-1.5 rounded-md text-xs bg-purple-600 text-white hover:bg-purple-500 transition-all disabled:opacity-50"
                disabled={!codeDirty || isValidatingCode}
                data-testid={selectors.workflowBuilder.code.applyButton}
              >
                {isValidatingCode ? "Validating…" : "Apply Changes"}
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

function ViewportDialog({
  isOpen,
  onDismiss,
  onSave,
  initialValue,
}: ViewportDialogProps) {
  const [widthValue, setWidthValue] = useState("");
  const [heightValue, setHeightValue] = useState("");
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
    setWidthValue(value.replace(/[^0-9]/g, ""));
  };

  const handleHeightChange = (value: string) => {
    setHeightValue(value.replace(/[^0-9]/g, ""));
  };

  const handleSave = () => {
    const parsedWidth = Number.parseInt(widthValue, 10);
    const parsedHeight = Number.parseInt(heightValue, 10);

    if (!Number.isFinite(parsedWidth) || parsedWidth < MIN_VIEWPORT_DIMENSION) {
      setError(
        `Width must be between ${MIN_VIEWPORT_DIMENSION} and ${MAX_VIEWPORT_DIMENSION} pixels.`,
      );
      return;
    }

    if (
      !Number.isFinite(parsedHeight) ||
      parsedHeight < MIN_VIEWPORT_DIMENSION
    ) {
      setError(
        `Height must be between ${MIN_VIEWPORT_DIMENSION} and ${MAX_VIEWPORT_DIMENSION} pixels.`,
      );
      return;
    }

    if (
      parsedWidth > MAX_VIEWPORT_DIMENSION ||
      parsedHeight > MAX_VIEWPORT_DIMENSION
    ) {
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

  const presetButtons: Array<{
    id: ViewportPreset;
    label: string;
    viewport: ExecutionViewportSettings;
  }> = [
    { id: "desktop", label: "Desktop", viewport: DEFAULT_DESKTOP_VIEWPORT },
    { id: "mobile", label: "Mobile", viewport: DEFAULT_MOBILE_VIEWPORT },
  ];

  return (
    <ResponsiveDialog
      isOpen={isOpen}
      onDismiss={onDismiss}
      ariaLabel="Configure execution dimensions"
      className="bg-flow-node border border-gray-800 rounded-lg shadow-2xl w-[360px] max-w-[90vw]"
      data-testid={selectors.viewport.dialog.root}
    >
      <div className="px-6 py-4 border-b border-gray-800">
        <h2
          className="text-lg font-semibold text-white"
          data-testid={selectors.viewport.dialog.title}
        >
          Execution dimensions
        </h2>
        <p className="mt-1 text-sm text-gray-400">
          Apply these dimensions to workflow runs and preview screenshots.
        </p>
      </div>

      <div className="px-6 py-5 space-y-5">
        <div>
          <span className="text-xs font-semibold uppercase tracking-wide text-gray-400">
            Presets
          </span>
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
                      ? "border-flow-accent bg-flow-accent/20 text-white"
                      : "border-gray-700 text-gray-300 hover:border-flow-accent hover:text-white"
                  }`}
                  data-testid={selectors.viewport.dialog.presetButton({
                    preset: id,
                  })}
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
              data-testid={selectors.viewport.dialog.widthInput}
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
              data-testid={selectors.viewport.dialog.heightInput}
            />
          </label>
        </div>

        <p className="text-xs text-gray-500">
          Recommended desktop preset works well for most workflows. Use custom
          dimensions for responsive testing or narrow layouts.
        </p>

        {error && (
          <div
            className="rounded-md border border-red-500/60 bg-red-500/10 px-3 py-2 text-xs text-red-300"
            data-testid={selectors.viewport.dialog.error}
          >
            {error}
          </div>
        )}
      </div>

      <div className="flex items-center justify-end gap-3 border-t border-gray-800 px-6 py-4">
        <button
          type="button"
          className="rounded-md border border-gray-700 bg-flow-bg px-4 py-2 text-sm font-semibold text-gray-300 hover:border-gray-500 hover:text-white"
          onClick={onDismiss}
          data-testid={selectors.viewport.dialog.cancelButton}
        >
          Cancel
        </button>
        <button
          type="button"
          className="rounded-md bg-flow-accent px-4 py-2 text-sm font-semibold text-white hover:bg-blue-500 disabled:opacity-50"
          onClick={handleSave}
          data-testid={selectors.viewport.dialog.saveButton}
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
