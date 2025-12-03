import { cn } from "../../lib/utils";

interface StatusPillProps {
  status: string;
  className?: string;
}

export function StatusPill({ status, className }: StatusPillProps) {
  const normalized = status.toLowerCase();
  const style =
    normalized === "completed" || normalized === "passed"
      ? "bg-emerald-500/20 text-emerald-300"
      : normalized === "running"
        ? "bg-cyan-500/20 text-cyan-200"
        : normalized === "queued" || normalized === "delegated"
          ? "bg-amber-400/20 text-amber-200"
          : normalized === "idle"
            ? "bg-white/10 text-slate-200"
            : "bg-red-500/20 text-red-200";

  return (
    <span className={cn("rounded-full px-3 py-1 text-xs uppercase tracking-wide", style, className)}>
      {status}
    </span>
  );
}
