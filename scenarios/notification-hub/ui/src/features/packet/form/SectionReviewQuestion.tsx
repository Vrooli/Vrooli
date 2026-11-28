import { BookOpen } from "lucide-react";

import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";
import { ANSWER_LIMITS, SECTION_VERDICTS, isSectionVerdict, unansweredVerdict, type SectionVerdict } from "../model/questions";
import { resolveAttachmentRef } from "../viewer/attachments";
import { NoteDisclosure } from "./NoteDisclosure";
import { VERDICT_LABEL } from "./verdictLabels";
import { TextArea } from "./TextArea";
import type { QuestionProps } from "./QuestionField";

const VERDICT_STYLE: Record<SectionVerdict, string> = {
  approve: "peer-checked:border-app-success peer-checked:bg-app-success/15",
  change: "peer-checked:border-app-warning peer-checked:bg-app-warning/15",
  drop: "peer-checked:border-app-danger peer-checked:bg-app-danger/10",
};

/**
 * Approve, change or drop each section of a document, with a note. A section
 * the operator leaves alone takes the question's default verdict, which is the
 * one shown selected, so what they see is what is sent.
 */
export function SectionReviewQuestion({ ask, question, value, disabled, onChange, onOpenAttachment }: QuestionProps) {
  const { t } = useTranslation();
  const fallback = unansweredVerdict(question);
  const verdictLabel = (verdict: SectionVerdict) => t(VERDICT_LABEL[verdict]);
  return (
    <div className="flex flex-col gap-3">
      <p className="text-sm text-app-muted-foreground">{t(strings.packet.form.reviewHint, { verdict: verdictLabel(fallback) })}</p>
      {question.sections.map((section) => {
        const entry = value.sections[section.key];
        const verdict = entry && isSectionVerdict(entry.verdict) ? entry.verdict : fallback;
        const note = entry?.note ?? "";
        const file = section.attachment ? resolveAttachmentRef(ask.attachments, section.attachment) : null;
        const set = (next: Partial<{ verdict: string; note: string }>) =>
          onChange({ ...value, sections: { ...value.sections, [section.key]: { verdict, note, ...next } } });
        return (
          <div key={section.key} data-section-key={section.key} className="flex flex-col gap-2 rounded-panel border border-app-border bg-app-surface p-3">
            <div className="flex flex-col gap-1">
              <h4 className="break-words font-semibold leading-snug">{section.title}</h4>
              {section.summary && <p className="break-words text-sm text-app-muted-foreground">{section.summary}</p>}
              {file && (
                <button type="button" onClick={() => onOpenAttachment(section.anchor ? { ...file.target, anchor: section.anchor } : file.target)} className="inline-flex min-h-11 items-center gap-1.5 self-start text-sm font-medium underline underline-offset-4">
                  <BookOpen size={16} aria-hidden="true" />
                  {t(strings.packet.form.readSection)}
                </button>
              )}
            </div>
            <div role="radiogroup" aria-label={section.title} className="grid grid-cols-3 gap-2">
              {SECTION_VERDICTS.map((choice) => (
                <label key={choice} className="cursor-pointer">
                  <input
                    type="radio"
                    className="peer sr-only"
                    name={`review-${question.id}-${section.key}`}
                    value={choice}
                    checked={verdict === choice}
                    disabled={disabled}
                    onChange={() => set({ verdict: choice })}
                  />
                  <span
                    className={`flex min-h-11 items-center justify-center rounded-control border border-app-border px-2 text-sm font-medium peer-focus-visible:outline peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-app-focus ${VERDICT_STYLE[choice]}`}
                  >
                    {verdictLabel(choice)}
                  </span>
                </label>
              ))}
            </div>
            <NoteDisclosure label={t(strings.packet.form.questionNote)} open={verdict !== "approve" || Boolean(note)}>
              <TextArea
                label={t(strings.packet.form.sectionNoteFor, { title: section.title })}
                hideLabel
                rows={2}
                disabled={disabled}
                maxLength={ANSWER_LIMITS.responseTextChars}
                value={note}
                onChange={(text) => set({ note: text })}
              />
            </NoteDisclosure>
          </div>
        );
      })}
    </div>
  );
}
