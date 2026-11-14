import type { Reading } from "../lib/api";
import { formatCompactNumber } from "../lib/format";

export function TrendIndicator({ trend }: { trend: Reading["trend"] }) {
  if (!trend || (trend.state !== "meaningful" && trend.state !== "neutral") || !trend.movement) return null;
  const arrow = trend.movement === "up" ? "↑" : trend.movement === "down" ? "↓" : "→";
  const percent = typeof trend.percent === "number" ? ` ${trend.percent >= 0 ? "+" : ""}${trend.percent.toFixed(1)}%` : "";
  const label = `Trend ${trend.movement}${percent}${trend.polarity ? `, ${trend.polarity}` : ""}`;
  return <span className="cc-trend" data-trend={trend.movement} data-polarity={trend.polarity ?? "neutral"} aria-label={label} title={`${label} versus the previous comparison window`}><span aria-hidden="true">{arrow}</span>{percent || (typeof trend.delta === "number" ? ` ${formatCompactNumber(trend.delta)}` : "")}</span>;
}
