import type { ReactNode } from "react";
import {
  ArrowUpRight,
  BookmarkPlus,
  Check,
  Copy,
  FileCode2,
  Forward,
  ListMusic,
  Play,
  Volume2,
  type LucideIcon,
} from "lucide-react";
import type { ConversationEvent } from "../../api/conversation";
import type { TTSPlaybackState } from "../../audio-integration";
import type { PlaybackVersion } from "../../domains/tts-playback/types";
import type { SummarizationLevel } from "../tts/PlaybackModeControl";
import { strings } from "../../consts/strings";

// DOC: docs/internal/SNIPPETS-AND-MESSAGE-ACTIONS-UX.md

export type MessageAudioSettings = Pick<
  TTSPlaybackState,
  "volume" | "isMuted" | "playbackRate" | "capabilities"
>;

export interface MessageActionContext {
  event: ConversationEvent;
  sessionId: string;
  readOnly: boolean;
  copied: boolean;
  isPlaintext: boolean;
  isAudioLoading: boolean;
  isTtsSpeaking: boolean;
  activeSpeakingEventId: string | null;
  summarizeLevel: SummarizationLevel;
  selectedVersion: PlaybackVersion;
  summarizingEventId: string | null;
  getSummarizeError: (eventId: string) => string | null;
  onClearSummarizeError: (eventId: string) => void;
  onToggleSummarized: (eventId: string, useSummarized: boolean) => void;
  onChangeLevel: (eventId: string, level: SummarizationLevel) => void;
  audioSettings: MessageAudioSettings;
  onSetPlaybackRate: (rate: number) => void;
  onSetVolume: (level: number) => void;
  onSetMuted: (next: boolean) => void;
  isMobile: boolean;
  onCopy: (eventId: string, text: string) => void;
  onPlayFromHere: (eventId: string) => void;
  onPlayEvent: (eventId: string) => void;
  onToggleRenderMode: (eventId: string) => void;
  /** Present only where the surface can reach a composer. */
  onSendToComposer?: (text: string) => void;
  onSaveAsSnippet?: (text: string) => void;
  onHandoff?: (sessionId: string, payload: string) => void;
  onOpenPlaybackMode?: () => void;
  onOpenAudio?: () => void;
  renderPlaybackAction?: () => ReactNode;
  renderAudioAction?: () => ReactNode;
}

export interface MessageAction {
  /** Stable registry id. */
  id: string;
  /** Existing i18n key used for the control label. */
  labelKey: string | ((ctx: MessageActionContext) => string);
  icon: (ctx: MessageActionContext) => LucideIcon;
  /** Primary actions render inline; the rest render in the overflow menu. */
  placement: "primary" | "overflow";
  placementFor?: (ctx: MessageActionContext) => "primary" | "overflow";
  appliesTo: (ctx: MessageActionContext) => boolean;
  run: (ctx: MessageActionContext) => void;
  testId: (ctx: MessageActionContext) => string;
  pressed?: (ctx: MessageActionContext) => boolean;
  disabled?: (ctx: MessageActionContext) => boolean;
  /** Composite controls retain their own portals and state behind this hook. */
  render?: (ctx: MessageActionContext) => ReactNode;
}

export function actionPlacement(action: MessageAction, ctx: MessageActionContext): "primary" | "overflow" {
  return action.placementFor?.(ctx) ?? action.placement;
}

export function actionLabelKey(action: MessageAction, ctx: MessageActionContext): string {
  return typeof action.labelKey === "function" ? action.labelKey(ctx) : action.labelKey;
}

export function actionIcon(action: MessageAction, ctx: MessageActionContext): LucideIcon {
  return action.icon(ctx);
}

export const MESSAGE_ACTIONS: readonly MessageAction[] = [
  {
    id: "copy",
    labelKey: strings.messageActions.copy,
    icon: (ctx) => (ctx.copied ? Check : Copy),
    placement: "primary",
    appliesTo: () => true,
    run: (ctx) => { ctx.onCopy(ctx.event.id, ctx.event.text); },
    testId: (ctx) => `msg-copy-${ctx.event.id}`,
  },
  {
    id: "read-from-here",
    labelKey: strings.messageActions.readFromHere,
    icon: () => Play,
    placement: "primary",
    appliesTo: (ctx) => !ctx.readOnly && ctx.event.role !== "user",
    run: (ctx) => { ctx.onPlayFromHere(ctx.event.id); },
    testId: (ctx) => `msg-speak-from-${ctx.event.id}`,
    disabled: (ctx) => ctx.isAudioLoading,
  },
  {
    id: "save-as-snippet",
    labelKey: strings.messageActions.saveAsSnippet,
    icon: () => BookmarkPlus,
    placement: "primary",
    placementFor: (ctx) => (ctx.event.role === "user" ? "primary" : "overflow"),
    appliesTo: (ctx) => ctx.onSaveAsSnippet != null,
    run: (ctx) => { ctx.onSaveAsSnippet?.(ctx.event.text); },
    testId: (ctx) => `msg-save-snippet-${ctx.event.id}`,
  },
  {
    id: "handoff",
    labelKey: strings.messageActions.handoff,
    icon: () => Forward,
    placement: "overflow",
    appliesTo: (ctx) => ctx.onHandoff != null,
    run: (ctx) => { ctx.onHandoff?.(ctx.sessionId, ctx.event.text); },
    testId: (ctx) => `msg-handoff-${ctx.event.id}`,
  },
  {
    id: "send-to-composer",
    labelKey: strings.messageActions.sendToComposer,
    icon: () => ArrowUpRight,
    placement: "overflow",
    appliesTo: (ctx) => ctx.onSendToComposer != null,
    run: (ctx) => { ctx.onSendToComposer?.(ctx.event.text); },
    testId: (ctx) => `msg-send-to-composer-${ctx.event.id}`,
  },
  {
    id: "render-mode",
    labelKey: (ctx) => ctx.isPlaintext
      ? strings.messageActions.viewAsMarkdown
      : strings.messageActions.viewAsPlainText,
    icon: () => FileCode2,
    placement: "overflow",
    appliesTo: () => true,
    run: (ctx) => { ctx.onToggleRenderMode(ctx.event.id); },
    testId: (ctx) => `msg-render-toggle-${ctx.event.id}`,
    pressed: (ctx) => ctx.isPlaintext,
  },
  {
    id: "playback-mode",
    labelKey: strings.messageActions.playbackMode,
    icon: () => ListMusic,
    placement: "overflow",
    appliesTo: (ctx) => !ctx.readOnly && ctx.event.role !== "user",
    run: (ctx) => { ctx.onOpenPlaybackMode?.(); },
    testId: (ctx) => `msg-${ctx.event.id}-mode-control`,
    render: (ctx) => ctx.renderPlaybackAction?.(),
  },
  {
    id: "audio-settings",
    labelKey: strings.messageActions.audioSettings,
    icon: () => Volume2,
    placement: "overflow",
    appliesTo: (ctx) => !ctx.readOnly && ctx.event.role !== "user",
    run: (ctx) => { ctx.onOpenAudio?.(); },
    testId: (ctx) => `msg-audio-${ctx.event.id}`,
    disabled: (ctx) => ctx.isAudioLoading,
    render: (ctx) => ctx.renderAudioAction?.(),
  },
] as const;
