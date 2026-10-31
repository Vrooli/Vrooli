import { Info } from "lucide-react";
import { cn } from "../../lib/utils";

interface InsufficientDataCardProps {
  label: string;
  reason: string;
  have?: number;
  required?: number;
  testId?: string;
  compact?: boolean;
}

export function InsufficientDataCard({ label, reason, have, required, testId, compact }: InsufficientDataCardProps) {
  return (
    <div
      className={cn(
        "rounded-lg border border-border/60 bg-card/40 p-3",
        compact ? "" : "min-h-[72px]",
      )}
      data-testid={testId}
    >
      <p className="text-xs text-muted-foreground">{label}</p>
      <div className="mt-1 flex items-start gap-1.5 text-sm text-muted-foreground">
        <Info className="mt-0.5 h-3.5 w-3.5 shrink-0" />
        <div>
          <p className="text-foreground">Not enough data yet</p>
          <p className="text-xs">
            {reason}
            {typeof have === "number" && typeof required === "number" && (
              <span className="ml-1 opacity-70">
                ({have} of {required} needed)
              </span>
            )}
          </p>
        </div>
      </div>
    </div>
  );
}
