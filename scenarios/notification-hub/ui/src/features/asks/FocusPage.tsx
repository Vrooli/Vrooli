import { useEffect, useRef, useState } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";
import { ListAsksRequest_Order, type Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";
import { CheckCircle2 } from "lucide-react";

import { conversationsClient } from "../../api/notifications";
import { EmptyNotice, QueryError } from "../../components/QueryNotice";
import { SkeletonRows } from "../../components/Skeleton";
import { Time } from "../../components/Time";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useNow } from "../../hooks/useNow";
import { useTranslation } from "../../i18n";
import { formatDueIn } from "../../lib/ago";
import { useClaimNavSection } from "../../layout/navSection";
import { preserveRoutedTestMode } from "../routedTestMode";
import { SessionUnavailable } from "../session/SessionGate";
import { SignedOutCard } from "../session/SignedOutCard";
import { useSession } from "../session/useSession";
import { askHeadline, DECISION_KIND_LABEL, optionLabel, senderLabel, urgencyLabel } from "./askView";
import { Chip } from "../packet/Chip";
import { DecisionText } from "../packet/DecisionText";
import { ExplainerFrame } from "../packet/explainer/ExplainerFrame";
import { explainerOf } from "../packet/viewer/attachments";
import { AnswerPanel } from "../packet/form/AnswerPanel";
import { blockerFor, withPrefill, withRecommendations } from "../packet/model/answer";
import { Recommendation } from "../packet/Recommendation";
import { PacketEvidence } from "../packet/PacketEvidence";
import { useAnswerFlow } from "../packet/useAnswerFlow";
import { usePacketDraft } from "../packet/usePacketDraft";

const FOCUS_PAGE_SIZE = 20;

/**
 * One decision at a time, in attention order, carrying the whole decision: the
 * requester's HTML page (its charts, forms and prose), the recommendation, the
 * answer form and the evidence. It reads the queue once; answering advances
 * through that review rather than refetching under the reader.
 */
export function FocusPage() {
  const { t } = useTranslation();
  const { search } = useLocation();
  const session = useSession();
  useClaimNavSection("asks");
  const signedIn = session.isSuccess && session.data.signedIn;
  const subject = signedIn ? session.data.subject : "";
  const [position, setPosition] = useState(0);
  const queue = useQuery({
    // Deliberately outside the `asks` cache root: answering invalidates that
    // root, and a review must not reflow under the reader mid-decision.
    queryKey: ["focus-review", subject],
    enabled: signedIn,
    staleTime: 0,
    refetchOnWindowFocus: false,
    retry: false,
    queryFn: async ({ signal }) => conversationsClient.listAsks({
      limit: FOCUS_PAGE_SIZE,
      stateFilter: "awaiting",
      order: ListAsksRequest_Order.ATTENTION,
    }, { signal }),
  });

  const pageTitle = <h1 className="sr-only">{t(strings.asks.focus.title)}</h1>;
  if (session.isPending) return <>{pageTitle}<SkeletonRows rows={3} /></>;
  if (session.isError) return <>{pageTitle}<SessionUnavailable retry={() => { void session.refetch(); }} /></>;
  if (!signedIn) return <>{pageTitle}<SignedOutCard /></>;
  if (queue.isPending) return <>{pageTitle}<SkeletonRows rows={3} /></>;
  if (queue.isError && !queue.data) {
    return <>{pageTitle}<QueryError error={queue.error} title={t(strings.asks.list.loadError)} onRetry={() => { void queue.refetch(); }} retrying={queue.isFetching} /></>;
  }
  const asks = queue.data.asks;
  const current = asks[position];
  if (!current) {
    return (
      <section aria-labelledby="focus-done-heading" data-testid={selectors.pages.focus} className="mx-auto flex w-full max-w-3xl flex-col gap-4">
        <h1 id="focus-done-heading" className="sr-only">{t(strings.asks.focus.title)}</h1>
        <EmptyNotice
          icon={<CheckCircle2 size={32} />}
          title={t(strings.asks.focus.doneTitle)}
          detail={queue.data.totalAwaiting > asks.length
            ? t(strings.asks.list.waiting, { count: queue.data.totalAwaiting - asks.length })
            : t(strings.asks.focus.doneDetail)}
        >
          <Link to={preserveRoutedTestMode("/asks", search)} className="inline-flex min-h-11 items-center text-sm font-medium text-app-primary hover:underline">
            {t(strings.asks.focus.later)}
          </Link>
        </EmptyNotice>
      </section>
    );
  }
  return (
    <FocusCard
      key={current.id}
      ask={current}
      subject={subject}
      position={position}
      total={Math.max(asks.length, queue.data.totalAwaiting)}
      onNext={() => setPosition((value) => value + 1)}
    />
  );
}

function FocusCard({ ask, subject, position, total, onNext }: { ask: Ask; subject: string; position: number; total: number; onNext: () => void }) {
  const { t } = useTranslation();
  const { search } = useLocation();
  const navigate = useNavigate();
  const now = useNow();
  const headingRef = useRef<HTMLHeadingElement>(null);
  useEffect(() => { headingRef.current?.focus({ preventScroll: true }); }, [ask.id]);
  // The list row is enough to start reading, but the page, attachments, report
  // and related decisions come from the full detail read.
  const detail = useQuery({
    queryKey: ["ask", ask.id, subject],
    queryFn: async () => (await conversationsClient.getAsk({ askId: ask.id })).ask,
    staleTime: 0,
    refetchOnWindowFocus: false,
    retry: false,
  });
  const full = detail.data ?? ask;
  const [draft, updateDraft] = usePacketDraft(subject, ask.id);
  const flow = useAnswerFlow({ ask: full, subject, attemptSubject: subject, draft });
  const [prefilled, setPrefilled] = useState(false);
  const fullPath = preserveRoutedTestMode(`/asks/${encodeURIComponent(ask.id)}`, search);
  const openFull = () => navigate(fullPath);
  const links = { attachments: full.attachments, onOpenAttachment: openFull, onOpenLinkedArtifact: openFull };
  const explainer = explainerOf(full.attachments);
  const hasExplainer = Boolean(explainer || full.templateName);
  const recommended = full.recommended && full.options.some((option) => option.key === full.recommended) ? full.recommended : "";
  const recommendedDraft = withRecommendations(full, draft);
  const canAccept = Boolean(recommended) && blockerFor(full, recommendedDraft) === null;
  const canSend = blockerFor(full, draft) === null;
  const open = full.state === "pending" || full.state === "escalated";
  const canAnswer = open || full.state === "defaulted";
  const due = formatDueIn(full.deadline, now);
  const urgency = urgencyLabel(full.urgency);
  const finished = flow.outcome === "recorded" || flow.outcome === "dismissed";
  const kindLabel = full.decisionKind ? DECISION_KIND_LABEL[full.decisionKind as keyof typeof DECISION_KIND_LABEL] : undefined;
  const frozen = flow.settling || flow.busy;

  return (
    <section aria-labelledby="focus-heading" data-testid={selectors.pages.focus} className="mx-auto flex w-full max-w-6xl flex-col gap-5">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
        <span aria-hidden="true" className="flex gap-1">
          {Array.from({ length: Math.min(total, 8) }, (_, index) => (
            <i
              key={index}
              className={`h-[3px] w-[18px] rounded-full ${index === position ? "bg-app-primary" : index < position ? "bg-app-primary/50" : "bg-app-surface-muted ring-1 ring-inset ring-app-border"}`}
            />
          ))}
        </span>
        <span role="status" className="font-mono text-xs text-app-muted-foreground">{t(strings.asks.focus.progress, { current: position + 1, total })}</span>
        <Link to={preserveRoutedTestMode("/asks", search)} className="ms-auto inline-flex min-h-11 items-center gap-1 text-sm font-medium text-app-muted-foreground hover:text-app-foreground">
          {t(strings.asks.focus.later)}
        </Link>
      </div>

      <div className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_18rem] lg:items-start">
        <article className="min-w-0 rounded-panel border border-app-border bg-app-surface p-4 md:p-6">
          <div className="flex flex-wrap items-center gap-1.5">
            {urgency && <Chip tone={full.urgency === "critical" ? "danger" : full.urgency === "high" ? "attention" : "neutral"}>{t(urgency)}</Chip>}
            {due && <Chip>{t(full.closingSoon ? strings.asks.row.closesIn : strings.asks.row.dueIn, { when: due })}</Chip>}
            {kindLabel && <Chip>{t(kindLabel)}</Chip>}
            {hasExplainer && <Chip>{t(strings.asks.focus.pageAttached)}</Chip>}
          </div>
          <p className="mt-3 font-mono text-[10px] uppercase tracking-[0.14em] text-app-muted-foreground">
            {senderLabel(full)}
          </p>
          <h1 ref={headingRef} tabIndex={-1} id="focus-heading" className="mt-1.5 break-words text-xl font-semibold leading-tight tracking-tight outline-none focus:outline-none focus-visible:outline-none md:text-2xl">{askHeadline(full)}</h1>
          {full.bigPicture && (
            <DecisionText
              value={full.bigPicture}
              links={links}
              className="mt-2.5 block max-w-[62ch] text-sm leading-relaxed text-app-muted-foreground md:text-[13.5px]"
            />
          )}

          {detail.isPending && <div className="mt-4"><SkeletonRows rows={3} /></div>}
          {detail.isError && (
            <p role="alert" className="mt-4 flex flex-wrap items-center gap-2 text-sm text-app-muted-foreground">
              {t(strings.asks.focus.contextFailed)}
              <button type="button" onClick={() => void detail.refetch()} className="inline-flex min-h-11 items-center text-sm font-medium text-app-primary hover:underline">
                {t(strings.common.retry)}
              </button>
            </p>
          )}
          {hasExplainer && !detail.isPending && (
            <div className="mt-4">
              <ExplainerFrame
                askId={full.id}
                subject={subject}
                kind={full.kind}
                recurrence={full.recurrence}
                templateName={full.templateName}
                templateData={full.templateData}
                answerChanges={draft.answerChanges}
                explainer={explainer}
                attachments={full.attachments}
                onOpen={openFull}
                onOpenAsk={openFull}
                onPrefill={(questionId, value) => {
                  updateDraft((current) => withPrefill(full, current, questionId, value));
                  setPrefilled(true);
                }}
                onAnswerChange={(change) => updateDraft((current) => {
                  const { [change.key]: _removed, ...remaining } = current.answerChanges;
                  const answerChanges = change.operation === "remove"
                    ? remaining
                    : { ...remaining, [change.key]: { label: change.label, value: change.value } };
                  return { ...current, answerChanges };
                })}
              />
              {prefilled && <p role="status" className="mt-2 text-sm text-app-muted-foreground">{t(strings.packet.explainer.prefilled)}</p>}
            </div>
          )}

          <div className="mt-5">
            <Recommendation ask={full} links={links} />
          </div>

          {canAnswer ? (
            <div className="mt-5">
              <AnswerPanel
                ask={full}
                draft={draft}
                onDraftChange={updateDraft}
                locked={frozen}
                frozen={flow.settling}
                paged={false}
                hideActions
                open={open}
                dismissing={flow.dismissing}
                onSubmit={flow.submit}
                onDismiss={flow.dismiss}
                onRequestContext={openFull}
                onOpenAttachment={openFull}
              />
            </div>
          ) : (
            <Link to={fullPath} className="mt-5 inline-flex min-h-11 items-center text-sm font-medium text-app-primary hover:underline">
              {t(strings.asks.focus.openPacket)}
            </Link>
          )}
        </article>

        <aside aria-label={t(strings.asks.focus.context)} className="flex min-w-0 flex-col gap-4 lg:sticky lg:top-4">
          <section className="rounded-panel border border-app-border bg-app-surface p-4">
            <h2 className="font-mono text-[9.5px] font-semibold uppercase tracking-[0.16em] text-app-muted-foreground">{t(strings.asks.focus.context)}</h2>
            <dl className="mt-3 grid gap-2 text-xs">
              <div className="flex items-baseline justify-between gap-3">
                <dt className="text-app-muted-foreground">{t(strings.asks.focus.source)}</dt>
                <dd className="min-w-0 break-words text-end font-medium">{senderLabel(full) || t(strings.asks.focus.unknown)}</dd>
              </div>
              <div className="flex items-baseline justify-between gap-3">
                <dt className="text-app-muted-foreground">{t(strings.asks.focus.age)}</dt>
                <dd className="text-end font-medium"><Time value={full.createdAt} /></dd>
              </div>
              <div className="flex items-baseline justify-between gap-3">
                <dt className="text-app-muted-foreground">{t(strings.asks.focus.closes)}</dt>
                <dd className="text-end font-medium">{due ?? t(strings.asks.focus.unknown)}</dd>
              </div>
              {full.costOfWaiting && (
                <div className="flex flex-col gap-0.5">
                  <dt className="text-app-muted-foreground">{t(strings.asks.focus.ifIgnored)}</dt>
                  <dd className="min-w-0 break-words font-medium">{full.costOfWaiting}</dd>
                </div>
              )}
            </dl>
          </section>
          <PacketEvidence ask={full} />
          {hasExplainer && (
            <Link to={fullPath} className="inline-flex min-h-11 items-center text-sm font-medium text-app-primary hover:underline">
              {t(strings.asks.focus.openPacket)}
            </Link>
          )}
        </aside>
      </div>

      {flow.sending && <p role="status" className="text-sm text-app-muted-foreground">{t(strings.packet.outcome.sending)}</p>}
      {flow.checking && <p role="status" className="text-sm text-app-muted-foreground">{t(strings.packet.outcome.checking)}</p>}
      {flow.outcome === "recorded" && (
        <p role="status" className="flex items-center gap-2 text-sm font-medium text-app-success">
          <CheckCircle2 size={18} aria-hidden="true" />
          {t(strings.packet.outcome.recorded, { option: flow.recordedLabel })}
        </p>
      )}
      {flow.outcome === "dismissed" && <p role="status" className="text-sm text-app-muted-foreground">{t(strings.packet.status.dismissed)}</p>}
      {flow.outcome === "uncertain" && <p role="alert" className="rounded-panel border border-app-warning/60 bg-app-warning/15 p-3 text-sm">{t(strings.packet.outcome.uncertain)}</p>}
      {flow.outcome === "changed" && <p role="alert" className="rounded-panel border border-app-warning/60 bg-app-warning/15 p-3 text-sm">{t(strings.packet.outcome.changed)}</p>}
      {flow.outcome === "open" && <p role="status" className="text-sm text-app-muted-foreground">{t(strings.packet.outcome.stillOpen)}</p>}
      {flow.outcome === "defaulted" && <p role="status" className="text-sm text-app-muted-foreground">{t(strings.packet.outcome.defaultPending)}</p>}
      {flow.outcome === "conflict" && <p role="alert" className="text-sm">{t(strings.packet.outcome.conflict, { option: flow.recordedLabel, attempted: optionLabel(full, draft.answer) })}</p>}
      {flow.readError && <p role="alert" className="text-sm text-app-danger">{t(strings.packet.outcome.readbackFailed)}</p>}
      {(flow.readError || flow.outcome === "open" || flow.outcome === "changed") && (
        <div className="flex flex-wrap items-center gap-3">
          <Button type="button" variant="outline" onClick={flow.checkStatus} disabled={flow.checking}>
            {t(strings.packet.outcome.checkStatus)}
          </Button>
          <Link to={fullPath} className="inline-flex min-h-11 items-center text-sm font-medium text-app-primary hover:underline">
            {t(strings.asks.focus.openPacket)}
          </Link>
        </div>
      )}

      <div className="sticky bottom-0 -mx-1 flex flex-wrap items-center gap-2 border-t border-app-border bg-app-background px-1 py-3">
        {finished ? (
          <Button type="button" variant="primary" onClick={onNext}>{t(strings.asks.focus.next)}</Button>
        ) : canAnswer ? (
          <>
            {recommended && (
              <Button type="button" variant="primary" disabled={!canAccept || frozen} onClick={() => { updateDraft(recommendedDraft); flow.submitDraft(recommendedDraft); }}>
                {t(strings.asks.focus.accept)}
              </Button>
            )}
            <Button type="button" variant={recommended ? "outline" : "primary"} disabled={!canSend || frozen} onClick={() => flow.submit()}>
              {t(strings.asks.focus.send)}
            </Button>
            <Link to={fullPath} className="ms-auto inline-flex min-h-11 items-center text-sm font-medium text-app-muted-foreground hover:text-app-foreground">
              {t(strings.asks.focus.openPacket)}
            </Link>
          </>
        ) : (
          <Link to={fullPath} className="inline-flex min-h-11 items-center rounded-control bg-app-primary px-4 text-sm font-semibold text-app-primary-foreground">
            {t(strings.asks.focus.openPacket)}
          </Link>
        )}
      </div>
    </section>
  );
}
