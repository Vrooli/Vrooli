import { useQuery } from "@tanstack/react-query";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { Card } from "@vrooli/react-component-library/Card/1";
import type { FallbackPairRow } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import type { TimeWindow } from "@vrooli/proto-types/measures/v1/measures_pb";
import { DataState } from "../../../components/feedback/DataState";
import { REQUEST_DEADLINE_MS } from "../../../lib/api";
import { formatCode, formatCount } from "../../../lib/format";
import { measuresClient } from "../../usage/measuresClient";

function window7d(): TimeWindow {
  const to = new Date(Math.floor(Date.now() / 60_000) * 60_000);
  return { window: { case: "custom", value: { from: timestampFromDate(new Date(to.getTime() - 7 * 86_400_000)), to: timestampFromDate(to) } } } as TimeWindow;
}

function Pairs({ title, rows }: { title: string; rows: FallbackPairRow[] }) {
  if (rows.length === 0) return null;
  return (
    <div className="space-y-1">
      <h4 className="text-sm font-medium">{title}</h4>
      <ul className="m-0 list-none space-y-1 p-0 text-sm text-muted-foreground">
        {rows.slice(0, 5).map((row) => <li key={`${row.from}>${row.to}>${row.reason}`}>{row.from} → {row.to} · {formatCode(row.reason)} · {formatCount(row.count)}×</li>)}
      </ul>
    </div>
  );
}

/** Runner and model fallbacks over the last 7 days (moved here from Usage › Details). */
export function FallbackInsights() {
  const query = useQuery({ queryKey: ["health", "fallback-insights"], queryFn: () => measuresClient.fallbackInsights({ window: window7d() }, { timeoutMs: REQUEST_DEADLINE_MS }), retry: false, refetchInterval: 60_000 });
  const data = query.data;
  return (
    <Card className="space-y-3 p-4" data-testid="fallback-insights">
      <h3 className="text-base font-semibold">Fallbacks · last 7 days</h3>
      <DataState compact subject="Fallback insights" loading={query.isLoading} error={query.error} isEmpty={!data} onRetry={() => void query.refetch()} empty={null}>
        {data ? (
          <div className="space-y-3">
            <p className="text-sm">
              Runner: {formatCount(data.runnerAttempts)} {data.runnerAttempts === 1n ? "fallback" : "fallbacks"}, {formatCount(data.runnerExhausted)} ran out of options.
              {" "}Model: {formatCount(data.modelAttempts)}, {formatCount(data.modelExhausted)} ran out.
            </p>
            <div className="grid gap-3 sm:grid-cols-2">
              <Pairs title="Runner fallbacks" rows={data.runnerPairs} />
              <Pairs title="Model fallbacks" rows={data.modelPairs} />
            </div>
          </div>
        ) : null}
      </DataState>
    </Card>
  );
}
