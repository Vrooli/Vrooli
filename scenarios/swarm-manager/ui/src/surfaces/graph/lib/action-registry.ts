/**
 * Action Registry
 *
 * Defines inspector actions per lens × entity type combination.
 * Each action maps to an existing API endpoint or navigation route.
 */

import type { LucideIcon } from "lucide-react";
import {
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
import { getGraphNodeData, getGraphNodeStatus, type GraphNode } from "../types";
import { parseNodeId } from "./node-id-parser";
import { API_ENDPOINTS } from "../../../lib/api-endpoints";
import { defaultApiClient } from "../../../lib/api-client";
import type { DetailSelection } from "../../../stores/detail-selection-store";

export interface InspectorAction {
  id: string;
  label: string;
  icon: LucideIcon;
  variant: "default" | "destructive";
  /** Execute the action. Returns void on success, throws on failure. */
  handler: (node: GraphNode) => Promise<void>;
  /** If provided, determines whether this action is available for the given node. */
  enabled?: (node: GraphNode) => boolean;
  /** If set, this action opens a detail page instead of calling an API. */
  navigateTo?: (node: GraphNode) => DetailSelection | null;
}

type ActionRegistry = Record<GraphLens, Partial<Record<EntityType, InspectorAction[]>>>;

/** Check if a node has a terminal execution status eligible for retry/review. */
function isTerminalExecution(node: GraphNode): boolean {
  const status = getGraphNodeStatus(node);
  return status === "completed" || status === "failed" || status === "canceled";
}

/** Check if execution is active (can be cancelled). */
function isActiveExecution(node: GraphNode): boolean {
  const status = getGraphNodeStatus(node);
  return status === "pending" || status === "starting" || status === "in_progress" || status === "running" || status === "needs_review" || status === "validating" || status === "needs_fixup";
}

// ---------------------------------------------------------------------------
// Action factory helpers
// ---------------------------------------------------------------------------

/** Statuses where queuing is not applicable (already queued, in progress, or terminal). */
const NON_QUEUEABLE_STATUSES = new Set(["queued", "in_progress", "completed", "archived"]);

function makeQueueAction(): InspectorAction {
  return {
    id: "queue",
    label: "Queue",
    icon: ListPlus,
    variant: "default",
    enabled(node: GraphNode) {
      const status = getGraphNodeStatus(node);
      return typeof status === "string" && !NON_QUEUEABLE_STATUSES.has(status);
    },
    async handler(node: GraphNode) {
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
    navigateTo(node: GraphNode) {
      const parsed = parseNodeId(node.id);
      if (!parsed?.kind || !parsed?.name) return null;
      return { entityType: "backlog", kind: parsed.kind, name: parsed.name };
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
    navigateTo(node: GraphNode) {
      const parsed = parseNodeId(node.id);
      if (!parsed) return null;
      return { entityType: "execution", identifier: parsed.identifier };
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
    navigateTo(node: GraphNode) {
      const parsed = parseNodeId(node.id);
      if (!parsed) return null;
      return { entityType: "execution", identifier: parsed.identifier };
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
    async handler(node: GraphNode) {
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
    async handler(node: GraphNode) {
      const parsed = parseNodeId(node.id);
      if (!parsed) throw new Error("Cannot determine execution identity");
      await defaultApiClient.post(API_ENDPOINTS.executionRetry(parsed.identifier), {});
    },
  };
}

function makeTriggerReviewAction(): InspectorAction {
  return {
    id: "trigger-review",
    label: "Run Post-Run Checks",
    icon: ClipboardCheck,
    variant: "default",
    enabled: isTerminalExecution,
    async handler(node: GraphNode) {
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
    async handler(node: GraphNode) {
      const parsed = parseNodeId(node.id);
      if (!parsed) throw new Error("Cannot determine execution identity");
      await defaultApiClient.post(API_ENDPOINTS.executionCancel(parsed.identifier), {});
    },
  };
}

function makeOpenActivityOwnerAction(): InspectorAction {
  return {
    id: "open-owner",
    label: "Open Owner",
    icon: Eye,
    variant: "default",
    async handler() { /* navigation handled by navigateTo */ },
    navigateTo(node: GraphNode) {
      const data = getGraphNodeData(node);
      if (data.entityType !== "agent-activity") return null;
      if (data.ownerType === "backlog" && data.ownerKind) {
        return { entityType: "backlog", kind: data.ownerKind, name: data.ownerName };
      }
      if (data.ownerType === "scenario") {
        return { entityType: "scenario", name: data.ownerName };
      }
      return null;
    },
  };
}

function makeViewRecordedExecutionAction(): InspectorAction {
  return {
    id: "view-recorded-execution",
    label: "Execution",
    icon: Eye,
    variant: "default",
    enabled(node: GraphNode) {
      const data = getGraphNodeData(node);
      return data.entityType === "agent-activity" && Boolean(data.executionId);
    },
    async handler() { /* navigation handled by navigateTo */ },
    navigateTo(node: GraphNode) {
      const data = getGraphNodeData(node);
      if (data.entityType !== "agent-activity" || !data.executionId) {
        return null;
      }
      return { entityType: "execution", identifier: data.executionId };
    },
  };
}

function makeStopActivityRunAction(): InspectorAction {
  return {
    id: "stop-activity-run",
    label: "Stop",
    icon: Square,
    variant: "destructive",
    enabled(node: GraphNode) {
      const data = getGraphNodeData(node);
      return data.entityType === "agent-activity" && Boolean(data.runId);
    },
    async handler(node: GraphNode) {
      const data = getGraphNodeData(node);
      if (data.entityType !== "agent-activity" || !data.runId) {
        throw new Error("Cannot determine run identity");
      }
      await defaultApiClient.post(API_ENDPOINTS.agentManagerStopRun(data.runId), {});
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
    async handler(node: GraphNode) {
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
    enabled(node: GraphNode) {
      const status = getGraphNodeStatus(node);
      return status === "classified";
    },
    async handler(node: GraphNode) {
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
    async handler(node: GraphNode) {
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
    navigateTo(node: GraphNode) {
      const parsed = parseNodeId(node.id);
      if (!parsed?.kind || !parsed?.name) return null;
      return { entityType: "backlog", kind: parsed.kind, name: parsed.name };
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
    navigateTo(node: GraphNode) {
      const parsed = parseNodeId(node.id);
      if (!parsed?.kind || !parsed?.name) return null;
      return { entityType: "backlog", kind: parsed.kind, name: parsed.name, tab: "workshop" };
    },
  };
}

function makeBacklogAddDependencyAction(): InspectorAction {
  return {
    id: "add-dependency",
    label: "Add Dependency",
    icon: Link,
    variant: "default",
    async handler() { /* navigation handled by navigateTo */ },
    navigateTo(node: GraphNode) {
      const parsed = parseNodeId(node.id);
      if (!parsed?.kind || !parsed?.name) return null;
      return { entityType: "backlog", kind: parsed.kind, name: parsed.name, tab: "dependencies" };
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
    navigateTo(node: GraphNode) {
      const parsed = parseNodeId(node.id);
      if (!parsed?.kind || !parsed?.name) return null;
      return { entityType: "backlog", kind: parsed.kind, name: parsed.name, tab: "initiative" };
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
    navigateTo(node: GraphNode) {
      const parsed = parseNodeId(node.id);
      if (!parsed?.kind || !parsed?.name) return null;
      return { entityType: "backlog", kind: parsed.kind, name: parsed.name, tab: "files" };
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
    navigateTo(node: GraphNode) {
      const parsed = parseNodeId(node.id);
      if (!parsed?.name) return null;
      return { entityType: "initiative", name: parsed.name };
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
    navigateTo(node: GraphNode) {
      const parsed = parseNodeId(node.id);
      if (!parsed?.name) return null;
      return { entityType: "initiative", name: parsed.name, tab: "items" };
    },
  };
}

function makeInitiativeArchiveAction(): InspectorAction {
  return {
    id: "archive-initiative",
    label: "Archive",
    icon: Archive,
    variant: "destructive",
    async handler(node: GraphNode) {
      const parsed = parseNodeId(node.id);
      if (!parsed?.name) throw new Error("Cannot determine initiative name");
      await defaultApiClient.put(API_ENDPOINTS.initiativeByName(parsed.name), {
        status: "archived",
      });
    },
    enabled(node: GraphNode) {
      const status = getGraphNodeStatus(node);
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
    navigateTo(node: GraphNode) {
      const parsed = parseNodeId(node.id);
      if (!parsed?.name) return null;
      return { entityType: "scenario", name: parsed.name, tab: "files" };
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
    navigateTo(node: GraphNode) {
      const parsed = parseNodeId(node.id);
      if (!parsed?.name) return null;
      return { entityType: "scenario", name: parsed.name };
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
    "agent-activity": [
      makeOpenActivityOwnerAction(),
      makeViewRecordedExecutionAction(),
      makeStopActivityRunAction(),
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
    backlog: [
      makeQueueAction(),
      makeBacklogWorkshopAction(),
      makeBacklogViewFilesAction(),
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
};

/**
 * Get the list of actions for a given lens and entity type.
 * Returns an empty array if no actions are defined.
 */
export function getActionsForNode(lens: GraphLens, entityType: EntityType): InspectorAction[] {
  return actionRegistry[lens]?.[entityType] ?? [];
}
