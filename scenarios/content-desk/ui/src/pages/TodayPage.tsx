import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1";
import type { Draft } from "@vrooli/proto-types/content-desk/v1/artifacts/artifacts_pb";

import { artifactsClient } from "../api/artifacts";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

type LoadState = "loading" | "ready" | "error";
const ACTION_STATUSES = new Set(["requested", "drafting", "drafted", "checking", "blocked", "reviewed", "approved"]);

function draftDescription(draft: Draft, t: ReturnType<typeof useTranslation>["t"]): string {
  const scope = draft.campaignId ? t(strings.today.campaign, { id: draft.campaignId }) : t(strings.today.noCampaign);
  const channel = draft.channel ? ` · ${draft.channel}` : "";
  return `${draft.status} · ${scope}${channel}`;
}

export function TodayPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [state, setState] = useState<LoadState>(() => import.meta.env.MODE === "test" ? "ready" : "loading");
  const [drafts, setDrafts] = useState<Draft[]>([]);
  const [error, setError] = useState("");

  const refresh = useCallback(async () => {
    setState("loading");
    try {
      const response = await artifactsClient.listDrafts({});
      setDrafts(response.drafts);
      setError("");
      setState("ready");
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Content Desk could not load saved drafts.");
      setState("error");
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const activeDrafts = useMemo(() => drafts.filter((draft) => ACTION_STATUSES.has(draft.status.toLowerCase())), [drafts]);
  const suggestedDraft = activeDrafts[0];
  const reviewDraft = activeDrafts.find((draft) => draft.status.toLowerCase() === "reviewed");
  const reviewCount = activeDrafts.filter((draft) => draft.status.toLowerCase() === "reviewed").length;
  const pendingDrafts = activeDrafts.filter((draft) => !["reviewed", "approved"].includes(draft.status.toLowerCase()));
  const releaseReadyDrafts = state === "ready" ? drafts.filter((draft) => draft.status.toLowerCase() === "approved") : [];

  function openDraft(draft: Draft) {
    navigate(`/?draft=${encodeURIComponent(draft.id)}`);
  }

  function openForReview(draft: Draft) {
    navigate(`/review?draft=${encodeURIComponent(draft.id)}`);
  }

  return (
    <section className="mx-auto w-full max-w-7xl space-y-6 px-4 py-6 sm:px-6 lg:px-8" data-experience-surface="today" data-testid="page-today">
      <header className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <p className="text-sm font-semibold uppercase tracking-[0.1em] text-app-primary">{t(strings.today.eyebrow)}</p>
          <h1 className="mt-1 font-serif text-4xl font-medium tracking-tight">{t(strings.today.title)}</h1>
          <p className="mt-2 max-w-2xl text-sm text-app-muted-foreground">{t(strings.today.subtitle)}</p>
        </div>
        <div className="flex items-center gap-2">
          <span className="rounded-full border border-app-border bg-app-surface px-3 py-1 text-sm">{t(strings.today.openItems, { count: state === "ready" ? activeDrafts.length : 0 })}</span>
          <Button variant="secondary" size="sm" disabled={state === "loading"} onClick={() => void refresh()}>{t(strings.today.refresh)}</Button>
        </div>
      </header>

      {state === "loading" ? (
        <section aria-live="polite" className="rounded-panel border border-app-border bg-app-surface p-5 text-sm text-app-muted-foreground">{t(strings.today.loading)}</section>
      ) : state === "error" ? (
        <section role="alert" className="rounded-panel border border-app-danger/40 bg-app-surface p-5">
          <h2 className="font-semibold">{t(strings.today.unavailableTitle)}</h2>
          <p className="mt-1 text-sm text-app-muted-foreground">{error}</p>
          <Button className="mt-4" onClick={() => void refresh()}>{t(strings.today.tryAgain)}</Button>
        </section>
      ) : activeDrafts.length === 0 ? (
        <section className="grid gap-4 lg:grid-cols-[1.4fr_1fr]">
          <Card>
            <CardHeader><CardTitle as="h2">{t(strings.today.quietTitle)}</CardTitle></CardHeader>
            <CardContent className="space-y-3">
              <p className="text-sm text-app-muted-foreground">{t(strings.today.quietDescription)}</p>
              <p className="text-sm text-app-muted-foreground">{t(strings.today.viewWorkDescription)}</p>
              <Button variant="secondary" onClick={() => navigate("/content")}>{t(strings.today.viewWork)}</Button>
            </CardContent>
          </Card>
        </section>
      ) : (
        <section className="grid gap-4 lg:grid-cols-[0.9fr_1.4fr_0.9fr]">
          <Card>
            <CardHeader><CardTitle as="h2">{t(strings.today.resumeTitle)}</CardTitle></CardHeader>
            <CardContent>
              {suggestedDraft ? <>
                <p className="font-serif text-xl leading-snug">{suggestedDraft.id}</p>
                <p className="mt-2 text-sm text-app-muted-foreground">{draftDescription(suggestedDraft, t)}</p>
                <Button className="mt-4 w-full" onClick={() => openDraft(suggestedDraft)}>{t(strings.today.resumeButton)}</Button>
              </> : <p className="text-sm text-app-muted-foreground">{t(strings.today.noDraft)}</p>}
              <div className="mt-5 border-l-2 border-app-primary pl-3">
                <h3 className="font-semibold">{t(strings.today.whyNext)}</h3>
                <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.today.whyText)}</p>
              </div>
            </CardContent>
          </Card>

          <Card className="bg-app-background">
            <CardHeader>
              <p className="text-xs font-semibold uppercase tracking-[0.1em] text-app-primary">{t(strings.today.independentDecision)}</p>
              <CardTitle as="h2">{reviewDraft ? t(strings.today.reviewTitle) : t(strings.today.noApprovalTitle)}</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-app-muted-foreground">{reviewDraft ? t(strings.today.reviewDescription) : t(strings.today.noApprovalDescription)}</p>
              <p className="mt-3 text-sm">{reviewCount === 1 ? t(strings.today.reviewCount_one, { count: reviewCount }) : t(strings.today.reviewCount_other, { count: reviewCount })}</p>
              <Button variant="secondary" className="mt-4" onClick={() => reviewDraft ? openForReview(reviewDraft) : suggestedDraft ? openDraft(suggestedDraft) : navigate("/create")}>{reviewDraft ? t(strings.today.reviewWorkbench) : t(strings.today.openWorkbench)}</Button>
            </CardContent>
          </Card>

          <Card>
            <CardHeader><CardTitle as="h2">{t(strings.today.waiting)}</CardTitle></CardHeader>
            <CardContent className="space-y-3">
              {pendingDrafts.slice(0, 3).map((draft) => <button key={draft.id} type="button" onClick={() => openDraft(draft)} className="block w-full rounded-control border border-app-border p-3 text-left hover:bg-app-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary">
                <span className="block font-medium">{draft.id}</span><span className="text-sm text-app-muted-foreground">{draftDescription(draft, t)}</span>
              </button>)}
              {pendingDrafts.length === 0 ? <p className="text-sm text-app-muted-foreground">{t(strings.today.noWaiting)}</p> : null}
              <p className="text-xs text-app-muted-foreground">{t(strings.today.localStateNote)}</p>
            </CardContent>
          </Card>
        </section>
      )}
      {state === "ready" && releaseReadyDrafts.length > 0 ? <Card>
        <CardHeader><CardTitle as="h2">{t(strings.today.releaseReadyTitle)}</CardTitle></CardHeader>
        <CardContent className="space-y-3">
          <p className="text-sm text-app-muted-foreground">{t(strings.today.releaseReadyDescription)}</p>
          {releaseReadyDrafts.slice(0, 3).map((draft) => <div key={draft.id} className="flex flex-wrap items-center justify-between gap-3 rounded-control border border-app-border p-3">
            <span className="font-medium">{draft.id} · {draftDescription(draft, t)}</span>
            <Button variant="secondary" onClick={() => openDraft(draft)}>{t(strings.today.openWorkbench)}</Button>
          </div>)}
        </CardContent>
      </Card> : null}
    </section>
  );
}
