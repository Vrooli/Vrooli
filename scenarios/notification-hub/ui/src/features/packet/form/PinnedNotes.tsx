import { Pin, Trash2 } from "lucide-react";
import type { Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";
import { ANSWER_LIMITS } from "../model/questions";
import { withAttachmentNote, type PacketDraft } from "../model/draft";
import { attachmentLabel, targetPath } from "../viewer/attachments";
import { TextArea } from "./TextArea";

/** The notes the operator pinned to files while viewing them; each can be edited or removed before sending. */
export function PinnedNotes({ ask, draft, disabled, onChange }: { ask: Ask; draft: PacketDraft; disabled: boolean; onChange: (next: PacketDraft) => void }) {
  const { t } = useTranslation();
  if (draft.attachmentNotes.length === 0) return null;
  return (
    <section aria-labelledby="pinned-notes-heading" className="flex flex-col gap-2">
      <h3 id="pinned-notes-heading" className="text-sm font-semibold">{t(strings.packet.form.pinnedNotes)}</h3>
      {draft.attachmentNotes.map((item) => {
        const attachment = ask.attachments.find((candidate) => candidate.name === item.attachment);
        const name = attachment ? (item.path ? targetPath({ name: item.attachment, path: item.path }) : attachmentLabel(attachment)) : item.attachment;
        return (
          <div key={`${item.attachment}\u0000${item.path}`} className="flex flex-col gap-1 rounded-panel border border-app-border bg-app-surface p-3">
            <div className="flex items-center justify-between gap-2">
              <span className="inline-flex min-w-0 items-center gap-1.5 text-sm font-medium">
                <Pin size={14} aria-hidden="true" className="shrink-0" />
                <span className="break-all">{name}</span>
              </span>
              <button
                type="button"
                disabled={disabled}
                onClick={() => onChange(withAttachmentNote(draft, item.attachment, item.path, ""))}
                aria-label={t(strings.packet.form.removeNote, { name })}
                className="touch-target inline-flex items-center justify-center rounded-control text-app-muted-foreground hover:text-app-foreground disabled:opacity-60"
              >
                <Trash2 size={18} aria-hidden="true" />
              </button>
            </div>
            <TextArea
              label={t(strings.packet.form.noteOn, { name })}
              hideLabel
              rows={2}
              disabled={disabled}
              maxLength={ANSWER_LIMITS.attachmentNoteChars}
              value={item.note}
              onChange={(note) => onChange(withAttachmentNote(draft, item.attachment, item.path, note))}
            />
          </div>
        );
      })}
    </section>
  );
}
