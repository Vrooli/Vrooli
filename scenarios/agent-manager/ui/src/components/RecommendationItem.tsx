import { Checkbox } from "./ui/checkbox";
import type { InvestigationRecommendation } from "../types";

const SEVERITY_STYLES: Record<string, string> = {
  Critical: "bg-destructive/10 text-destructive border-destructive/30",
  Major: "bg-amber-500/10 text-amber-600 border-amber-500/30",
  Gap: "bg-blue-500/10 text-blue-600 border-blue-500/30",
  Minor: "bg-muted text-muted-foreground border-border",
};

interface RecommendationItemProps {
  recommendation: InvestigationRecommendation;
  selected: boolean;
  onToggle: () => void;
}

export function RecommendationItem({
  recommendation,
  selected,
  onToggle,
}: RecommendationItemProps) {
  return (
    <div
      className={`flex flex-col gap-2 rounded-lg border p-3 transition-all ${
        selected ? "border-border bg-background" : "border-border/50 bg-muted/30"
      }`}
    >
      <div className="flex items-start gap-3">
        <Checkbox checked={selected} onCheckedChange={onToggle} className="mt-0.5" />

        <div className="flex-1 min-w-0 space-y-1">
          <button
            type="button"
            onClick={onToggle}
            className={`text-left text-sm transition-colors hover:text-primary ${
              selected ? "text-foreground" : "text-muted-foreground line-through"
            }`}
          >
            {recommendation.text}
          </button>

          {recommendation.severity && (
            <span
              className={`ml-0 inline-block rounded border px-1.5 py-0.5 text-xs font-medium ${
                SEVERITY_STYLES[recommendation.severity] ?? SEVERITY_STYLES.Minor
              }`}
            >
              {recommendation.severity}
            </span>
          )}

          {recommendation.evidence && (
            <p className="text-xs text-muted-foreground">{recommendation.evidence}</p>
          )}
        </div>
      </div>
    </div>
  );
}
