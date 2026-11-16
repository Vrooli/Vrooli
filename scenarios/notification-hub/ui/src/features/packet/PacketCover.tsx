import type { ReactNode } from "react";
import type { Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { safeContextUrl, senderLabel, urgencyLabel } from "../asks/askView";
import { useNow } from "../../hooks/useNow";
import { formatDueIn, formatExact } from "../../lib/ago";
import { Chip } from "./Chip";
import { DecisionText } from "./DecisionText";
import { isLocalFileReference, type TextLinks } from "./decisionLinks";
import { Recommendation } from "./Recommendation";
import { findAttachmentByPath, type AttachmentTarget } from "./viewer/attachments";

interface PacketCoverProps {
  ask: Ask;
  links: TextLinks;
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
export function PacketCover({ ask, links, onOpenAttachment, outcome, afterQuestion }: PacketCoverProps) {
  const { t } = useTranslation();
  const now = useNow();
  const contextRef = ask.contextUrl;
  const attachedContext = contextRef ? findAttachmentByPath(ask.attachments, contextRef) : null;
  const linkedContext = contextRef && !attachedContext && isLocalFileReference(contextRef) ? contextRef : "";
  const contextHref = contextRef && !attachedContext ? safeContextUrl(contextRef) : null;
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
        {linkedContext && links.onOpenLinkedArtifact && (
          <button type="button" onClick={() => links.onOpenLinkedArtifact?.(linkedContext)} className="inline-flex min-h-11 items-center self-start text-sm underline">
            {t(strings.packet.cover.more)}
          </button>
        )}
        {contextHref && (
          <a href={contextHref} className="inline-flex min-h-11 items-center self-start text-sm underline" rel="noopener noreferrer" target="_blank">
            {t(strings.packet.cover.more)}
          </a>
        )}
      </div>
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
