import type { EvalRun } from "../../api/evals";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

/**
 * Trend renders two sparklines over a suite's runs in chronological (oldest →
 * newest) order: mean strong-query top-1 score (higher is better) and the worst
 * gibberish leakage (lower is better). It is descriptive, not a verdict — the
 * whole point of the harness is watching these move across tagged experiments.
 *
 * Pure SVG, no chart dependency: each series is normalized to [0,1] (scores
 * already live there) and drawn as a polyline in a fixed viewbox.
 */
export function Trend({ runs }: { runs: readonly EvalRun[] }) {
  const { t } = useTranslation();
  // ListRuns returns newest-first; the trend reads oldest → newest.
  const ordered = [...runs].reverse();
  if (ordered.length < 2) return null;

  const strong = ordered.map((r) => r.aggregate?.meanStrongTop1 ?? 0);
  const gibberish = ordered.map((r) => r.aggregate?.maxGibberishScore ?? 0);

  return (
    <div data-testid="evals-trend-inner" className="grid grid-cols-1 gap-3 sm:grid-cols-2">
      <Sparkline label={t(strings.evals.trendStrong)} values={strong} tone="positive" />
      <Sparkline label={t(strings.evals.trendGibberish)} values={gibberish} tone="negative" />
    </div>
  );
}

function Sparkline({
  label,
  values,
  tone,
}: {
  label: string;
  values: number[];
  tone: "positive" | "negative";
}) {
  const w = 200;
  const h = 40;
  const pad = 2;
  const n = values.length;
  const points = values
    .map((v, i) => {
      const clamped = Math.max(0, Math.min(1, v));
      const x = pad + (i * (w - 2 * pad)) / Math.max(1, n - 1);
      const y = h - pad - clamped * (h - 2 * pad);
      return `${x.toFixed(1)},${y.toFixed(1)}`;
    })
    .join(" ");
  const last = values[values.length - 1] ?? 0;
  const stroke = tone === "positive" ? "var(--color-app-primary)" : "var(--color-app-destructive)";

  return (
    <div className="flex flex-col gap-1">
      <div className="flex items-baseline justify-between text-xs text-app-muted-foreground">
        <span>{label}</span>
        <span className="font-mono text-app-foreground">{last.toFixed(3)}</span>
      </div>
      <svg
        viewBox={`0 0 ${w} ${h}`}
        role="img"
        aria-label={`${label}: ${last.toFixed(3)}`}
        className="h-10 w-full rounded-control border border-app-border bg-app-surface"
        preserveAspectRatio="none"
      >
        <polyline points={points} fill="none" stroke={stroke} strokeWidth="1.5" />
      </svg>
    </div>
  );
}
