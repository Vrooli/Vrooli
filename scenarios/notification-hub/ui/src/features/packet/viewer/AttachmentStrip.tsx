import { Pin } from "lucide-react";
import type { AskAttachment } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { attachmentUrl } from "../../../api/attachments";
import { selectors } from "../../../consts/selectors";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";
import { Chip } from "../Chip";
import { AttachmentIcon } from "./AttachmentIcon";
import { attachmentLabel, evidenceAttachments, isFolder, isInlineImage, previewKindOf, type AttachmentTarget } from "./attachments";

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(bytes < 10 * 1024 * 1024 ? 1 : 0)} MB`;
}

/**
 * The files attached to the decision, as tiles. An image shows itself; the
 * rest show what kind of file they are. Each opens inside the Hub, from the
 * Hub's own stored copy, so the operator sees exactly what the agent saw.
 */
export function AttachmentStrip({
  askId,
  attachments,
  notedAttachments,
  onOpen,
}: {
  askId: string;
  attachments: readonly AskAttachment[];
  /** Names of attachments the operator has pinned a note to. */
  notedAttachments: ReadonlySet<string>;
  onOpen: (target: AttachmentTarget) => void;
}) {
  const { t } = useTranslation();
  const evidence = evidenceAttachments(attachments);
  if (evidence.length === 0) return null;
  return (
    <section aria-labelledby="attachments-heading" data-testid={selectors.packet.attachments} className="flex flex-col gap-2">
      <h2 id="attachments-heading" className="text-sm font-semibold">
        {t(strings.packet.attachments.title, { count: evidence.length })}
      </h2>
      <ul className="grid grid-cols-3 gap-2 sm:grid-cols-4 lg:grid-cols-5">
        {evidence.map((attachment) => {
          const folder = isFolder(attachment);
          const kind = folder ? "directory" : previewKindOf(attachment.previewKind, attachment.mediaType);
          const label = attachmentLabel(attachment);
          const status = attachment.sourceStatus;
          const size = folder ? t(strings.packet.attachments.files, { count: attachment.entries.length }) : formatSize(Number(attachment.sizeBytes));
          // The tile's chip is not read out with its name, so the status joins the detail.
          const detail = status === "changed" ? `${size}, ${t(strings.packet.attachments.changedLong)}` : status === "missing" ? `${size}, ${t(strings.packet.attachments.missingLong)}` : size;
          return (
            <li key={attachment.name} className="min-w-0">
              <button
                type="button"
                data-testid={selectors.packet.attachment}
                data-attachment-name={attachment.name}
                onClick={() => onOpen({ name: attachment.name, path: "" })}
                aria-label={t(strings.packet.attachments.open, { label, detail })}
                className="group relative flex aspect-square w-full flex-col overflow-hidden rounded-panel border border-app-border bg-app-surface text-start transition-colors hover:border-app-primary focus-visible:outline-offset-2"
              >
                <span className="flex min-h-0 flex-1 items-center justify-center overflow-hidden bg-app-surface-muted text-app-muted-foreground">
                  {!folder && isInlineImage(kind) ? (
                    <img src={attachmentUrl(askId, attachment.name)} alt="" loading="lazy" className="h-full w-full object-cover" />
                  ) : (
                    <AttachmentIcon kind={kind} folder={folder} size={32} strokeWidth={1.5} />
                  )}
                </span>
                <span dir="ltr" className="block truncate border-t border-app-border px-1.5 py-1 text-start font-mono text-[11px] leading-4" aria-hidden="true">
                  {attachment.name}{folder ? "/" : ""}
                </span>
                {(status === "changed" || status === "missing") && (
                  <span className="absolute start-1 top-1">
                    <Chip tone={status === "changed" ? "attention" : "neutral"}>{t(status === "changed" ? strings.packet.attachments.changed : strings.packet.attachments.missing)}</Chip>
                  </span>
                )}
                {notedAttachments.has(attachment.name) && (
                  <span className="absolute end-1 top-1 inline-flex h-6 w-6 items-center justify-center rounded-pill bg-app-primary text-app-primary-foreground" role="img" aria-label={t(strings.packet.attachments.hasNote)}>
                    <Pin size={12} aria-hidden="true" />
                  </span>
                )}
              </button>
            </li>
          );
        })}
      </ul>
      {evidence.some((attachment) => attachment.truncated) && <p className="text-xs text-app-muted-foreground">{t(strings.packet.attachments.truncated)}</p>}
    </section>
  );
}
