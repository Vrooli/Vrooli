import { cn } from "../../lib/utils";

interface InfoCardProps {
  label: string;
  value: React.ReactNode;
  className?: string;
}

/**
 * A simple card displaying a label and value, used throughout
 * the deployment manager UI for consistent information display.
 */
export function InfoCard({ label, value, className }: InfoCardProps) {
  return (
    <div className={cn("rounded-lg border border-white/10 bg-white/5 p-3", className)}>
      <p className="text-xs text-slate-400">{label}</p>
      <p className="font-semibold">{value}</p>
    </div>
  );
}
