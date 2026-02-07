type KnownStatusVariant =
  | "pending"
  | "starting"
  | "running"
  | "needs_review"
  | "complete"
  | "failed"
  | "cancelled";

type StatusBadgeVariant = KnownStatusVariant | "secondary";

const KNOWN_STATUS_VARIANTS: readonly KnownStatusVariant[] = [
  "pending",
  "starting",
  "running",
  "needs_review",
  "complete",
  "failed",
  "cancelled",
];

function isKnownStatusVariant(status: string): status is KnownStatusVariant {
  return KNOWN_STATUS_VARIANTS.includes(status as KnownStatusVariant);
}

export function statusBadgeVariant(status: string): StatusBadgeVariant {
  if (isKnownStatusVariant(status)) {
    return status;
  }
  return "secondary";
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
