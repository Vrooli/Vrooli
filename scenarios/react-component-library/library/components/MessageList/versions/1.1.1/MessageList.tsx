/**
 * @libraryId react-component-library:MessageList
 * @displayName MessageList
 * @description The message collection preserving streaming scroll position, supporting branching, grouping, history loading, a new-message indicator, and stable anchoring while content grows above and below.
 * @version 1.1.1
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource ai.message-list */
import { useEffect, useRef, useState, type CSSProperties } from "react";
import { Message, type MessageProps } from "@vrooli/react-component-library/Message/1";
import {
  VirtualList,
  type VirtualListHandle,
  type VirtualListPosition,
} from "@vrooli/react-component-library/VirtualList/1";
import {
  AsyncBoundary,
  type AsyncBoundaryStatus,
} from "@vrooli/react-component-library/AsyncBoundary/1";
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";
import { useStrings } from "@vrooli/react-component-library/useLocale/1";

export type MessageListState =
  | "default"
  | "loading"
  | "refreshing"
  | "empty"
  | "partial"
  | "stale"
  | "request-error"
  | "offline"
  | "retry";
export interface MessageListItem extends Omit<MessageProps, "onRetry"> {
  id: string;
  group?: string;
  /** Changes when rich content changes without changing its item identity. */
  revision?: string;
}
export interface MessageListBranch {
  id: string;
  label: string;
  disabled?: boolean;
}
export interface MessageListProps {
  presentation?: "card" | "transcript";
  messages?: MessageListItem[];
  state?: MessageListState;
  label?: string;
  height?: number | string;
  emptyLabel?: string;
  errorMessage?: string;
  onRetry?: () => void | Promise<void>;
  onRetryMessage?: (id: string) => void;
  hasEarlier?: boolean;
  loadingEarlier?: boolean;
  historyError?: string;
  onLoadEarlier?: () => void;
  earlierLabel?: string;
  loadingEarlierLabel?: string;
  latestLabel?: string;
  newMessagesLabel?: string;
  branches?: MessageListBranch[];
  branchId?: string;
  branchLabel?: string;
  onBranchChange?: (id: string) => void;
  initialScrollTop?: number;
  onViewportChange?: (position: VirtualListPosition) => void;
  className?: string;
  style?: CSSProperties;
}
const none: MessageListItem[] = [];
const noBranches: MessageListBranch[] = [];
const keyOf = (message: MessageListItem) => message.id;
const states: Record<MessageListState, AsyncBoundaryStatus> = {
  default: "success",
  loading: "pending",
  refreshing: "refreshing",
  empty: "idle",
  partial: "partial-error",
  stale: "stale",
  "request-error": "error",
  offline: "offline",
  retry: "refreshing",
};
const css = `
[data-rcl-message-list][data-presentation="transcript"] [data-rcl-virtual-list] { background: transparent; }
[data-rcl-message-list][data-presentation="transcript"] [data-rcl-virtual-list-status] { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip-path: inset(50%); white-space: nowrap; }

[data-rcl-message-list] { min-inline-size: 0; color: var(--color-foreground); }
[data-rcl-message-list][data-presentation="transcript"] > [data-rcl-async-boundary] { border: 0; border-radius: 0; box-shadow: none; background: transparent; }
[data-rcl-message-list] > [data-rcl-async-boundary] > [data-rcl-async-content] { padding: 0; }
[data-rcl-message-list] [data-rcl-virtual-list] { border: 0; border-radius: 0; box-shadow: none; }
[data-rcl-message-list] [data-rcl-virtual-list-row] { padding: var(--space-sm); border: 0; background: transparent; }
[data-rcl-message-list-toolbar], [data-rcl-message-list-history], [data-rcl-message-list-latest] { display: flex; flex-wrap: wrap; align-items: center; gap: var(--space-xs); padding: var(--space-sm); }
[data-rcl-message-list] button[data-message-list-action], [data-rcl-message-list] select { min-block-size: var(--tap-target-min); max-inline-size: 100%; padding: var(--space-2xs) var(--space-sm); border: 1px solid var(--color-border); border-radius: var(--radius-control); background: var(--color-surface); color: inherit; font: inherit; cursor: pointer; }
[data-rcl-message-list] button:focus-visible, [data-rcl-message-list] select:focus-visible { outline: 2px solid var(--color-focus); outline-offset: 2px; }
[data-rcl-message-list] button:disabled { cursor: wait; opacity: .65; }
[data-rcl-message-list-group] { margin: 0 0 var(--space-xs); font: inherit; font-weight: 600; }
[data-rcl-message-list-latest] { justify-content: center; }
[data-rcl-message-list-history-error] { margin: 0; padding-inline: var(--space-sm); }
`;

export const MessageList = withClassName(function MessageList({
  presentation = "card",
  messages = none,
  state = "default",
  label,
  height = 480,
  emptyLabel,
  errorMessage,
  onRetry,
  onRetryMessage,
  hasEarlier = false,
  loadingEarlier = false,
  historyError,
  onLoadEarlier,
  earlierLabel,
  loadingEarlierLabel,
  latestLabel,
  newMessagesLabel,
  branches = noBranches,
  branchId,
  branchLabel,
  onBranchChange,
  initialScrollTop,
  onViewportChange,
  className,
  style,
}: MessageListProps) {
  const strings = useStrings();
  label = label ?? strings("ai.message-list.label", "Conversation messages");
  emptyLabel = emptyLabel ?? strings("ai.message-list.empty", "No messages yet.");
  earlierLabel = earlierLabel ?? strings("ai.message-list.earlier", "Load earlier messages");
  loadingEarlierLabel =
    loadingEarlierLabel ?? strings("ai.message-list.loading-earlier", "Loading earlier messages…");
  latestLabel = latestLabel ?? strings("ai.message-list.latest", "Return to latest");
  newMessagesLabel =
    newMessagesLabel ?? strings("ai.message-list.new-messages", "Show new messages ({count})");
  branchLabel = branchLabel ?? strings("ai.message-list.branch", "Conversation branch");
  const controller = useRef<VirtualListHandle>(null);
  const atEnd = useRef(true);
  const [unseen, setUnseen] = useState(0);
  const [changed, setChanged] = useState(false);
  const previous = useRef({ messages, branchId });
  useEffect(() => {
    const old = previous.current;
    previous.current = { messages, branchId };
    if (old.branchId !== branchId || atEnd.current) {
      setUnseen(0);
      setChanged(false);
      return;
    }
    const last = old.messages.at(-1);
    const lastIndex = last ? messages.findIndex((message) => message.id === last.id) : -1;
    const oldKeys = new Set(old.messages.map(keyOf));
    const added =
      lastIndex >= 0
        ? messages.slice(lastIndex + 1).filter((message) => !oldKeys.has(message.id)).length
        : 0;
    if (added) setUnseen((count) => count + added);
    const currentLast = lastIndex >= 0 ? messages[lastIndex] : undefined;
    if (
      last &&
      currentLast &&
      (last.content !== currentLast.content ||
        last.revision !== currentLast.revision ||
        last.state !== currentLast.state)
    )
      setChanged(true);
  }, [messages, branchId]);
  const positionChanged = (position: VirtualListPosition) => {
    atEnd.current = position.atEnd;
    if (position.atEnd) {
      setUnseen(0);
      setChanged(false);
    }
    onViewportChange?.(position);
  };
  const boundaryStatus =
    state === "request-error" && messages.length ? "partial-error" : states[state];
  return (
    <section
      data-rcl-message-list
      data-presentation={presentation}
      data-state={state}
      aria-label={label}
      className={className}
      style={style}
    >
      <StyleSheet libraryId="react-component-library:MessageList" version="1.1.1" css={css} />
      {branches.length > 0 && (
        <label data-rcl-message-list-toolbar>
          {branchLabel}
          <select
            value={branchId ?? ""}
            disabled={!onBranchChange}
            onChange={(event) => onBranchChange?.(event.currentTarget.value)}
          >
            {!branchId && (
              <option value="" disabled>
                {branchLabel}
              </option>
            )}
            {branches.map((branch) => (
              <option key={branch.id} value={branch.id} disabled={branch.disabled}>
                {branch.label}
              </option>
            ))}
          </select>
        </label>
      )}
      <AsyncBoundary
        status={boundaryStatus}
        preserveContent={messages.length > 0}
        detectOffline={false}
        offline={state === "offline"}
        error={errorMessage}
        retry={onRetry}
        loadingDelay={0}
      >
        {(hasEarlier || loadingEarlier || historyError) && (
          <div data-rcl-message-list-history>
            <button
              type="button"
              data-message-list-action="history"
              disabled={loadingEarlier || !onLoadEarlier}
              aria-busy={loadingEarlier || undefined}
              onClick={onLoadEarlier}
            >
              {loadingEarlier ? loadingEarlierLabel : earlierLabel}
            </button>
            {historyError && (
              <p data-rcl-message-list-history-error role="alert">
                {historyError}
              </p>
            )}
          </div>
        )}
        <VirtualList
          key={branchId ?? "$default"}
          items={messages}
          getItemKey={keyOf}
          label={label}
          height={height}
          empty={emptyLabel}
          estimateItemHeight={220}
          followEnd
          controllerRef={controller}
          initialScrollTop={initialScrollTop}
          onViewportChange={positionChanged}
          renderItem={(message, index) => {
            const { id, group, revision: _revision, ...props } = message;
            return (
              <div role={group ? "group" : undefined} aria-label={group}>
                {group && messages[index - 1]?.group !== group && (
                  <h3 data-rcl-message-list-group>{group}</h3>
                )}
                <Message
                  presentation={presentation}
                  {...props}
                  onRetry={onRetryMessage ? () => onRetryMessage(id) : undefined}
                />
              </div>
            );
          }}
        />
      </AsyncBoundary>
      {(unseen > 0 || changed) && (
        <div data-rcl-message-list-latest>
          <button
            type="button"
            data-message-list-action="latest"
            onClick={() => controller.current?.scrollToEnd()}
          >
            {unseen > 0 ? newMessagesLabel.replace("{count}", String(unseen)) : latestLabel}
          </button>
        </div>
      )}
    </section>
  );
});
