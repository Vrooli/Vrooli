import { Link } from "react-router-dom";
import { ArrowUpRight, Component, FileText, Layers, Layout, Puzzle, Settings2, Sparkles } from "lucide-react";
import type { LucideIcon } from "lucide-react";

import { Badge } from "../../components/ui/Badge";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { ROUTES } from "../../routes.generated";
import type { ProvenanceTag, SearchHit, SurfaceKind } from "../../api/search";

const KIND_ICON: Record<SurfaceKind, LucideIcon> = {
  component: Puzzle,
  page: FileText,
  feature: Sparkles,
  hook: Settings2,
  layout: Layout,
  other: Layers,
  unspecified: Component,
};

const PROVENANCE_TONE: Record<
  ProvenanceTag,
  "neutral" | "info" | "success" | "warn" | "error"
> = {
  custom: "info",
  "adopted-unmodified": "success",
  "adopted-modified": "warn",
  unknown: "neutral",
  unspecified: "neutral",
};

function formatScore(score: number): string {
  if (!Number.isFinite(score)) return "—";
  return score.toFixed(2);
}

function highlight(text: string, query: string): React.ReactNode {
  const trimmed = query.trim();
  if (trimmed.length === 0 || text.length === 0) return text;
  const safe = trimmed.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const parts = text.split(new RegExp(`(${safe})`, "ig"));
  return parts.map((part, idx) =>
    part.toLowerCase() === trimmed.toLowerCase() ? (
      <mark key={idx} className="bg-app-primary/20 px-0.5 text-app-foreground">
        {part}
      </mark>
    ) : (
      <span key={idx}>{part}</span>
    ),
  );
}

export interface ResultCardProps {
  hit: SearchHit;
  index: number;
  query: string;
}

export function ResultCard({ hit, index, query }: ResultCardProps) {
  const { t } = useTranslation();
  const Icon = KIND_ICON[hit.kind];
  const provenanceTone = PROVENANCE_TONE[hit.provenance];

  const kindLabel = t(
    hit.kind === "component"
      ? strings.pages.search.kind.component
      : hit.kind === "page"
      ? strings.pages.search.kind.page
      : hit.kind === "feature"
      ? strings.pages.search.kind.feature
      : hit.kind === "hook"
      ? strings.pages.search.kind.hook
      : hit.kind === "layout"
      ? strings.pages.search.kind.layout
      : hit.kind === "other"
      ? strings.pages.search.kind.other
      : strings.pages.search.kind.unspecified,
  );

  const provenanceLabel = t(
    hit.provenance === "custom"
      ? strings.pages.search.provenance.custom
      : hit.provenance === "adopted-unmodified"
      ? strings.pages.search.provenance.adoptedUnmodified
      : hit.provenance === "adopted-modified"
      ? strings.pages.search.provenance.adoptedModified
      : hit.provenance === "unknown"
      ? strings.pages.search.provenance.unknown
      : strings.pages.search.provenance.unspecified,
  );

  return (
    <article
      data-testid={selectors.search.resultRow({ index })}
      className="flex flex-col gap-2 rounded-panel border border-app-border bg-app-surface p-4 hover:border-app-primary/40"
      aria-labelledby={`search-result-${index}-title`}
    >
      <header className="flex items-start justify-between gap-2">
        <div className="flex min-w-0 flex-col gap-1">
          <h3
            id={`search-result-${index}-title`}
            className="flex items-center gap-2 text-sm font-semibold tracking-tight"
          >
            <Icon aria-hidden className="h-4 w-4 text-app-muted-foreground" />
            <span className="break-all">{highlight(hit.displayName || hit.slot, query)}</span>
          </h3>
          <p className="text-xs text-app-muted-foreground">
            <span className="font-mono">{hit.scenario}</span>
            {hit.slot ? (
              <>
                <span aria-hidden> · </span>
                <span className="font-mono">{hit.slot}</span>
              </>
            ) : null}
          </p>
        </div>
        <div
          className="text-xs font-mono text-app-muted-foreground tabular-nums"
          aria-label={t(strings.pages.search.results.score, { score: formatScore(hit.score) })}
          data-testid={selectors.search.resultScore({ index })}
        >
          {formatScore(hit.score)}
        </div>
      </header>

      {hit.description ? (
        <p className="text-sm text-app-foreground/90">{highlight(hit.description, query)}</p>
      ) : null}

      {hit.filePath ? (
        <p className="font-mono text-xs text-app-muted-foreground break-all">
          <span className="text-app-muted-foreground/80">
            {t(strings.pages.search.results.path)}:{" "}
          </span>
          {hit.filePath}
        </p>
      ) : null}

      <footer className="flex flex-wrap items-center justify-between gap-2 pt-1">
        <div className="flex flex-wrap items-center gap-2">
          <Badge tone="neutral">{kindLabel}</Badge>
          <Badge
            tone={provenanceTone}
            data-testid={selectors.search.resultProvenance({ index })}
          >
            {provenanceLabel}
          </Badge>
        </div>
        <Link
          to={ROUTES.surfaceDetail(`${hit.scenario}__${hit.slot || "_"}`)}
          className="inline-flex items-center gap-1 text-xs font-medium text-app-primary hover:underline"
          data-testid={selectors.search.resultOpen({ index })}
        >
          {t(strings.pages.search.results.openInInventory)}
          <ArrowUpRight aria-hidden className="h-3.5 w-3.5" />
        </Link>
      </footer>
    </article>
  );
}
