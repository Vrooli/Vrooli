import { strings } from "../../../consts/strings";
import { attachmentUrl } from "../../../api/attachments";
import { useTranslation } from "../../../i18n";
import { AttachmentIcon } from "../viewer/AttachmentIcon";
import { Chip } from "../Chip";
import { isInlineImage, resolveAttachmentRef, targetPath } from "../viewer/attachments";
import type { QuestionProps } from "./QuestionField";

/**
 * Choices side by side, each showing its attachment: an image inline, any
 * other file as a tile that opens the viewer. Tapping the picture looks at it;
 * tapping the name chooses it, so looking never answers by accident.
 */
export function VariantQuestion({ ask, question, value, disabled, onChange, onOpenAttachment, labelledBy }: QuestionProps & { labelledBy: string }) {
  const { t } = useTranslation();
  const recommended = question.recommended?.selected ?? [];
  return (
    <div role="radiogroup" aria-labelledby={labelledBy} className="grid grid-cols-2 gap-3 sm:grid-cols-[repeat(auto-fill,minmax(12rem,1fr))]">
      {question.choices.map((choice) => {
        const file = choice.attachment ? resolveAttachmentRef(ask.attachments, choice.attachment) : null;
        const open = file ? () => onOpenAttachment(file.target) : undefined;
        return (
          <div
            key={choice.key}
            className="flex min-w-0 flex-col overflow-hidden rounded-panel border border-app-border bg-app-surface has-[:checked]:border-app-primary has-[:checked]:ring-2 has-[:checked]:ring-app-primary has-[:focus-visible]:outline has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-2 has-[:focus-visible]:outline-app-focus"
          >
            {file && (
              <button
                type="button"
                onClick={open}
                aria-label={t(strings.packet.form.viewChoice, { label: choice.label })}
                className="flex aspect-[4/3] w-full items-center justify-center overflow-hidden bg-app-surface-muted text-app-muted-foreground"
              >
                {isInlineImage(file.kind) ? (
                  <img src={attachmentUrl(ask.id, file.target.name, file.target.path)} alt="" loading="lazy" className="h-full w-full object-cover" />
                ) : (
                  <span className="flex flex-col items-center gap-1 p-2 text-xs">
                    <AttachmentIcon kind={file.kind} size={32} />
                    <span className="break-all">{targetPath(file.target)}</span>
                  </span>
                )}
              </button>
            )}
            <label className="flex min-h-11 flex-1 cursor-pointer items-start gap-2 p-2.5">
              <input
                type="radio"
                className="peer sr-only"
                name={`question-${question.id}`}
                value={choice.key}
                checked={value.selected.includes(choice.key)}
                disabled={disabled}
                onChange={() => onChange({ ...value, selected: [choice.key] })}
              />
              <span
                aria-hidden="true"
                className="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-pill border-2 border-app-muted-foreground bg-app-surface peer-checked:border-app-primary peer-checked:bg-app-primary after:block after:h-2 after:w-2 after:scale-0 after:rounded-pill after:bg-app-primary-foreground after:transition-transform peer-checked:after:scale-100"
              />
              <span className="flex min-w-0 flex-1 flex-col gap-0.5">
                <span className="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm font-medium leading-snug">
                  <span dir="auto" className="min-w-0 break-words">{choice.label}</span>
                  {recommended.includes(choice.key) && <Chip tone="attention">{t(strings.packet.form.recommended)}</Chip>}
                </span>
                {choice.detail && <span dir="auto" className="break-words text-xs text-app-muted-foreground">{choice.detail}</span>}
              </span>
            </label>
          </div>
        );
      })}
    </div>
  );
}
