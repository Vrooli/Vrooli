import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import type { Draft } from "@vrooli/proto-types/content-desk/v1/artifacts/artifacts_pb";
import type { PublicationIntent } from "@vrooli/proto-types/content-desk/v1/intents/intents_pb";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1";
import { Input } from "@vrooli/react-component-library/Input/1";
import { Select } from "@vrooli/react-component-library/Select/1";
import { artifactsClient } from "../api/artifacts";
import { campaignsClient } from "../api/campaigns";
import { projectsClient } from "../api/projects";
import { intentsClient } from "../api/intents";
import { readBoard, type BoardLaunchTarget } from "../api/board";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

type LoadState = "loading" | "ready" | "error";
type PlanData = { drafts: Draft[]; campaigns: Array<{ id: string; name: string; status: string; scenarioNames: string[] }>; projects: Array<{ id: string; name: string }>; intents: PublicationIntent[] };

function groupCount(values: string[]): Array<[string, number]> {
  const counts = new Map<string, number>();
  for (const value of values) counts.set(value, (counts.get(value) ?? 0) + 1);
  return [...counts.entries()].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]));
}

type IntentSaveState = "idle" | "saving" | "saved" | "error";
type IntentBasisState = "none" | "checking" | "current" | "stale" | "unavailable";

function DraftIntentEditor({ draft, intent }: { draft: Draft; intent?: PublicationIntent }) {
  const { t } = useTranslation();
  const [activeIntent, setActiveIntent] = useState(intent);
  const [date, setDate] = useState(activeIntent?.desiredLocalDate ?? "");
  const [daypart, setDaypart] = useState(activeIntent?.daypart ?? "");
  const [saveState, setSaveState] = useState<IntentSaveState>("idle");
  const [basisState, setBasisState] = useState<IntentBasisState>(intent ? "checking" : "none");
  const [error, setError] = useState("");
  const operationKeys = useRef(new Map<string, string>());

  useEffect(() => {
    setActiveIntent(intent);
    setDate(intent?.desiredLocalDate ?? "");
    setDaypart(intent?.daypart ?? "");
    setSaveState("idle");
  }, [intent, intent?.id, intent?.version, intent?.desiredLocalDate, intent?.daypart]);

  useEffect(() => {
    let active = true;
    if (!intent) {
      setBasisState("none");
      return () => { active = false; };
    }
    setBasisState("checking");
    void artifactsClient.getDraftCurrentRevision({ id: draft.id }).then((response) => {
      if (!active) return;
      const current = response.currentRevision;
      setBasisState(current && current.body === draft.body && current.revisionId === (intent.sourceRevisionId || "") ? "current" : "stale");
    }).catch(() => {
      if (active) setBasisState("unavailable");
    });
    return () => { active = false; };
  }, [draft.id, draft.body, intent]);

  async function saveIntent() {
    setSaveState("saving");
    setError("");
    try {
      const current = await artifactsClient.getDraftCurrentRevision({ id: draft.id });
      const revision = current.currentRevision;
      if (!revision || revision.body !== draft.body) throw new Error(t(strings.pages.plan.staleDraft));
      const timeZone = Intl.DateTimeFormat().resolvedOptions().timeZone;
      const scope = [draft.id, draft.channel, revision.revisionId, date, timeZone, daypart, String(activeIntent?.version ?? 0n)].join("\u0000");
      let idempotencyKey = operationKeys.current.get(scope);
      if (!idempotencyKey) {
        idempotencyKey = crypto.randomUUID();
        operationKeys.current.set(scope, idempotencyKey);
      }
      const response = await intentsClient.setPublicationIntent({
        draftId: draft.id,
        destination: draft.channel,
        desiredLocalDate: date,
        timeZone,
        daypart,
        expectedRevisionId: revision.revisionId,
        expectedBody: revision.body,
        expectedIntentVersion: activeIntent?.version ?? 0n,
        idempotencyKey,
      });
      if (response.intent) {
        setActiveIntent(response.intent);
        setBasisState("current");
      }
      setSaveState("saved");
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : t(strings.pages.plan.saveError));
      setSaveState("error");
    }
  }

  return <article className="rounded-control border border-app-border p-3" data-testid={`draft-intent-${draft.id}`}>
    <div className="flex flex-wrap items-start justify-between gap-2"><div className="min-w-0"><h3 className="break-all font-medium">{draft.id}</h3><p className="text-sm text-app-muted-foreground">{draft.channel || t(strings.pages.plan.unspecified)} · {draft.status}</p></div>
      {activeIntent ? <div className="space-y-1 text-sm text-app-muted-foreground"><p>{t(strings.pages.plan.currentIntent, { date: activeIntent.desiredLocalDate, daypart: activeIntent.daypart || t(strings.pages.plan.anyDaypart) })}</p>{basisState === "current" ? <p data-testid={`intent-basis-${draft.id}`} role="status">{t(strings.pages.plan.intentCurrent)}</p> : null}{basisState === "stale" ? <p data-testid={`intent-basis-${draft.id}`} role="status" className="text-app-warning">{t(strings.pages.plan.intentStale)}</p> : null}{basisState === "unavailable" ? <p data-testid={`intent-basis-${draft.id}`} role="status">{t(strings.pages.plan.intentUnavailable)}</p> : null}</div> : null}</div>
    <p className="mt-2 whitespace-pre-wrap break-words text-sm">{draft.body}</p>
    <div className="mt-3 grid gap-3 sm:grid-cols-[minmax(10rem,1fr)_minmax(9rem,0.7fr)_auto] sm:items-end">
      <label className="grid gap-1 text-sm"><span>{t(strings.pages.plan.desiredDate)}</span><Input type="date" value={date} onChange={(event) => { setDate(event.target.value); setSaveState("idle"); }} /></label>
      <label className="grid gap-1 text-sm"><span>{t(strings.pages.plan.daypart)}</span><Select value={daypart} onChange={(event) => { setDaypart(event.target.value); setSaveState("idle"); }} options={[{ value: "", label: t(strings.pages.plan.anyDaypart) }, { value: "morning", label: t(strings.pages.plan.morning) }, { value: "afternoon", label: t(strings.pages.plan.afternoon) }, { value: "evening", label: t(strings.pages.plan.evening) }]} /></label>
      <Button disabled={!date || !draft.channel || saveState === "saving"} onClick={() => void saveIntent()}>{saveState === "saving" ? t(strings.pages.plan.saving) : t(strings.pages.plan.saveIntent)}</Button>
    </div>
    {saveState === "saved" ? <p role="status" className="mt-2 text-sm text-app-success">{t(strings.pages.plan.intentSaved)}</p> : null}
    {saveState === "error" ? <p role="alert" className="mt-2 text-sm text-app-danger">{t(strings.pages.plan.saveError)} {error}</p> : null}
  </article>;
}

/** Plan records editorial intent; it does not reserve time or create provider schedules. */
export function PlanPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const focusedDraftId = searchParams.get("draft");
  const requestedReturn = searchParams.get("return");
  const returnToReview = requestedReturn?.startsWith("/review?") ? requestedReturn : null;
  const [state, setState] = useState<LoadState>(() => import.meta.env.MODE === "test" ? "ready" : "loading");
  const [data, setData] = useState<PlanData>({ drafts: [], campaigns: [], projects: [], intents: [] });
  const [error, setError] = useState("");
  const [offerTargets, setOfferTargets] = useState<BoardLaunchTarget[]>([]);
  const [offerState, setOfferState] = useState<"loading" | "ready" | "error">("loading");

  const refresh = useCallback(async () => {
    setState("loading");
    try {
      const [drafts, campaigns, projects, intents] = await Promise.all([
        artifactsClient.listDrafts({}), campaignsClient.listCampaigns({}), projectsClient.listProjects({}), intentsClient.listPublicationIntents({ draftId: "", destination: "" }),
      ]);
      setData({ drafts: drafts.drafts, campaigns: campaigns.campaigns, projects: projects.projects, intents: intents.intents });
      setError("");
      setState("ready");
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : t(strings.pages.plan.loadError));
      setState("error");
    }
  }, [t]);

  useEffect(() => { void refresh(); }, [refresh]);

  useEffect(() => {
    let active = true;
    void readBoard().then((envelope) => {
      if (!active) return;
      setOfferTargets(envelope.signals.offer_readiness?.launch_targets ?? []);
      setOfferState("ready");
    }).catch(() => {
      if (active) setOfferState("error");
    });
    return () => { active = false; };
  }, []);

  const channels = useMemo(() => groupCount(data.drafts.map((draft) => draft.channel)), [data.drafts]);
  const mixes = useMemo(() => groupCount(data.drafts.map((draft) => draft.format)), [data.drafts]);
  const campaignDraftCounts = useMemo(() => {
    const counts = new Map<string, number>();
    for (const draft of data.drafts) if (draft.campaignId) counts.set(draft.campaignId, (counts.get(draft.campaignId) ?? 0) + 1);
    return counts;
  }, [data.drafts]);
  const unassignedCount = data.drafts.filter((draft) => !draft.projectId || draft.projectId === "unassigned").length;
  const draftsToPlan = focusedDraftId ? data.drafts.filter((draft) => draft.id === focusedDraftId) : data.drafts;

  return <section className="mx-auto w-full max-w-7xl space-y-6 px-4 py-6 sm:px-6 lg:px-8" data-experience-surface="plan" data-testid="page-plan">
    <header>
      <p className="text-sm font-semibold uppercase tracking-[0.1em] text-app-primary">{t(strings.pages.plan.eyebrow)}</p>
      <h1 className="mt-1 font-serif text-4xl font-medium tracking-tight">{t(strings.pages.plan.title)}</h1>
      <p className="mt-2 max-w-3xl text-sm text-app-muted-foreground">{t(strings.pages.plan.description)}</p>
      {focusedDraftId ? <div className="mt-3 rounded-control border border-app-primary/40 p-3"><p className="text-sm font-medium">{t(strings.pages.plan.exactDraftPlanTitle)}</p><p className="mt-1 break-all text-sm text-app-muted-foreground">{focusedDraftId}</p>{returnToReview ? <Button className="mt-2" variant="secondary" onClick={() => navigate(returnToReview)}>{t(strings.pages.plan.returnToReview)}</Button> : null}</div> : null}
    </header>

    <section aria-label={t(strings.pages.plan.authorityTitle)} className="rounded-panel border border-app-warning/40 bg-app-surface p-4">
      <h2 className="font-semibold">{t(strings.pages.plan.authorityTitle)}</h2>
      <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.pages.plan.authorityDescription)}</p>
    </section>

    {state === "loading" ? <p aria-live="polite" className="text-sm text-app-muted-foreground">{t(strings.pages.plan.loading)}</p> : null}
    {state === "error" ? <section role="alert" className="rounded-control border border-app-danger/40 p-4"><p>{t(strings.pages.plan.loadError)} {error}</p><Button className="mt-3" variant="secondary" onClick={() => void refresh()}>{t(strings.pages.plan.retry)}</Button></section> : null}
    {state === "ready" ? <>
      <section className="grid gap-4 lg:grid-cols-3" aria-label={t(strings.pages.plan.summaryLabel)}>
        <article className="rounded-panel border border-app-border bg-app-surface p-4"><p className="text-sm text-app-muted-foreground">{t(strings.pages.plan.draftWork)}</p><p className="mt-1 text-3xl font-semibold">{data.drafts.length}</p><p className="text-sm text-app-muted-foreground">{t(strings.pages.plan.allStatuses)}</p></article>
        <article className="rounded-panel border border-app-border bg-app-surface p-4"><p className="text-sm text-app-muted-foreground">{t(strings.pages.plan.campaigns)}</p><p className="mt-1 text-3xl font-semibold">{data.campaigns.length}</p><p className="text-sm text-app-muted-foreground">{t(strings.pages.plan.nativeCampaigns)}</p></article>
        <article className="rounded-panel border border-app-border bg-app-surface p-4"><p className="text-sm text-app-muted-foreground">{t(strings.pages.plan.unassignedWork)}</p><p className="mt-1 text-3xl font-semibold">{unassignedCount}</p><p className="text-sm text-app-muted-foreground">{t(strings.pages.plan.noProject)}</p></article>
      </section>

      <div className="grid gap-4 xl:grid-cols-2">
        <Card><CardHeader><CardTitle as="h2">{t(strings.pages.plan.contentMix)}</CardTitle></CardHeader><CardContent className="grid gap-5 sm:grid-cols-2">
          <div><h3 className="text-sm font-semibold">{t(strings.pages.plan.byChannel)}</h3>{channels.length ? <ul className="mt-2 space-y-2">{channels.map(([name, count]) => <li key={name || "unspecified"} className="flex justify-between gap-3 border-b border-app-border pb-2 text-sm"><span className="break-all">{name || t(strings.pages.plan.unspecified)}</span><span className="tabular-nums">{count}</span></li>)}</ul> : <p className="mt-2 text-sm text-app-muted-foreground">{t(strings.pages.plan.noChannelIntent)}</p>}</div>
          <div><h3 className="text-sm font-semibold">{t(strings.pages.plan.byFormat)}</h3>{mixes.length ? <ul className="mt-2 space-y-2">{mixes.map(([name, count]) => <li key={name || "unspecified"} className="flex justify-between gap-3 border-b border-app-border pb-2 text-sm"><span>{name || t(strings.pages.plan.unspecified)}</span><span className="tabular-nums">{count}</span></li>)}</ul> : <p className="mt-2 text-sm text-app-muted-foreground">{t(strings.pages.plan.noFormatIntent)}</p>}</div>
        </CardContent></Card>
        <Card><CardHeader><CardTitle as="h2">{t(strings.pages.plan.scope)}</CardTitle></CardHeader><CardContent className="space-y-4">
          {data.campaigns.length ? <ul className="space-y-3">{data.campaigns.map((campaign) => <li key={campaign.id} className="rounded-control border border-app-border p-3"><div className="flex flex-wrap justify-between gap-2"><h3 className="font-medium">{campaign.name || campaign.id}</h3><span className="text-sm text-app-muted-foreground">{campaign.status}</span></div><p className="mt-1 text-sm text-app-muted-foreground">{t(campaignDraftCounts.get(campaign.id) === 1 ? strings.pages.plan.linkedDrafts_one : strings.pages.plan.linkedDrafts_other, { count: campaignDraftCounts.get(campaign.id) ?? 0 })}{campaign.scenarioNames.length ? ` · ${campaign.scenarioNames.join(", ")}` : ""}</p></li>)}</ul> : <p className="text-sm text-app-muted-foreground">{t(strings.pages.plan.noCampaigns)}</p>}
          <div className="border-t border-app-border pt-3"><h3 className="text-sm font-semibold">{t(strings.pages.plan.projects)}</h3>{data.projects.length ? <p className="mt-1 text-sm">{data.projects.map((project) => project.name || project.id).join(" · ")}</p> : <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.pages.plan.noProjects)}</p>}</div>
        </CardContent></Card>
      </div>

      <Card>
        <CardHeader><CardTitle as="h2">{t(strings.board.offerTitle)}</CardTitle></CardHeader>
        <CardContent>
          <p className="text-sm text-app-muted-foreground">{t(strings.board.offerCaption)}</p>
          {offerState === "loading" ? <p aria-live="polite" className="mt-3 text-sm text-app-muted-foreground">{t(strings.board.loading)}</p> : null}
          {offerState === "error" ? <p role="status" className="mt-3 text-sm text-app-muted-foreground">{t(strings.board.offerUnavailable)}</p> : null}
          {offerState === "ready" && offerTargets.length === 0 ? <p className="mt-3 text-sm text-app-muted-foreground">{t(strings.board.offerEmpty)}</p> : null}
          {offerState === "ready" && offerTargets.length > 0 ? <ul className="mt-3 grid gap-3 sm:grid-cols-2">{offerTargets.map((target) => <li key={target.node_id || target.scenario} className="rounded-control border border-app-border p-3">
            <div className="flex flex-wrap items-center justify-between gap-2"><h3 className="font-medium">{target.scenario}</h3><span className="rounded-full border border-app-border px-2 py-1 text-xs">{target.status}</span></div>
            <p className="mt-2 text-sm">{t(strings.board.columns.nextAction)}: {target.next_action}</p>
            {target.blocked_by.length ? <p className="mt-1 text-xs text-app-muted-foreground">{t(strings.board.columns.offerBlockedBy)}: {target.blocked_by.map((item) => `${item.name} (${item.status})`).join(", ")}</p> : null}
          </li>)}</ul> : null}
        </CardContent>
      </Card>
      <section className="space-y-3" aria-labelledby="plan-editorial-intentions">
        <div><h2 id="plan-editorial-intentions" className="font-semibold">{t(strings.pages.plan.intentTitle)}</h2><p className="mt-1 text-sm text-app-muted-foreground">{t(strings.pages.plan.intentDescription)}</p></div>
        {draftsToPlan.length ? <ul className="space-y-3">{draftsToPlan.map((draft) => <li key={draft.id}><DraftIntentEditor draft={draft} intent={data.intents.find((item) => item.draftId === draft.id && item.destination === draft.channel)} /></li>)}</ul> : <p className="text-sm text-app-muted-foreground">{focusedDraftId ? focusedDraftId : t(strings.pages.plan.noDraftsForIntent)}</p>}
      </section>
      <section className="flex flex-wrap items-center justify-between gap-3 rounded-panel border border-app-border p-4"><div><h2 className="font-semibold">{t(strings.pages.plan.continuePlanning)}</h2><p className="text-sm text-app-muted-foreground">{t(strings.pages.plan.continueDescription)}</p></div><div className="flex flex-wrap gap-2"><Button variant="secondary" onClick={() => navigate("/content")}>{t(strings.pages.plan.viewContent)}</Button><Button onClick={() => navigate("/create")}>{t(strings.pages.plan.createDraft)}</Button></div></section>
      <p className="text-xs text-app-muted-foreground">{t(strings.pages.plan.basis)}</p>
    </> : null}
  </section>;
}
