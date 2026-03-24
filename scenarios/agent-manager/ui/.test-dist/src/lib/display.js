const KNOWN_STATUS_VARIANTS = [
    "pending",
    "starting",
    "running",
    "needs_review",
    "complete",
    "failed",
    "cancelled",
];
function isKnownStatusVariant(status) {
    return KNOWN_STATUS_VARIANTS.includes(status);
}
export function statusBadgeVariant(status) {
    if (isKnownStatusVariant(status)) {
        return status;
    }
    return "secondary";
}
export function formatStatusLabel(status) {
    return status
        .split("_")
        .map((word) => (word && word[0] ? word[0].toUpperCase() + word.slice(1) : word))
        .join(" ");
}
export function formatHyphenatedLabel(value) {
    return value
        .split("-")
        .map((word) => (word && word[0] ? word[0].toUpperCase() + word.slice(1) : word))
        .join(" ");
}
export function formatUnknownLabel(value) {
    if (!value || value === "unknown") {
        return "Unknown";
    }
    return value;
}
