import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Code, ConnectError } from "@connectrpc/connect";
import type { Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { conversationsClient } from "../../api/notifications";
import { REASON_ASK_CHANGED, hasHubReason } from "./hubError";
import { blockerFor, buildAnswer, type AnswerPayload } from "./model/answer";
import type { PacketDraft } from "./model/draft";
import { clearAskAttempt, clearAskDraft, readAskAttempt, writeAskAttempt } from "./model/draftStore";

export type Outcome = "uncertain" | "open" | "defaulted" | "recorded" | "dismissed" | "conflict" | "changed";

const isAskChanged = (error: unknown): boolean => hasHubReason(error, REASON_ASK_CHANGED);

/**
 * Sending the operator's answer, and never losing it. The payload is frozen
 * into an attempt before it leaves, so after any uncertain response the page
 * only reads the decision back from the server; it never sends the answer a
 * second time. Everything shown as the result comes from that readback.
 */
export function useAnswerFlow({ ask, subject, attemptSubject, draft }: { ask: Ask; subject: string; attemptSubject: string; draft: PacketDraft }) {
  const queryClient = useQueryClient();
  const [attempt, setAttempt] = useState(() => readAskAttempt(attemptSubject, ask.id));
  const [dismissAttempt, setDismissAttempt] = useState(false);
  const [outcome, setOutcome] = useState<Outcome | null>(() => (attempt ? "uncertain" : null));
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
      queryClient.setQueryData(["ask", ask.id, subject], current);
      if (current.state === "answered" && attempt && current.answer === attempt.answer && current.note === attempt.note) {
        setOutcome("recorded");
        setRecordedLabel(current.answerLabel || current.answer);
        clearAskDraft(subject, ask.id);
      } else if (current.state === "pending" || current.state === "escalated") {
        setOutcome("open");
      } else if (current.state === "defaulted") {
        setOutcome("defaulted");
      } else if (current.state === "dismissed") {
        setOutcome("dismissed");
        setDismissAttempt(false);
        clearAskDraft(subject, ask.id);
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
    mutationFn: (payload: AnswerPayload) => {
      const { attachments, ...answer } = payload;
      return conversationsClient.answer({
      askId: ask.id,
      ...answer,
      images: payload.images.map((image) => ({ ...image, sizeBytes: BigInt(image.sizeBytes) })),
      ...(attachments?.length ? { attachments: attachments.map((file) => ({ ...file, sizeBytes: BigInt(file.sizeBytes) })) } : {}),
    });
    },
    // A successful mutation response is not the displayed terminal state.
    // Confirm the saved answer from the recipient-scoped read path first.
    onSuccess: () => readback.mutate(),
    onError: (error) => {
      if (isAskChanged(error)) {
        // Nothing was stored: the answers were checked against a form that no
        // longer exists. Show the revised ask and keep the draft to re-check.
        setAttempt(null);
        clearAskAttempt(subject, ask.id);
        setOutcome("changed");
        void queryClient.invalidateQueries({ queryKey: ["ask", ask.id, subject] });
        return;
      }
      setOutcome("uncertain");
    },
  });
  const dismissMutation = useMutation({
    retry: false,
    mutationFn: (note: string) => conversationsClient.dismiss({ askId: ask.id, note }),
    onSuccess: () => readback.mutate(),
    onError: () => setOutcome("uncertain"),
  });
  const busy = answer.isPending || dismissMutation.isPending;

  return {
    attempt,
    outcome,
    recordedLabel,
    readError,
    /** An answer was sent or a dismissal attempted: the answer is frozen and only readback is offered. */
    settling: Boolean(attempt) || dismissAttempt,
    sending: answer.isPending,
    sent: answer.isSuccess,
    dismissing: dismissMutation.isPending,
    checking: readback.isPending,
    busy,
    submit: () => {
      if (!subject || attempt || busy || blockerFor(ask, draft)) return;
      setOutcome(null);
      const payload = buildAnswer(ask, draft);
      const saved = { ...payload, subject };
      setAttempt(saved);
      writeAskAttempt(subject, ask.id, saved);
      answer.mutate(payload);
    },
    dismiss: () => {
      if (busy) return;
      setDismissAttempt(true);
      dismissMutation.mutate(draft.note.trim());
    },
    checkStatus: () => {
      setReadError(false);
      readback.mutate();
    },
  };
}
