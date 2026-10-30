import { Link } from "react-router-dom";
import type { UsageAction } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { useUsageForecast, useUsageRecommendations } from "../../features/usage/api";
import { windowKey } from "../../features/usage/format";
import { formatPoints } from "../../lib/format";

/** The highest-scoring quota lever stated in one window (actions arrive ranked); unproven ranges and per-trim figures are not savings. */
function biggestLever(actions: readonly UsageAction[], scope: { provider: string; pool: string; window: string }) {
  return actions.find((action) => action.source === "lever" && action.score > 0 && action.effect?.unit === "points" && action.effect.basis !== "range" && !action.effect.basis.startsWith("per ") && windowKey(action) === windowKey(scope));
}

/** Under a window that will run out: the biggest lever and where to act on it. Nothing when no window is at risk. */
export function UsageLeverLine() {
  const forecast = useUsageForecast();
  const recommendations = useUsageRecommendations();
  const exhausting = (forecast.data?.windows ?? []).find((window) => window.status === "will_exhaust");
  const top = exhausting ? biggestLever(recommendations.data?.actions ?? [], exhausting) : undefined;
  if (!exhausting || !top) return null;
  return (
    <p className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1 px-3 text-sm" data-testid="dashboard-usage-lever">
      <span className="min-w-0 break-words text-muted-foreground">
        Biggest lever: {top.action}, ≈ −{formatPoints(top.score)}/wk.
      </span>
      <Link to={`/usage?window=${encodeURIComponent(windowKey(exhausting))}`} className="inline-flex min-h-11 items-center font-medium text-primary hover:underline sm:min-h-8">
        What to do →
      </Link>
    </p>
  );
}
