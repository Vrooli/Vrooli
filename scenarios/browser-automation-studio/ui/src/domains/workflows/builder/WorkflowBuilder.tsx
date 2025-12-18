import { Circle, Code, Eye } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import toast from "react-hot-toast";
import ReactFlow, {
  addEdge,
  Background,
  BackgroundVariant,
  Connection,
  ConnectionMode,
  Edge,
  EdgeChange,
  MarkerType,
  MiniMap,
  Node,
  NodeChange,
  NodeTypes,
  ReactFlowProvider,
  useEdgesState,
  useNodesState,
  useReactFlow,
} from "reactflow";
import "reactflow/dist/style.css";
import { selectors } from "@constants/selectors";
import { useRegisterShortcuts } from "@hooks/useKeyboardShortcuts";
import { useReactFlowReady } from "@hooks/useReactFlowReady";
import type { ExecutionViewportSettings } from "@stores/workflowStore";
import { useWorkflowStore } from "@stores/workflowStore";
import { useSettingsStore } from "@stores/settingsStore";
import type {
  WorkflowDefinition,
  WorkflowValidationResult,
} from "@/types/workflow";
import { logger } from "@utils/logger";
import { autoLayoutNodes, normalizeEdges, normalizeNodes } from "../utils/normalizers";
import { buildActionDefinition } from "@utils/actionBuilder";
import { validateWorkflowDefinition } from "../validation/workflowValidation";
import { CustomConnectionLine } from "../components";
import AssertNode from "../nodes/AssertNode";
import BlurNode from "../nodes/BlurNode";
import BrowserActionNode from "../nodes/BrowserActionNode";
import ClearCookieNode from "../nodes/ClearCookieNode";
import ClearStorageNode from "../nodes/ClearStorageNode";
import ClickNode from "../nodes/ClickNode";
import ConditionalNode from "../nodes/ConditionalNode";
import DragDropNode from "../nodes/DragDropNode";
import ExtractNode from "../nodes/ExtractNode";
import FocusNode from "../nodes/FocusNode";
import FrameSwitchNode from "../nodes/FrameSwitchNode";
import GestureNode from "../nodes/GestureNode";
import GetCookieNode from "../nodes/GetCookieNode";
import GetStorageNode from "../nodes/GetStorageNode";
import HoverNode from "../nodes/HoverNode";
import KeyboardNode from "../nodes/KeyboardNode";
import LoopNode from "../nodes/LoopNode";
import NavigateNode from "../nodes/NavigateNode";
import NetworkMockNode from "../nodes/NetworkMockNode";
import RotateNode from "../nodes/RotateNode";
import ScreenshotNode from "../nodes/ScreenshotNode";
import ScriptNode from "../nodes/ScriptNode";
import ScrollNode from "../nodes/ScrollNode";
import SelectNode from "../nodes/SelectNode";
import SetCookieNode from "../nodes/SetCookieNode";
import SetStorageNode from "../nodes/SetStorageNode";
import SetVariableNode from "../nodes/SetVariableNode";
import ShortcutNode from "../nodes/ShortcutNode";
import TabSwitchNode from "../nodes/TabSwitchNode";
import TypeNode from "../nodes/TypeNode";
import UploadFileNode from "../nodes/UploadFileNode";
import UseVariableNode from "../nodes/UseVariableNode";
import WaitNode from "../nodes/WaitNode";
import SubflowNode from "../nodes/SubflowNode";
import WorkflowToolbar from "./WorkflowToolbar";
import { ViewportDialog, normalizeViewportSetting, CodeEditorPanel } from "./components";

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
  subflow: SubflowNode,
  loop: LoopNode,
};

// Edge marker colors by theme - darker for light mode, lighter for dark mode
const EDGE_MARKER_COLORS = {
  dark: "#6b7280",  // gray-500 - visible on dark backgrounds
  light: "#4b5563", // gray-600 - visible on light backgrounds
} as const;

const createEdgeOptions = (theme: "light" | "dark") => ({
  animated: true,
  type: "smoothstep",
  markerEnd: {
    type: MarkerType.ArrowClosed,
    width: 20,
    height: 20,
    color: EDGE_MARKER_COLORS[theme],
  },
});

interface WorkflowBuilderProps {
  projectId?: string;
  onStartRecording?: () => void;
}

function WorkflowBuilderInner({ projectId, onStartRecording }: WorkflowBuilderProps) {
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

  // Theme-aware edge options
  const getEffectiveTheme = useSettingsStore((state) => state.getEffectiveTheme);
  const effectiveTheme = getEffectiveTheme();
  const defaultEdgeOptions = useMemo(
    () => createEdgeOptions(effectiveTheme),
    [effectiveTheme],
  );

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
  const hasInitiallyFitView = useRef(false);

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

  // Auto-zoom to fit all nodes on initial workflow load
  useEffect(() => {
    // Only run once per workflow load when we have nodes and instance ready
    if (
      !hasInitiallyFitView.current &&
      storeNodes &&
      storeNodes.length > 0 &&
      reactFlowInstance &&
      typeof reactFlowInstance.fitView === "function"
    ) {
      // Small delay to ensure React Flow has rendered the nodes
      const timeoutId = setTimeout(() => {
        reactFlowInstance.fitView({ padding: 0.2, duration: 300 });
        hasInitiallyFitView.current = true;
      }, 100);
      return () => clearTimeout(timeoutId);
    }
  }, [storeNodes, reactFlowInstance]);

  // Reset fit view flag when workflow changes
  useEffect(() => {
    hasInitiallyFitView.current = false;
  }, [currentWorkflow?.id]);

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

  // React Flow ready state for test automation
  const isReactFlowReady = useReactFlowReady({
    reactFlowInstance,
    viewMode,
    nodeCount: nodes.length,
    workflowId: currentWorkflow?.id,
  });

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
      const baseDefinition = currentWorkflow?.flowDefinition as WorkflowDefinition | undefined;
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
        const initialNodes = normalizeNodes(parsedDefinition?.nodes ?? []);
        parsedEdges = normalizeEdges(parsedDefinition?.edges ?? []);
        parsedNodes = autoLayoutNodes(initialNodes, parsedEdges);
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

  // Select all nodes
  const selectAll = useCallback(() => {
    setNodes((nds) => nds.map((node) => ({ ...node, selected: true })));
    setEdges((eds) => eds.map((edge) => ({ ...edge, selected: true })));
  }, [setNodes, setEdges]);

  // Get zoom functions from React Flow
  const { zoomIn, zoomOut, fitView } = reactFlowInstance;

  // Register keyboard shortcuts for workflow builder
  const shortcutActions = useMemo(
    () => ({
      // Editing shortcuts
      'undo': undo,
      'redo': redo,
      'delete-selected': deleteSelected,
      'delete-selected-alt': deleteSelected,
      'select-all': selectAll,
      'duplicate-selected': duplicateSelected,
      // Canvas shortcuts
      'zoom-in': () => zoomIn(),
      'zoom-out': () => zoomOut(),
      'fit-view': () => fitView(),
      'toggle-lock': () => setLocked((prev) => !prev),
      // Workflow shortcuts
      'save-workflow': () => {
        // Trigger autosave immediately
        if (currentWorkflow && isDirty) {
          scheduleAutosave({ source: "manual", changeDescription: "Manual save" });
          toast.success("Workflow saved");
        } else if (currentWorkflow && !isDirty) {
          toast.success("No changes to save");
        }
      },
      'execute-workflow': () => {
        // Dispatch event to trigger execution in Header component
        window.dispatchEvent(new CustomEvent('execute-workflow'));
      },
    }),
    [
      undo,
      redo,
      deleteSelected,
      selectAll,
      duplicateSelected,
      zoomIn,
      zoomOut,
      fitView,
      currentWorkflow,
      isDirty,
      scheduleAutosave,
    ]
  );

  useRegisterShortcuts(shortcutActions, viewMode === 'visual' && !locked);

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

      const newNodeId = `node-${Date.now()}`;
      const nodeData = { label: type.charAt(0).toUpperCase() + type.slice(1) };
      const newNode: Node = {
        id: newNodeId,
        type,
        position,
        data: nodeData,
        // V2 action field for type-safe execution
        action: buildActionDefinition(type, nodeData),
      } as Node;

      // Auto-connect: Find nodes that have no outgoing edges (end of chain)
      // If there's exactly one such node, auto-connect to the new node
      const currentNodes = nodes;
      const currentEdges = edges;

      // Get all node IDs that have outgoing edges
      const nodesWithOutgoingEdges = new Set(currentEdges.map((e) => e.source));

      // Find nodes without outgoing edges (potential chain ends)
      const chainEndNodes = currentNodes.filter(
        (node) => !nodesWithOutgoingEdges.has(node.id)
      );

      setNodes((nds) => nds.concat(newNode));

      // If there's exactly one chain end, auto-connect it to the new node
      if (chainEndNodes.length === 1) {
        const sourceNode = chainEndNodes[0];
        const newEdge: Edge = {
          id: `edge-${sourceNode.id}-${newNodeId}`,
          source: sourceNode.id,
          target: newNodeId,
          type: "smoothstep",
          animated: true,
          markerEnd: {
            type: MarkerType.ArrowClosed,
            width: 20,
            height: 20,
            color: EDGE_MARKER_COLORS[effectiveTheme],
          },
        };
        setEdges((eds) => eds.concat(newEdge));
      }
    },
    [reactFlowInstance, setNodes, setEdges, nodes, edges, effectiveTheme],
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
      data-builder-ready={viewMode === "visual" && isReactFlowReady ? "true" : "false"}
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
        {onStartRecording && (
          <button
            onClick={onStartRecording}
            className="toolbar-button flex items-center gap-1 bg-red-500/20 hover:bg-red-500/30 text-red-100 border border-red-500/40"
            title="Start Record Mode"
          >
            <Circle size={14} className="text-red-300 fill-red-300" />
            <span className="text-xs font-medium">Record</span>
          </button>
        )}
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
        >
          {/* Empty canvas guidance overlay */}
          {nodes.length === 0 && (
            <div
              className="absolute inset-0 flex items-center justify-center pointer-events-none z-10"
              data-testid="workflow-empty-state"
            >
              <div className="text-center max-w-md px-6">
                <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-flow-node/80 border border-flow-border/60 flex items-center justify-center">
                  <svg
                    className="w-8 h-8 text-flow-text-muted"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={1.5}
                      d="M12 4v16m8-8H4"
                    />
                  </svg>
                </div>
                <h3 className="text-lg font-semibold text-flow-text mb-2">
                  Start building your workflow
                </h3>
                <p className="text-sm text-flow-text-muted mb-4">
                  Drag a <span className="text-blue-400 font-medium">Navigate</span> node from the
                  sidebar to begin, then chain additional actions like Click, Type, or Screenshot.
                </p>
                <div className="flex items-center justify-center gap-2 text-xs text-flow-text-secondary">
                  <kbd className="px-2 py-1 bg-flow-node rounded border border-flow-border/60 font-mono">
                    Drag
                  </kbd>
                  <span>from sidebar</span>
                  <span className="text-flow-text-muted">→</span>
                  <span>drop here</span>
                </div>
              </div>
            </div>
          )}
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
                case "subflow":
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
                case "subflow":
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
        <CodeEditorPanel
          codeValue={codeValue}
          codeError={codeError}
          codeDirty={codeDirty}
          validationResult={validationResult}
          isValidatingCode={isValidatingCode}
          onCodeChange={handleCodeChange}
          onResetCode={handleResetCode}
          onApplyChanges={() => void applyCodeChanges()}
        />
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

// Wrap with ReactFlowProvider to use hooks
function WorkflowBuilder(props: WorkflowBuilderProps) {
  return (
    <ReactFlowProvider>
      <WorkflowBuilderInner {...props} />
    </ReactFlowProvider>
  );
}

export default WorkflowBuilder;
