import { Menu as MenuIcon, Activity } from "lucide-react";
import { useQueries, useQuery } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { fetchHealth } from "../../api/health";
import { goldenClient } from "../../api/golden";
import { reportClient } from "../../api/report";
import { stalenessClient } from "../../api/staleness";
import { summarizeVerdicts } from "../../lib/verdict";
import { Badge, type BadgeProps } from "../ui/primitives/Badge";
import { cn } from "../lib/utils";

interface TopHeaderProps {
  onMenuToggle?: () => void;
}

/**
 * Sticky top header.
 *
 * Surfaces left-to-right:
 *   - hamburger toggle (mobile only)
 *   - app title
 *   - convergence chip (fanout of report.getGoldenSummary; counts
 *     goldens whose every verdict is PASS and not stale)
 *   - stale flag count chip (staleness.listStale length)
 *   - health status dot (live via /health poll)
 */
export function TopHeader({ onMenuToggle }: TopHeaderProps): ReactNode {
  const { t } = useTranslation();

  const healthQuery = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
    refetchInterval: 30000,
    retry: false,
  });

  const goldensQuery = useQuery({
    queryKey: ["goldens"],
    queryFn: () => goldenClient.listGoldens({}),
    staleTime: 30_000,
  });

  const goldens = goldensQuery.data?.goldens ?? [];
  const summaryQueries = useQueries({
    queries: goldens.map((g) => ({
      queryKey: ["goldenSummary", g.slug] as const,
      queryFn: () => reportClient.getGoldenSummary({ goldenSlug: g.slug }),
      staleTime: 30_000,
    })),
  });

  const stalenessQuery = useQuery({
    queryKey: ["staleness", "all"],
    queryFn: () => stalenessClient.listStale({}),
    refetchInterval: 60_000,
  });

  // Convergence: a golden "converges" when it has at least one verdict and
  // every verdict is PASS (not stale, not failed, not unexpected).
  let convergedCount = 0;
  for (const q of summaryQueries) {
    const sum = q.data?.summary;
    if (!sum) continue;
    const counts = summarizeVerdicts([
      ...sum.skillVerdicts,
      ...sum.toolVerdicts,
    ]);
    if (counts.total > 0 && counts.pass === counts.total) {
      convergedCount += 1;
    }
  }
  const totalGoldens = goldens.length;
  const convergenceText =
    totalGoldens === 0
      ? t(strings.nav.convergencePlaceholder)
      : t(strings.nav.convergenceCounts, {
          converged: convergedCount,
          total: totalGoldens,
        });
  const convergenceVariant: BadgeProps["variant"] =
    totalGoldens > 0 && convergedCount === totalGoldens
      ? "verdict-pass"
      : "neutral";

  const staleCount = stalenessQuery.data?.entries.length ?? 0;
  const staleText = stalenessQuery.isLoading
    ? "…"
    : staleCount === 0
      ? t(strings.nav.stalePlaceholder)
      : String(staleCount);
  const staleVariant: BadgeProps["variant"] =
    staleCount > 0 ? "verdict-stale" : "neutral";

  const healthLabel = healthQuery.isLoading
    ? "…"
    : healthQuery.error
      ? "offline"
      : healthQuery.data?.status ?? "unknown";
  const healthVariant: "verdict-pass" | "verdict-failure" | "neutral" = healthQuery.error
    ? "verdict-failure"
    : healthQuery.data?.status === "healthy"
      ? "verdict-pass"
      : healthQuery.data?.status === "degraded" || healthQuery.data?.status === "unhealthy"
        ? "verdict-failure"
        : "neutral";

  return (
    <header
      data-testid={selectors.nav.topHeader}
      className="sticky top-0 z-10 flex items-center justify-between gap-3 border-b border-app-border bg-app-shell px-4 py-2"
    >
      <div className="flex min-w-0 items-center gap-3">
        {onMenuToggle ? (
          <button
            type="button"
            data-testid={selectors.nav.topHeaderMenu}
            aria-label={t(strings.nav.menuToggle)}
            onClick={onMenuToggle}
            className="rounded-control p-2 text-app-foreground hover:bg-app-surface-muted md:hidden"
          >
            <MenuIcon className="h-5 w-5" />
          </button>
        ) : null}
        <h1 className="truncate text-sm font-semibold text-app-foreground">
          {t(strings.app.title)}
        </h1>
      </div>

      <div className="flex shrink-0 items-center gap-2">
        <Badge
          data-testid={selectors.nav.topHeaderConvergence}
          variant={convergenceVariant}
          className="hidden sm:inline-flex"
        >
          <span className="text-app-muted-foreground">{t(strings.nav.convergenceLabel)}:</span>
          <span>{convergenceText}</span>
        </Badge>
        <Badge
          data-testid={selectors.nav.topHeaderStale}
          variant={staleVariant}
          className="hidden sm:inline-flex"
        >
          <span className="text-app-muted-foreground">{t(strings.nav.staleLabel)}:</span>
          <span>{staleText}</span>
        </Badge>
        <span
          data-testid={selectors.nav.topHeaderHealth}
          aria-label={t(strings.nav.healthLabel)}
          className="flex items-center gap-1 rounded-pill border border-app-border-subtle bg-app-surface-muted px-2 py-0.5 text-xs"
        >
          <Activity
            className={cn(
              "h-3 w-3",
              healthVariant === "verdict-pass"
                ? "text-status-pass"
                : healthVariant === "verdict-failure"
                  ? "text-status-failure"
                  : "text-status-neutral",
            )}
          />
          <span className="text-app-muted-foreground">{healthLabel}</span>
        </span>
      </div>
    </header>
  );
}
