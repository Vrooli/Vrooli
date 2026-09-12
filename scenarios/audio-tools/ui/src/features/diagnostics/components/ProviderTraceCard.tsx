import { Panel } from "../../../components/ui/panel";
import { Badge } from "../../../components/ui/badge";
import { cn } from "../../../lib/utils";
import type { ProviderTrace, SuiteCapability } from "../../../services/diagnostics";
import { useTranslation } from "../../../i18n";
import { strings } from "../../../consts/strings";

export interface TraceEntry extends ProviderTrace {
  capability: string;
  emittedAt: number;
}

export type TraceFilter = "all" | SuiteCapability;

const FILTERS: TraceFilter[] = ["all", "stt", "tts", "summarize", "transcode"];

type Translate = (key: string, options?: Record<string, unknown>) => string;

interface Props {
  entries: TraceEntry[];
  filter?: TraceFilter;
  onFilterChange?: (f: TraceFilter) => void;
  /** When true, sticky-positioned on lg+ screens (page lays out 2-col). */
  sticky?: boolean;
}

export function ProviderTraceCard({ entries, filter = "all", onFilterChange, sticky }: Props) {
  const { t } = useTranslation();
  const tr = t as unknown as Translate;
  const visible = filter === "all" ? entries : entries.filter((e) => e.capability === filter);

  return (
    <aside
      className={cn(
        "flex flex-col gap-2",
        sticky && "lg:sticky lg:top-4 lg:max-h-[calc(100%-2rem)] lg:overflow-y-auto",
      )}
    >
      <Panel
        title={tr(strings.diagnostics.traceTitle)}
        description={tr(strings.diagnostics.traceDescription)}
        bodyless
      >
        <div className="flex flex-wrap gap-1 border-b border-app-border px-3 py-2" role="tablist" aria-label={tr(strings.diagnostics.suite.traceFilterTitle)}>
          {FILTERS.map((f) => {
            const active = filter === f;
            return (
              <button
                key={f}
                type="button"
                role="tab"
                aria-selected={active}
                onClick={() => onFilterChange?.(f)}
                disabled={!onFilterChange}
                data-testid={`trace-filter-${f}`}
                className={cn(
                  "rounded-control px-2 py-0.5 text-xs transition",
                  active
                    ? "bg-app-foreground text-app-background"
                    : "bg-app-surface-muted text-app-muted-foreground hover:text-app-foreground",
                  !onFilterChange && "cursor-default opacity-60",
                )}
              >
                {filterLabel(tr, f)}
              </button>
            );
          })}
        </div>
        {visible.length === 0 ? (
          <div className="p-4 text-sm text-app-muted-foreground">{tr(strings.diagnostics.traceEmpty)}</div>
        ) : (
          <ul className="divide-y divide-app-border">
            {visible.map((e) => (
              <li key={`${e.emittedAt}-${e.providerId}`} className="flex flex-col gap-1 px-4 py-2 text-xs">
                <div className="flex items-center gap-2">
                  <Badge variant="info">{e.capability}</Badge>
                  <span className="font-mono">{e.providerTier}</span>
                  <span className="text-app-muted-foreground">{e.providerId}</span>
                </div>
                <div className="flex items-center justify-between text-app-muted-foreground">
                  <span className="font-mono">{e.modelId || tr(strings.common.dash)}</span>
                  <span>{tr(strings.common.millisSuffix, { ms: Math.round(e.latencyMs) })}</span>
                </div>
              </li>
            ))}
          </ul>
        )}
      </Panel>
    </aside>
  );
}

function filterLabel(t: Translate, f: TraceFilter): string {
  switch (f) {
    case "all": return t(strings.diagnostics.suite.traceFilterAll);
    case "stt": return t(strings.diagnostics.suite.traceFilterSTT);
    case "tts": return t(strings.diagnostics.suite.traceFilterTTS);
    case "summarize": return t(strings.diagnostics.suite.traceFilterSummarize);
    case "transcode": return t(strings.diagnostics.suite.traceFilterTranscode);
  }
}
