import { useEffect, useRef, useState, type ReactNode, type RefObject } from "react";
import { Link, useLocation, useNavigate, useSearchParams } from "react-router-dom";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { Code, ConnectError } from "@connectrpc/connect";
import { Button } from "@vrooli/react-component-library/Button/2";
import { IconButton } from "@vrooli/react-component-library/IconButton/3";
import type { Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";
import { CheckCircle2, Clock, FileQuestion, Play, Search, SearchX, X } from "lucide-react";

import { conversationsClient } from "../../api/notifications";
import { EmptyNotice, QueryError } from "../../components/QueryNotice";
import { SkeletonRows } from "../../components/Skeleton";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useNow } from "../../hooks/useNow";
import { useTranslation } from "../../i18n";
import { isUnauthenticated } from "../../lib/apiError";
import { formatDueIn, formatExact } from "../../lib/ago";
import { useClaimNavSection } from "../../layout/navSection";
import { Chip } from "../packet/Chip";
import { DecisionText } from "../packet/DecisionText";
import { MainOptions } from "../packet/form/MainOptions";
import { ExplainerFrame } from "../packet/explainer/ExplainerFrame";
import { explainerOf } from "../packet/viewer/attachments";
import { blockerFor, missingRequired, withPrefill, withRecommendations } from "../packet/model/answer";
import { FollowthroughBanner, PacketEvidence } from "../packet/PacketEvidence";
import { READINESS_LABEL, readinessGrade } from "../packet/readiness";
import { Recommendation } from "../packet/Recommendation";
import { OutcomeNotices } from "../packet/status/OutcomeNotices";
import { PacketStatus } from "../packet/status/PacketStatus";
import { useAnswerFlow } from "../packet/useAnswerFlow";
import { usePacketDraft } from "../packet/usePacketDraft";
import { preserveRoutedTestMode } from "../routedTestMode";
import { SessionUnavailable } from "../session/SessionGate";
import { SignedOutCard } from "../session/SignedOutCard";
import { useSession } from "../session/useSession";
import {
  closingHours as hoursOf,
  filterCount,
  groupByAttentionAndFamily,
  listViewRequest,
  readListView,
  TIER_LABEL,
  writeListView,
  type ListView,
} from "./askListView";
import { askHeadline, askTone, isAwaiting, senderLabel } from "./askView";
import { askStateKey, STATE_LABEL } from "./askState";
import { ListControls } from "./ListControls";
import { NativeConsents } from "./NativeConsents";
import { useNeedsYou } from "./useNeedsYou";
import { useNativeConsents } from "./useNativeConsents";

const PAGE_SIZE = 50;
const HEADING_ID = "desk-heading";
const LIST_HEADING_ID = "desk-list-heading";

/**
 * The desktop decision room: the queue keeps its own column and the decision
 * being read sits beside it, as one field. Choosing a row selects it in place
 * rather than navigating, so the operator can move through the backlog without
 * losing their place. Below the desk width the queue keeps its single-column
 * composition; a direct `/asks/:id` link still opens the full packet.
 */
export function DeskPage() {
  const { t } = useTranslation();
  const location = useLocation();
  const [params, setParams] = useSearchParams();
  const session = useSession();
  useClaimNavSection("asks");
  const signedIn = session.isSuccess && session.data.signedIn;
  const subject = signedIn ? session.data.subject : "";
  const search = params.get("q") ?? "";
  const view = readListView(params, true);
  const [query, setQuery] = useState(search.trim());
  const [searchOpen, setSearchOpen] = useState(false);
  const searchInput = useRef<HTMLInputElement>(null);
  const [needsSearchFocus, setNeedsSearchFocus] = useState(false);
  const needsYou = useNeedsYou();

  useEffect(() => {
    if (search.trim() === query) return;
    const timer = setTimeout(() => setQuery(search.trim()), 200);
    return () => clearTimeout(timer);
  }, [search, query]);
  useEffect(() => {
    if (needsSearchFocus) {
      searchInput.current?.focus();
      setNeedsSearchFocus(false);
    }
  }, [needsSearchFocus]);

  const searching = query !== search.trim();
  const list = useInfiniteQuery({
    queryKey: ["asks", subject, "open", query, "awaiting", view],
    enabled: signedIn,
    initialPageParam: "",
    queryFn: async ({ pageParam, signal }) => conversationsClient.listAsks({
      limit: PAGE_SIZE, cursor: pageParam, query, stateFilter: "awaiting",
      ...listViewRequest(view, true),
    }, { signal }),
    placeholderData: (previousData, previousQuery) => previousQuery?.queryKey[1] === subject ? previousData : undefined,
    getNextPageParam: (page) => page.nextCursor || undefined,
    retry: false,
    staleTime: 30_000,
    refetchInterval: 60_000,
  });
  const answered = list.isPlaceholderData ? undefined : list.data;
  const facets = list.data?.pages[0]?.facets;
  const closingHours = hoursOf(list.data?.pages[0]?.closingWindowSeconds);
  const rows = [...new Map(answered?.pages.flatMap((page) => page.asks).map((ask) => [ask.id, ask]) ?? []).values()];
  const settled = list.isSuccess && !list.isPlaceholderData && !searching;
  const pending = list.isPending || list.isPlaceholderData || searching;
  const initialError = list.isError && !list.data;
  const refreshError = list.isError && Boolean(list.data) && !list.isFetchNextPageError;
  const consents = useNativeConsents(query, signedIn, subject);
  const consentsFailed = signedIn && consents.isError && !isUnauthenticated(consents.error);
  const pendingConsents = consents.data?.pages.flatMap((page) => page.consents).filter((consent) => consent.state === "pending").length ?? 0;
  const totalAwaiting = answered?.pages[0]?.totalAwaiting ?? 0;
  const matching = (facets?.urgencies ?? [])
    .filter((facet) => view.urgencies.length === 0 || view.urgencies.some((urgency) => urgency === facet.value))
    .reduce((sum, facet) => sum + facet.count, 0);
  const hasFilters = Boolean(search || filterCount(view) > 0);
  const groups = groupByAttentionAndFamily(rows, view.sort === "attention");
  const selectedId = params.get("ask") || rows[0]?.id || "";
  const selected = useSelectedAsk(selectedId, subject, signedIn);

  const changeSearch = (value: string) => setParams((current) => {
    const next = new URLSearchParams(current);
    if (value) next.set("q", value); else next.delete("q");
    return next;
  }, { replace: true });
  const changeView = (next: ListView) => setParams((current) => writeListView(current, next, true), { replace: true });
  const select = (id: string) => setParams((current) => {
    const next = new URLSearchParams(current);
    next.set("ask", id);
    return next;
  }, { replace: true });
  const clear = () => {
    setParams((current) => {
      const next = writeListView(current, { ...view, urgencies: [], senders: [] }, true);
      next.delete("q");
      return next;
    }, { replace: true });
    searchInput.current?.focus();
  };

  if (session.isPending) {
    return <DeskFrame><div className="p-6"><SkeletonRows /></div></DeskFrame>;
  }
  if (session.isError) {
    return <DeskFrame><div className="p-6"><SessionUnavailable retry={() => { void session.refetch(); }} /></div></DeskFrame>;
  }
  if (!signedIn) {
    return <DeskFrame><div className="mx-auto w-full max-w-md p-6"><SignedOutCard /></div></DeskFrame>;
  }

  const waiting = Math.max(rows.length, totalAwaiting, needsYou.total);
  const dueToday = rows.filter((ask) => ask.closingSoon).length;
  const summary = settled
    ? hasFilters
      ? t(strings.asks.list.matching, { count: matching, total: totalAwaiting })
      : waiting === 0
        ? t(strings.asks.list.allClear)
        : [
          t(strings.asks.desk.needYou, { count: waiting }),
          dueToday > 0 ? t(strings.asks.desk.dueToday, { count: dueToday }) : null,
          pendingConsents > 0 ? t(strings.asks.desk.permissions, { count: pendingConsents }) : null,
        ].filter(Boolean).join(" · ")
    : t(strings.asks.desk.needYou, { count: waiting });

  const reviewSearch = new URLSearchParams(location.search);
  reviewSearch.delete("ask");

  return (
    <DeskFrame>
      <header className="flex flex-none flex-wrap items-center gap-3 border-b border-app-border px-6 py-3.5">
        <div className="min-w-0 flex-1">
          <h1 id={HEADING_ID} className="text-[19px] font-semibold leading-tight tracking-[-0.02em]">{t(strings.asks.list.title)}</h1>
          <p className="mt-0.5 text-xs text-app-muted-foreground">{summary}</p>
        </div>
        <div className="flex items-center gap-1.5">
          <IconButton
            aria-label={t(searchOpen || search ? strings.asks.list.searchClose : strings.asks.list.searchOpen)}
            aria-pressed={searchOpen || search !== ""}
            surface="ghost"
            shape="rounded"
            size="lg"
            onClick={() => {
              if (searchOpen || search) { setSearchOpen(false); changeSearch(""); } else { setSearchOpen(true); setNeedsSearchFocus(true); }
            }}
          >
            <Search size={18} aria-hidden="true" />
          </IconButton>
          {rows.length > 0 && (
            <Link
              to={preserveRoutedTestMode(`/focus${reviewSearch.size ? `?${reviewSearch}` : ""}`, location.search)}
              className="inline-flex min-h-11 items-center gap-1.5 rounded-control bg-app-primary px-3.5 text-sm font-semibold text-app-primary-foreground hover:opacity-90"
            >
              <Play size={14} aria-hidden="true" />
              {t(strings.asks.focus.start)}
            </Link>
          )}
        </div>
      </header>

      {(searchOpen || search !== "") && (
        <div className="flex-none border-y border-app-border px-6 py-2">
          <div className="flex min-h-10 items-center gap-2.5 rounded-control border border-app-border bg-app-surface px-2.5">
            <Search size={16} className="shrink-0 text-app-muted-foreground" aria-hidden="true" />
            <input
              ref={searchInput}
              id="desk-search"
              type="search"
              maxLength={500}
              aria-label={t(strings.pages.history.searchLabel)}
              value={search}
              onChange={(event) => changeSearch(event.currentTarget.value)}
              placeholder={t(strings.pages.history.searchHint)}
              className="min-h-10 min-w-0 flex-1 bg-transparent text-sm text-app-foreground outline-none placeholder:text-app-muted-foreground"
            />
            {search && (
              <button type="button" onClick={() => changeSearch("")} className="inline-flex min-h-9 items-center gap-1 rounded-control px-1 text-xs text-app-muted-foreground hover:text-app-foreground">
                <X size={13} aria-hidden="true" />{t(strings.pages.history.clearFilters)}
              </button>
            )}
          </div>
        </div>
      )}

      <div className="grid min-h-0 flex-1 grid-cols-[minmax(17rem,22rem)_minmax(0,1fr)]">
        <section
          data-testid={selectors.asks.desk.list}
          aria-labelledby={LIST_HEADING_ID}
          className="flex min-h-0 flex-col overflow-y-auto border-e border-app-border px-2.5 pb-8 pt-3"
        >
          <h2 id={LIST_HEADING_ID} className="sr-only">{t(strings.asks.desk.listLabel)}</h2>
          <div className="px-1.5 pb-2">
            <ListControls
              view={view}
              facets={facets}
              matching={matching}
              closingHours={closingHours}
              onChange={changeView}
              onClear={clear}
              compact
            />
          </div>
          <NativeConsents list={consents} />
          {(initialError || refreshError || consentsFailed) && (
            <div className="px-1.5 pb-2">
              <QueryError
                error={initialError || refreshError ? list.error : consents.error}
                title={t(initialError || refreshError
                  ? consentsFailed ? strings.asks.list.loadErrorBoth : strings.asks.list.loadError
                  : strings.nativeConsent.unavailable)}
                detail={initialError || refreshError ? (refreshError ? t(strings.asks.list.stale) : t(strings.asks.list.notCleared)) : undefined}
                onRetry={() => {
                  if (initialError || refreshError) void list.refetch();
                  if (consentsFailed) void consents.refetch();
                }}
                retrying={list.isFetching || consents.isFetching}
                retryTestId={selectors.asks.loadRetry}
              />
            </div>
          )}
          {pending && <div className="px-1.5"><SkeletonRows /></div>}
          {settled && rows.length === 0 && (
            <div className="px-1.5 pt-2">
              <EmptyNotice
                icon={hasFilters ? <SearchX size={28} /> : <CheckCircle2 size={28} />}
                title={t(hasFilters ? strings.pages.history.noMatches : strings.pages.history.noAwaiting)}
                detail={t(hasFilters ? strings.pages.history.trySearch : strings.pages.history.awaitingEmptyDetail)}
              >
                {hasFilters && <Button type="button" variant="outline" onClick={clear}>{t(strings.pages.history.clearFilters)}</Button>}
              </EmptyNotice>
            </div>
          )}
          {rows.length > 0 && groups.map(({ tier, families }) => {
            const tierAsks = families.flatMap((family) => family.asks);
            const grouped = view.sort === "attention";
            return (
              <section key={tier} aria-labelledby={grouped ? `desk-group-${tier}` : undefined} className="pb-2">
                {grouped && (
                  <div className="flex items-baseline gap-2 px-3 pb-1.5 pt-3">
                    <b id={`desk-group-${tier}`} className="font-mono text-[9.5px] font-semibold uppercase tracking-[0.15em] text-app-muted-foreground">
                      {t(TIER_LABEL[tier], { count: Math.max(1, closingHours) })}
                    </b>
                    <span className="font-mono text-[10px] font-semibold text-app-muted-foreground/80">{tierAsks.length}</span>
                    <i aria-hidden="true" className="h-px flex-1 bg-app-border" />
                  </div>
                )}
                <ul className="flex flex-col">
                  {tierAsks.map((ask) => (
                    <DeskRow key={ask.id} ask={ask} selected={ask.id === selectedId} onSelect={() => select(ask.id)} />
                  ))}
                </ul>
              </section>
            );
          })}
          {list.isFetchNextPageError && <p role="alert" className="px-3 py-2 text-sm">{t(strings.pages.history.moreError)}</p>}
          {list.hasNextPage && (
            <div className="px-1.5 pt-2">
              <Button type="button" variant="outline" className="w-full" disabled={list.isFetching || searching} onClick={() => void list.fetchNextPage()}>
                {t(list.isFetchingNextPage ? strings.asks.desk.loadingMore : strings.asks.desk.more)}
              </Button>
            </div>
          )}
        </section>

        <section data-testid={selectors.asks.desk.detail} aria-label={t(strings.asks.desk.detailLabel)} className="min-h-0 overflow-hidden">
          <DeskDetail
            ask={selected.data}
            askId={selectedId}
            pending={selected.isPending}
            error={selected.error}
            retrying={selected.isFetching}
            onRetry={() => void selected.refetch()}
            rows={rows}
            subject={subject}
            search={location.search}
            onSelect={select}
            onNext={() => {
              const index = rows.findIndex((ask) => ask.id === selectedId);
              const next = rows[(index + 1) % Math.max(rows.length, 1)];
              if (next) select(next.id);
            }}
          />
        </section>
      </div>
    </DeskFrame>
  );
}

/** The window-height field the split fills; the shell hands it the pane in `fill` mode. */
function DeskFrame({ children }: { children: ReactNode }) {
  return <div data-testid={selectors.asks.desk.page} className="flex min-h-0 flex-1 flex-col overflow-hidden bg-app-background">{children}</div>;
}

function useSelectedAsk(askId: string, subject: string, signedIn: boolean) {
  return useQuery({
    queryKey: ["ask", askId, subject],
    enabled: signedIn && askId !== "",
    queryFn: async () => (await conversationsClient.getAsk({ askId })).ask,
    retry: (count, error) => !(error instanceof ConnectError && (error.code === Code.NotFound || error.code === Code.Unauthenticated)) && count < 2,
    staleTime: 0,
  });
}

/** One compact queue row: title, and the source · cause · close line under it. */
function DeskRow({ ask, selected, onSelect }: { ask: Ask; selected: boolean; onSelect: () => void }) {
  const { t } = useTranslation();
  const now = useNow();
  const tone = askTone(ask, now);
  const due = isAwaiting(ask) ? formatDueIn(ask.deadline, now) : null;
  const meta = [
    ask.decisionCause.trim(),
    senderLabel(ask),
    due ? t(ask.closingSoon ? strings.asks.row.closesIn : strings.asks.row.dueIn, { when: due }) : null,
  ].filter(Boolean).join(" · ");
  return (
    <li>
      <button
        type="button"
        data-testid={selectors.asks.desk.row}
        aria-current={selected ? "true" : undefined}
        onClick={onSelect}
        className={`flex min-h-[3.25rem] w-full items-start gap-2.5 rounded-control px-3 py-2 text-start transition-colors ${
          selected ? "bg-app-surface-muted text-app-foreground" : "text-app-muted-foreground hover:bg-app-surface hover:text-app-foreground"
        }`}
      >
        <span
          aria-hidden="true"
          className={`mt-[0.45rem] h-[5px] w-[5px] shrink-0 rounded-full ${
            tone === "critical" ? "bg-app-danger" : tone === "attention" ? "bg-app-warning" : "bg-app-border"
          }`}
        />
        <span className="flex min-w-0 flex-1 flex-col gap-0.5">
          <span className="line-clamp-2 break-words text-[13.5px] font-medium leading-snug">{askHeadline(ask)}</span>
          {meta && <span className="truncate text-[11px] leading-snug text-app-muted-foreground">{meta}</span>}
        </span>
      </button>
    </li>
  );
}

function DeskDetail({ ask, askId, pending, error, retrying, onRetry, rows, subject, search, onSelect, onNext }: {
  ask: Ask | null | undefined;
  askId: string;
  pending: boolean;
  error: unknown;
  retrying: boolean;
  onRetry: () => void;
  rows: readonly Ask[];
  subject: string;
  search: string;
  onSelect: (id: string) => void;
  onNext: () => void;
}) {
  const { t } = useTranslation();
  const scrollRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    scrollRef.current?.scrollTo({ top: 0 });
  }, [askId]);

  if (askId === "") {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-2 p-8 text-center">
        <CheckCircle2 size={24} aria-hidden="true" className="text-app-muted-foreground" />
        <h2 className="text-base font-semibold">{t(strings.asks.desk.choose)}</h2>
        <p className="max-w-sm text-sm text-app-muted-foreground">{t(strings.asks.desk.chooseDetail)}</p>
      </div>
    );
  }
  if (pending) {
    return <div className="p-6"><SkeletonRows rows={4} /></div>;
  }
  if (error || !ask) {
    const gone = error instanceof ConnectError && error.code === Code.NotFound;
    return (
      <div className="p-6">
        {gone ? (
          <EmptyNotice icon={<FileQuestion size={28} />} title={t(strings.asks.desk.detailGone)} />
        ) : (
          <QueryError error={error} title={t(strings.asks.desk.detailError)} onRetry={onRetry} retrying={retrying} retryTestId={selectors.packet.loadRetry} />
        )}
      </div>
    );
  }
  return (
    <DeskDecision
      key={ask.id}
      ask={ask}
      rows={rows}
      subject={subject}
      search={search}
      scrollRef={scrollRef}
      onSelect={onSelect}
      onNext={onNext}
    />
  );
}

function DeskDecision({ ask, rows, subject, search, scrollRef, onSelect, onNext }: {
  ask: Ask;
  rows: readonly Ask[];
  subject: string;
  search: string;
  scrollRef: RefObject<HTMLDivElement>;
  onSelect: (id: string) => void;
  onNext: () => void;
}) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const now = useNow();
  const [draft, updateDraft] = usePacketDraft(subject, ask.id);
  const flow = useAnswerFlow({ ask, subject, attemptSubject: subject, draft });
  const [prefilled, setPrefilled] = useState(false);
  const fullPath = preserveRoutedTestMode(`/asks/${encodeURIComponent(ask.id)}`, search);
  const openFull = () => navigate(fullPath);
  // Every file or linked artifact opens in the full packet; the desk is a
  // reading surface, not a second copy of the viewer.
  const links = { attachments: ask.attachments, onOpenAttachment: openFull, onOpenLinkedArtifact: openFull };
  const awaiting = isAwaiting(ask);
  const frozen = flow.settling || flow.busy;
  const recommended = ask.recommended && ask.options.some((option) => option.key === ask.recommended) ? ask.recommended : "";
  const recommendedDraft = withRecommendations(ask, draft);
  const canAccept = Boolean(recommended) && blockerFor(ask, recommendedDraft) === null;
  const blocker = blockerFor(ask, draft);
  const hint =
    blocker === "no-answer" ? t(strings.packet.form.blockedNoAnswer)
    : blocker === "other-needs-note" ? t(strings.packet.form.blockedOther)
    : blocker === "required-question" ? t(strings.packet.form.blockedRequired, { prompt: missingRequired(ask, draft)[0]?.prompt ?? "" })
    : undefined;
  const readiness = readinessGrade(ask.qualityGrade);
  const due = awaiting ? formatDueIn(ask.deadline, now) : null;
  const explainer = explainerOf(ask.attachments);
  const hasExplainer = Boolean(explainer || ask.templateName);
  const family = ask.decisionFamily.trim();
  const peers = family ? rows.filter((row) => row.decisionFamily === family && row.id !== ask.id) : [];
  const fileCount = ask.attachments.length;
  const multi = ask.questions.length > 0;
  const kicker = [
    ask.decisionCause.trim(),
    senderLabel(ask),
    ask.deadline ? `${t(strings.packet.cover.due)} ${formatDueIn(ask.deadline, now) ?? ""}` : "",
  ].filter(Boolean).join(" · ");

  const accept = () => {
    updateDraft(recommendedDraft);
    flow.submitDraft(recommendedDraft);
  };

  return (
    <article aria-labelledby="desk-detail-heading" className="flex h-full min-h-0 flex-col">
      <div ref={scrollRef} className="min-h-0 flex-1 overflow-y-auto px-6 pb-6 pt-5 xl:px-8">
        <div className="flex flex-wrap gap-1.5">
          {ask.closingSoon && due && <Chip tone="attention"><span aria-hidden="true" className="h-1.5 w-1.5 rounded-full bg-app-warning" />{t(strings.asks.row.closesIn, { when: due })}</Chip>}
          {readiness && readiness !== "ready" && <Chip tone={readiness === "refused" ? "danger" : "attention"}>{t(READINESS_LABEL[readiness])}</Chip>}
          {family && peers.length > 0 && <Chip>{t(strings.asks.desk.family, { count: peers.length + 1 })}</Chip>}
          {!awaiting && <Chip>{t(STATE_LABEL[askStateKey(ask)])}</Chip>}
        </div>

        {kicker && (
          <p className="mt-4 font-mono text-[10px] uppercase tracking-[0.14em] text-app-muted-foreground">{kicker}</p>
        )}
        <h2
          id="desk-detail-heading"
          data-testid={selectors.asks.desk.title}
          className="mt-2 max-w-[34ch] break-words text-[26px] font-semibold leading-[1.12] tracking-[-0.034em]"
        >
          <DecisionText value={askHeadline(ask)} inheritText dense links={links} />
        </h2>
        {ask.title && ask.title !== ask.question && (
          <p className="mt-2 max-w-[52ch] break-words text-sm leading-relaxed text-app-muted-foreground">
            <DecisionText value={ask.question} inheritText dense links={links} />
          </p>
        )}
        {ask.bigPicture && (
          <DecisionText
            value={ask.bigPicture}
            links={links}
            className="mt-2.5 block max-w-[52ch] text-[13.5px] leading-relaxed text-app-muted-foreground"
          />
        )}
        <FollowthroughBanner ask={ask} />

        {hasExplainer && (
          <div className="mt-4">
            <ExplainerFrame
              askId={ask.id}
              subject={subject}
              kind={ask.kind}
              recurrence={ask.recurrence}
              templateName={ask.templateName}
              templateData={ask.templateData}
              answerChanges={draft.answerChanges}
              explainer={explainer}
              attachments={ask.attachments}
              onOpen={openFull}
              onOpenAsk={openFull}
              onPrefill={(questionId, value) => {
                updateDraft((current) => withPrefill(ask, current, questionId, value));
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

        {peers.length > 0 && (
          <section aria-labelledby="desk-family-heading" className="mt-4 overflow-hidden rounded-panel border border-app-border bg-app-surface">
            <div className="flex items-baseline justify-between gap-3 border-b border-app-border px-4 py-2.5">
              <b id="desk-family-heading" className="text-[11.5px] font-semibold text-app-foreground">
                {ask.decisionCause.trim() || t(strings.asks.desk.family, { count: peers.length + 1 })}
              </b>
              <span className="font-mono text-[9.5px] font-semibold uppercase tracking-[0.1em] text-app-muted-foreground">
                {t(strings.asks.desk.familyHint)}
              </span>
            </div>
            {[ask, ...peers].map((peer, index) => (
              <FamilyRow key={peer.id} peer={peer} current={index === 0} onSelect={() => onSelect(peer.id)} />
            ))}
          </section>
        )}

        <Recommendation ask={ask} links={links} />
        {awaiting && ask.costOfWaiting && (
          <p className="mt-2 max-w-[52ch] text-xs leading-relaxed text-app-muted-foreground">
            <b className="font-medium text-app-foreground">{t(strings.asks.desk.ifIgnored)}</b> {ask.costOfWaiting}
          </p>
        )}

        {awaiting ? (
          <div className="mt-5">
            {multi ? (
              <div className="flex flex-col items-start gap-2 rounded-panel border border-app-border bg-app-surface p-4">
                <p className="text-sm text-app-muted-foreground">{t(strings.asks.desk.questions, { count: ask.questions.length })}</p>
                <Link to={fullPath} className="inline-flex min-h-11 items-center text-sm font-semibold text-app-primary hover:underline">
                  {t(strings.asks.desk.answerQuestions, { count: ask.questions.length })}
                </Link>
              </div>
            ) : (
              <MainOptions ask={ask} draft={draft} disabled={frozen} onChange={updateDraft} />
            )}
          </div>
        ) : (
          <div className="mt-5"><PacketStatus ask={ask} now={now} links={links} /></div>
        )}

        <div className="mt-5">
          <PacketEvidence ask={ask} />
        </div>

        <div className="mt-4 flex flex-wrap items-center justify-between gap-3 border-t border-app-border pt-3 text-xs text-app-muted-foreground">
          <span className="inline-flex items-center gap-1.5">
            <Clock size={13} aria-hidden="true" />
            {ask.deadline ? <time dateTime={ask.deadline}>{formatExact(ask.deadline) ?? t(strings.common.timeUnavailable)}</time> : t(strings.common.timeUnavailable)}
          </span>
          <Link to={fullPath} className="inline-flex min-h-11 items-center gap-1.5 font-medium text-app-muted-foreground hover:text-app-foreground">
            {t(fileCount === 1 ? strings.asks.desk.files_one : strings.asks.desk.files, { count: fileCount })}
          </Link>
        </div>

      </div>

      {(flow.outcome || flow.sending || flow.readError) && (
        <div className="flex-none px-6 pt-2">
          <OutcomeNotices
            ask={ask}
            outcome={flow.outcome}
            recordedLabel={flow.recordedLabel}
            attemptedAnswer={flow.attempt?.answer ?? ""}
            sending={flow.sending}
            checking={flow.sent && flow.checking}
            readError={flow.readError}
            returnTo={preserveRoutedTestMode("/asks", search)}
          />
        </div>
      )}

      <div className="flex flex-none flex-wrap items-center gap-2 border-t border-app-border bg-app-background/85 px-6 py-3 backdrop-blur">
        {awaiting && !multi ? (
          <>
            {recommended && (
              <Button type="button" variant="primary" size="lg" disabled={!canAccept || frozen} onClick={accept}>
                {t(strings.asks.focus.accept)}
              </Button>
            )}
            <Button type="button" variant={recommended ? "outline" : "primary"} size="lg" disabled={blocker !== null || frozen} onClick={() => flow.submit()}>
              {t(strings.asks.focus.send)}
            </Button>
            {hint && (!recommended || blocker !== "no-answer") && <span role="status" className="min-w-0 text-xs leading-snug text-app-muted-foreground">{hint}</span>}
            <IconButton
              aria-label={t(strings.asks.focus.later)}
              title={t(strings.asks.focus.later)}
              surface="ghost"
              shape="rounded"
              size="lg"
              onClick={onNext}
            >
              <Clock size={16} aria-hidden="true" />
            </IconButton>
          </>
        ) : awaiting && multi ? (
          <Link
            to={fullPath}
            className="inline-flex min-h-11 items-center rounded-control bg-app-primary px-4 text-sm font-semibold text-app-primary-foreground"
          >
            {t(strings.asks.desk.answerQuestions, { count: ask.questions.length })}
          </Link>
        ) : (
          <Link to={fullPath} className="inline-flex min-h-11 items-center rounded-control bg-app-primary px-4 text-sm font-semibold text-app-primary-foreground">
            {t(strings.asks.desk.openFull)}
          </Link>
        )}
        {awaiting && !multi && (
          <Link to={fullPath} className="ms-auto inline-flex min-h-11 items-center text-xs font-medium text-app-muted-foreground hover:text-app-foreground">
            {t(strings.asks.desk.openFull)}
          </Link>
        )}
      </div>
    </article>
  );
}


/** One row of the "one cause" block: the current decision reads, its siblings select. */
function FamilyRow({ peer, current, onSelect }: { peer: Ask; current: boolean; onSelect: () => void }) {
  const body = (
    <>
      <span aria-hidden="true" className={`mt-1 h-3.5 w-3.5 shrink-0 rounded-full border-2 border-app-primary ${current ? "bg-app-primary" : ""}`} />
      <span className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className="break-words text-[12.5px] font-medium leading-snug text-app-foreground">{askHeadline(peer)}</span>
        <span className="line-clamp-1 break-words text-[11px] text-app-muted-foreground">{peer.question}</span>
      </span>
    </>
  );
  const className = "flex w-full items-start gap-3 border-b border-app-border px-4 py-2.5 text-start last:border-b-0";
  return current ? (
    <div aria-current="true" className={className}>{body}</div>
  ) : (
    <button type="button" onClick={onSelect} className={`${className} hover:bg-app-surface-muted`}>{body}</button>
  );
}
