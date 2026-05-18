import { Menu as MenuIcon, Activity } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { fetchHealth } from "../../api/health";
import { Badge } from "../ui/primitives/Badge";
import { cn } from "../lib/utils";

interface TopHeaderProps {
  onMenuToggle?: () => void;
}

/**
 * Sticky top header.
 *
 * Surfaces left-to-right:
 *   - hamburger toggle (mobile only)
 *   - app title + active surface breadcrumb (todo: derive from route)
 *   - convergence chip (TODO: no backend API yet — neutral placeholder)
 *   - stale flag count chip (TODO: no backend API yet — static "0")
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

  // TODO: replace with convergence summary client once the backend Connect-RPC
  // service (verdicts.proto convergence summary RPC) lands. For now we show a
  // muted placeholder so the surface area is reserved but doesn't claim
  // false data.
  const convergencePlaceholder = t(strings.nav.convergencePlaceholder);

  // TODO: replace with stale-flag count from stale.proto once the API ships.
  // Per plan §scope: render a static "0" placeholder so the chip is visible
  // and the surface area is reserved.
  const stalePlaceholder = t(strings.nav.stalePlaceholder);

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
          variant="neutral"
          className="hidden sm:inline-flex"
        >
          <span className="text-app-muted-foreground">{t(strings.nav.convergenceLabel)}:</span>
          <span>{convergencePlaceholder}</span>
        </Badge>
        <Badge
          data-testid={selectors.nav.topHeaderStale}
          variant="neutral"
          className="hidden sm:inline-flex"
        >
          <span className="text-app-muted-foreground">{t(strings.nav.staleLabel)}:</span>
          <span>{stalePlaceholder}</span>
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
