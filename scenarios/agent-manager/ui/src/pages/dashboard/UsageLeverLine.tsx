import { Link } from "react-router-dom";
import type { UsageRecommendationGroup } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { useUsageForecast, useUsageRecommendations } from "../../features/usage/api";
import { windowKey } from "../../features/usage/format";
import { formatPoints } from "../../lib/format";

/** The largest lever of the top-ranked target for one window (groups arrive ranked, levers largest first). */
function biggestLever(groups: readonly UsageRecommendationGroup[], scope: { provider: string; pool: string; window: string }) {
  const group = groups.find((candidate) => windowKey(candidate) === windowKey(scope) && candidate.levers.length > 0);
  const lever = group?.levers[0];
  return group && lever ? { action: lever.action, target: group.targetLabel, savedPoints: lever.savedPoints } : undefined;
}

/** Under a window that will run out: the biggest lever and where to act on it. Nothing when no window is at risk. */
export function UsageLeverLine() {
  const forecast = useUsageForecast();
  const recommendations = useUsageRecommendations();
  const exhausting = (forecast.data?.windows ?? []).find((window) => window.status === "will_exhaust");
  const top = exhausting ? biggestLever(recommendations.data?.groups ?? [], exhausting) : undefined;
  if (!exhausting || !top) return null;
  const windowId = windowKey(exhausting);
  // The server's action usually names its target already; add it only when it does not.
  const where = top.target && !top.action.includes(top.target) ? ` in "${top.target}"` : "";
  return (
    <p className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1 px-3 text-sm" data-testid="dashboard-usage-lever">
      <span className="min-w-0 break-words text-muted-foreground">
        Biggest lever: {top.action}{where}, ≈ −{formatPoints(top.savedPoints)}/wk.
      </span>
      <Link to={`/usage?window=${encodeURIComponent(windowId)}`} className="inline-flex min-h-11 items-center font-medium text-primary hover:underline sm:min-h-8">
        What to do →
      </Link>
    </p>
  );
}
