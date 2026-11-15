import type { ReactNode } from "react";
import { Button } from "@vrooli/react-component-library/Button/2";
import type { UseQueryResult } from "@tanstack/react-query";
import type { Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { safeContextUrl, senderLabel, urgencyLabel } from "../asks/askView";
import { useNow } from "../../hooks/useNow";
import { formatDueIn, formatExact } from "../../lib/ago";
import { Chip } from "./Chip";
import { DecisionText } from "./DecisionText";
import { fileReferenceHref, hasLocalFileReference, type TextLinks } from "./decisionLinks";
import { Recommendation } from "./Recommendation";
import { findAttachmentByPath, type AttachmentTarget } from "./viewer/attachments";

interface PacketCoverProps {
  ask: Ask;
  links: TextLinks;
  webConsole: { needed: boolean; query: UseQueryResult<string> };
  onOpenAttachment: (target: AttachmentTarget) => void;
  /** How a closed decision ended, shown straight under the question so the outcome is the first thing read. */
  outcome?: ReactNode;
  /** What belongs right under the question: the requester's page, and any open questions to it. */
  afterQuestion?: ReactNode;
}

/**
 * The cover of a decision packet: what is being asked, why it matters, what the
 * requester recommends. A cold reader should have the whole decision from this
 * alone; technical details stay one tap away.
 */
export function PacketCover({ ask, links, webConsole, onOpenAttachment, outcome, afterQuestion }: PacketCoverProps) {
  const { t } = useTranslation();
  const now = useNow();
  const { query: consoleUrl } = webConsole;
  const contextRef = ask.contextUrl;
  const attachedContext = contextRef ? findAttachmentByPath(ask.attachments, contextRef) : null;
  const contextPath = contextRef && !attachedContext && hasLocalFileReference(contextRef) ? contextRef : "";
  let contextHref: string | null = null;
  if (contextPath) contextHref = fileReferenceHref(contextPath, consoleUrl.data);
  else if (contextRef && !attachedContext) contextHref = safeContextUrl(contextRef);
  const urgency = urgencyLabel(ask.urgency);
  const sender = senderLabel(ask);

  return (
    <>
      <div className="flex flex-col gap-2 md:gap-3">
        {(sender || urgency || ask.deadline) && (
          <p className="flex flex-wrap items-center gap-2 text-sm text-app-muted-foreground">
            {sender && <span className="[overflow-wrap:anywhere]">{sender}</span>}
            {urgency && <Chip tone="attention">{t(urgency)}</Chip>}
            {ask.deadline && (
              <Chip>
                {t(strings.packet.cover.due)} <time dateTime={ask.deadline} title={formatExact(ask.deadline) ?? undefined}>{formatDueIn(ask.deadline, now) ?? ask.deadline}</time>
              </Chip>
            )}
          </p>
        )}
        <h1 id="ask-heading" dir="auto" data-testid={selectors.asks.question} className="break-words text-xl font-semibold leading-snug md:text-2xl">
          <DecisionText value={ask.question} inheritText dense links={links} />
        </h1>
        {outcome}
        {attachedContext && (
          <button type="button" onClick={() => onOpenAttachment(attachedContext)} className="inline-flex min-h-11 items-center self-start text-sm underline">
            {t(strings.packet.cover.more)}
          </button>
        )}
        {contextHref && (
          <a href={contextHref} className="inline-flex min-h-11 items-center self-start text-sm underline" rel="noopener noreferrer" target={contextPath ? "_blank" : undefined}>
            {t(strings.packet.cover.more)}
          </a>
        )}
      </div>
      {webConsole.needed && !consoleUrl.data && (
        <div className="flex flex-wrap items-center gap-3 text-sm text-app-muted-foreground">
          <p role="status">{consoleUrl.isPending ? t(strings.packet.text.fileLinksLoading) : t(strings.packet.text.fileLinksUnavailable)}</p>
          {consoleUrl.isError && (
            <Button data-testid={selectors.asks.retryFileLinks} className="min-h-11" type="button" variant="outline" disabled={consoleUrl.isFetching} onClick={() => void consoleUrl.refetch()}>
              {consoleUrl.isFetching ? t(strings.packet.text.fileLinksLoading) : t(strings.packet.text.retryFileLinks)}
            </Button>
          )}
        </div>
      )}
      {afterQuestion}
      {ask.bigPicture && (
        <details open className="rounded-panel border border-app-border bg-app-surface px-3 open:pb-3">
          <summary className="min-h-11 cursor-pointer py-2.5 font-medium">{t(strings.packet.cover.bigPicture)}</summary>
          <DecisionText value={ask.bigPicture} links={links} className="block text-base leading-relaxed text-app-muted-foreground" />
        </details>
      )}
      <Recommendation ask={ask} links={links} />
      {ask.technicalDetails && (
        <details className="rounded-panel border border-app-border bg-app-surface px-3 open:pb-3">
          <summary className="min-h-11 cursor-pointer py-2.5 font-medium">{t(strings.packet.cover.technicalDetails)}</summary>
          <DecisionText value={ask.technicalDetails} links={links} className="block text-app-muted-foreground" />
        </details>
      )}
    </>
  );
}
