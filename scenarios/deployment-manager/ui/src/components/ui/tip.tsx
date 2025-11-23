import { type ReactNode } from "react";
import { Info } from "lucide-react";
import { cn } from "../../lib/utils";

type Tone = "info" | "success" | "warning";

const toneStyles: Record<Tone, string> = {
  info: "border-cyan-400/60 bg-cyan-500/5",
  success: "border-emerald-400/60 bg-emerald-500/5",
  warning: "border-amber-400/60 bg-amber-500/5",
};

interface TipProps {
  title?: string;
  children?: ReactNode;
  icon?: ReactNode;
  tone?: Tone;
  className?: string;
  action?: ReactNode;
}

export function Tip({
  title,
  children,
  icon,
  tone = "info",
  className,
  action,
}: TipProps) {
  return (
    <div
      className={cn(
        "flex gap-3 rounded-lg border border-l-4 border-white/10 p-4 text-sm text-slate-200",
        toneStyles[tone],
        className,
      )}
    >
      <div className="mt-0.5 text-cyan-300">
        {icon ?? <Info className="h-5 w-5" />}
      </div>
      <div className="flex-1 space-y-1">
        {title && <p className="font-semibold">{title}</p>}
        {children}
      </div>
      {action && <div className="flex items-start">{action}</div>}
    </div>
  );
}
