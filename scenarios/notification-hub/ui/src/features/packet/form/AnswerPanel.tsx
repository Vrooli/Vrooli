import { useEffect, useRef, useState } from "react";
import { Button } from "@vrooli/react-component-library/Button/2";
import type { Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";
import { ChevronLeft, Sparkles } from "lucide-react";

import { selectors } from "../../../consts/selectors";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";
import { blockerFor, hasQuestionRecommendations, missingRequired, withRecommendations } from "../model/answer";
import { emptyResponse, withResponse, type PacketDraft } from "../model/draft";
import type { AttachmentTarget } from "../viewer/attachments";
import { ActionBar } from "./ActionBar";
import { SecondaryActions } from "./SecondaryActions";
import { MainOptions } from "./MainOptions";
import { PinnedNotes } from "./PinnedNotes";
import { QuestionField } from "./QuestionField";
import { ReviewSummary } from "./ReviewSummary";
import { TextArea } from "./TextArea";
import { AnswerAttachments } from "./AnswerAttachments";

export interface AnswerPanelProps {
  ask: Ask;
  draft: PacketDraft;
  onDraftChange: (update: PacketDraft | ((current: PacketDraft) => PacketDraft)) => void;
  /** Inputs are disabled: an answer was sent, or a send is in flight. */
  locked: boolean;
  /** An answer was sent: the form stays readable as the record of what was attempted, with no actions. */
  frozen: boolean;
  /** Show one step at a time (several questions on a phone). */
  paged: boolean;
  /** Leave the actions to the caller (the focus review owns one bar). */
  hideActions?: boolean;
  /** The decision is open: it can be dismissed or sent back for more context. */
  open: boolean;
  dismissing: boolean;
  onSubmit: () => void;
  onDismiss: () => void;
  onRequestContext: () => void;
  onOpenAttachment: (target: AttachmentTarget) => void;
  onBackToCover?: () => void;
}

type Step = "answer" | { question: string } | "review";

/**
 * The Hub's own answer form: the decision's main options, then the typed
 * questions, a place for notes, and the actions. On a wide screen it is one
 * column beside the cover. With several questions on a phone it is a short
 * guided flow, one question per screen, ending in a review of what will be
 * sent. Only this panel's Send button answers.
 */
export function AnswerPanel(props: AnswerPanelProps) {
  const { ask, draft, onDraftChange, locked, frozen, paged, hideActions = false, open, dismissing, onSubmit, onDismiss, onRequestContext, onOpenAttachment, onBackToCover } = props;
  const { t } = useTranslation();
  const [stepIndex, setStepIndex] = useState(0);
  const [imageUploading, setImageUploading] = useState(false);
  const stepRoot = useRef<HTMLDivElement>(null);
  const firstRender = useRef(true);
  const steps: Step[] = ["answer", ...ask.questions.map((question) => ({ question: question.id })), "review"];
  const index = Math.min(stepIndex, steps.length - 1);
  // Moving to another step moves keyboard and screen-reader focus with it.
  useEffect(() => {
    if (firstRender.current) firstRender.current = false;
    else stepRoot.current?.focus({ preventScroll: false });
  }, [index]);
  const blocker = blockerFor(ask, draft);
  const firstMissing = missingRequired(ask, draft)[0];
  const blockedHint =
    blocker === "no-answer" ? t(strings.packet.form.blockedNoAnswer)
    : blocker === "other-needs-note" ? t(strings.packet.form.blockedOther)
    : blocker === "required-question" && firstMissing ? t(strings.packet.form.blockedRequired, { prompt: firstMissing.prompt })
    : undefined;

  const accept = () => {
    onDraftChange(withRecommendations(ask, draft));
    if (paged) setStepIndex(steps.length - 1);
  };
  const goTo = (target: "answer" | { question: string }) => {
    setStepIndex(steps.findIndex((step) => (typeof step === "string" ? step === target : typeof target !== "string" && step.question === target.question)));
  };

  const acceptButton = hasQuestionRecommendations(ask) && !locked && (
    <Button data-testid={selectors.packet.acceptRecommended} type="button" variant="outline" className="min-h-11 self-start border border-app-border bg-app-surface" onClick={accept}>
      <Sparkles size={16} aria-hidden="true" className="me-1.5" />
      {t(strings.packet.form.acceptRecommended)}
    </Button>
  );

  const mainOptions = <MainOptions ask={ask} draft={draft} disabled={locked} onChange={onDraftChange} />;
  const questions = ask.questions.map((question) => (
    <QuestionField
      key={question.id}
      ask={ask}
      question={question}
      value={draft.responses[question.id] ?? emptyResponse()}
      disabled={locked}
      onChange={(next) => onDraftChange(withResponse(draft, question.id, next))}
      onOpenAttachment={onOpenAttachment}
    />
  ));
  const pinned = <PinnedNotes ask={ask} draft={draft} disabled={locked} onChange={onDraftChange} />;
  const answerImages = <AnswerAttachments images={draft.answerImages} attachments={draft.answerAttachments} disabled={locked || frozen} onBusyChange={setImageUploading} onChange={onDraftChange} />;

  const send = { label: t(strings.packet.form.submit), onClick: onSubmit, disabled: blocker !== null || imageUploading, testId: selectors.packet.send };
  const secondary = open && !frozen && <SecondaryActions canRequestContext={ask.canRequestContext} onRequestContext={onRequestContext} onDismiss={onDismiss} dismissing={dismissing} disabled={locked} />;

  // The body and the pinned bar are siblings: the bar sticks within the tall
  // page content, so it cannot be nested inside the short form.
  if (!paged) {
    return (
      <>
        <div data-testid={selectors.packet.form} className="flex flex-col gap-5">
          {acceptButton}
          {mainOptions}
          {questions.length > 0 && <div className="flex flex-col gap-6 border-t border-app-border pt-5">{questions}</div>}
          {pinned}
          {answerImages}
          {secondary}
        </div>
        {!frozen && !hideActions && <ActionBar primary={send} hint={blockedHint} disabled={locked || imageUploading} />}
      </>
    );
  }

  const current = steps[index] ?? "answer";
  const last = index === steps.length - 1;
  // A question step shows its own prompt as the card heading.
  const title = current === "answer" ? t(strings.packet.form.stepDecision) : current === "review" ? t(strings.packet.form.stepReview) : "";
  const overallNote = draft.answer === "other" ? null : (
    <TextArea label={t(strings.packet.form.noteLabel)} testId={selectors.asks.note} value={draft.note} disabled={locked} onChange={(note) => onDraftChange({ ...draft, note })} />
  );

  return (
    <>
    <div ref={stepRoot} tabIndex={-1} data-testid={selectors.packet.form} className="flex flex-col gap-5 outline-none">
      <header className="flex flex-col gap-3">
        <div className="flex items-center justify-between gap-3">
          <button type="button" onClick={onBackToCover} className="inline-flex min-h-11 items-center gap-1 text-sm font-medium">
            <ChevronLeft size={18} aria-hidden="true" className="rtl:rotate-180" />
            {t(strings.packet.form.backToCover)}
          </button>
          <span className="text-sm text-app-muted-foreground">{t(strings.packet.form.stepOf, { current: index + 1, total: steps.length })}</span>
        </div>
        <div
          role="progressbar"
          aria-label={t(strings.packet.form.progress)}
          aria-valuemin={1}
          aria-valuemax={steps.length}
          aria-valuenow={index + 1}
          aria-valuetext={t(strings.packet.form.stepOf, { current: index + 1, total: steps.length })}
          className="flex gap-1"
        >
          {steps.map((step, position) => (
            <span key={typeof step === "string" ? step : step.question} className={`h-1.5 flex-1 rounded-pill ${position <= index ? "bg-app-primary" : "bg-app-surface-muted ring-1 ring-inset ring-app-border"}`} />
          ))}
        </div>
        {title && <h2 className="break-words text-lg font-semibold leading-snug">{title}</h2>}
      </header>
      {current === "answer" && (
        <>
          {acceptButton}
          {mainOptions}
        </>
      )}
      {typeof current === "object" && questions.find((node) => node.key === current.question)}
      {current === "review" && (
        <>
          <ReviewSummary ask={ask} draft={draft} onEdit={goTo} />
          {pinned}
          {overallNote}
          {answerImages}
        </>
      )}
      {last && secondary}
    </div>
    <ActionBar
      primary={last ? send : { label: t(strings.packet.form.next), onClick: () => setStepIndex(index + 1) }}
      back={index > 0 ? { label: t(strings.packet.form.previous), onClick: () => setStepIndex(index - 1) } : undefined}
      hint={last ? blockedHint : undefined}
      disabled={locked || imageUploading}
    />
    </>
  );
}
