import { intentAccent, type Intent } from "../../lib/constants";
import type { LucideIcon } from "lucide-react";

export const StatusTile = ({
  icon: Icon,
  label,
  value,
  meta,
  intent
}: {
  icon: LucideIcon;
  label: string;
  value: string;
  meta?: string;
  intent: Intent;
}) => (
  <div className={`rounded-2xl border px-4 py-5 shadow-lg shadow-black/20 ${intentAccent[intent]}`}>
    <div className="flex items-center justify-between text-xs uppercase tracking-[0.2em] text-white/70">
      <span>{label}</span>
      <Icon className="h-4 w-4" />
    </div>
    <p className="mt-3 text-2xl font-semibold">{value}</p>
    {meta ? <p className="text-sm text-white/70">{meta}</p> : null}
  </div>
);
