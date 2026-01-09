import { cn } from "../../lib/utils";

interface StatCardProps {
  label: string;
  value: string | number;
  description?: string;
  className?: string;
}

export function StatCard({ label, value, description, className }: StatCardProps) {
  return (
    <article className={cn("rounded-2xl border border-white/5 bg-black/30 p-4", className)}>
      <p className="text-xs uppercase tracking-[0.3em] text-slate-400">{label}</p>
      <p className="mt-2 text-3xl font-semibold">
        {typeof value === "number" ? value.toString().padStart(2, "0") : value}
      </p>
      {description && <p className="mt-1 text-sm text-slate-300">{description}</p>}
    </article>
  );
}
