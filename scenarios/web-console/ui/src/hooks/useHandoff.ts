import { useCallback } from "react";

import type { GateResult } from "../components/terminal/inputGate";
import { renderHandoffPrompt, type HandoffResult } from "../lib/handoff";
import type { SessionActivity } from "../lib/workspaceNavigation";
import { useWorkspaceStore, type PaneMetadata, type RoleMeta, type TabGroupMeta } from "../stores/useWorkspaceStore";

// DOC: docs/internal/SNIPPETS-AND-MESSAGE-ACTIONS-UX.md

// [REQ:P0-014d] Handoff Between Sessions In A Group
//
// This module is the SEND path. It imports nothing from the capture-rule
// matcher or its API client, and it never will: deleting every capture rule
// must not change one line of behaviour here. A rule only ever produces a
// suggestion that opens the same composer a button opens.

/**
 * Where a handoff is going.
 *
 * A descriptor, not a raw session id, because a target may be a role that has
 * not started yet — and the difference decides whether sending has to launch
 * a process first. Taking descriptors from the start is what let the
 * waiting-role case be an addition rather than a rewrite.
 */
export type HandoffTarget = (
  | { kind: "session"; sessionId: string; label: string; incomingPrompt: string }
  | { kind: "role"; role: RoleMeta; label: string; incomingPrompt: string }
  /** Create a session in the group, then send to it. */
  | { kind: "new-session"; groupId: string | null; label: string; incomingPrompt: string }
) & { meta?: HandoffTargetMeta };

/**
 * Display-only description of a target row. The send path never reads it.
 *
 * Four terminals launched the same way all render as "/bin/bash", and the
 * composer used to carry nothing but that name — so the list read as four
 * identical rows. These are the signals the SIDEBAR already uses to tell the
 * same panes apart (accent colour, group, last activity), deliberately
 * reused rather than a second vocabulary invented for this one surface.
 */
export interface HandoffTargetMeta {
  /** The pane's own accent; the group's is carried separately as the fallback. */
  color?: string | null;
  groupColor?: string | null;
  /** Which group the session lives in. Worth showing outside "this group". */
  groupName?: string;
  /** "2m", "Visited 14m" — empty for a session that has never been seen. */
  activityLabel?: string;
  /** Sort key: epoch ms of that moment, 0 when there is none. */
  activityAt?: number;
  /** Unread assistant messages. Always 0 for a plain terminal. */
  unreadCount?: number;
}

export type HandoffTargetSectionKind = "group" | "other" | "new";

export interface HandoffTargetSection {
  kind: HandoffTargetSectionKind;
  /** i18n key for the section heading. */
  labelKey: string;
  targets: HandoffTarget[];
}

/** A stable id for a target, used for result keys and test ids. */
export function targetKey(target: HandoffTarget): string {
  if (target.kind === "session") return target.sessionId;
  if (target.kind === "role") return target.role.id;
  return `new-${target.groupId ?? "ungrouped"}`;
}

/** The exact text one target will receive, before the operator edits it. */
export function textForTarget(target: HandoffTarget, payload: string): string {
  return renderHandoffPrompt(target.incomingPrompt, payload);
}

export interface SendHandoffDeps {
  /** Deliver text to a specific session's terminal. */
  submit: (data: string, intent: "bulk_text", targetId: string) => GateResult;
  /** Start a command and resolve to the new session id, or null on failure. */
  launch: (options: { command?: string; workingDir?: string; backend?: string; targetId?: string; groupId?: string | null }) => Promise<string | null>;
  /** Queue text under a session id that has no mounted terminal yet. */
  queueForSession: (sessionId: string, text: string) => void;
  /** Attach a started role to its new session. */
  attachRole: (roleId: string, sessionId: string) => void;
}

/**
 * Send one message to each target.
 *
 * Targets are processed SEQUENTIALLY, not concurrently. `launchSession`
 * guards against concurrent creation with an in-flight flag and returns null
 * on the second overlapping call, so a parallel fan-out to three waiting
 * roles would silently start one. Awaiting each in turn is the whole fix.
 *
 * Every target gets a result. `queued` is never collapsed into `sent`: a
 * session created moments ago has no mounted terminal, and reporting success
 * there would tell the operator their message arrived when it has not.
 */
export async function sendHandoff(
  targets: readonly HandoffTarget[],
  textFor: (target: HandoffTarget) => string,
  deps: SendHandoffDeps,
): Promise<HandoffResult[]> {
  const results: HandoffResult[] = [];

  for (const target of targets) {
    const key = targetKey(target);
    const text = textFor(target);

    if (!text.trim()) {
      results.push({ targetId: key, label: target.label, status: "failed", reason: "empty" });
      continue;
    }

    if (target.kind === "session") {
      results.push(deliver(target.sessionId, text, key, target.label, deps));
      continue;
    }

    // Both remaining kinds start a process first. If the start fails the
    // message is NOT queued anywhere — reporting `queued` for a process that
    // never began would leave the operator waiting on nothing.
    const launchOptions = target.kind === "role"
      ? {
          command: target.role.command || undefined,
          workingDir: target.role.workingDir || undefined,
          backend: target.role.backend || undefined,
          targetId: target.role.targetId || undefined,
        }
      : { groupId: target.groupId };

    let sessionId: string | null = null;
    try {
      sessionId = await deps.launch(launchOptions);
    } catch (error) {
      console.error("Handoff could not start its target:", error);
      sessionId = null;
    }

    if (!sessionId) {
      results.push({ targetId: key, label: target.label, status: "failed", reason: "start-failed" });
      continue;
    }

    if (target.kind === "role") deps.attachRole(target.role.id, sessionId);

    // A just-created session has no terminal handle, so this is the queue's
    // ordinary path rather than an error case.
    deps.queueForSession(sessionId, text);
    results.push({ targetId: key, label: target.label, status: "queued", reason: "not-ready" });
  }

  return results;
}

/** Hand text to a running session, mapping the gate's verdict onto a result. */
function deliver(
  sessionId: string,
  text: string,
  key: string,
  label: string,
  deps: SendHandoffDeps,
): HandoffResult {
  const gate = deps.submit(text, "bulk_text", sessionId);
  if (gate.status === "sent") {
    return { targetId: key, label, status: "sent" };
  }
  if (gate.status === "queued") {
    return { targetId: key, label, status: "queued", reason: gate.reason };
  }
  // A rejection means no terminal handle exists for this id yet — the pane is
  // unmounted, or the session is still coming up. Queue it rather than
  // dropping it: a handoff must never disappear without telling the operator.
  if (gate.reason === "disposed") {
    deps.queueForSession(sessionId, text);
    return { targetId: key, label, status: "queued", reason: "not-ready" };
  }
  return { targetId: key, label, status: "failed", reason: gate.reason };
}

/**
 * Per-session activity, as the sidebar computes it. Supplied by the caller
 * because the workspace STORE does not hold it: conversation events and
 * last-visited times are Workspace state, so a store-only builder cannot see
 * them. Defaulting to empty keeps every target valid without it.
 */
export type HandoffActivity = Record<string, SessionActivity | undefined>;

/** Newest first, then by label, so identically named rows still have an order. */
function byRecency(a: HandoffTarget, b: HandoffTarget): number {
  const delta = (b.meta?.activityAt ?? 0) - (a.meta?.activityAt ?? 0);
  return delta !== 0 ? delta : a.label.localeCompare(b.label);
}

/**
 * The targets a session may hand off to: every other member of its group.
 *
 * Excludes the source, and excludes running roles whose session is already
 * listed as a pane, so a session never appears twice.
 */
function groupTargetsForSession(
  sourceSessionId: string,
  groupId: string | null,
  activity: HandoffActivity,
): HandoffTarget[] {
  if (!groupId) return [];
  const { panes, roles, groups } = useWorkspaceStore.getState();
  const groupBy = new Map(groups.map((group) => [group.id, group]));
  const roleBySession = new Map(
    roles.filter((r) => r.sessionId !== null).map((r) => [r.sessionId as string, r]),
  );

  const sessions: HandoffTarget[] = [];
  for (const pane of panes) {
    if (pane.groupId !== groupId || pane.sessionId === sourceSessionId) continue;
    const role = roleBySession.get(pane.sessionId);
    sessions.push({
      kind: "session",
      sessionId: pane.sessionId,
      label: role?.label ?? pane.name,
      // A running role still supplies its own framing: the prompt lives on
      // the receiver, so a hand-made role and a templated one behave alike.
      incomingPrompt: role?.incomingPrompt ?? "",
      meta: paneMeta(pane, groupBy, activity),
    });
  }
  // Waiting roles keep their own block after the running sessions: they are
  // not ordered by activity because they have none, and interleaving them
  // would put a placeholder above a session that just produced output.
  const waiting: HandoffTarget[] = [];
  for (const role of roles) {
    if (role.groupId !== groupId || role.sessionId !== null) continue;
    waiting.push({
      kind: "role",
      role,
      label: role.label,
      incomingPrompt: role.incomingPrompt,
      meta: { groupColor: groupBy.get(groupId)?.color },
    });
  }
  return [...sessions.sort(byRecency), ...waiting];
}

/** The row description for one pane. */
function paneMeta(
  pane: PaneMetadata,
  groupBy: Map<string, TabGroupMeta>,
  activity: HandoffActivity,
): HandoffTargetMeta {
  const group = pane.groupId ? groupBy.get(pane.groupId) : undefined;
  const seen = activity[pane.sessionId];
  return {
    color: pane.headerColor,
    groupColor: group?.color,
    groupName: group?.name,
    activityLabel: seen?.label ?? "",
    activityAt: seen?.at ?? 0,
    unreadCount: seen?.unreadCount ?? 0,
  };
}

/** Ordered handoff choices: the source group, other live panes, then a new session. */
export function handoffTargetSections(
  sourceSessionId: string,
  groupId: string | null,
  activity: HandoffActivity = {},
): HandoffTargetSection[] {
  const { panes, roles, groups } = useWorkspaceStore.getState();
  const groupBy = new Map(groups.map((group) => [group.id, group]));
  const groupTargets = groupTargetsForSession(sourceSessionId, groupId, activity);
  const groupedSessionIds = new Set(
    groupTargets.flatMap((target) => target.kind === "session" ? [target.sessionId] : []),
  );
  const roleBySession = new Map(
    roles.filter((role) => role.sessionId !== null).map((role) => [role.sessionId as string, role]),
  );
  const otherTargets: HandoffTarget[] = panes.flatMap((pane) => {
    if (pane.sessionId === sourceSessionId || groupedSessionIds.has(pane.sessionId)) return [];
    const role = roleBySession.get(pane.sessionId);
    return [{
      kind: "session" as const,
      sessionId: pane.sessionId,
      label: role?.label ?? pane.name,
      incomingPrompt: role?.incomingPrompt ?? "",
      meta: paneMeta(pane, groupBy, activity),
    }];
  }).sort(byRecency);

  return [
    ...(groupId && groupTargets.length > 0
      ? [{ kind: "group" as const, labelKey: "handoff.sections.group", targets: groupTargets }]
      : []),
    { kind: "other", labelKey: "handoff.sections.other", targets: otherTargets },
    {
      kind: "new",
      labelKey: "handoff.sections.new",
      targets: [{ kind: "new-session", groupId, label: "New session", incomingPrompt: "" }],
    },
  ];
}

/** Bind sendHandoff to the app's seams. */
export function useHandoff(deps: SendHandoffDeps) {
  return useCallback(
    (targets: readonly HandoffTarget[], textFor: (target: HandoffTarget) => string) =>
      sendHandoff(targets, textFor, deps),
    [deps],
  );
}
