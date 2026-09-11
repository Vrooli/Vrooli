import {
  SessionActivitySource,
  SessionActivityState,
  type SessionActivity,
} from "@vrooli/proto-types/web-console/v1/sessions/sessions_pb";

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#session-activity-contract

/** What a session's agent is doing between messages, as the server reads it. */
export type ActivityState = "working" | "idle" | "waiting" | "unknown";

/** The evidence behind an activity. */
export type ActivitySource = "screen" | "output_clock" | "hook" | "harness_event";

export interface PromptOptionView {
  key: string;
  label: string;
  selected: boolean;
}

export interface PendingPromptView {
  kind: string;
  /** Empty when the prompt was detected but its content could not be read. */
  text: string;
  options: PromptOptionView[];
  answerable: boolean;
  freeTextHint?: string;
  /** Identifies the prompt for an answer (kind, text, options; not the selection). */
  hash?: string;
  /** The harness lets Escape dismiss it. */
  cancellable?: boolean;
}

export interface SessionActivityView {
  state: ActivityState;
  source: ActivitySource;
  confidence: number;
  /** RFC 3339; when the current state began. */
  since: string;
  lastOutputAt?: string;
  prompt?: PendingPromptView;
  harness: string;
  harnessVersion?: string;
}

/** The snake_case payload of a `session_activity` hub envelope. */
export interface SessionActivityPayload {
  state?: string;
  source?: string;
  confidence?: number;
  since?: string;
  last_output_at?: string;
  prompt?: {
    kind?: string;
    text?: string;
    options?: { key?: string; label?: string; selected?: boolean }[];
    answerable?: boolean;
    free_text_hint?: string;
    hash?: string;
    cancellable?: boolean;
  };
  harness?: string;
  harness_version?: string;
}

const STATES: ReadonlySet<string> = new Set<ActivityState>(["working", "idle", "waiting", "unknown"]);
const SOURCES: ReadonlySet<string> = new Set<ActivitySource>(["screen", "output_clock", "hook", "harness_event"]);

function asState(value: string | undefined): ActivityState {
  return value && STATES.has(value) ? (value as ActivityState) : "unknown";
}

function asSource(value: string | undefined): ActivitySource {
  return value && SOURCES.has(value) ? (value as ActivitySource) : "output_clock";
}

export function decodeActivityPayload(payload: SessionActivityPayload): SessionActivityView {
  const prompt = payload.prompt;
  return {
    state: asState(payload.state),
    source: asSource(payload.source),
    confidence: payload.confidence ?? 0,
    since: payload.since ?? "",
    ...(payload.last_output_at ? { lastOutputAt: payload.last_output_at } : {}),
    ...(prompt ? {
      prompt: {
        kind: prompt.kind ?? "unknown",
        text: prompt.text ?? "",
        options: (prompt.options ?? []).map((option) => ({
          key: option.key ?? "",
          label: option.label ?? "",
          selected: option.selected ?? false,
        })),
        answerable: prompt.answerable ?? false,
        ...(prompt.free_text_hint ? { freeTextHint: prompt.free_text_hint } : {}),
        ...(prompt.hash ? { hash: prompt.hash } : {}),
        ...(prompt.cancellable ? { cancellable: true } : {}),
      },
    } : {}),
    harness: payload.harness ?? "",
    ...(payload.harness_version ? { harnessVersion: payload.harness_version } : {}),
  };
}

const PROTO_STATES: Readonly<Record<number, ActivityState>> = {
  [SessionActivityState.WORKING]: "working",
  [SessionActivityState.IDLE]: "idle",
  [SessionActivityState.WAITING]: "waiting",
};

const PROTO_SOURCES: Readonly<Record<number, ActivitySource>> = {
  [SessionActivitySource.SCREEN]: "screen",
  [SessionActivitySource.OUTPUT_CLOCK]: "output_clock",
  [SessionActivitySource.HOOK]: "hook",
  [SessionActivitySource.HARNESS_EVENT]: "harness_event",
};

/** Decode `Session.activity` from the sessions list. */
export function decodeActivityProto(activity: SessionActivity): SessionActivityView {
  return decodeActivityPayload({
    state: PROTO_STATES[activity.state] ?? "unknown",
    source: PROTO_SOURCES[activity.source] ?? "output_clock",
    confidence: activity.confidence,
    since: activity.since,
    last_output_at: activity.lastOutputAt,
    ...(activity.prompt ? {
      prompt: {
        kind: activity.prompt.kind,
        text: activity.prompt.text,
        options: activity.prompt.options.map((option) => ({ key: option.key, label: option.label, selected: option.selected })),
        answerable: activity.prompt.answerable,
        free_text_hint: activity.prompt.freeTextHint,
        hash: activity.prompt.hash,
        cancellable: activity.prompt.cancellable,
      },
    } : {}),
    harness: activity.harness,
    harness_version: activity.harnessVersion,
  });
}
