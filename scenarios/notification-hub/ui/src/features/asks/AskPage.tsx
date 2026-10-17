import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Code, ConnectError } from "@connectrpc/connect";
import { Button } from "@vrooli/react-component-library/Button/2";
import type { Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { conversationsClient } from "../../api/notifications";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { SessionGate } from "../session/SessionGate";
import {
  optionLabel,
  orderedOptions,
  remainingUntil,
  safeContextUrl,
} from "./askView";

/**
 * One decision, built for a phone: a push opens this page, and one tap on an
 * option answers it. iOS home-screen apps show no notification action
 * buttons, so this page is the answer path there.
 */
export function AskPage() {
  return (
    <section
      data-testid={selectors.pages.ask}
      className="mx-auto flex w-full max-w-xl flex-col gap-4"
    >
      <SessionGate>
        <AskDetail />
      </SessionGate>
    </section>
  );
}

function AskDetail() {
  const { askId = "" } = useParams();
  const { t } = useTranslation();
  const ask = useQuery({
    queryKey: ["ask", askId],
    queryFn: async () => (await conversationsClient.getAsk({ askId })).ask,
    retry: (count, error) =>
      !(error instanceof ConnectError && error.code === Code.NotFound) &&
      count < 2,
  });

  if (ask.isPending)
    return (
      <p className="text-app-muted-foreground">
        {t(strings.asks.detail.loading)}
      </p>
    );
  if (!ask.data) return <p role="alert">{t(strings.asks.detail.notFound)}</p>;
  return <AskBody ask={ask.data} />;
}

function AskBody({ ask }: { ask: Ask }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [note, setNote] = useState("");
  const answer = useMutation({
    mutationFn: (key: string) =>
      conversationsClient.answer({
        askId: ask.id,
        answer: key,
        note: note.trim(),
      }),
    onSuccess: () =>
      queryClient
        .invalidateQueries({ queryKey: ["ask", ask.id] })
        .then(() => queryClient.invalidateQueries({ queryKey: ["asks"] })),
  });
  const now = useNow();
  const open = ask.state === "pending" || ask.state === "escalated";
  const canAnswer = open || ask.state === "defaulted";
  const context = ask.contextUrl ? safeContextUrl(ask.contextUrl) : null;

  return (
    <>
      <Link to="/asks" className="text-sm text-app-muted-foreground underline">
        {t(strings.asks.detail.back)}
      </Link>
      <h2
        data-testid={selectors.asks.question}
        className="text-2xl font-semibold leading-snug"
      >
        {ask.question}
      </h2>
      {context && (
        <a href={context} className="text-sm underline" rel="noreferrer">
          {t(strings.asks.detail.more)}
        </a>
      )}
      <AskStatus ask={ask} now={now} />
      {canAnswer && (
        <div className="flex flex-col gap-3">
          {orderedOptions(ask).map((option) => {
            const recommended = option.key === ask.recommended;
            return (
              <div key={option.key} className="flex flex-col gap-1">
                <Button
                  data-testid={selectors.asks.option}
                  data-option-key={option.key}
                  type="button"
                  variant={recommended ? "primary" : "outline"}
                  className="min-h-14 w-full justify-center text-base"
                  disabled={answer.isPending}
                  onClick={() => answer.mutate(option.key)}
                >
                  {option.label}
                  {recommended && (
                    <span className="ms-2 text-xs opacity-80">
                      ({t(strings.asks.detail.recommended)})
                    </span>
                  )}
                </Button>
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
              value={note}
              onChange={(event) => setNote(event.target.value)}
            />
          </details>
          {answer.isPending && (
            <p role="status">{t(strings.asks.detail.sending)}</p>
          )}
          {answer.error && (
            <p role="alert" className="text-app-danger">
              {t(strings.asks.detail.saveError, {
                message:
                  answer.error instanceof ConnectError
                    ? answer.error.rawMessage
                    : String(answer.error),
              })}
            </p>
          )}
        </div>
      )}
    </>
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
        {ask.note && (
          <p className="mt-1 text-sm text-app-muted-foreground">{ask.note}</p>
        )}
      </div>
    );
  }
  if (ask.state === "defaulted") {
    return (
      <p data-testid={selectors.asks.status} role="status">
        {t(strings.asks.detail.defaulted, {
          option: optionLabel(ask, ask.defaultAnswer),
        })}
      </p>
    );
  }
  if (ask.state === "expired") {
    return (
      <p data-testid={selectors.asks.status} role="status">
        {t(strings.asks.detail.expired)}
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

function useNow(): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 30_000);
    return () => window.clearInterval(timer);
  }, []);
  return now;
}
