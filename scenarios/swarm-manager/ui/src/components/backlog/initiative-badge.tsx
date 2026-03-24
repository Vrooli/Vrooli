/**
 * Initiative Badge
 *
 * Small pill showing which initiative a backlog item belongs to.
 */

interface InitiativeBadgeProps {
  initiative?: string;
}

export function InitiativeBadge({ initiative }: InitiativeBadgeProps) {
  if (!initiative) return null;

  const label = initiative.length > 20 ? initiative.slice(0, 18) + "\u2026" : initiative;

  return (
    <span
      className="inline-flex items-center rounded-full bg-blue-500/15 px-2 py-0.5 text-[10px] font-medium text-blue-400"
      title={`Initiative: ${initiative}`}
    >
      {label}
    </span>
  );
}
