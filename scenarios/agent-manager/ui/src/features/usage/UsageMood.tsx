import { useState } from "react";
import { Card } from "@vrooli/react-component-library/Card/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { ButtonGroup } from "@vrooli/react-component-library/ButtonGroup/1";
import { Toggle } from "@vrooli/react-component-library/Toggle/1";
import { DataState } from "../../components/feedback/DataState";
import { formatCode, formatCount } from "../../lib/format";
import { ANNOTATION_WINDOWS, useAnnotationGate, useAnnotationSeries, type AnnotationWindow, type MoodSeries } from "./api";
import { gateWord, type AnnotationGate, type GateLabel } from "./annotationGate";
import { formatDayRange } from "./format";
import { MoodChart } from "./MoodChart";
import { Proof } from "./UsageAnnotations";

const TONES = { proven: "success", partial: "warning", unproven: "neutral" } as const;

const mean = (value: number | null | undefined) => (typeof value === "number" ? value.toFixed(2) : "—");

/** What the series can honestly say about this label: nothing seen, nothing proven, or some of it proven. */
export function moodVerdict(series: Pick<MoodSeries, "messages" | "proven_messages" | "points">): string {
  if (series.messages === 0) return "No human-written message was observed in this window: unknown, not neutral.";
  if (series.points.every((point) => typeof point.mean !== "number" && typeof point.proven_mean !== "number")) return "No per-message value is stored for this label, so there is nothing to chart.";
  if (series.proven_messages === 0) return "No proven model yet: the dashed line is unproven evidence of mood, not a measurement.";
  if (series.proven_messages < series.messages) return "Solid line: messages read by a proven model. Dashed line: every message, which is unproven evidence.";
  return "Every message in the window was read by a proven model.";
}

/** The numbers behind the chart, one row per bucket; the reader who cannot see a chart gets the same facts. */
function MoodTable({ series, label }: { series: MoodSeries; label: string }) {
  return (
    <details className="text-xs">
      <summary className="inline-flex min-h-11 cursor-pointer items-center text-primary sm:min-h-8">Show the numbers as a table</summary>
      <div className="overflow-x-auto">
        <table className="mt-1 w-full text-left tabular-nums" data-testid="mood-table">
          <caption className="sr-only">{label} per day</caption>
          <thead><tr><th scope="col" className="py-1 pr-3 font-medium">Day</th><th scope="col" className="py-1 pr-3 font-medium">Messages</th><th scope="col" className="py-1 pr-3 font-medium">Proven</th><th scope="col" className="py-1 pr-3 font-medium">Proven mean</th><th scope="col" className="py-1 pr-3 font-medium">All-model mean</th></tr></thead>
          <tbody>
            {series.points.map((point) => (
              <tr key={point.start} className="border-t border-border">
                <th scope="row" className="py-1 pr-3 font-normal">{formatDayRange(point.start)}</th>
                <td className="py-1 pr-3">{formatCount(point.messages)}</td>
                <td className="py-1 pr-3">{formatCount(point.proven_messages)}</td>
                <td className="py-1 pr-3">{mean(point.proven_mean)}</td>
                <td className="py-1 pr-3">{mean(point.mean)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </details>
  );
}

function MoodSeriesView({ label, window }: { label: GateLabel; window: AnnotationWindow }) {
  const series = useAnnotationSeries(label.id, window);
  const word = gateWord(label);
  const quantity = formatCode(label.id.split(".").pop());
  return (
    <section className="flex min-w-0 flex-col gap-2" data-testid="mood-label" aria-label={label.id}>
      <div className="flex flex-wrap items-center gap-2">
        <h4 className="min-w-0 break-all text-sm font-semibold">{label.id}</h4>
        <StatusBadge tone={TONES[word]}>{word}</StatusBadge>
      </div>
      <Proof label={label} />
      <DataState subject="Mood series" compact loading={series.isLoading} error={series.error} isEmpty={!series.data} onRetry={() => void series.refetch()} empty={null}>
        {series.data ? (
          <>
            <p className="text-xs text-muted-foreground" data-testid="mood-verdict">
              {formatCount(series.data.messages)} human-written {series.data.messages === 1 ? "message" : "messages"}, {formatCount(series.data.proven_messages)} from a proven model. {moodVerdict(series.data)}
            </p>
            <MoodChart points={series.data.points} since={series.data.window?.since} until={series.data.window?.until} label={`Mean probability a message is ${quantity.toLowerCase()}`} />
            {series.data.points.length > 0 ? (
              <>
                <ul className="m-0 flex list-none flex-wrap gap-x-4 gap-y-1 p-0 text-xs" aria-label="Chart key">
                  <li className="inline-flex items-center gap-1.5"><i className="inline-block h-1 w-5 rounded bg-primary" aria-hidden="true" />Proven model only</li>
                  <li className="inline-flex items-center gap-1.5"><i className="inline-block h-0 w-5 border-t-2 border-dotted border-muted-foreground" aria-hidden="true" />Every message (unproven)</li>
                </ul>
                <MoodTable series={series.data} label={`Mean probability a message is ${quantity.toLowerCase()}`} />
              </>
            ) : null}
          </>
        ) : null}
      </DataState>
    </section>
  );
}

/** Mood labels over time: each label that is asked of every human message, drawn from what a person wrote. */
export function Mood({ data, window, onWindow }: { data: AnnotationGate; window: AnnotationWindow; onWindow: (window: AnnotationWindow) => void }) {
  const labels = data.labels.filter((label) => label.scope === "message");
  const span = ANNOTATION_WINDOWS.find((entry) => entry.id === window)?.label ?? window;
  return (
    <Card className="flex flex-col gap-3 p-4" data-testid="usage-mood">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <h3 className="text-base font-semibold">Operator mood</h3>
          <p className="text-sm text-muted-foreground" data-testid="usage-mood-population">
            Read from human-written messages only; workflow and delegation briefs are excluded.{" "}
            {data.humanMessages === undefined
              ? `Nothing was annotated in the last ${span}, so the number of messages is unknown.`
              : `Last ${span}: ${formatCount(data.humanMessages)} messages in ${formatCount(data.humanTextRuns ?? 0)} runs.`}
            {data.staleRuns ? ` ${formatCount(data.staleRuns)} runs not yet classified are left out until the recompute finishes.` : ""}
          </p>
        </div>
        <ButtonGroup label="Window" collapse="never" className="flex-wrap">
          {ANNOTATION_WINDOWS.map((entry) => <Toggle key={entry.id} size="lg" pressed={window === entry.id} onPressedChange={() => onWindow(entry.id)}>{entry.label}</Toggle>)}
        </ButtonGroup>
      </div>
      {labels.length === 0 ? (
        <p className="text-sm" role="status">This pack defines no per-message mood label, so there is no series to draw.</p>
      ) : (
        <div className="grid gap-4 lg:grid-cols-3">
          {labels.map((label) => <MoodSeriesView key={label.id} label={label} window={window} />)}
        </div>
      )}
      <p className="text-xs text-muted-foreground">Each point is the mean probability over a day's messages. The solid line counts only messages read by a model whose hold-out measurement cleared the label's targets; the dashed line counts every message and is evidence, not a measurement.</p>
    </Card>
  );
}

/** Usage › Friction › Operator mood, read from the annotation gate and the per-message series. */
export function MoodSection() {
  const [window, setWindow] = useState<AnnotationWindow>("30d");
  const gate = useAnnotationGate(window);
  return (
    <DataState subject="Operator mood" compact loading={gate.isLoading} error={gate.error} isEmpty={!gate.data} onRetry={() => void gate.refetch()} empty={null}>
      {gate.data ? <Mood data={gate.data} window={window} onWindow={setWindow} /> : null}
    </DataState>
  );
}
