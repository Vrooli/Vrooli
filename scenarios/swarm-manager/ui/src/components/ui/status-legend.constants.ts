import type { StatusItem } from "./status-legend.types";

/**
 * Pre-configured status items for Backlog
 */
export const BACKLOG_STATUS_LEGEND_ITEMS: StatusItem[] = [
  {
    status: "backlog",
    label: "Backlog",
    colorClass: "bg-slate-600",
    description: "New backlog item, not yet started",
  },
  {
    status: "researching",
    label: "Researching",
    colorClass: "bg-blue-600",
    description: "Gathering information",
  },
  {
    status: "ready",
    label: "Ready",
    colorClass: "bg-green-600",
    description: "Ready to be queued",
  },
  {
    status: "queued",
    label: "Queued",
    colorClass: "bg-yellow-600",
    description: "Waiting to be processed",
  },
  {
    status: "in_progress",
    label: "In Progress",
    colorClass: "bg-purple-600",
    description: "Being implemented",
  },
  {
    status: "completed",
    label: "Completed",
    colorClass: "bg-emerald-600",
    description: "Implementation done",
  },
  {
    status: "failed",
    label: "Failed",
    colorClass: "bg-red-600",
    description: "Last execution failed",
  },
  {
    status: "archived",
    label: "Archived",
    colorClass: "bg-gray-600",
    description: "No longer active",
  },
];
