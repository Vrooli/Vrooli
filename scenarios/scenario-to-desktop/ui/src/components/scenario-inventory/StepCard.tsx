import { ReactNode } from "react";
import { CheckCircle, Circle } from "lucide-react";
import { cn } from "../../lib/utils";

interface StepCardProps {
  step: number;
  title: string;
  description: string;
  complete?: boolean;
  disabled?: boolean;
  children: ReactNode;
}

export function StepCard({ step, title, description, complete, disabled, children }: StepCardProps) {
  return (
    <div
      className={cn(
        "rounded-xl border p-4",
        complete ? "border-green-700 bg-green-950/5" : "border-slate-700 bg-slate-900/30",
        disabled && "opacity-50"
      )}
    >
      <div className="flex items-start gap-3">
        <div className="flex h-8 w-8 items-center justify-center rounded-full border border-slate-600 bg-slate-900 text-sm font-semibold">
          {complete ? <CheckCircle className="h-4 w-4 text-green-400" /> : <Circle className="h-4 w-4 text-slate-500" />}
        </div>
        <div className="flex-1 space-y-1">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs uppercase tracking-wide text-slate-400">Step {step}</p>
              <p className="font-semibold text-slate-100">{title}</p>
            </div>
          </div>
          <p className="text-xs text-slate-400">{description}</p>
          <div className="pt-3">{children}</div>
        </div>
      </div>
    </div>
  );
}
