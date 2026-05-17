import { Panel } from "../../components/ui/panel";
import { Badge } from "../../components/ui/badge";
import type { ProviderTrace } from "../../services/diagnostics";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";

export interface TraceEntry extends ProviderTrace {
  capability: string;
  emittedAt: number;
}

export function ProviderTraceCard({ entries }: { entries: TraceEntry[] }) {
  const { t } = useTranslation();
  return (
    <aside className="flex flex-col gap-2">
      <Panel title={t(strings.diagnostics.traceTitle)} description={t(strings.diagnostics.traceDescription)} bodyless>
        {entries.length === 0 ? (
          <div className="p-4 text-sm text-app-muted-foreground">{t(strings.diagnostics.traceEmpty)}</div>
        ) : (
          <ul className="divide-y divide-app-border">
            {entries.map((e) => (
              <li key={`${e.emittedAt}-${e.providerId}`} className="flex flex-col gap-1 px-4 py-2 text-xs">
                <div className="flex items-center gap-2">
                  <Badge variant="info">{e.capability}</Badge>
                  <span className="font-mono">{e.providerTier}</span>
                  <span className="text-app-muted-foreground">{e.providerId}</span>
                </div>
                <div className="flex items-center justify-between text-app-muted-foreground">
                  <span className="font-mono">{e.modelId || t(strings.common.dash)}</span>
                  <span>{t(strings.common.millisSuffix, { ms: Math.round(e.latencyMs) })}</span>
                </div>
              </li>
            ))}
          </ul>
        )}
      </Panel>
    </aside>
  );
}
