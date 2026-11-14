import type { StatusTone } from "@vrooli/react-component-library/StatusBadge/1";

// One mapping from a run/task/check status word to the shared status-badge tone.
const STATUS_TONES: Record<string, StatusTone> = {
  complete: "success",
  completed: "success",
  approved: "success",
  succeeded: "success",
  ok: "success",
  pending: "neutral",
  queued: "neutral",
  cancelled: "neutral",
  starting: "info",
  running: "info",
  needs_review: "warning",
  rejected: "warning",
  parked: "warning",
  failed: "danger",
};

export function statusTone(status: string): StatusTone {
  return STATUS_TONES[status] ?? "neutral";
}

export function formatStatusLabel(status: string): string {
  return status
    .split("_")
    .map((word) => (word && word[0] ? word[0].toUpperCase() + word.slice(1) : word))
    .join(" ");
}

export function formatHyphenatedLabel(value: string): string {
  return value
    .split("-")
    .map((word) => (word && word[0] ? word[0].toUpperCase() + word.slice(1) : word))
    .join(" ");
}

export function formatUnknownLabel(value: string): string {
  if (!value || value === "unknown") {
    return "Unknown";
  }
  return value;
}
