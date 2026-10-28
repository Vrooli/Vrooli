import { useEffect, useState } from "react";
import { Link, useLocation, useNavigate, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Code, ConnectError } from "@connectrpc/connect";
import { Button } from "@vrooli/react-component-library/Button/2";
import type { Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { conversationsClient } from "../../api/notifications";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { SignInForm, useSession } from "../session/SessionGate";
import {
  optionLabel,
  orderedOptions,
  remainingUntil,
  safeContextUrl,
} from "./askView";
import {
  readAskAttempt,
  readAskDraft,
  writeAskAttempt,
  writeAskDraft,
  clearAskDraft,
} from "./askDrafts";
import { preserveRoutedTestMode } from "../routedTestMode";

/**
 * One decision, built for a phone: a push opens this page, option selection
 * stays a draft, and the recipient deliberately submits the answer. iOS
 * home-screen apps show no notification action buttons, so this page is the
 * answer path there.
 */
export function AskPage() {
  return <AskDetail />;
}

function AskDetail() {
  const { askId = "" } = useParams();
  const { t } = useTranslation();
  const session = useSession();
  const subject = session.data?.subject ?? "";
  const [expiredAttemptSubject, setExpiredAttemptSubject] = useState("");
  useEffect(() => {
    const onExpired = () => {
      if (subject) setExpiredAttemptSubject(subject);
    };
    window.addEventListener("notification-hub:session-expired", onExpired);
    return () => window.removeEventListener("notification-hub:session-expired", onExpired);
  }, [subject]);
  const attemptSubject = subject || expiredAttemptSubject;
  const ask = useQuery({
    queryKey: ["ask", askId, subject],
    enabled: !session.isPending,
    queryFn: async () => (await conversationsClient.getAsk({ askId })).ask,
    retry: (count, error) =>
      !(error instanceof ConnectError && error.code === Code.NotFound) &&
      count < 2,
  });

  if (session.isPending || ask.isPending)
    return (
      <p className="text-app-muted-foreground">
        {t(strings.asks.detail.loading)}
      </p>
    );
  if (ask.error instanceof ConnectError && ask.error.code === Code.Unauthenticated) return <SignInForm />;
  if (!ask.data) return <p role="alert">{t(strings.asks.detail.notFound)}</p>;
  return (
    <AskBody
      key={`${askId}\u0000${subject}`}
      ask={ask.data}
      subject={subject}
      attemptSubject={attemptSubject}
      signedIn={Boolean(session.data?.signedIn)}
    />
  );
}

function AskBody({
  ask,
  subject,
  attemptSubject,
  signedIn,
}: {
  ask: Ask;
  subject: string;
  attemptSubject: string;
  signedIn: boolean;
}) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const location = useLocation();
  const navigate = useNavigate();
  const [draft, setDraft] = useState(() => readAskDraft(subject, ask.id));
  const [attempt, setAttempt] = useState(() => readAskAttempt(attemptSubject, ask.id));
  const [outcome, setOutcome] = useState<"uncertain" | "open" | "defaulted" | "recorded" | "conflict" | null>(() =>
    readAskAttempt(attemptSubject, ask.id) ? "uncertain" : null,
  );
  const [recordedLabel, setRecordedLabel] = useState("");
  const [readError, setReadError] = useState(false);
  const readback = useMutation({
    retry: false,
    mutationFn: () => conversationsClient.getAsk({ askId: ask.id }),
    onSuccess: async (response) => {
      const current = response.ask;
      if (!current) {
        setReadError(true);
        return;
      }
      setReadError(false);
      await queryClient.setQueryData(["ask", ask.id, subject], current);
      if (current.state === "answered" && attempt && current.answer === attempt.answer && current.note === attempt.note) {
        setOutcome("recorded");
        setRecordedLabel(current.answerLabel || current.answer);
        clearAskDraft(subject, ask.id);
      } else if (current.state === "pending" || current.state === "escalated") {
        setOutcome("open");
      } else if (current.state === "defaulted") {
        setOutcome("defaulted");
      } else if (current.state === "answered" || current.state === "expired" || current.state === "withdrawn") {
        setOutcome("conflict");
        setRecordedLabel(current.answerLabel || current.answer || current.state);
      }
      await queryClient.invalidateQueries({ queryKey: ["asks"] });
    },
    onError: (error) => {
      setReadError(true);
      if (error instanceof ConnectError && error.code === Code.Unauthenticated) setOutcome("uncertain");
    },
  });
  const answer = useMutation({
    retry: false,
    mutationFn: (payload: { answer: string; note: string }) =>
      conversationsClient.answer({
        askId: ask.id,
        answer: payload.answer,
        note: payload.note,
      }),
    onSuccess: () => {
      // A successful mutation response is not the displayed terminal state.
      // Confirm the saved answer from the recipient-scoped read path first.
      readback.mutate();
    },
    onError: () => setOutcome("uncertain"),
  });
  const now = useNow();
  const open = ask.state === "pending" || ask.state === "escalated";
  const canAnswer = open || ask.state === "defaulted";
  const context = ask.contextUrl ? safeContextUrl(ask.contextUrl) : null;
  const returnState = location.state as { from?: unknown } | null;
  const returnTo = typeof returnState?.from === "string" && /^\/(?:asks|history)(?:\?|$)/.test(returnState.from)
    ? returnState.from
    : preserveRoutedTestMode("/asks", location.search);
  const updateDraft = (next: typeof draft) => {
    setDraft(next);
    writeAskDraft(subject, ask.id, next);
  };
  const submit = () => {
    if (!subject || !draft.answer || attempt || answer.isPending) return;
    const immutable = { answer: draft.answer, note: draft.note.trim() };
    const savedAttempt = { ...immutable, subject };
    setAttempt(savedAttempt);
    writeAskAttempt(subject, ask.id, savedAttempt);
    answer.mutate(immutable);
  };

  return (
    <section aria-labelledby="ask-heading" className="mx-auto flex w-full max-w-3xl flex-col gap-5">
      <header className="-mx-4 -mt-4 flex min-h-14 items-center gap-5 border-b border-app-border bg-app-surface-muted px-4 py-3 sm:-mx-8 sm:px-8">
        <Link to={returnTo} state={{ restoreAskId: ask.id }} onClick={(event) => { event.preventDefault(); navigate(returnTo, { state: { restoreAskId: ask.id } }); }} className="shrink-0 text-sm font-medium underline">
          {returnTo.startsWith("/history") ? t("pages.history.back") : t(strings.asks.detail.back)}
        </Link>
        <span className="truncate text-sm font-semibold" aria-label={t("asks.detail.requestId", { defaultValue: "Request ID" })}>{ask.id}</span>
      </header>
      {ask.source && <p className="text-sm text-app-muted-foreground">{ask.source}</p>}
      <h1
        id="ask-heading"
        data-testid={selectors.asks.question}
        className="break-words text-3xl font-bold leading-tight tracking-tight sm:text-5xl"
      >
        {ask.question}
      </h1>
      {context && (
        <a href={context} className="text-sm underline" rel="noreferrer">
          {t(strings.asks.detail.more)}
        </a>
      )}
      {ask.bigPicture && (
        <details className="rounded-panel border border-app-border p-3">
          <summary className="cursor-pointer font-medium">
            {t(strings.asks.detail.bigPicture)}
          </summary>
          <p className="mt-2 whitespace-pre-wrap break-words text-sm text-app-muted-foreground">
            {ask.bigPicture}
          </p>
        </details>
      )}
      {ask.technicalDetails && (
        <details className="rounded-panel border border-app-border p-3">
          <summary className="cursor-pointer font-medium">
            {t(strings.asks.detail.technicalDetails)}
          </summary>
          <p className="mt-2 whitespace-pre-wrap break-words text-sm text-app-muted-foreground">
            {ask.technicalDetails}
          </p>
        </details>
      )}
      <AskStatus ask={ask} now={now} />
      {answer.isSuccess && readback.isPending && <p role="status">{t(strings.asks.detail.checking)}</p>}
      {outcome === "recorded" && <p role="status">{t(strings.asks.detail.recorded, { option: recordedLabel })}</p>}
      {outcome === "uncertain" && <p role="alert">{t(strings.asks.detail.uncertain)}</p>}
      {outcome === "open" && <p role="status">{t(strings.asks.detail.stillOpen)}</p>}
      {outcome === "defaulted" && <p role="status">{t("asks.detail.defaultAnswerPending", { defaultValue: "The default is still recorded. Your later answer may still finish. Check status; do not submit again." })}</p>}
      {outcome === "conflict" && <p role="alert">{t(strings.asks.detail.conflict, { option: recordedLabel, attempted: attempt ? optionLabel(ask, attempt.answer) : "" })}</p>}
      {readError && <p role="alert">{t(strings.asks.detail.readbackFailed)}</p>}
      {canAnswer && !signedIn && (
        <div className="flex flex-col gap-3">
          <SignInForm />
          {attempt && outcome !== "recorded" && (
            <Button className="min-h-12 w-full" type="button" variant="outline" disabled={readback.isPending} onClick={() => { setReadError(false); readback.mutate(); }}>
              {readback.isPending ? t(strings.asks.detail.checking) : t(strings.asks.detail.checkStatus)}
            </Button>
          )}
        </div>
      )}
      {canAnswer && signedIn && (
        <div className="flex flex-col gap-3">
          {orderedOptions(ask).map((option) => {
            const recommended = option.key === ask.recommended;
            return (
              <div key={option.key} className={`flex flex-col gap-1 rounded-panel border p-3 ${draft.answer === option.key ? "border-app-accent bg-app-surface-muted" : "border-app-border"}`}>
                <label className="flex min-h-11 cursor-pointer items-center gap-3">
                  <input data-testid={selectors.asks.option} data-option-key={option.key} type="radio" name={`answer-${ask.id}`} value={option.key} checked={draft.answer === option.key} disabled={Boolean(attempt) || answer.isPending} onChange={() => updateDraft({ ...draft, answer: option.key })} />
                  <span className="font-medium">{option.label}{recommended && <span className="ms-2 text-xs text-app-muted-foreground">({t(strings.asks.detail.recommended)})</span>}</span>
                </label>
                {recommended && ask.recommendationReason && (
                  <p className="text-sm text-app-muted-foreground">
                    {ask.recommendationReason}
                  </p>
                )}
              </div>
            );
          })}
          <details className="text-sm">
            <summary className="cursor-pointer py-2">
              {t(strings.asks.detail.noteLabel)}
            </summary>
            <textarea
              data-testid={selectors.asks.note}
              aria-label={t(strings.asks.detail.noteLabel)}
              className="mt-1 min-h-24 w-full rounded-control border border-app-border bg-app-surface p-2"
              value={draft.note}
              disabled={Boolean(attempt) || answer.isPending}
              onChange={(event) => updateDraft({ ...draft, note: event.target.value })}
            />
          </details>
        </div>
      )}
      {answer.isPending && <p role="status">{t(strings.asks.detail.sending)}</p>}
      {canAnswer && signedIn && !attempt && (
        <footer className="sticky bottom-0 z-10 -mx-4 border-t border-app-border bg-app-surface/95 px-4 py-3 pb-[calc(0.75rem+env(safe-area-inset-bottom))] backdrop-blur sm:-mx-8 sm:px-8">
          <Button className="min-h-12 w-full" type="button" variant="primary" disabled={!draft.answer || answer.isPending} onClick={submit}>{t(strings.asks.detail.submit)}</Button>
        </footer>
      )}
      {attempt && outcome !== "recorded" && signedIn && (
        <footer className="sticky bottom-0 z-10 -mx-4 border-t border-app-border bg-app-surface/95 px-4 py-3 pb-[calc(0.75rem+env(safe-area-inset-bottom))] backdrop-blur sm:-mx-8 sm:px-8">
          <Button className="min-h-12 w-full" type="button" variant="outline" disabled={readback.isPending} onClick={() => { setReadError(false); readback.mutate(); }}>
            {readback.isPending ? t(strings.asks.detail.checking) : t(strings.asks.detail.checkStatus)}
          </Button>
        </footer>
      )}
    </section>
  );
}

function AskStatus({ ask, now }: { ask: Ask; now: number }) {
  const { t } = useTranslation();
  if (ask.state === "answered") {
    return (
      <div
        data-testid={selectors.asks.status}
        role="status"
        className="rounded-panel border border-app-border p-3"
      >
        <p className="font-medium">
          {t(strings.asks.detail.answered, {
            option: ask.answerLabel || ask.answer,
          })}
        </p>
        {ask.late && ask.defaultAnswer && (
          <div className="mt-3 border-t border-app-border pt-3 text-sm">
            <p className="font-medium">
              {t("asks.detail.lateAnswerRecorded", {
                defaultValue: "A later answer was recorded after the default.",
              })}
            </p>
            <dl className="mt-2 grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-app-muted-foreground">
              <dt>{t("asks.detail.recordedDefault", { defaultValue: "Recorded default" })}</dt>
              <dd className="text-right">{optionLabel(ask, ask.defaultAnswer)}</dd>
              <dt>{t("asks.detail.defaultApplied", { defaultValue: "Default applied" })}</dt>
              <dd className="text-right">
                {ask.defaultAppliedAt ? (
                  <time dateTime={ask.defaultAppliedAt}>{formatAskTime(ask.defaultAppliedAt)}</time>
                ) : t("asks.detail.timeUnavailable", { defaultValue: "Time unavailable" })}
              </dd>
            </dl>
            <p className="mt-3 text-app-muted-foreground">
              {t("asks.detail.lateAnswerDoesNotUndo", {
                defaultValue: "The requester may already have acted on the default. This answer does not automatically undo that work.",
              })}
            </p>
          </div>
        )}
        {ask.note && (
          <p className="mt-1 text-sm text-app-muted-foreground">{ask.note}</p>
        )}
      </div>
    );
  }
  if (ask.state === "defaulted") {
    return (
      <div data-testid={selectors.asks.status} role="status" className="rounded-panel border border-app-border p-3">
        <p className="font-medium">
          {t(strings.asks.detail.defaulted, { option: optionLabel(ask, ask.defaultAnswer) })}
        </p>
        {ask.defaultAppliedAt && <p className="mt-1 text-sm text-app-muted-foreground"><time dateTime={ask.defaultAppliedAt}>{formatAskTime(ask.defaultAppliedAt)}</time></p>}
        {ask.reversible && <p className="mt-2 text-sm text-app-muted-foreground">{t("asks.detail.lateAnswerAvailable", { defaultValue: "You can still give one later answer. The requester may already have acted on this default." })}</p>}
      </div>
    );
  }
  if (ask.state === "expired") {
    return (
      <p data-testid={selectors.asks.status} role="status">
        {t(strings.asks.detail.expired)}
      </p>
    );
  }
  if (ask.state === "withdrawn") {
    return (
      <p data-testid={selectors.asks.status} role="status">
        {t(strings.asks.detail.withdrawn)}
      </p>
    );
  }
  if (!ask.defaultAnswer) return null;
  const remaining = remainingUntil(ask.defaultEligibleAt, now);
  const option = optionLabel(ask, ask.defaultAnswer);
  return (
    <p
      data-testid={selectors.asks.defaultNotice}
      className="rounded-panel bg-app-surface-muted p-3 text-sm"
    >
      {remaining
        ? t(strings.asks.detail.defaultIn, { option, time: remaining })
        : t(strings.asks.detail.defaultWaiting, { option })}
    </p>
  );
}

function formatAskTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat(undefined, {
    timeZone: "UTC",
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

function useNow(): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 30_000);
    return () => window.clearInterval(timer);
  }, []);
  return now;
}
