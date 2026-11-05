import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import type { TFunction } from "i18next";
import { Check, X, Search, AlertTriangle, ShieldCheck } from "lucide-react";
import { SwipeActions } from "@vrooli/react-component-library/SwipeActions";
import { FullPageDrawer } from "@vrooli/react-component-library/FullPageDrawer/1";
import { AlertDialog } from "@vrooli/react-component-library/AlertDialog/2";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { ErrorState } from "@vrooli/react-component-library/ErrorState/1";

import { Button } from "../ui/button";
import { cn } from "../../lib/classnames";
import { strings } from "../../consts/strings";
import { useMediaQuery } from "../../hooks/useMediaQuery";
import { useApprovals } from "../../hooks/useApprovals";
import { useApprovalsStore } from "../../stores/useApprovalsStore";
import {
  answerApproval,
  getApproval,
  ApprovalAlreadyResolvedError,
  ApprovalsUnauthorizedError,
  ApprovalsUnavailableError,
  type ApprovalDecision,
} from "../../api/approvals";
import { useAuthStore } from "../../stores/authStore";
import ApprovalContextPanel from "./ApprovalContextPanel";
import {
  approveOptionKey,
  isActionable,
  recommendedOption,
  relativeTime,
  secondaryOptions,
  sortDecisions,
  sourceAccent,
  sourceLabel,
} from "./approvalsFormat";

type FilterMode = "pending" | "all" | "resolved" | "expired";

interface ApprovalsDrawerProps {
  open: boolean;
  onClose: () => void;
}

/**
 * The approvals inbox.
 *
 * Container is RCL FullPageDrawer (like SettingsModal / ArchiveDrawer); the
 * structure — search + filters in headerExtra, a list that drills into a detail
 * — is templated on ArchiveDrawer. It composes EXISTING RCL primitives only
 * (FullPageDrawer, SwipeActions, AlertDialog, EmptyState, ErrorState) plus the
 * app's Button; no new shared RCL primitive is introduced (the answer-options
 * block is intentionally web-console-local — the ratified decision).
 *
 * Desktop = master-detail (list left, detail right). Mobile = card stack that
 * drills into a full-screen detail; approve/reject via SwipeActions on
 * REVERSIBLE cards only — an irreversible decision has no swipe and must be
 * opened to confirm (the ratified safety rule).
 */
export default function ApprovalsDrawer({ open, onClose }: ApprovalsDrawerProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const isMobile = useMediaQuery("(max-width: 767px)");

  const [filter, setFilter] = useState<FilterMode>("pending");
  const [query, setQuery] = useState("");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [swipeOpenId, setSwipeOpenId] = useState<string | null>(null);
  const [note, setNote] = useState("");
  const [confirming, setConfirming] = useState<{ decision: ApprovalDecision; optionKey: string } | null>(null);
  const [answering, setAnswering] = useState(false);
  const [answerError, setAnswerError] = useState<string | null>(null);
  // Typed terminal states that replace the generic error banner with a tailored
  // action: session expired (sign in), or already resolved by someone else.
  const [sessionExpired, setSessionExpired] = useState(false);
  const [alreadyResolved, setAlreadyResolved] = useState(false);
  // A deep-linked decision that is NOT in the current list (resolved / outside
  // the filter): fetched directly and held here so the detail actually renders.
  const [deepLinkDecision, setDeepLinkDecision] = useState<ApprovalDecision | null>(null);
  // Optimistic: ids answered this session, hidden from the actionable list
  // immediately so a row never briefly shows as still-actionable before refetch.
  const [answeredIds, setAnsweredIds] = useState<ReadonlySet<string>>(() => new Set());
  const signIn = useAuthStore((s) => s.signIn);

  // The pending queue AND the resolved view come from one hook; openOnly=false
  // when a history filter is active so resolved/expired rows are available.
  const wantsHistory = filter === "resolved" || filter === "all" || filter === "expired";
  const { data, isLoading, error, refetch } = useApprovals({ openOnly: !wantsHistory, enabled: open });

  // A deep link asks the drawer to open and land on one decision.
  const pendingDeepLinkId = useApprovalsStore((s) => s.pendingDeepLinkId);
  const clearDeepLinkIntent = useApprovalsStore((s) => s.clearDeepLinkIntent);

  const unavailable = error instanceof ApprovalsUnavailableError;
  // A 401 can arrive from the background list poll too, not only from an answer.
  const listSessionExpired = error instanceof ApprovalsUnauthorizedError;

  const decisions = useMemo(() => {
    const rows = data ?? [];
    const filtered = rows.filter((d) => {
      // Optimistically drop rows answered this session from the pending/all view
      // so an answered row never flashes as still-actionable before the refetch.
      if (answeredIds.has(d.id) && (filter === "pending" || filter === "all")) return false;
      if (filter === "pending") return d.state === "pending";
      if (filter === "resolved") return d.state === "answered";
      if (filter === "expired") return d.state === "expired";
      return true; // all
    });
    const q = query.trim().toLowerCase();
    const searched = q
      ? filtered.filter(
          (d) =>
            d.question.toLowerCase().includes(q) ||
            sourceLabel(d.source).toLowerCase().includes(q) ||
            d.kind.toLowerCase().includes(q),
        )
      : filtered;
    return sortDecisions(searched);
  }, [data, filter, query, answeredIds]);

  // Resolve a deep link once rows are present: select it (fetching it directly
  // if it's not in the current list), open the drawer, then clear the intent so
  // a later navigation doesn't reopen it. Same deferred-intent discipline the
  // workspace uses for a reopened session.
  useEffect(() => {
    if (!open || !pendingDeepLinkId) return;
    const targetId = pendingDeepLinkId;
    setSelectedId(targetId);
    const present = (data ?? []).some((d) => d.id === targetId);
    if (!present) {
      // Pull the one decision directly (it may be resolved / outside the filter)
      // and HOLD it so the detail actually renders — the prior version threw the
      // fetched decision away, leaving a blank detail on a deep link.
      void getApproval(targetId)
        .then((decision) => { setDeepLinkDecision(decision); })
        .catch(() => { /* a missing deep-linked decision simply leaves nothing selected */ });
    }
    clearDeepLinkIntent();
  }, [open, pendingDeepLinkId, data, clearDeepLinkIntent]);

  const selected = useMemo(() => {
    if (!selectedId) return null;
    const fromList = (data ?? []).find((d) => d.id === selectedId);
    if (fromList) return fromList;
    // Fall back to a directly-fetched deep-link decision not in the list.
    if (deepLinkDecision && deepLinkDecision.id === selectedId) return deepLinkDecision;
    return null;
  }, [data, selectedId, deepLinkDecision]);

  // On desktop, default-select the first row so the detail pane is never blank.
  useEffect(() => {
    if (!open || isMobile) return;
    const first = decisions[0];
    if (!selectedId && first) setSelectedId(first.id);
  }, [open, isMobile, selectedId, decisions]);

  useEffect(() => { setNote(""); setAnswerError(null); }, [selectedId]);

  // One in-flight answer at a time. A ref (not just the `answering` state) is the
  // guard a fast second swipe/tap hits before React re-renders, so a double
  // gesture cannot fire two answers.
  const inFlightRef = useRef(false);

  const submitAnswer = useCallback(async (decision: ApprovalDecision, optionKey: string) => {
    if (inFlightRef.current) return;
    inFlightRef.current = true;
    setAnswering(true);
    setAnswerError(null);
    try {
      await answerApproval(decision.id, optionKey, note.trim());
      setConfirming(null);
      setNote("");
      // Optimistic: hide this row from the actionable list immediately.
      setAnsweredIds((prev) => new Set(prev).add(decision.id));
      // A stale pending detail must not remain actionable while refetch runs.
      // Desktop selects the next available decision; mobile returns to the list.
      setSelectedId(null);
      await Promise.all([
        refetch(),
        queryClient.invalidateQueries({ queryKey: ["approvals-badge"] }),
      ]);
    } catch (cause) {
      if (cause instanceof ApprovalsUnauthorizedError) {
        // Session lapsed: switch the whole surface to the sign-in state rather
        // than a per-answer error with a dead retry.
        setSessionExpired(true);
        setConfirming(null);
      } else if (cause instanceof ApprovalAlreadyResolvedError) {
        // Someone else (or the deadline default) answered it first. Show the
        // friendly note, drop the selection, and STILL refetch so the stale row
        // leaves the list and the operator stops hitting 409.
        setAlreadyResolved(true);
        setConfirming(null);
        setAnsweredIds((prev) => new Set(prev).add(decision.id));
        setSelectedId(null);
        await Promise.all([
          refetch(),
          queryClient.invalidateQueries({ queryKey: ["approvals-badge"] }),
        ]);
      } else {
        setAnswerError(cause instanceof Error ? cause.message : String(cause));
      }
    } finally {
      inFlightRef.current = false;
      setAnswering(false);
    }
  }, [note, refetch, queryClient]);

  // An answer on an irreversible decision always confirms first; a reversible
  // one answers directly (the swipe / button is the confirmation). Guarded so a
  // second gesture while one is in flight is a no-op.
  const chooseOption = useCallback((decision: ApprovalDecision, optionKey: string) => {
    if (inFlightRef.current || answering) return;
    if (!isActionable(decision) || !optionKey) return;
    if (decision.reversible) {
      void submitAnswer(decision, optionKey);
    } else {
      setConfirming({ decision, optionKey });
    }
  }, [submitAnswer, answering]);

  const headerExtra = (
    <div className="mt-3 space-y-2">
      <label className="flex items-center gap-2 rounded-lg border border-wc-default bg-wc-surface-input px-3 py-2">
        <Search className="h-4 w-4 shrink-0 text-wc-text-muted" aria-hidden="true" />
        <input
          data-testid="approvals-search"
          value={query}
          onChange={(e) => { setQuery(e.target.value); }}
          className="min-w-0 flex-1 bg-transparent text-sm text-wc-text-primary outline-none"
          placeholder={t(strings.approvals.searchPlaceholder)}
        />
      </label>
      <div className="flex flex-wrap items-center gap-2 text-xs">
        {(["pending", "all", "resolved", "expired"] as FilterMode[]).map((mode) => (
          <button
            key={mode}
            type="button"
            data-testid={`approvals-filter-${mode}`}
            aria-pressed={filter === mode}
            onClick={() => { setFilter(mode); setSelectedId(null); }}
            className={cn(
              "rounded-full border px-3 py-1",
              filter === mode
                ? "border-wc-accent/50 bg-wc-accent/15 text-wc-text-primary"
                : "border-wc-default text-wc-text-muted hover:text-wc-text-secondary",
            )}
          >
            {mode === "pending"
              ? t(strings.approvals.filterPending)
              : mode === "all"
                ? t(strings.approvals.filterAll)
                : mode === "resolved"
                  ? t(strings.approvals.filterResolved)
                  : t(strings.approvals.filterExpired)}
          </button>
        ))}
      </div>
    </div>
  );

  return (
    <>
      <FullPageDrawer
        open={open}
        onClose={onClose}
        title={t(strings.approvals.title)}
        closeLabel={t(strings.approvals.close)}
        testId="approvals-drawer"
        avoidKeyboard
        headerExtra={headerExtra}
      >
        <div className="flex h-full min-h-0 flex-col md:flex-row">
          {/* ── List ───────────────────────────────────────────────── */}
          <section
            aria-label={t(strings.approvals.title)}
            data-testid="approvals-list"
            className={cn(
              "min-h-0 overflow-y-auto border-wc-default md:w-[24rem] md:border-e",
              // On mobile the list hides once a decision is open (drill-in).
              isMobile && selected ? "hidden" : "block",
              isMobile ? "max-h-none flex-1" : "shrink-0",
            )}
          >
            {renderListBody()}
          </section>

          {/* ── Detail ─────────────────────────────────────────────── */}
          <section
            data-testid="approvals-detail"
            className={cn(
              "min-h-0 flex-1 overflow-y-auto",
              isMobile && !selected ? "hidden" : "block",
            )}
          >
            {renderDetailBody()}
          </section>
        </div>
      </FullPageDrawer>

      {confirming && (
        <AlertDialog
          open
          title={t(strings.approvals.confirmTitle)}
          description={
            confirming.decision.reversible
              ? t(strings.approvals.confirmBody, {
                  option: optionLabel(confirming.decision, confirming.optionKey),
                  question: confirming.decision.question,
                })
              : t(strings.approvals.confirmIrreversible, {
                  option: optionLabel(confirming.decision, confirming.optionKey),
                  question: confirming.decision.question,
                })
          }
          destructive={!confirming.decision.reversible}
          busy={answering}
          confirmLabel={t(strings.approvals.confirm)}
          cancelLabel={t(strings.approvals.cancel)}
          busyLabel={t(strings.approvals.answering)}
          onCancel={() => { if (!answering) setConfirming(null); }}
          onConfirm={() => void submitAnswer(confirming.decision, confirming.optionKey)}
          testIdPrefix="approvals-confirm"
        />
      )}
    </>
  );

  // ── render helpers (closures so they read the local state) ───────────────

  function renderListBody() {
    if (isLoading) return <ListSkeleton />;
    // Session expired (from an answer OR the background list poll): offer sign-in,
    // never a dead retry.
    if (sessionExpired || listSessionExpired) {
      return (
        <div className="p-6" data-testid="approvals-session-expired">
          <EmptyState
            icon={<AlertTriangle className="h-7 w-7 text-wc-warning" aria-hidden="true" />}
            title={t(strings.approvals.sessionExpiredTitle)}
            description={t(strings.approvals.sessionExpiredBody)}
            actionLabel={t(strings.approvals.signIn)}
            onAction={() => {
              setSessionExpired(false);
              void signIn();
            }}
          />
        </div>
      );
    }
    if (unavailable) {
      return (
        <div className="p-4" data-testid="approvals-unavailable">
          <ErrorState
            title={t(strings.approvals.errorTitle)}
            message={t(strings.approvals.unavailable)}
            onRetry={() => void refetch()}
          />
        </div>
      );
    }
    if (error) {
      return (
        <div className="p-4" data-testid="approvals-error">
          <ErrorState
            title={t(strings.approvals.errorTitle)}
            message={error.message}
            onRetry={() => void refetch()}
          />
        </div>
      );
    }
    const resolvedBanner = alreadyResolved ? (
      <div
        role="status"
        data-testid="approvals-already-resolved"
        className="m-3 flex items-start gap-2 rounded-lg border border-wc-default bg-wc-surface-raised p-3 text-xs text-wc-text-secondary"
      >
        <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-wc-warning" aria-hidden="true" />
        <span className="flex-1">{t(strings.approvals.alreadyResolved)}</span>
        <button
          type="button"
          data-testid="approvals-already-resolved-dismiss"
          className="text-wc-text-faint hover:text-wc-text-secondary"
          onClick={() => { setAlreadyResolved(false); }}
          aria-label={t(strings.approvals.cancel)}
        >
          <X className="h-3.5 w-3.5" aria-hidden="true" />
        </button>
      </div>
    ) : null;

    if (decisions.length === 0) {
      return (
        <>
          {resolvedBanner}
          <div className="p-6" data-testid="approvals-empty">
            <EmptyState
              icon={<ShieldCheck className="h-7 w-7 text-wc-success" aria-hidden="true" />}
              title={t(strings.approvals.empty)}
              description={t(strings.approvals.emptyBody)}
            />
          </div>
        </>
      );
    }
    return (
      <>
        {resolvedBanner}
        {decisions.map((decision) => (
          <ApprovalRow
            key={decision.id}
            decision={decision}
            selected={decision.id === selectedId}
            isMobile={isMobile}
            swipeOpen={swipeOpenId === decision.id}
            onSwipeOpenChange={(next) => { setSwipeOpenId(next ? decision.id : null); }}
            onSelect={() => { setSelectedId(decision.id); }}
            onApprove={() => {
              // Approve submits the producer's recommended option ONLY when it is
              // a valid option; otherwise open the detail to choose explicitly.
              const key = approveOptionKey(decision);
              if (key) chooseOption(decision, key);
              else setSelectedId(decision.id);
            }}
            onReject={() => {
              // Reject = the non-recommended option when there are exactly two;
              // else open the detail to choose explicitly.
              const others = secondaryOptions(decision);
              const only = others.length === 1 ? others[0] : undefined;
              if (only) chooseOption(decision, only.key);
              else setSelectedId(decision.id);
            }}
            t={t}
          />
        ))}
      </>
    );
  }

  function renderDetailBody() {
    if (sessionExpired || listSessionExpired) return null;
    if (!selected) {
      return (
        <div className="hidden h-full items-center justify-center p-8 text-sm text-wc-text-muted md:flex" data-testid="approvals-detail-empty">
          {t(strings.approvals.selectPrompt)}
        </div>
      );
    }
    return (
      <ApprovalDetail
        decision={selected}
        isMobile={isMobile}
        note={note}
        onNoteChange={setNote}
        answering={answering}
        answerError={answerError}
        onBack={() => { setSelectedId(null); }}
        onChoose={(key) => { chooseOption(selected, key); }}
        t={t}
      />
    );
  }
}

function optionLabel(decision: ApprovalDecision, key: string): string {
  return decision.options.find((o) => o.key === key)?.label ?? key;
}

// ── List row ────────────────────────────────────────────────────────────────

function ApprovalRow({
  decision,
  selected,
  isMobile,
  swipeOpen,
  onSwipeOpenChange,
  onSelect,
  onApprove,
  onReject,
  t,
}: {
  decision: ApprovalDecision;
  selected: boolean;
  isMobile: boolean;
  swipeOpen: boolean;
  onSwipeOpenChange: (open: boolean) => void;
  onSelect: () => void;
  onApprove: () => void;
  onReject: () => void;
  t: TFunction;
}) {
  const accent = sourceAccent(decision.source);
  const rec = recommendedOption(decision);
  const expired = decision.state === "expired";
  const resolved = decision.state === "answered";

  const card = (
    <button
      type="button"
      data-testid={`approval-row-${decision.id}`}
      onClick={onSelect}
      className={cn(
        "block w-full border-b border-wc-default/60 p-3 text-start transition",
        selected ? "bg-wc-accent/10" : "hover:bg-wc-surface-raised",
      )}
    >
      <div className="mb-1 flex items-center gap-2">
        <span className={cn("inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-[10.5px] font-semibold uppercase tracking-wide", accent.bg, accent.text)}>
          <span className={cn("h-1.5 w-1.5 rounded-full", accent.dot)} aria-hidden="true" />
          {sourceLabel(decision.source)}
        </span>
        <span className="ms-auto text-[10.5px] text-wc-text-faint">{relativeTime(decision.createdAt)}</span>
      </div>
      <div className="truncate text-sm font-semibold text-wc-text-primary">{decision.question}</div>
      <div className="mt-1.5 flex flex-wrap items-center gap-1.5">
        {decision.reversible ? (
          <span className="rounded-full bg-wc-success/12 px-2 py-0.5 text-[10.5px] font-semibold text-wc-success">{t(strings.approvals.reversible)}</span>
        ) : (
          <span className="rounded-full border border-wc-error-border bg-wc-error-surface/50 px-2 py-0.5 text-[10.5px] font-semibold text-wc-error-text">{t(strings.approvals.notReversible)}</span>
        )}
        {rec && !resolved && !expired && (
          <span className="rounded-full bg-wc-accent/12 px-2 py-0.5 text-[10.5px] font-semibold text-wc-accent">{t(strings.approvals.recommends, { label: rec.label })}</span>
        )}
        {expired && (
          <span className="rounded-full bg-wc-error-surface/50 px-2 py-0.5 text-[10.5px] font-semibold text-wc-error-text">{t(strings.approvals.expired, { when: relativeTime(decision.deadline) })}</span>
        )}
        {resolved && (
          <span className="rounded-full bg-wc-surface-input px-2 py-0.5 text-[10.5px] font-semibold text-wc-text-muted">{t(strings.approvals.answered, { label: decision.answerLabel || decision.answer })}</span>
        )}
        {!expired && !resolved && decision.deadline && (
          <span className="rounded-full bg-wc-warning/10 px-2 py-0.5 text-[10.5px] font-semibold text-wc-warning">{t(strings.approvals.expiresIn, { when: relativeTime(decision.deadline) })}</span>
        )}
      </div>
    </button>
  );

  // Swipe is a MOBILE accelerator, and ONLY on a reversible, still-actionable
  // decision. Irreversible or resolved/expired rows get the plain card: the
  // irreversible ones must be opened to confirm (the ratified safety rule).
  if (!isMobile || !decision.reversible || !isActionable(decision)) {
    return card;
  }
  return (
    <SwipeActions
      testId={`approval-swipe-${decision.id}`}
      label={t(strings.approvals.swipeAria, { subject: decision.question })}
      open={swipeOpen}
      onOpenChange={onSwipeOpenChange}
      releaseMode="rest"
      actions={[
        { id: "approve", label: t(strings.approvals.swipeApprove), icon: <Check className="h-4 w-4" aria-hidden="true" />, tone: "primary", onSelect: onApprove },
        { id: "reject", label: t(strings.approvals.swipeReject), icon: <X className="h-4 w-4" aria-hidden="true" />, tone: "destructive", onSelect: onReject },
      ]}
    >
      {card}
    </SwipeActions>
  );
}

// ── Detail ──────────────────────────────────────────────────────────────────

function ApprovalDetail({
  decision,
  isMobile,
  note,
  onNoteChange,
  answering,
  answerError,
  onBack,
  onChoose,
  t,
}: {
  decision: ApprovalDecision;
  isMobile: boolean;
  note: string;
  onNoteChange: (v: string) => void;
  answering: boolean;
  answerError: string | null;
  onBack: () => void;
  onChoose: (optionKey: string) => void;
  t: TFunction;
}) {
  const accent = sourceAccent(decision.source);
  const rec = recommendedOption(decision);
  const others = secondaryOptions(decision);
  const actionable = isActionable(decision);
  const expired = decision.state === "expired";
  const resolved = decision.state === "answered";

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="border-b border-wc-default/60 p-4 md:p-6">
        {isMobile && (
          <button type="button" data-testid="approvals-detail-back" onClick={onBack} className="mb-3 inline-flex items-center gap-1 text-sm text-wc-text-muted">
            ‹ {t(strings.approvals.viewAll)}
          </button>
        )}
        <div className="mb-2 flex flex-wrap items-center gap-2">
          <span className={cn("inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-[11px] font-semibold uppercase tracking-wide", accent.bg, accent.text)}>
            <span className={cn("h-1.5 w-1.5 rounded-full", accent.dot)} aria-hidden="true" />
            {sourceLabel(decision.source)}{decision.kind ? ` · ${decision.kind}` : ""}
          </span>
          {decision.reversible ? (
            <span className="rounded-full bg-wc-success/12 px-2 py-0.5 text-[11px] font-semibold text-wc-success">{t(strings.approvals.reversible)}</span>
          ) : (
            <span className="rounded-full border border-wc-error-border bg-wc-error-surface/50 px-2 py-0.5 text-[11px] font-semibold text-wc-error-text">{t(strings.approvals.notReversible)}</span>
          )}
          {decision.deadline && !resolved && (
            <span className={cn("ms-auto rounded-full px-2.5 py-1 text-xs font-semibold", expired ? "bg-wc-error-surface/50 text-wc-error-text" : "bg-wc-warning/10 text-wc-warning")}>
              {expired ? t(strings.approvals.expired, { when: relativeTime(decision.deadline) }) : t(strings.approvals.expiresIn, { when: relativeTime(decision.deadline) })}
            </span>
          )}
        </div>
        <h2 className="text-lg font-semibold text-wc-text-primary md:text-xl">{decision.question}</h2>
      </div>

      <div className="flex-1 space-y-5 p-4 md:p-6">
        <ApprovalContextPanel decision={decision} contextUrl={decision.contextUrl} />

        {resolved ? (
          <div className="rounded-lg border border-wc-default bg-wc-surface-raised p-4 text-sm" data-testid="approvals-resolved-note">
            <div className="font-semibold text-wc-text-primary">{t(strings.approvals.answered, { label: decision.answerLabel || decision.answer })}</div>
            {(decision.answeredBy || decision.answeredAt) && (
              <div className="mt-1 text-wc-text-muted">{t(strings.approvals.answeredBy, { who: decision.answeredBy || "you", when: relativeTime(decision.answeredAt) })}</div>
            )}
            {decision.note && <div className="mt-2 text-wc-text-secondary">“{decision.note}”</div>}
          </div>
        ) : expired ? (
          <div className="rounded-lg border border-wc-error-border bg-wc-error-surface/50 p-4" data-testid="approvals-expired-note">
            <div className="flex items-center gap-2 text-sm font-semibold text-wc-error-text">
              <AlertTriangle className="h-4 w-4" aria-hidden="true" />
              {t(strings.approvals.notActionable)}
            </div>
          </div>
        ) : (
          <div data-testid="approvals-answer-block">
            <h3 className="mb-2.5 text-[11px] font-bold uppercase tracking-wide text-wc-text-faint">{t(strings.approvals.yourDecision)}</h3>
            {rec && (
              <p className="mb-3 text-sm leading-relaxed text-wc-text-muted">
                {decision.recommendationReason
                  ? t(strings.approvals.recommendedReason, { reason: decision.recommendationReason })
                  : t(strings.approvals.recommends, { label: rec.label })}
              </p>
            )}
            <div className="flex flex-wrap gap-2.5">
              {rec && (
                <Button variant="default" size="lg" shape="square" pending={answering} disabled={!actionable} data-testid="approvals-answer-recommended" onClick={() => { onChoose(rec.key); }}>
                  <Check className="me-1.5 h-4 w-4" aria-hidden="true" /> {rec.label}
                </Button>
              )}
              {others.map((opt) => (
                <Button
                  key={opt.key}
                  variant={isRejectLike(opt.key, opt.label) ? "danger" : "secondary"}
                  size="lg"
                  shape="square"
                  disabled={!actionable || answering}
                  data-testid={`approvals-answer-${opt.key}`}
                  onClick={() => { onChoose(opt.key); }}
                >
                  {opt.label}
                </Button>
              ))}
            </div>
            <label className="mt-4 block">
              <span className="mb-1.5 block text-[11px] font-bold uppercase tracking-wide text-wc-text-faint">{t(strings.approvals.noteLabel)}</span>
              <textarea
                data-testid="approvals-note"
                value={note}
                onChange={(e) => { onNoteChange(e.target.value); }}
                disabled={!actionable || answering}
                rows={3}
                className="w-full resize-y rounded-lg border border-wc-default bg-wc-surface-input p-3 text-sm text-wc-text-primary outline-none focus:border-wc-accent/60"
                placeholder={t(strings.approvals.notePlaceholder)}
              />
            </label>
            {decision.defaultAnswer && decision.reversible && (
              <p className="mt-2 text-xs text-wc-text-muted">
                {t(strings.approvals.autoApplies, { when: relativeTime(decision.deadline) })} · {optionLabel(decision, decision.defaultAnswer)}
              </p>
            )}
            {answerError && <p className="mt-2 text-xs text-wc-error-text" role="alert">{answerError}</p>}
          </div>
        )}
      </div>
    </div>
  );
}

/** Heuristic: an option reads as a reject/deny/stop so it takes the danger variant. */
function isRejectLike(key: string, label: string): boolean {
  return /reject|deny|decline|stop|cancel|no\b/i.test(`${key} ${label}`);
}

// ── Loading skeleton ─────────────────────────────────────────────────────────

function ListSkeleton() {
  return (
    <div className="space-y-3 p-3" data-testid="approvals-loading">
      {[0, 1, 2, 3].map((i) => (
        <div key={i} className="flex gap-3">
          <div className="h-16 w-1 shrink-0 animate-pulse rounded bg-wc-surface-input" />
          <div className="flex-1 space-y-2">
            <div className="h-3 w-2/5 animate-pulse rounded bg-wc-surface-input" />
            <div className="h-4 w-4/5 animate-pulse rounded bg-wc-surface-input" />
            <div className="h-3 w-3/5 animate-pulse rounded bg-wc-surface-input" />
          </div>
        </div>
      ))}
    </div>
  );
}
