/**
 * @libraryId react-component-library:Message
 * @displayName Message
 * @description A structured conversational item that keeps identity, content, evidence, activity, recovery, and actions in one readable surface.
 * @version 1.0.9
 * @tags ["ai","conversation","content","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource ai.message */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import { useId, type CSSProperties, type ReactNode } from "react";
import {
  Avatar,
  type AvatarPresence,
} from "@vrooli/react-component-library/Avatar/1";
import { RelativeTime } from "@vrooli/react-component-library/RelativeTime/1";
import {
  StatusIndicator,
  type StatusCertainty,
  type StatusUrgency,
} from "@vrooli/react-component-library/StatusIndicator/1";
import { Text } from "@vrooli/react-component-library/Text/1";

export type MessageState =
  | "default"
  | "loading"
  | "partial"
  | "success"
  | "request-error"
  | "retry";

export interface MessageActor {
  name: string;
  role?: string;
  src?: string;
  presence?: AvatarPresence;
}

export interface MessageAttachment {
  id: string;
  name: string;
  detail?: string;
  href?: string;
  status?: "ready" | "processing" | "error";
}

export interface MessageCitation {
  id: string;
  label: string;
  href?: string;
  source?: string;
  excerpt?: string;
}

export interface MessageActivity {
  label: string;
  detail?: string;
  status?: "idle" | "pending" | "success" | "error" | "offline";
  certainty?: StatusCertainty;
  urgency?: StatusUrgency;
}

export interface MessageAction {
  id: string;
  label: string;
  onClick?: () => void;
  disabled?: boolean;
  primary?: boolean;
}

export interface MessageProps {
  actor: MessageActor;
  content?: ReactNode;
  state?: MessageState;
  timestamp?: string;
  certainty?: StatusCertainty;
  urgency?: StatusUrgency;
  attachments?: MessageAttachment[];
  citations?: MessageCitation[];
  activity?: MessageActivity;
  actions?: MessageAction[];
  footer?: ReactNode;
  errorMessage?: ReactNode;
  onRetry?: () => void;
  retryLabel?: string;
  className?: string;
  style?: CSSProperties;
}

const styles = `
[data-rcl-message] { --rcl-message-accent: var(--color-primary, #2563eb); display: grid; gap: var(--space-md, 24px); min-inline-size: 0; padding: var(--space-lg, 32px); border: var(--border-hairline, 1px) solid var(--color-border, #cbd5e1); border-radius: var(--radius-panel, 0.5rem); background: var(--color-surface-raised, #ffffff); color: var(--color-foreground, #0f172a); box-shadow: var(--elev-raised, 0 1px 2px rgba(9, 18, 22, .06), 0 1px 3px rgba(9, 18, 22, .10)); }
[data-rcl-message][data-state="partial"], [data-rcl-message][data-state="loading"] { border-color: color-mix(in srgb, var(--rcl-message-accent) 36%, var(--color-border, #cbd5e1)); }
[data-rcl-message][data-state="request-error"], [data-rcl-message][data-state="retry"] { border-color: color-mix(in srgb, var(--color-danger, #dc2626) 44%, var(--color-border, #cbd5e1)); }
[data-rcl-message-header] { display: flex; align-items: center; gap: var(--space-sm, 16px); min-inline-size: 0; }
[data-rcl-message-actor] { display: grid; gap: var(--space-3xs, 4px); min-inline-size: 0; flex: 1 1 auto; }
[data-rcl-message-actor-line] { display: flex; align-items: baseline; flex-wrap: wrap; gap: var(--space-2xs, 8px); min-inline-size: 0; }
[data-rcl-message-actor-name] { color: var(--color-foreground, #0f172a); font: var(--text-label, 500 var(--text-label-size) / var(--text-label-line) var(--font-sans)); }
[data-rcl-message-actor-role] { color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans)); }
[data-rcl-message-time] { color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans)); white-space: nowrap; }
[data-rcl-message-body] { display: grid; gap: var(--space-sm, 16px); min-inline-size: 0; }
[data-rcl-message-content] { max-inline-size: 72ch; min-inline-size: 0; color: var(--color-foreground, #0f172a); font: var(--text-body, 400 var(--text-body-size) / var(--text-body-line) var(--font-sans)); overflow-wrap: anywhere; }
[data-rcl-message-content] p { margin: 0; }
[data-rcl-message-content] p + p { margin-block-start: var(--space-sm, 16px); }
[data-rcl-message-state] { display: flex; align-items: center; flex-wrap: wrap; gap: var(--space-xs, 12px); min-inline-size: 0; color: var(--color-muted-foreground, #64748b); font: var(--text-body-sm, 400 var(--text-body-sm-size) / var(--text-body-sm-line) var(--font-sans)); }
[data-rcl-message-state-copy] { min-inline-size: 0; }
[data-rcl-message-error] { display: grid; gap: var(--space-xs, 12px); padding: var(--space-sm, 16px); border: var(--border-hairline, 1px) solid color-mix(in srgb, var(--color-danger, #dc2626) 42%, var(--color-border, #cbd5e1)); border-radius: var(--radius-panel, 0.5rem); background: color-mix(in srgb, var(--color-danger, #dc2626) 7%, var(--color-surface-muted, #f1f5f9)); color: var(--color-danger, #dc2626); font: var(--text-body-sm, 400 var(--text-body-sm-size) / var(--text-body-sm-line) var(--font-sans)); }
[data-rcl-message-error-actions], [data-rcl-message-actions] { display: flex; flex-wrap: wrap; align-items: center; gap: var(--space-2xs, 8px); }
[data-rcl-message-action] { min-block-size: var(--tap-target-min, 44px); padding: var(--space-2xs, 8px) var(--space-sm, 16px); border: var(--border-hairline, 1px) solid var(--color-border-strong, color-mix(in srgb, var(--color-border) 72%, var(--color-foreground))); border-radius: var(--radius-control, 0.375rem); background: var(--color-surface, #ffffff); color: var(--color-foreground, #0f172a); font: var(--text-label, 500 var(--text-label-size) / var(--text-label-line) var(--font-sans)); cursor: pointer; }
[data-rcl-message-action]:hover { border-color: var(--rcl-message-accent); color: var(--rcl-message-accent); }
[data-rcl-message-action][data-primary="true"] { border-color: var(--rcl-message-accent); background: var(--rcl-message-accent); color: var(--color-primary-foreground, #ffffff); }
[data-rcl-message-action]:disabled { cursor: not-allowed; opacity: .58; }
[data-rcl-message-list] { display: grid; gap: var(--space-2xs, 8px); margin: 0; padding: 0; list-style: none; }
[data-rcl-message-list-title] { margin-block-end: var(--space-2xs, 8px); color: var(--color-muted-foreground, #64748b); font: var(--text-overline, 700 var(--text-caption-size) / var(--text-caption-line) var(--font-sans)); letter-spacing: .08em; text-transform: uppercase; }
[data-rcl-message-attachment], [data-rcl-message-citation] { display: flex; align-items: flex-start; gap: var(--space-xs, 12px); min-inline-size: 0; padding: var(--space-xs, 12px) var(--space-sm, 16px); border: var(--border-hairline, 1px) solid var(--color-border, #cbd5e1); border-radius: var(--radius-panel, 0.5rem); background: var(--color-surface-muted, #f1f5f9); }
[data-rcl-message-attachment-copy], [data-rcl-message-citation-copy] { display: grid; gap: var(--space-3xs, 4px); min-inline-size: 0; }
[data-rcl-message-link] { min-inline-size: 0; color: var(--rcl-message-accent); font: var(--text-label, 500 var(--text-label-size) / var(--text-label-line) var(--font-sans)); overflow-wrap: anywhere; }
[data-rcl-message-detail] { color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans)); overflow-wrap: anywhere; }
[data-rcl-message-marker] { display: grid; place-items: center; inline-size: var(--space-md, 24px); block-size: var(--space-md, 24px); flex: 0 0 auto; margin-block-start: .15rem; border-radius: var(--radius-pill, 9999px); background: color-mix(in srgb, var(--rcl-message-accent) 16%, var(--color-surface-muted, #f1f5f9)); color: var(--rcl-message-accent); font: var(--text-caption, 600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans)); }
[data-rcl-message-divider] { block-size: var(--border-hairline, 1px); background: var(--color-border, #cbd5e1); }
[data-rcl-message-footer] { display: flex; align-items: center; flex-wrap: wrap; gap: var(--space-xs, 12px); color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans)); }
@media (max-width: 34rem) { [data-rcl-message] { padding: var(--space-md, 24px); } [data-rcl-message-header] { display: grid; grid-template-columns: auto minmax(0, 1fr); align-items: start; } [data-rcl-message-header] > [data-rcl-status-indicator] { grid-column: 2; justify-self: start; } [data-rcl-message-actor-line] { display: grid; gap: var(--space-3xs, 4px); } [data-rcl-message-actions] { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); } [data-rcl-message-action] { inline-size: 100%; } }


`;

function stateStatus(state: MessageState) {
  if (state === "success")
    return { status: "success" as const, label: "Delivered" };
  if (state === "request-error")
    return { status: "error" as const, label: "Response unavailable" };
  if (state === "loading")
    return { status: "pending" as const, label: "Generating response" };
  if (state === "partial")
    return { status: "pending" as const, label: "Streaming response" };
  if (state === "retry")
    return { status: "error" as const, label: "Ready to retry" };
  return null;
}

function AttachmentList({ items }: { items: MessageAttachment[] }) {
  const strings = useStrings();
  return (
    <section aria-label={strings("ai.message.attachments", "Attachments")}>
      <div data-rcl-message-list-title>
        {strings("ai.message.attachments", "Attachments")}
      </div>
      <ul data-rcl-message-list>
        {items.map((item) => (
          <li key={item.id} data-rcl-message-attachment>
            <span data-rcl-message-marker aria-hidden="true">
              ↗
            </span>
            <span data-rcl-message-attachment-copy>
              {item.href ? (
                <a
                  data-testid="ai.message"
                  data-rcl-message-link
                  href={item.href}
                >
                  {item.name}
                </a>
              ) : (
                <Text textStyle="label">{item.name}</Text>
              )}
              {item.detail ? (
                <span data-rcl-message-detail>{item.detail}</span>
              ) : null}
              {item.status && item.status !== "ready" ? (
                <span data-rcl-message-detail>
                  {item.status === "processing"
                    ? "Processing"
                    : "Needs attention"}
                </span>
              ) : null}
            </span>
          </li>
        ))}
      </ul>
    </section>
  );
}

function CitationList({ items }: { items: MessageCitation[] }) {
  const strings = useStrings();
  return (
    <section aria-label={strings("ai.message.citations", "Citations")}>
      <div data-rcl-message-list-title>
        {strings("ai.message.sources", "Sources")}
      </div>
      <ol data-rcl-message-list>
        {items.map((item, index) => (
          <li key={item.id} data-rcl-message-citation>
            <span data-rcl-message-marker aria-hidden="true">
              {index + 1}
            </span>
            <span data-rcl-message-citation-copy>
              {item.href ? (
                <a
                  data-testid="ai.message"
                  data-rcl-message-link
                  href={item.href}
                >
                  {item.label}
                </a>
              ) : (
                <Text textStyle="label">{item.label}</Text>
              )}
              {item.source ? (
                <span data-rcl-message-detail>{item.source}</span>
              ) : null}
              {item.excerpt ? (
                <span data-rcl-message-detail>{item.excerpt}</span>
              ) : null}
            </span>
          </li>
        ))}
      </ol>
    </section>
  );
}

export const Message = withClassName(function Message({
  actor,
  content,
  state = "default",
  timestamp,
  certainty = "observed",
  urgency = "ambient",
  attachments = [],
  citations = [],
  activity,
  actions = [],
  footer,
  errorMessage = "The response could not be completed. Your conversation is still safe to retry.",
  onRetry,
  retryLabel = "Retry response",
  className,
  style,
}: MessageProps) {
  const strings = useStrings();
  const id = useId().replace(/:/g, "");
  const titleId = `rcl-message-title-${id}`;
  const descriptionId = `rcl-message-content-${id}`;
  const status = stateStatus(state);
  const isLoading = state === "loading";
  const hasBody = content !== undefined || isLoading;
  return (
    <article
      data-rcl-message
      data-state={state}
      className={className}
      style={style}
      aria-labelledby={titleId}
      aria-describedby={hasBody ? descriptionId : undefined}
      aria-busy={isLoading || undefined}
    >
      <StyleSheet name="message-1-0-4-1" css={styles} />
      <header data-rcl-message-header>
        <Avatar
          name={actor.name}
          src={actor.src}
          presence={actor.presence}
          size="sm"
        />
        <div data-rcl-message-actor>
          <div data-rcl-message-actor-line>
            <Text
              as="strong"
              id={titleId}
              textStyle="label"
              data-rcl-message-actor-name
            >
              {actor.name}
            </Text>
            {actor.role ? (
              <span data-rcl-message-actor-role>{actor.role}</span>
            ) : null}
          </div>
          {timestamp ? (
            <span data-rcl-message-time data-rcl-message-time-value>
              <RelativeTime value={timestamp} />
            </span>
          ) : null}
        </div>
        {status ? (
          <StatusIndicator
            status={status.status}
            label={status.label}
            certainty={certainty}
            urgency={urgency}
          />
        ) : null}
      </header>
      <div data-rcl-message-body>
        {isLoading ? (
          <div
            id={descriptionId}
            data-rcl-message-state
            role="status"
            aria-label={strings(
              "ai.message.generating-response",
              "Generating response",
            )}
          >
            <span data-rcl-message-marker aria-hidden="true">
              ✦
            </span>
            <span data-rcl-message-state-copy>
              {strings(
                "ai.message.gathering-a-considered-response",
                "Gathering a considered response…",
              )}
            </span>
          </div>
        ) : content !== undefined ? (
          <div id={descriptionId} data-rcl-message-content>
            <Text as="div" textStyle="body">
              {content}
            </Text>
          </div>
        ) : null}
        {activity ? (
          <div data-rcl-message-state aria-label={activity.label}>
            <StatusIndicator
              status={activity.status}
              label={activity.label}
              certainty={activity.certainty}
              urgency={activity.urgency}
            />
            {activity.detail ? (
              <span data-rcl-message-state-copy>{activity.detail}</span>
            ) : null}
          </div>
        ) : null}
        {attachments.length > 0 ? <AttachmentList items={attachments} /> : null}
        {citations.length > 0 ? <CitationList items={citations} /> : null}
        {state === "request-error" || state === "retry" ? (
          <div
            data-rcl-message-error
            role={state === "request-error" ? "alert" : "status"}
          >
            <span>{errorMessage}</span>
            <div data-rcl-message-error-actions>
              {onRetry ? (
                <button
                  data-testid="ai.message"
                  type="button"
                  data-rcl-message-action
                  data-primary="true"
                  onClick={onRetry}
                >
                  {retryLabel}
                </button>
              ) : null}
            </div>
          </div>
        ) : null}
      </div>
      {actions.length > 0 ? (
        <>
          <div data-rcl-message-divider aria-hidden="true" />
          <div
            data-rcl-message-actions
            aria-label={strings(
              "ai.message.message-actions",
              "Message actions",
            )}
          >
            {actions.map((action) => (
              <button
                data-testid="ai.message"
                key={action.id}
                type="button"
                data-rcl-message-action
                data-primary={action.primary || undefined}
                onClick={action.onClick}
                disabled={action.disabled}
              >
                {action.label}
              </button>
            ))}
          </div>
        </>
      ) : null}
      {footer ? <footer data-rcl-message-footer>{footer}</footer> : null}
    </article>
  );
});

export const MessageParts = {
  AttachmentList,
  CitationList,
};
