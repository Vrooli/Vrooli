import { HelpCircle, ListChecks, Paperclip } from "lucide-react";
import type { Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

/**
 * What a decision carries beyond its question, shown on its row so the
 * operator can tell a quick yes/no from a packet to read before opening it:
 * attachment and question counts, and whether the operator already asked the
 * requester for more context and is waiting.
 */
export function RequestPacketMeta({ ask }: { ask: Pick<Ask, "attachmentCount" | "questionCount" | "openContextRequestCount"> }) {
  const { t } = useTranslation();
  return (
    <>
      {ask.attachmentCount > 0 && (
        <span className="inline-flex items-center gap-1 text-sm text-app-muted-foreground">
          <Paperclip size={14} aria-hidden="true" />
          <span aria-hidden="true">{ask.attachmentCount}</span>
          <span className="sr-only">{t(strings.pages.history.attachments, { count: ask.attachmentCount })}</span>
        </span>
      )}
      {ask.questionCount > 0 && (
        <span className="inline-flex items-center gap-1 text-sm text-app-muted-foreground">
          <ListChecks size={14} aria-hidden="true" />
          <span aria-hidden="true">{ask.questionCount}</span>
          <span className="sr-only">{t(strings.pages.history.questions, { count: ask.questionCount })}</span>
        </span>
      )}
      {ask.openContextRequestCount > 0 && (
        <span className="inline-flex items-center gap-1 rounded-pill bg-app-warning/15 px-2 py-0.5 text-sm font-medium">
          <HelpCircle size={14} aria-hidden="true" />
          {t(strings.pages.history.askedContext)}
        </span>
      )}
    </>
  );
}
