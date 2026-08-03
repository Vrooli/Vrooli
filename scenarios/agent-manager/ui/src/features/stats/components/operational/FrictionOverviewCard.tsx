import { useQuery } from "@tanstack/react-query";
import { useTimeWindow } from "../../hooks/useTimeWindow";
import {
  fetchExternalToolShare,
  fetchFileRereadRate,
  fetchFindingRecurrenceRate,
  fetchHelpRecoveryRate,
  fetchRepeatedWorkRate,
  fetchRetryRate,
  type ExternalToolShareMeasure,
  type FileRereadRateMeasure,
  type FindingRecurrenceRateMeasure,
  type MeasureWindow,
  type RateMeasure,
} from "../../api/statsClient";
import { MeasureFrame } from "../measure/MeasureFrame";
import { useMeasureDefinitions } from "../../hooks/useMeasureDefinitions";

type FrictionMetrics = {
  external: ExternalToolShareMeasure;
  retry: RateMeasure;
  helpRecovery: RateMeasure;
  repeatedWork: RateMeasure;
  rereads: FileRereadRateMeasure;
  findings: FindingRecurrenceRateMeasure;
};

const percent = (value: number) => `${(value * 100).toFixed(1)}%`;

export function FrictionOverviewCard() {
  const { preset } = useTimeWindow();
  const definitions = useMeasureDefinitions();
  const query = useQuery<FrictionMetrics, Error>({
    queryKey: ["typed-friction-measures", preset],
    queryFn: async () => {
      const window: MeasureWindow = { window: { custom: { from: windowStart(preset).toISOString(), to: new Date().toISOString() } } };
      const [external, retry, helpRecovery, repeatedWork, rereads, findings] = await Promise.all([
        fetchExternalToolShare(window), fetchRetryRate(window), fetchHelpRecoveryRate(window),
        fetchRepeatedWorkRate(window), fetchFileRereadRate(window), fetchFindingRecurrenceRate(window),
      ]);
      return { external, retry, helpRecovery, repeatedWork, rereads, findings };
    },
    staleTime: 30_000,
  });
  return (
    <section className="rounded-lg border border-border bg-card/40 p-4" data-testid="friction-overview-card">
      <h3 className="mb-1 text-sm font-semibold">Friction overview</h3>
      <p className="mb-3 text-xs text-muted-foreground">Durable invocation evidence; unknown ownership is excluded from external-tool share.</p>
      {query.isLoading && <p className="text-sm text-muted-foreground" data-testid="friction-overview-loading">Loading friction metrics…</p>}
      {query.error && <p className="text-sm text-destructive" role="alert" data-testid="friction-overview-error">Failed to load friction metrics: {query.error.message}</p>}
      {query.data && (
        <>
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3" data-testid="friction-overview-values">
            <MeasureFrame label="External tools" result={query.data.external} definition={definitions.data?.find((item) => item.id === "friction.external_tool_share")}><Metric label="External tools" value={percent(query.data.external.share)} /></MeasureFrame>
            <MeasureFrame label="Retries" result={query.data.retry} definition={definitions.data?.find((item) => item.id === "friction.retry_rate")}><Metric label="Retries" value={percent(query.data.retry.rate)} /></MeasureFrame>
            <MeasureFrame label="Help recovery" result={query.data.helpRecovery} definition={definitions.data?.find((item) => item.id === "friction.help_recovery_rate")}><Metric label="Help recovery" value={percent(query.data.helpRecovery.rate)} /></MeasureFrame>
            <MeasureFrame label="Repeated work" result={query.data.repeatedWork} definition={definitions.data?.find((item) => item.id === "friction.repeated_work_rate")}><Metric label="Repeated work" value={percent(query.data.repeatedWork.rate)} /></MeasureFrame>
            <MeasureFrame label="File rereads" result={query.data.rereads} definition={definitions.data?.find((item) => item.id === "friction.file_reread_rate")}><Metric label="File rereads" value={percent(query.data.rereads.rate)} /></MeasureFrame>
            <MeasureFrame label="Finding recurrence" result={query.data.findings} definition={definitions.data?.find((item) => item.id === "friction.finding_recurrence_rate")}><Metric label="Finding recurrence" value={percent(query.data.findings.rate)} /></MeasureFrame>
          </div>
          <p className="mt-3 text-xs text-muted-foreground">{query.data.external.resolvedCalls} resolved calls · {query.data.external.unknownCalls} unknown ownership · {query.data.rereads.readCalls} file reads</p>
          {query.data.retry.validity.largestFingerprintShare > 0 && <p className="mt-1 text-xs text-muted-foreground">Largest retry fingerprint: {(query.data.retry.validity.largestFingerprintShare * 100).toFixed(1)}% of the usable sample.</p>}
        </>
      )}
    </section>
  );
}

function windowStart(preset: "6h" | "12h" | "24h" | "7d" | "30d"): Date {
  const hours = { "6h": 6, "12h": 12, "24h": 24, "7d": 24 * 7, "30d": 24 * 30 }[preset];
  return new Date(Date.now() - hours * 60 * 60 * 1000);
}

function Metric({ label, value }: { label: string; value: string }) {
  return <div data-testid={`friction-metric-${label.toLowerCase().replaceAll(" ", "-")}`}><p className="text-xs text-muted-foreground">{label}</p><p className="text-lg font-semibold tabular-nums">{value}</p></div>;
}
