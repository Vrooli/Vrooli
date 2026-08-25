import { useMemo, useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { TriggerComposition, Verdict } from "@vrooli/proto-types/offer-desk/v1/offers/offers_pb";
import type { TFunction } from "i18next";

import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { FormSection } from "@vrooli/react-component-library/FormSection/1.0.0";
import { DirtyStateGuard } from "@vrooli/react-component-library/DirtyStateGuard/1.0.0";
import { Button } from "@vrooli/react-component-library/Button/1.2.0";
import { Input } from "../components/ui/input";
import { Select } from "@vrooli/react-component-library/Select/1.1.0";
import { DataTable } from "@vrooli/react-component-library/DataTable/1.2.0";
import { useSurfaceState } from "../hooks/useSurfaceState";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { addFact, declareTrigger, evaluateTriggers } from "../api/offers";

const enumLabel = (value: number | string | undefined, values: Record<number, string>, fallback: string) => {
  if (typeof value === "number") return values[value] ?? fallback;
  return value || fallback;
};
const verdictName = (value: number | string | undefined) => enumLabel(value, Verdict, "VERDICT_UNSPECIFIED");
const verdictCopy = (value: number | string | undefined, translate: TFunction) => {
  switch (verdictName(value)) {
    case "SATISFIED": return translate(strings.pages.triggers.verdictSatisfied);
    case "UNSATISFIED": return translate(strings.pages.triggers.verdictUnsatisfied);
    case "UNKNOWN": return translate(strings.pages.triggers.verdictUnknown);
    default: return verdictName(value);
  }
};
const factAge = (value: bigint | number | undefined) => value === undefined ? "unknown" : `${value.toString()}s`;
const timestampLabel = (timestamp?: { seconds: bigint | number; nanos?: number }) => timestamp ? new Date(Number(timestamp.seconds) * 1000 + Math.floor((timestamp.nanos ?? 0) / 1_000_000)).toISOString() : "unknown";

type EvaluationLike = { id: string; nodeId?: string; verdict: Verdict | string; factName?: string; factNames?: string[]; explanation?: string; evaluatedAt?: { seconds: bigint | number; nanos?: number }; factAgeSeconds?: bigint | number };

export function TriggersPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const evaluation = useQuery({ queryKey: ["trigger-evaluation"], queryFn: () => evaluateTriggers(true), retry: false });
  const [latestEvaluation, setLatestEvaluation] = useState<typeof evaluation.data>();
  const [triggerForm, setTriggerForm] = useState({ nodeId: "", factName: "", operator: ">=", threshold: "", secondFactName: "", secondOperator: ">=", secondThreshold: "", composition: String(TriggerComposition.ALL) });
  const [factForm, setFactForm] = useState({ name: "", value: "", observedAt: new Date().toISOString().slice(0, 10), staleAfterDays: "30", dimension: "" });
  const [message, setMessage] = useState("");
  const [changeError, setChangeError] = useState(false);
  const evaluated = latestEvaluation ?? evaluation.data;
  const evaluations = useMemo(() => (evaluated?.evaluations ?? []) as EvaluationLike[], [evaluated?.evaluations]);
  const parseError = evaluation.isError || changeError;
  const surface = useSurfaceState({ query: { isLoading: evaluation.isLoading, isFetching: evaluation.isFetching, isError: evaluation.isError, error: evaluation.error }, empty: Boolean(evaluated && evaluations.length === 0) });

  const declareMutation = useMutation({
    mutationFn: () => declareTrigger({
      nodeId: triggerForm.nodeId.trim(),
      factName: triggerForm.factName.trim(),
      operator: triggerForm.operator,
      threshold: Number(triggerForm.threshold),
      clauses: triggerForm.secondFactName.trim() ? [{ factName: triggerForm.secondFactName.trim(), operator: triggerForm.secondOperator, threshold: Number(triggerForm.secondThreshold) }] : [],
      composition: Number(triggerForm.composition),
    }),
    onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ["trigger-evaluation"] }); setTriggerForm({ nodeId: "", factName: "", operator: ">=", threshold: "", secondFactName: "", secondOperator: ">=", secondThreshold: "", composition: String(TriggerComposition.ALL) }); setChangeError(false); setMessage(t(strings.pages.triggers.savedNotice)); },
    onError: () => setChangeError(true),
  });
  const factMutation = useMutation({
    mutationFn: () => addFact({ name: factForm.name.trim(), value: Number(factForm.value), observedAt: new Date(factForm.observedAt), staleAfterDays: Number(factForm.staleAfterDays), dimension: factForm.dimension.trim() }),
    onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ["trigger-evaluation"] }); setFactForm({ name: "", value: "", observedAt: new Date().toISOString().slice(0, 10), staleAfterDays: "30", dimension: "" }); setChangeError(false); setMessage(t(strings.pages.triggers.savedNotice)); },
    onError: () => setChangeError(true),
  });
  const dryRunMutation = useMutation({
    mutationFn: () => evaluateTriggers(true),
    onSuccess: (result) => { setLatestEvaluation(result); setChangeError(false); },
    onError: () => setChangeError(true),
  });

  const traces = useMemo(() => {
    const result = new Map<string, { name: string; age: string }>();
    evaluations.forEach((item) => {
      const names = item.factNames?.length ? item.factNames : item.factName ? [item.factName] : [];
      names.forEach((name) => result.set(name, { name, age: factAge(item.factAgeSeconds) }));
    });
    return [...result.values()];
  }, [evaluations]);
  const unknownCount = evaluations.filter((item) => verdictName(item.verdict) === "UNKNOWN").length;
  const latestTimestamp = evaluations.map((item) => item.evaluatedAt).filter(Boolean).sort((left, right) => Number(right?.seconds ?? 0) - Number(left?.seconds ?? 0))[0];
  const stalled = evaluations.length === 0;
  const factColumns = [
    { id: "name", header: t(strings.pages.triggers.factNameLabel), accessor: (trace: { name: string; age: string }) => trace.name, searchValue: (trace: { name: string; age: string }) => trace.name, className: "break-words" },
    { id: "age", header: t(strings.pages.triggers.factAge, { age: "" }), accessor: (trace: { name: string; age: string }) => t(strings.pages.triggers.factAge, { age: trace.age }), searchValue: (trace: { name: string; age: string }) => trace.age, className: "break-words" },
  ];

  const submitTrigger = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setMessage("");
    const threshold = Number(triggerForm.threshold);
    const secondThreshold = Number(triggerForm.secondThreshold);
    if (!triggerForm.nodeId.trim() || !triggerForm.factName.trim() || !Number.isFinite(threshold) || (triggerForm.secondFactName.trim() && !Number.isFinite(secondThreshold))) { setChangeError(true); return; }
    setChangeError(false);
    declareMutation.mutate();
  };
  const submitFact = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setMessage("");
    const value = Number(factForm.value);
    const staleAfterDays = Number(factForm.staleAfterDays);
    if (!factForm.name.trim() || !Number.isFinite(value) || !Number.isInteger(staleAfterDays) || staleAfterDays < 1 || Number.isNaN(new Date(factForm.observedAt).getTime())) { setChangeError(true); return; }
    setChangeError(false);
    factMutation.mutate();
  };

  return (
    <ExperienceSurface surfaceId="triggers" state={surface.state} statusMessage={surface.reason} data-testid={selectors.pages.triggers} aria-labelledby="triggers-heading" className="flex flex-col gap-4">
      <h2 id="triggers-heading" className="text-2xl font-semibold">{t(strings.pages.triggers.title)}</h2>
      <Card data-testid={selectors.pages.triggerEditor} role="form" aria-label={t(strings.pages.triggers.declareAction)}>
        <CardHeader><CardTitle>{t(strings.pages.triggers.cardTitle)}</CardTitle></CardHeader>
        <CardContent>
          <p className="text-app-muted-foreground">{t(strings.pages.triggers.description)}</p>
          <div data-testid={selectors.pages.triggerParseStatus} role="status" className="mt-4 rounded-md border p-3">{parseError ? t(strings.pages.triggers.parseError) : t(strings.pages.triggers.parseReady)}</div>
          <p data-testid={selectors.pages.triggerParseError} role="alert" className="text-sm text-app-danger">{parseError ? t(strings.pages.triggers.parseErrorDetail) : ""}</p>
          <Button type="button" data-testid={selectors.pages.triggerDryRun} className="mt-3" disabled={dryRunMutation.isPending} onClick={() => dryRunMutation.mutate()}>{t(strings.pages.triggers.dryRunAction)}</Button>
          <p data-testid={selectors.pages.triggerDryRunVerdict} role="status" className="text-sm">{evaluations.length ? evaluations.map((item) => verdictCopy(item.verdict, t)).join(", ") : t(strings.pages.triggers.dryRunVerdict)}</p>
          <ul data-testid={selectors.pages.triggerFactTrace} aria-label={t(strings.pages.triggers.factTrace)} className="text-sm text-app-muted-foreground">{evaluations.length ? evaluations.map((item) => <li key={item.id}>{item.nodeId || "—"} · {(item.factNames?.length ? item.factNames : [item.factName || "—"]).join(", ")} · {verdictCopy(item.verdict, t)} · {item.explanation || ""}</li>) : <li>{t(strings.pages.triggers.noFacts)}</li>}</ul>
          <p data-testid={selectors.pages.triggerMissingFact} role="note" aria-label={t(strings.pages.triggers.missingFact)} className="text-sm text-app-muted-foreground">{unknownCount ? t(strings.pages.triggers.verdictUnknown) : ""}</p>
          <DataTable rows={traces} columns={factColumns} getRowKey={(trace) => trace.name} caption={t(strings.pages.triggers.factRegistry)} searchLabel={t(strings.pages.triggers.factRegistry)} searchPlaceholder={t(strings.pages.triggers.factNameLabel)} emptyMessage={t(strings.pages.triggers.noFacts)} tableTestId={selectors.pages.factRegistry} className="mt-3 text-app-muted-foreground" />
          <p data-testid={selectors.pages.evaluationFreshness} role="status" aria-label={t(strings.pages.triggers.evaluationFreshness)} className="text-sm text-app-muted-foreground">{t(strings.pages.triggers.evaluationFreshness)}{latestTimestamp ? `: ${timestampLabel(latestTimestamp)}` : ""}</p>
          <p data-testid={selectors.pages.evaluationStalledAlert} role="alert" className="text-sm text-app-muted-foreground">{stalled ? t(strings.pages.triggers.stalledAlert) : ""}</p>
          <p data-testid={selectors.pages.triggersEmptyGuidance} role="note" className="mt-3 rounded-md border border-dashed p-3 text-app-muted-foreground">{t(strings.pages.triggers.emptyGuidance)}</p>
        </CardContent>
      </Card>

      <div className="grid gap-4 lg:grid-cols-2">
        <DirtyStateGuard isDirty={Boolean(triggerForm.nodeId || triggerForm.factName || triggerForm.threshold || triggerForm.secondFactName || triggerForm.secondThreshold)} protectUnload title={t(strings.pages.triggers.declareAction)} description={t(strings.pages.triggers.description)}>
          <FormSection title={t(strings.pages.triggers.declareAction)}>
            <form data-testid={selectors.pages.triggerDeclare} className="grid gap-3" onSubmit={submitTrigger}>
              <label className="grid gap-1" htmlFor="trigger-node"><span>{t(strings.pages.triggers.nodeLabel)}</span><Input id="trigger-node" value={triggerForm.nodeId} onChange={(event) => setTriggerForm({ ...triggerForm, nodeId: event.target.value })} /></label>
              <label className="grid gap-1" htmlFor="trigger-fact"><span>{t(strings.pages.triggers.factNameLabel)}</span><Input id="trigger-fact" value={triggerForm.factName} onChange={(event) => setTriggerForm({ ...triggerForm, factName: event.target.value })} /></label>
              <label className="grid gap-1" htmlFor="trigger-operator"><span>{t(strings.pages.triggers.operatorLabel)}</span><Select id="trigger-operator" value={triggerForm.operator} onChange={(event) => setTriggerForm({ ...triggerForm, operator: event.target.value })} options={[{ value: ">=", label: ">=" }, { value: ">", label: ">" }, { value: "<=", label: "<=" }, { value: "<", label: "<" }, { value: "==", label: "==" }]} /></label>
              <label className="grid gap-1" htmlFor="trigger-threshold"><span>{t(strings.pages.triggers.thresholdLabel)}</span><Input id="trigger-threshold" type="number" step="any" value={triggerForm.threshold} onChange={(event) => setTriggerForm({ ...triggerForm, threshold: event.target.value })} /></label>
              <label className="grid gap-1" htmlFor="trigger-composition"><span>{t(strings.pages.triggers.compositionLabel)}</span><Select id="trigger-composition" value={triggerForm.composition} onChange={(event) => setTriggerForm({ ...triggerForm, composition: event.target.value })} options={[{ value: String(TriggerComposition.ALL), label: "ALL" }, { value: String(TriggerComposition.ANY), label: "ANY" }]} /></label>
              <label className="grid gap-1" htmlFor="trigger-second-fact"><span>{t(strings.pages.triggers.factNameLabel)} 2</span><Input id="trigger-second-fact" value={triggerForm.secondFactName} onChange={(event) => setTriggerForm({ ...triggerForm, secondFactName: event.target.value })} /></label>
              <label className="grid gap-1" htmlFor="trigger-second-threshold"><span>{t(strings.pages.triggers.thresholdLabel)} 2</span><Input id="trigger-second-threshold" type="number" step="any" value={triggerForm.secondThreshold} onChange={(event) => setTriggerForm({ ...triggerForm, secondThreshold: event.target.value })} /></label>
              <Button type="submit" data-testid={selectors.pages.triggerDeclareAction} disabled={declareMutation.isPending}>{t(strings.pages.triggers.declareAction)}</Button>
            </form>
          </FormSection>
        </DirtyStateGuard>
        <DirtyStateGuard isDirty={Boolean(factForm.name || factForm.value || factForm.dimension)} protectUnload title={t(strings.pages.triggers.factEntryTitle)} description={t(strings.pages.triggers.description)}>
          <FormSection title={t(strings.pages.triggers.factEntryTitle)}>
            <form data-testid={selectors.pages.triggerAddFact} aria-label={t(strings.pages.triggers.factEntryTitle)} className="grid gap-3" onSubmit={submitFact}>
              <label className="grid gap-1" htmlFor="fact-name"><span>{t(strings.pages.triggers.factNameLabel)}</span><Input id="fact-name" value={factForm.name} onChange={(event) => setFactForm({ ...factForm, name: event.target.value })} /></label>
              <label className="grid gap-1" htmlFor="fact-value"><span>{t(strings.pages.triggers.factValueLabel)}</span><Input id="fact-value" type="number" step="any" value={factForm.value} onChange={(event) => setFactForm({ ...factForm, value: event.target.value })} /></label>
              <label className="grid gap-1" htmlFor="fact-observed-at"><span>{t(strings.pages.triggers.observedAtLabel)}</span><Input id="fact-observed-at" type="date" value={factForm.observedAt} onChange={(event) => setFactForm({ ...factForm, observedAt: event.target.value })} /></label>
              <label className="grid gap-1" htmlFor="fact-stale-after"><span>{t(strings.pages.triggers.staleAfterLabel)}</span><Input id="fact-stale-after" type="number" min="1" step="1" value={factForm.staleAfterDays} onChange={(event) => setFactForm({ ...factForm, staleAfterDays: event.target.value })} /></label>
              <label className="grid gap-1" htmlFor="fact-dimension"><span>{t(strings.pages.triggers.dimensionLabel)}</span><Input id="fact-dimension" value={factForm.dimension} onChange={(event) => setFactForm({ ...factForm, dimension: event.target.value })} /></label>
              <Button type="submit" data-testid={selectors.pages.triggerAddFactAction} disabled={factMutation.isPending}>{t(strings.pages.triggers.addFactAction)}</Button>
            </form>
          </FormSection>
        </DirtyStateGuard>
      </div>
      {changeError && <p role="alert" className="text-sm text-app-danger">{t(strings.pages.triggers.requestError)}</p>}
      {message && <p data-testid={selectors.pages.triggerChangeNotice} role="status" className="text-sm text-app-success">{message}</p>}
    </ExperienceSurface>
  );
}
