interface StatusBadgeProps {
  active: boolean;
  activeLabel?: string;
  inactiveLabel?: string;
}

export function StatusBadge({
  active,
  activeLabel = "On",
  inactiveLabel = "Off",
}: StatusBadgeProps) {
  return (
    <span
      className={`rounded px-2 py-0.5 text-xs font-medium ${
        active
          ? "bg-[var(--status-healthy)]/20 text-[var(--status-healthy)]"
          : "bg-[var(--status-unknown)]/20 text-[var(--text-muted)]"
      }`}
    >
      {active ? activeLabel : inactiveLabel}
    </span>
  );
}
