import type { ReactNode } from "react";
import type { Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";
import { History } from "lucide-react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { safeContextUrl, senderLabel, urgencyLabel } from "../asks/askView";
import { useNow } from "../../hooks/useNow";
import { formatDueIn, formatExact } from "../../lib/ago";
import { Chip } from "./Chip";
import { DecisionText } from "./DecisionText";
import { isLocalFileReference, type TextLinks } from "./decisionLinks";
import { FollowthroughBanner, PacketEvidence } from "./PacketEvidence";
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
  navigation?: ReactNode;
  /** The way back, drawn at the start of the title row. */
  back?: ReactNode;
}

/**
 * The cover of a decision packet: what is being asked, why it matters, what the
 * requester recommends. A cold reader should have the whole decision from this
 * alone; technical details stay one tap away.
 */
export function PacketCover({ ask, links, onOpenAttachment, outcome, afterQuestion, navigation, back }: PacketCoverProps) {
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
      <div className="flex flex-col gap-2">
        {/* One title row, as in the approved layout: back, the name with one quiet line under it, history, and ‹ › on desktop. */}
        <div className="flex items-start gap-1 md:gap-2">
          {back}
          <div className="min-w-0 flex-1 pt-1.5">
            <h1 id="ask-heading" dir="auto" data-testid={ask.title && ask.title !== ask.question ? undefined : selectors.asks.question} className="break-words text-lg font-semibold leading-snug md:text-2xl">
              <DecisionText value={ask.title || ask.question} inheritText dense links={links} />
            </h1>
            <p className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-sm text-app-muted-foreground">
              <span>{ask.kind === "status" ? t(strings.packet.cover.statusUpdate) : t(strings.packet.cover.decision)}</span>
              {sender && <><span aria-hidden="true">·</span><span className="[overflow-wrap:anywhere]">{sender}</span></>}
              {urgency && <Chip tone="attention">{t(urgency)}</Chip>}
              {ask.deadline && (
                <Chip>
                  {t(strings.packet.cover.due)} <time dateTime={ask.deadline} title={formatExact(ask.deadline) ?? undefined}>{formatDueIn(ask.deadline, now) ?? ask.deadline}</time>
                </Chip>
              )}
            </p>
          </div>
          <details className="relative flex-none">
            <summary aria-label={t(strings.packet.cover.history)} title={t(strings.packet.cover.history)} className="flex min-h-11 min-w-11 cursor-pointer list-none items-center justify-center rounded-control text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground"><History size={19} aria-hidden="true" /></summary>
            <div className="absolute end-0 z-10 w-72 rounded-panel border border-app-border bg-app-surface p-4 shadow-lg">
              <h2 className="font-semibold">{t(strings.packet.cover.history)}</h2>
              {ask.recurrence?.dates.length ? <ul className="mt-2 space-y-1 text-sm text-app-muted-foreground">{ask.recurrence.dates.map((date, index) => <li key={`${date}-${index}`}>{date}</li>)}</ul> : <p className="mt-2 text-sm text-app-muted-foreground">{t(strings.packet.cover.historyEmpty)}</p>}
              <p className="mt-3 border-t border-app-border pt-2 text-xs text-app-muted-foreground">{t(strings.packet.status.requestId)} <bdi dir="ltr" className="select-all break-all font-mono">{ask.id}</bdi></p>
            </div>
          </details>
          <div className="hidden flex-none lg:block">{navigation}</div>
        </div>
        {ask.title && ask.title !== ask.question && <p dir="auto" data-testid={selectors.asks.question} className="line-clamp-2 break-words text-sm leading-relaxed text-app-muted-foreground md:text-base"><DecisionText value={ask.question} inheritText dense links={links} /></p>}
        {outcome}
        <FollowthroughBanner ask={ask} />
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
      <PacketEvidence ask={ask} />
      {ask.technicalDetails && (
        <details className="rounded-panel border border-app-border bg-app-surface px-3 open:pb-3">
          <summary className="min-h-11 cursor-pointer py-2.5 font-medium">{t(strings.packet.cover.technicalDetails)}</summary>
          <DecisionText value={ask.technicalDetails} links={links} className="block text-app-muted-foreground" />
        </details>
      )}
    </>
  );
}
