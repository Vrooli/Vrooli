import { Badge } from "../../components/ui/badge";
import { cn } from "../../lib/utils";

const toneByStatus: Record<string, string> = {
  pass: "border-emerald-400/30 bg-emerald-500/15 text-emerald-100",
  warn: "border-amber-400/30 bg-amber-500/15 text-amber-100",
  fail: "border-rose-400/30 bg-rose-500/15 text-rose-100",
  error: "border-rose-400/30 bg-rose-500/15 text-rose-100",
  warning: "border-amber-400/30 bg-amber-500/15 text-amber-100",
  info: "border-sky-400/30 bg-sky-500/15 text-sky-100",
  approved: "border-emerald-400/30 bg-emerald-500/15 text-emerald-100",
  approved_with_constraints: "border-cyan-400/30 bg-cyan-500/15 text-cyan-100",
  needs_review: "border-amber-400/30 bg-amber-500/15 text-amber-100",
  denied: "border-rose-400/30 bg-rose-500/15 text-rose-100",
  blocked: "border-rose-400/30 bg-rose-500/15 text-rose-100",
  deprecated: "border-zinc-400/30 bg-zinc-500/15 text-zinc-100",
  unrecorded: "border-amber-400/30 bg-amber-500/15 text-amber-100"
};

export function DecisionStatusBadge({ value, className }: { value: string; className?: string }) {
  const label = value ? value.replaceAll("_", " ") : "unknown";
  return (
    <Badge
      variant="outline"
      className={cn("whitespace-nowrap border text-[11px] normal-case tracking-normal", toneByStatus[value] ?? toneByStatus.info, className)}
    >
      {label}
    </Badge>
  );
}
