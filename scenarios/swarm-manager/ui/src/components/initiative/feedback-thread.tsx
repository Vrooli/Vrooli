/**
 * FeedbackThread — chronological render of a feedback round's thread.
 *
 * Each Message is a user or agent turn; attachments render as thumbnails
 * that link to the server-hosted file. The thread is read-only; actions
 * (revise, accept, reject, dismiss) are owned by ProposalReview and the
 * decision buttons in FeedbackPanel.
 */

import { memo } from "react";
import { Bot, Sparkles, User } from "lucide-react";
import { feedbackService } from "../../services/feedback-service";
import { cn } from "../../lib/utils";
import { selectors } from "../../consts/selectors";
import { formatRelativeTime } from "../../lib";
import { renderMarkdown } from "../../lib/render-markdown";
import type { FeedbackMessage, FeedbackRound } from "../../types";

export interface FeedbackThreadProps {
  round: FeedbackRound;
  className?: string;
}

export const FeedbackThread = memo(function FeedbackThread({ round, className }: FeedbackThreadProps) {
  const messages = round.thread ?? [];
  if (messages.length === 0) {
    return (
      <p className="text-xs italic text-slate-500">No turns in this round yet.</p>
    );
  }
  return (
    <ol className={cn("space-y-3", className)}>
      {messages.map((message, idx) => (
        <li key={idx}>
          <ThreadMessage
            message={message}
            round={round}
            index={idx}
          />
        </li>
      ))}
    </ol>
  );
});

interface ThreadMessageProps {
  message: FeedbackMessage;
  round: FeedbackRound;
  index: number;
}

function ThreadMessage({ message, round, index }: ThreadMessageProps) {
  const isUser = message.role === "user";
  const carriesProposal = message.role === "agent" && !!message.proposal_id;
  const Icon = isUser ? User : carriesProposal ? Sparkles : Bot;
  return (
    <div
      className={cn(
        "flex gap-3 rounded-xl border p-3 text-sm",
        isUser
          ? "border-slate-700/70 bg-slate-900/60"
          : carriesProposal
            ? "border-cyan-500/30 bg-cyan-500/5"
            : "border-slate-700/60 bg-slate-900/40",
      )}
      data-testid={selectors.feedback.threadMessage}
      data-role={message.role}
      data-message-index={index}
    >
      <Icon className={cn("h-4 w-4 shrink-0", isUser ? "text-slate-400" : carriesProposal ? "text-cyan-300" : "text-slate-500")} />
      <div className="min-w-0 flex-1">
        <div className="mb-1 flex items-center justify-between gap-3 text-[11px] text-slate-500">
          <span className="uppercase tracking-wider">
            {isUser ? "You" : carriesProposal ? "Agent · proposal" : "Agent"}
          </span>
          <span>{formatRelativeTime(message.created_at)}</span>
        </div>
        <div
          className="prose-sm-slate break-words text-slate-200"
          dangerouslySetInnerHTML={{ __html: renderMarkdown(message.content) }}
        />
        {message.attachment_ids && message.attachment_ids.length > 0 && (
          <AttachmentList
            initiativeName={round.initiative_name}
            roundNumber={round.number}
            ids={message.attachment_ids}
          />
        )}
      </div>
    </div>
  );
}

function AttachmentList({
  initiativeName,
  roundNumber,
  ids,
}: {
  initiativeName: string;
  roundNumber: number;
  ids: string[];
}) {
  return (
    <ul className="mt-2 flex flex-wrap gap-2">
      {ids.map((id) => {
        const href = feedbackService.attachmentUrl(initiativeName, roundNumber, id);
        const isImage = /\.(png|jpe?g|gif|webp)$/i.test(id);
        if (isImage) {
          return (
            <li key={id}>
              <a
                href={href}
                target="_blank"
                rel="noopener noreferrer"
                className="block overflow-hidden rounded-lg border border-slate-700 hover:border-slate-500"
              >
                <img
                  src={href}
                  alt={id}
                  className="h-16 w-16 object-cover"
                  loading="lazy"
                />
              </a>
            </li>
          );
        }
        return (
          <li key={id}>
            <a
              href={href}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center rounded-md border border-slate-700 bg-slate-800 px-2 py-0.5 text-[11px] text-slate-300 hover:border-slate-500 hover:text-slate-100"
            >
              {id}
            </a>
          </li>
        );
      })}
    </ul>
  );
}
