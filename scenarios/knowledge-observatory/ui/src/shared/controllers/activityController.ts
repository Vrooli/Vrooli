// DOC: docs/concepts/ARCHITECTURE.md#ui-surface
import type { ActivityRecord, ActivityStatus } from "../lib/activityStore";

export type ActivityTone = "good" | "medium" | "poor";

export type ActivityView = {
  id: string;
  title: string;
  description?: string;
  metaLabel?: string;
  statusLabel: string;
  tone: ActivityTone;
  timestampLabel: string;
};

const formatRelativeTime = (value: string) => {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "Unknown";
  const diffMs = Date.now() - date.getTime();
  const seconds = Math.max(1, Math.round(diffMs / 1000));
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.round(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.round(hours / 24);
  return `${days}d ago`;
};

const statusLabel = (status?: ActivityStatus) => {
  switch (status) {
    case "running":
      return "Running";
    case "completed":
      return "Completed";
    case "failed":
      return "Failed";
    default:
      return "Updated";
  }
};

const statusTone = (status?: ActivityStatus): ActivityTone => {
  switch (status) {
    case "completed":
      return "good";
    case "failed":
      return "poor";
    case "running":
      return "medium";
    default:
      return "medium";
  }
};

const formatMeta = (record: ActivityRecord) => {
  if (!record.meta) return "";
  const entries = Object.entries(record.meta)
    .filter(([_, value]) => value && value.trim())
    .map(([key, value]) => `${key}: ${value}`);
  return entries.join(" • ");
};

export function buildActivityViews(records: ActivityRecord[]): ActivityView[] {
  return records.map((record) => {
    const metaLabel = formatMeta(record);
    return {
      id: record.id,
      title: record.title,
      description: record.description,
      metaLabel: metaLabel || undefined,
      statusLabel: statusLabel(record.status),
      tone: statusTone(record.status),
      timestampLabel: formatRelativeTime(record.createdAt),
    };
  });
}
