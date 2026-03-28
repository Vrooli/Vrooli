/**
 * Action Registry
 *
 * Defines inspector actions per lens × entity type combination.
 * Each action maps to an existing API endpoint or navigation route.
 */

import type { Node } from "@xyflow/react";
import type { LucideIcon } from "lucide-react";
import {
  Play,
  Square,
  RotateCcw,
  ListPlus,
  Eye,
  FileSearch,
  ClipboardCheck,
  XCircle,
  RefreshCw,
  Pencil,
  Trash2,
  PlusCircle,
  FolderOpen,
  Beaker,
  Link,
  Target,
  Archive,
  Users,
} from "lucide-react";
import type { GraphLens, EntityType } from "../stores/graph-data-store";
import { parseNodeId } from "./node-id-parser";
import { API_ENDPOINTS } from "../../../lib/api-endpoints";
import { defaultApiClient } from "../../../lib/api-client";

export interface InspectorAction {
  id: string;
  label: string;
  icon: LucideIcon;
  variant: "default" | "destructive";
  /** Execute the action. Returns void on success, throws on failure. */
  handler: (node: Node) => Promise<void>;
  /** If provided, determines whether this action is available for the given node. */
  enabled?: (node: Node) => boolean;
  /** If set, this action navigates to a route instead of calling an API. */
  navigateTo?: (node: Node) => string | null;
}

type ActionRegistry = Record<GraphLens, Partial<Record<EntityType, InspectorAction[]>>>;

/** Check if a node has a terminal execution status eligible for retry/review. */
function isTerminalExecution(node: Node): boolean {
  const status = (node.data as Record<string, unknown>).status as string | undefined;
  return status === "completed" || status === "failed" || status === "canceled";
}

/** Check if execution is active (can be cancelled). */
function isActiveExecution(node: Node): boolean {
  const status = (node.data as Record<string, unknown>).status as string | undefined;
  return status === "pending" || status === "scheduled" || status === "starting" || status === "in_progress" || status === "running" || status === "needs_review" || status === "validating" || status === "needs_fixup";
}

/** Check if a scenario is running. */
function isScenarioRunning(node: Node): boolean {
  const status = (node.data as Record<string, unknown>).status as string | undefined;
  return status === "running";
}

/** Check if a scenario is stopped. */
function isScenarioStopped(node: Node): boolean {
  const status = (node.data as Record<string, unknown>).status as string | undefined;
  return status === "stopped" || status === "error" || status === "unknown";
}

// ---------------------------------------------------------------------------
// Action factory helpers
// ---------------------------------------------------------------------------

function makeQueueAction(): InspectorAction {
  return {
    id: "queue",
    label: "Queue",
    icon: ListPlus,
    variant: "default",
    async handler(node: Node) {
      const parsed = parseNodeId(node.id);
      if (!parsed?.kind || !parsed?.name) throw new Error("Cannot determine backlog item identity");
      await defaultApiClient.post(API_ENDPOINTS.backlogQueue(parsed.kind, parsed.name), {});
    },
  };
}

function makeViewBacklogDetailsAction(): InspectorAction {
  return {
    id: "view-backlog-details",
    label: "View Details",
    icon: Eye,
    variant: "default",
    async handler() { /* navigation handled by navigateTo */ },
    navigateTo(node: Node) {
      const parsed = parseNodeId(node.id);
      if (!parsed?.kind || !parsed?.name) return null;
      return `/details/backlog/${parsed.kind}/${parsed.name}`;
    },
  };
}

function makeViewScenarioDetailsAction(): InspectorAction {
  return {
    id: "view-scenario-details",
    label: "View Details",
    icon: Eye,
    variant: "default",
    async handler() { /* navigation handled by navigateTo */ },
    navigateTo(node: Node) {
      const parsed = parseNodeId(node.id);
      if (!parsed?.name) return null;
      return `/details/scenario/${parsed.name}`;
    },
  };
}

function makeViewExecutionDetailsAction(): InspectorAction {
  return {
    id: "view-execution-details",
    label: "View Details",
    icon: Eye,
    variant: "default",
    async handler() { /* navigation handled by navigateTo */ },
    navigateTo(node: Node) {
      const parsed = parseNodeId(node.id);
      if (!parsed) return null;
      return `/details/execution/${parsed.identifier}`;
    },
  };
}

function makeViewPromptTraceAction(): InspectorAction {
  return {
    id: "view-prompt-trace",
    label: "Prompt Trace",
    icon: FileSearch,
    variant: "default",
    async handler() { /* navigation handled by navigateTo */ },
    navigateTo(node: Node) {
      const parsed = parseNodeId(node.id);
      if (!parsed) return null;
      return `/details/execution/${parsed.identifier}/prompt-trace`;
    },
  };
}

function makeFollowUpAction(): InspectorAction {
  return {
    id: "follow-up",
    label: "Follow-up",
    icon: RefreshCw,
    variant: "default",
    enabled: isTerminalExecution,
    async handler(node: Node) {
      const parsed = parseNodeId(node.id);
      if (!parsed) throw new Error("Cannot determine execution identity");
      await defaultApiClient.post(API_ENDPOINTS.executionFollowUp(parsed.identifier), {
        followUpType: "followup",
        runMode: "new",
      });
    },
  };
}

function makeRetryAction(): InspectorAction {
  return {
    id: "retry",
    label: "Retry",
    icon: RotateCcw,
    variant: "default",
    enabled: isTerminalExecution,
    async handler(node: Node) {
      const parsed = parseNodeId(node.id);
      if (!parsed) throw new Error("Cannot determine execution identity");
      await defaultApiClient.post(API_ENDPOINTS.executionRetry(parsed.identifier), {});
    },
  };
}

function makeTriggerReviewAction(): InspectorAction {
  return {
    id: "trigger-review",
    label: "Trigger Review",
    icon: ClipboardCheck,
    variant: "default",
    enabled: isTerminalExecution,
    async handler(node: Node) {
      const parsed = parseNodeId(node.id);
      if (!parsed) throw new Error("Cannot determine execution identity");
      await defaultApiClient.post(API_ENDPOINTS.executionTriggerReview(parsed.identifier), {});
    },
  };
}

function makeCancelExecutionAction(): InspectorAction {
  return {
    id: "cancel",
    label: "Cancel",
    icon: XCircle,
    variant: "destructive",
    enabled: isActiveExecution,
    async handler(node: Node) {
      const parsed = parseNodeId(node.id);
      if (!parsed) throw new Error("Cannot determine execution identity");
      await defaultApiClient.post(API_ENDPOINTS.executionCancel(parsed.identifier), {});
    },
  };
}

function makeScenarioStartAction(): InspectorAction {
  return {
    id: "start",
    label: "Start",
    icon: Play,
    variant: "default",
    enabled: isScenarioStopped,
    async handler(node: Node) {
      const parsed = parseNodeId(node.id);
      if (!parsed?.name) throw new Error("Cannot determine scenario name");
      await defaultApiClient.post(API_ENDPOINTS.scenarioStart(parsed.name), {});
    },
  };
}

function makeScenarioStopAction(): InspectorAction {
  return {
    id: "stop",
    label: "Stop",
    icon: Square,
    variant: "destructive",
    enabled: isScenarioRunning,
    async handler(node: Node) {
      const parsed = parseNodeId(node.id);
      if (!parsed?.name) throw new Error("Cannot determine scenario name");
      await defaultApiClient.post(API_ENDPOINTS.scenarioStop(parsed.name), {});
    },
  };
}

function makeScenarioRestartAction(): InspectorAction {
  return {
    id: "restart",
    label: "Restart",
    icon: RotateCcw,
    variant: "default",
    enabled: isScenarioRunning,
    async handler(node: Node) {
      const parsed = parseNodeId(node.id);
      if (!parsed?.name) throw new Error("Cannot determine scenario name");
      await defaultApiClient.post(API_ENDPOINTS.scenarioRestart(parsed.name), {});
    },
  };
}

function makeStopAgentRunAction(): InspectorAction {
  return {
    id: "stop-run",
    label: "Stop",
    icon: Square,
    variant: "destructive",
    async handler(node: Node) {
      const parsed = parseNodeId(node.id);
      if (!parsed) throw new Error("Cannot determine run identity");
      await defaultApiClient.post(API_ENDPOINTS.agentManagerStopRun(parsed.identifier), {});
    },
  };
}

// ---------------------------------------------------------------------------
// Topology action factory helpers
// ---------------------------------------------------------------------------

function makeCaptureClassifyAction(): InspectorAction {
  return {
    id: "classify",
    label: "Classify",
    icon: ClipboardCheck,
    variant: "default",
    async handler(node: Node) {
      const parsed = parseNodeId(node.id);
      if (!parsed) throw new Error("Cannot determine capture identity");
      await defaultApiClient.post(API_ENDPOINTS.captureClassify(parsed.identifier), {});
    },
  };
}

function makeCaptureCreateItemAction(): InspectorAction {
  return {
    id: "create-item",
    label: "Create Item",
    icon: PlusCircle,
    variant: "default",
    enabled(node: Node) {
      const status = (node.data as Record<string, unknown>).status as string | undefined;
      return status === "classified";
    },
    async handler(node: Node) {
      const parsed = parseNodeId(node.id);
      if (!parsed) throw new Error("Cannot determine capture identity");
      await defaultApiClient.post(API_ENDPOINTS.captureCreateItem(parsed.identifier), {});
    },
  };
}

function makeCaptureDeleteAction(): InspectorAction {
  return {
    id: "delete-capture",
    label: "Delete",
    icon: Trash2,
    variant: "destructive",
    async handler(node: Node) {
      const parsed = parseNodeId(node.id);
      if (!parsed) throw new Error("Cannot determine capture identity");
      await defaultApiClient.delete(API_ENDPOINTS.captureById(parsed.identifier));
    },
  };
}

function makeBacklogEditAction(): InspectorAction {
  return {
    id: "edit-backlog",
    label: "Edit",
    icon: Pencil,
    variant: "default",
    async handler() { /* navigation handled by navigateTo */ },
    navigateTo(node: Node) {
      const parsed = parseNodeId(node.id);
      if (!parsed?.kind || !parsed?.name) return null;
      return `/details/backlog/${parsed.kind}/${parsed.name}`;
    },
  };
}

function makeBacklogWorkshopAction(): InspectorAction {
  return {
    id: "workshop",
    label: "Workshop",
    icon: Beaker,
    variant: "default",
    async handler() { /* navigation handled by navigateTo */ },
    navigateTo(node: Node) {
      const parsed = parseNodeId(node.id);
      if (!parsed?.kind || !parsed?.name) return null;
      return `/details/backlog/${parsed.kind}/${parsed.name}?tab=workshop`;
    },
  };
}

function makeBacklogAddDependencyAction(): InspectorAction {
  return {
    id: "add-dependency",
    label: "Add Dependency",
    icon: Link,
    variant: "default",
    async handler() { /* navigation handled by navigateTo — opens detail page dependency section */ },
    navigateTo(node: Node) {
      const parsed = parseNodeId(node.id);
      if (!parsed?.kind || !parsed?.name) return null;
      return `/details/backlog/${parsed.kind}/${parsed.name}?tab=dependencies`;
    },
  };
}

function makeBacklogAssignInitiativeAction(): InspectorAction {
  return {
    id: "assign-initiative",
    label: "Assign Initiative",
    icon: Target,
    variant: "default",
    async handler() { /* navigation handled by navigateTo */ },
    navigateTo(node: Node) {
      const parsed = parseNodeId(node.id);
      if (!parsed?.kind || !parsed?.name) return null;
      return `/details/backlog/${parsed.kind}/${parsed.name}?tab=initiative`;
    },
  };
}

function makeBacklogViewFilesAction(): InspectorAction {
  return {
    id: "view-files",
    label: "View Files",
    icon: FolderOpen,
    variant: "default",
    async handler() { /* navigation handled by navigateTo */ },
    navigateTo(node: Node) {
      const parsed = parseNodeId(node.id);
      if (!parsed?.kind || !parsed?.name) return null;
      return `/details/backlog/${parsed.kind}/${parsed.name}?tab=files`;
    },
  };
}

function makeInitiativeEditAction(): InspectorAction {
  return {
    id: "edit-initiative",
    label: "Edit",
    icon: Pencil,
    variant: "default",
    async handler() { /* navigation handled by navigateTo */ },
    navigateTo(node: Node) {
      const parsed = parseNodeId(node.id);
      if (!parsed?.name) return null;
      return `/details/initiative/${parsed.name}`;
    },
  };
}

function makeInitiativeManageMembersAction(): InspectorAction {
  return {
    id: "manage-members",
    label: "Manage Items",
    icon: Users,
    variant: "default",
    async handler() { /* navigation handled by navigateTo */ },
    navigateTo(node: Node) {
      const parsed = parseNodeId(node.id);
      if (!parsed?.name) return null;
      return `/details/initiative/${parsed.name}?tab=items`;
    },
  };
}

function makeInitiativeArchiveAction(): InspectorAction {
  return {
    id: "archive-initiative",
    label: "Archive",
    icon: Archive,
    variant: "destructive",
    async handler(node: Node) {
      const parsed = parseNodeId(node.id);
      if (!parsed?.name) throw new Error("Cannot determine initiative name");
      await defaultApiClient.put(API_ENDPOINTS.initiativeByName(parsed.name), {
        status: "archived",
      });
    },
    enabled(node: Node) {
      const status = (node.data as Record<string, unknown>).status as string | undefined;
      return status !== "archived";
    },
  };
}

function makeScenarioViewFilesAction(): InspectorAction {
  return {
    id: "view-scenario-files",
    label: "View Files",
    icon: FolderOpen,
    variant: "default",
    async handler() { /* navigation handled by navigateTo */ },
    navigateTo(node: Node) {
      const parsed = parseNodeId(node.id);
      if (!parsed?.name) return null;
      return `/details/scenario/${parsed.name}?tab=files`;
    },
  };
}

function makeScenarioEditMetadataAction(): InspectorAction {
  return {
    id: "edit-scenario",
    label: "Edit Metadata",
    icon: Pencil,
    variant: "default",
    async handler() { /* navigation handled by navigateTo */ },
    navigateTo(node: Node) {
      const parsed = parseNodeId(node.id);
      if (!parsed?.name) return null;
      return `/details/scenario/${parsed.name}`;
    },
  };
}

// ---------------------------------------------------------------------------
// Registry
// ---------------------------------------------------------------------------

export const actionRegistry: ActionRegistry = {
  topology: {
    capture: [
      makeCaptureClassifyAction(),
      makeCaptureCreateItemAction(),
      makeCaptureDeleteAction(),
    ],
    backlog: [
      makeBacklogEditAction(),
      makeQueueAction(),
      makeBacklogWorkshopAction(),
      makeBacklogAddDependencyAction(),
      makeBacklogAssignInitiativeAction(),
      makeBacklogViewFilesAction(),
    ],
    initiative: [
      makeInitiativeEditAction(),
      makeInitiativeManageMembersAction(),
      makeInitiativeArchiveAction(),
    ],
    scenario: [
      makeScenarioViewFilesAction(),
      makeScenarioEditMetadataAction(),
    ],
  },
  flow: {
    backlog: [
      makeQueueAction(),
      makeViewBacklogDetailsAction(),
    ],
    execution: [
      makeViewExecutionDetailsAction(),
      makeViewPromptTraceAction(),
      makeFollowUpAction(),
      makeRetryAction(),
      makeTriggerReviewAction(),
      makeCancelExecutionAction(),
    ],
  },
  operations: {
    scenario: [
      makeScenarioStartAction(),
      makeScenarioStopAction(),
      makeScenarioRestartAction(),
      makeViewScenarioDetailsAction(),
    ],
    execution: [
      makeViewExecutionDetailsAction(),
      makeViewPromptTraceAction(),
      makeCancelExecutionAction(),
    ],
    "agent-run": [
      makeStopAgentRunAction(),
    ],
  },
};

/**
 * Get the list of actions for a given lens and entity type.
 * Returns an empty array if no actions are defined.
 */
export function getActionsForNode(lens: GraphLens, entityType: EntityType): InspectorAction[] {
  return actionRegistry[lens]?.[entityType] ?? [];
}
