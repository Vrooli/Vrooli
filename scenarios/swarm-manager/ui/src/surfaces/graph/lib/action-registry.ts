/**
 * Action Registry
 *
 * Defines inspector actions per lens × entity type combination.
 * Each action maps to an existing API endpoint or navigation route.
 */

import type { LucideIcon } from "lucide-react";
import {
  ListPlus,
  ClipboardCheck,
  Pencil,
  Trash2,
  PlusCircle,
  FolderOpen,
  Link,
  Target,
  Archive,
  Users,
} from "lucide-react";
import type { GraphLens } from "../stores/graph-data-store";
import type { EntityType } from "../stores/graph-settings-store";
import { getGraphNodeStatus, type GraphNode } from "../types";
import { parseNodeId } from "./node-id-parser";
import { API_ENDPOINTS } from "../../../lib/api-endpoints";
import { defaultApiClient } from "../../../lib/api-client";
import type { DetailRouteTarget } from "../../../app/routes/route-paths";

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
  navigateTo?: (node: GraphNode) => DetailRouteTarget | null;
}

type ActionRegistry = Record<GraphLens, Partial<Record<EntityType, InspectorAction[]>>>;

// ---------------------------------------------------------------------------
// Action factory helpers
// ---------------------------------------------------------------------------

/** Statuses where queuing is not applicable (already queued, in progress, or terminal). */
const NON_QUEUEABLE_STATUSES = new Set(["queued", "in_progress", "completed"]);

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

function makeBacklogAssignGoalAction(): InspectorAction {
  return {
    id: "assign-goal",
    label: "Assign Goal",
    icon: Target,
    variant: "default",
    async handler() { /* navigation handled by navigateTo */ },
    navigateTo(node: GraphNode) {
      const parsed = parseNodeId(node.id);
      if (!parsed?.kind || !parsed?.name) return null;
      return { entityType: "backlog", kind: parsed.kind, name: parsed.name, tab: "goal" };
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

function makeGoalEditAction(): InspectorAction {
  return {
    id: "edit-goal",
    label: "Edit",
    icon: Pencil,
    variant: "default",
    async handler() { /* navigation handled by navigateTo */ },
    navigateTo(node: GraphNode) {
      const parsed = parseNodeId(node.id);
      if (!parsed?.name) return null;
      return { entityType: "goal", name: parsed.name };
    },
  };
}

function makeGoalManageMembersAction(): InspectorAction {
  return {
    id: "manage-members",
    label: "Manage Items",
    icon: Users,
    variant: "default",
    async handler() { /* navigation handled by navigateTo */ },
    navigateTo(node: GraphNode) {
      const parsed = parseNodeId(node.id);
      if (!parsed?.name) return null;
      return { entityType: "goal", name: parsed.name, tab: "items" };
    },
  };
}

function makeGoalArchiveAction(): InspectorAction {
  return {
    id: "archive-goal",
    label: "Archive",
    icon: Archive,
    variant: "destructive",
    async handler(node: GraphNode) {
      const parsed = parseNodeId(node.id);
      if (!parsed?.name) throw new Error("Cannot determine goal name");
      await defaultApiClient.patch(API_ENDPOINTS.goalArchiveItem(parsed.name), {});
    },
    enabled(node: GraphNode) {
      // Check archivedAt from the node's extra data if available
      const data = node.data as Record<string, unknown> | undefined;
      return data?.archivedAt == null;
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
  // The plan lens renders the kanban board, not the node canvas — no
  // inspector actions apply.
  plan: {},
  focus: {
    capture: [
      makeCaptureClassifyAction(),
      makeCaptureCreateItemAction(),
      makeCaptureDeleteAction(),
    ],
    backlog: [
      makeBacklogEditAction(),
      makeQueueAction(),
      makeBacklogAddDependencyAction(),
      makeBacklogAssignGoalAction(),
      makeBacklogViewFilesAction(),
    ],
    goal: [
      makeGoalEditAction(),
      makeGoalManageMembersAction(),
      makeGoalArchiveAction(),
    ],
    scenario: [
      makeScenarioViewFilesAction(),
      makeScenarioEditMetadataAction(),
    ],
  },
  topology: {
    capture: [
      makeCaptureClassifyAction(),
      makeCaptureCreateItemAction(),
      makeCaptureDeleteAction(),
    ],
    backlog: [
      makeBacklogEditAction(),
      makeQueueAction(),
      makeBacklogAddDependencyAction(),
      makeBacklogAssignGoalAction(),
      makeBacklogViewFilesAction(),
    ],
    goal: [
      makeGoalEditAction(),
      makeGoalManageMembersAction(),
      makeGoalArchiveAction(),
    ],
    scenario: [
      makeScenarioViewFilesAction(),
      makeScenarioEditMetadataAction(),
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
