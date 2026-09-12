import { useState, type RefObject } from "react";
import { useTranslation } from "react-i18next";
import { ContextMenu, type ContextMenuItem } from "@vrooli/react-component-library/ContextMenu/1";
import { BottomSheet } from "@vrooli/react-component-library/BottomSheet/1";
import { strings } from "../../consts/strings";
import { cn } from "../../lib/classnames";
import { PlaybackModeControl } from "../tts/PlaybackModeControl";
import {
  MESSAGE_ACTIONS,
  actionIcon,
  actionLabelKey,
  actionPlacement,
  orderedActions,
  type MessageAction,
  type MessageActionContext,
} from "./messageActions";

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#the-row-and-action-reveal

/** Where the list opens: a pointer position, or (null) anchored to the row. */
export interface ActionsOrigin {
  x: number;
  y: number;
}

/**
 * A message's actions as a surface offers them: the context with the
 * playback-mode control wired in, the actions that apply (primary first), and
 * the composite controls they render. The row and the reader both use it, so
 * both offer the same list.
 */
export function useMessageActions(base: MessageActionContext, anchorRef: RefObject<HTMLElement | null>, isTall: boolean) {
  const [playbackModeOpen, setPlaybackModeOpen] = useState(false);
  const { event, selectedVersion, summarizingEventId, summarizeLevel, onToggleSummarized, onChangeLevel } = base;
  const hasSummary = event.summarized && event.originalSpeechParagraphs != null && event.originalSpeechParagraphs.length > 0;
  const ctx: MessageActionContext = {
    ...base,
    isTall,
    onOpenPlaybackMode: () => { setPlaybackModeOpen(true); },
    renderPlaybackAction: () => (
      <PlaybackModeControl
        testIdPrefix={`msg-${event.id}`}
        isSummarized={selectedVersion === "active" && hasSummary}
        hasOriginalVersion={hasSummary}
        canSummarize
        isSummarizing={summarizingEventId === event.id}
        currentLevel={summarizeLevel}
        onToggleSummarized={(use) => { onToggleSummarized(event.id, use); }}
        onChangeLevel={(level) => { onChangeLevel(event.id, level); }}
        open={playbackModeOpen}
        onOpenChange={setPlaybackModeOpen}
        hideTrigger
        anchorRef={anchorRef}
      />
    ),
  };
  const composites = MESSAGE_ACTIONS.filter((action) => action.render && action.appliesTo(ctx)).map((action) => (
    <span key={`${action.id}-composite`}>{action.render?.(ctx)}</span>
  ));
  return { ctx, actions: orderedActions(ctx), composites };
}

interface MessageActionListProps {
  /** Every applicable action, primary first. */
  actions: readonly MessageAction[];
  ctx: MessageActionContext;
  coarsePointer: boolean;
  origin: ActionsOrigin | null;
  anchorRef: RefObject<HTMLElement | null>;
  onClose: () => void;
}

/**
 * The full action list for one message: a bottom sheet on touch devices, a
 * context menu at the pointer (or the row) on fine pointers. One declared
 * model (messageActions.ts) feeds both.
 */
export function MessageActionList({ actions, ctx, coarsePointer, origin, anchorRef, onClose }: MessageActionListProps) {
  const { t } = useTranslation();
  const eventId = ctx.event.id;

  if (coarsePointer) {
    return (
      <BottomSheet
        // Sized to the app's viewport like every overlay. Action rows only, so
        // the keyboard never moves it.
        avoidKeyboard
        open
        onOpenChange={(open) => { if (!open) onClose(); }}
        title={t(strings.messageActions.sheetTitle)}
        closeLabel={t(strings.handoff.close)}
        testId="msg-action-sheet"
      >
        <div className="flex flex-col gap-0.5">
          {actions.map((action) => {
            const Icon = actionIcon(action, ctx);
            const primary = actionPlacement(action, ctx) === "primary";
            return (
              <button
                key={action.id}
                type="button"
                data-testid={action.testId(ctx)}
                data-action-row
                disabled={action.disabled?.(ctx)}
                aria-pressed={action.pressed?.(ctx)}
                onClick={() => { onClose(); action.run(ctx); }}
                className={cn(
                  "flex min-h-11 w-full items-center gap-3 rounded-lg px-3 text-start text-sm transition hover:bg-wc-surface-input disabled:opacity-60",
                  primary ? "text-wc-accent" : "text-wc-text-secondary",
                )}
              >
                <Icon className="h-4 w-4 shrink-0" />
                <span>{t(actionLabelKey(action, ctx) as never)}</span>
              </button>
            );
          })}
        </div>
      </BottomSheet>
    );
  }

  const items: ContextMenuItem[] = actions.map((action) => {
    const Icon = actionIcon(action, ctx);
    return {
      id: action.id,
      label: t(actionLabelKey(action, ctx) as never),
      // The render-mode toggle keeps its selector on the icon, where it has
      // always carried aria-pressed (D26: existing selectors are preserved).
      icon: action.id === "render-mode"
        ? <span data-testid={action.testId(ctx)} aria-pressed={action.pressed?.(ctx)}><Icon className="h-3.5 w-3.5" /></span>
        : <Icon className="h-3.5 w-3.5" />,
      testId: action.id === "render-mode" ? undefined : action.testId(ctx),
      disabled: action.disabled?.(ctx),
      pressed: action.pressed?.(ctx),
      onSelect: () => { action.run(ctx); },
    };
  });

  return (
    <ContextMenu
      open
      onOpenChange={(open) => { if (!open) onClose(); }}
      position={origin ?? undefined}
      anchorRef={origin ? undefined : anchorRef}
      placement="bottom-end"
      title={t(strings.messageActions.more)}
      closeLabel={t(strings.handoff.close)}
      testId={`msg-actions-menu-${eventId}`}
      items={items}
    />
  );
}
