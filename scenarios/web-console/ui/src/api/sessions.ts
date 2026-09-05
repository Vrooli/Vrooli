import { createClient } from "@connectrpc/connect";
import { ArchiveRestoreState, SessionsService, SessionOrigin } from "@vrooli/proto-types/web-console/v1/sessions/sessions_pb";
import type { Target } from "@vrooli/proto-types/web-console/v1/shared/target_pb";
import { buildWsUrl } from "@vrooli/api-base";

import { API_BASE_WITH_SUFFIX, transport } from "./client";
import { decodeTarget, type TerminalTarget } from "./targets";

// sessionsClient is the Connect-Web client for SessionsService. Consumers
// should prefer the typed wrappers below, which surface the snake_case
// shapes the UI components and stores expect.
export const sessionsClient = createClient(SessionsService, transport);

// ---------------------------------------------------------------------------
// Domain types
// ---------------------------------------------------------------------------

// [REQ:P1-001a] Expiration Policy types
export type PolicyMode = "never" | "preset" | "custom";

export interface ExpirationPolicy {
  mode: PolicyMode;
  duration?: string;
}

// CROSS-LANGUAGE COUPLING: Backend IDs must match BackendID constants in api/backend_registry.go
export type BackendID = "standard" | "persistent";

// Provenance of a session — who/what created it. Mirrors sessionstore.Origin in
// the API ("ui" | "programmatic" | "remote"); "unspecified" is the pre-Phase-1
// / unknown fallback. The sidebar buckets sessions by this (see the origin tabs
// work) so a CLI- or remotely-created session is visually distinct from a UI one.
export type SessionOriginName = "ui" | "programmatic" | "remote" | "unspecified";

export interface BackendOption {
  id: BackendID;
  display_name: string;
  description: string;
  survives_restart: boolean;
  available: boolean;
  reason?: string;
}

export interface SessionInfo {
  id: string;
  shell: string;
  created_at: string;
  cols: number;
  rows: number;
  backend: BackendID;
  survives_restart: boolean;
  policy: ExpirationPolicy;
  recovered?: boolean;
  /** Session provenance for sidebar bucketing (see SessionOriginName). */
  origin: SessionOriginName;
  /** Free-form owner tag set by the creator (empty for anonymous UI creates). */
  owner: string;
  /** Human-facing label set by the creator (empty when unset). */
  display_label: string;
  tracking_degraded?: boolean;
  target?: TerminalTarget;
}

export interface PolicyResponse {
  session_id: string;
  policy: ExpirationPolicy;
  expires_at?: string;
  ttl_seconds?: number;
}

// Closed set of agent kinds the recovery flow knows how to reattach. Mirrors
// sessionstore.Agent in the API; new runtimes require an explicit change on
// both sides.
export type AgentType = "none" | "codex" | "claude" | "opencode" | "grok";

// Persistent-session-recovery surface. See
// scenarios/web-console/docs/guides/SESSION_RECOVERY.md for the contract.
export interface RecoverableSession {
  id: string;
  backend?: string;
  shell?: string;
  cols?: number;
  rows?: number;
  created_at?: string;
  orphaned_at?: string;
  last_activity_at?: string;
  agent_type?: AgentType;
	owner?: string;
	display_label?: string;
  agent_session_id?: string;
  launch_command?: string;
  cwd?: string;
  last_rollout_path?: string;
  recoverable: boolean;
  not_recoverable_reason?: string;
  pane_name?: string;
  header_color?: string;
  group_name?: string;
}

export interface RecoverResult {
  old_session_id: string;
  new_session_id: string;
  agent_type?: string;
  command_sent?: string;
  codex_home_copied?: boolean;
}

export type ArchivedRestoreState = "reopenable" | "read_only" | "nothing_to_restore";

export interface ArchivedSession {
  id: string;
  archived_at: string;
  created_at: string;
  agent_type: AgentType;
  agent_session_id?: string;
  cwd?: string;
  pane_name: string;
  header_color?: string;
  group_name?: string;
  message_count: number;
  restore_state: ArchivedRestoreState;
  restore_state_reason?: string;
  awaiting_recovery?: boolean;
}

export interface ArchiveRetentionSnapshot {
	policy: {
		message_less_age_days: number;
		agent_home_age_days: number;
		max_bytes: number;
	};
	stats: {
		entry_count: number;
		message_count: number;
		transcript_bytes: number;
		agent_home_bytes: number;
		total_bytes: number;
	};
}

// ---------------------------------------------------------------------------
// Decoders — proto wire shape → domain shape
// ---------------------------------------------------------------------------

type ProtoSession = {
  id: string;
  shell: string;
  createdAt: string;
  cols: number;
  rows: number;
  backend: string;
  survivesRestart: boolean;
  policy?: { mode: string; duration: string };
  recovered: boolean;
  origin?: SessionOrigin;
  owner?: string;
  displayLabel?: string;
  trackingDegraded?: boolean;
  target?: Target;
};

type ProtoRecoverable = {
  id: string;
  backend: string;
  shell: string;
  cols: number;
  rows: number;
  createdAt: string;
  orphanedAt: string;
  lastActivityAt: string;
  agentType: string;
  agentSessionId: string;
  launchCommand: string;
  cwd: string;
  lastRolloutPath: string;
  recoverable: boolean;
  notRecoverableReason: string;
  paneName: string;
  headerColor: string;
  groupName: string;
};

type ProtoPolicyView = {
  sessionId: string;
  policy?: { mode: string; duration: string };
  expiresAt: string;
  ttlSeconds: number;
  hasExpiry: boolean;
};

/** Map the SessionOrigin proto enum to the string form the sidebar buckets on. */
export function originName(o: SessionOrigin | undefined): SessionOriginName {
  switch (o) {
    case SessionOrigin.UI:
      return "ui";
    case SessionOrigin.PROGRAMMATIC:
      return "programmatic";
    case SessionOrigin.REMOTE:
      return "remote";
    default:
      return "unspecified";
  }
}

/** Coerce a free-form origin string (e.g. from the SSE lifecycle payload) into a
 *  closed SessionOriginName, defaulting unknown/empty to "unspecified". */
export function coerceOriginName(s: string | undefined): SessionOriginName {
  switch (s) {
    case "ui":
    case "programmatic":
    case "remote":
      return s;
    default:
      return "unspecified";
  }
}

export function decodeSession(s: ProtoSession | undefined): SessionInfo {
  const policy = s?.policy;
  return {
    id: s?.id ?? "",
    shell: s?.shell ?? "",
    created_at: s?.createdAt ?? "",
    cols: s?.cols ?? 0,
    rows: s?.rows ?? 0,
    backend: (s?.backend as BackendID) ?? "standard",
    survives_restart: s?.survivesRestart ?? false,
    policy: {
      mode: (policy?.mode as PolicyMode) ?? "never",
      ...(policy?.duration ? { duration: policy.duration } : {}),
    },
    ...(s?.recovered ? { recovered: true } : {}),
    origin: originName(s?.origin),
    owner: s?.owner ?? "",
    display_label: s?.displayLabel ?? "",
    ...(s?.trackingDegraded ? { tracking_degraded: true } : {}),
    ...(s?.target ? { target: decodeTarget(s.target) } : {}),
  };
}

function decodeRecoverable(r: ProtoRecoverable): RecoverableSession {
  return {
    id: r.id,
    backend: r.backend || undefined,
    shell: r.shell || undefined,
    cols: r.cols || undefined,
    rows: r.rows || undefined,
    created_at: r.createdAt || undefined,
    orphaned_at: r.orphanedAt || undefined,
    last_activity_at: r.lastActivityAt || undefined,
    agent_type: (r.agentType as AgentType) || undefined,
    agent_session_id: r.agentSessionId || undefined,
    launch_command: r.launchCommand || undefined,
    cwd: r.cwd || undefined,
    last_rollout_path: r.lastRolloutPath || undefined,
    recoverable: r.recoverable,
    not_recoverable_reason: r.notRecoverableReason || undefined,
    pane_name: r.paneName || undefined,
    header_color: r.headerColor || undefined,
    group_name: r.groupName || undefined,
  };
}

function decodePolicyView(v: ProtoPolicyView | undefined): PolicyResponse {
  return {
    session_id: v?.sessionId ?? "",
    policy: {
      mode: (v?.policy?.mode as PolicyMode) ?? "never",
      ...(v?.policy?.duration ? { duration: v.policy.duration } : {}),
    },
    ...(v?.hasExpiry
      ? {
          expires_at: v.expiresAt,
          ttl_seconds: v.ttlSeconds,
        }
      : {}),
  };
}

function restoreStateName(state: ArchiveRestoreState): ArchivedRestoreState {
  switch (state) {
    case ArchiveRestoreState.REOPENABLE:
      return "reopenable";
    case ArchiveRestoreState.NOTHING_TO_RESTORE:
      return "nothing_to_restore";
    default:
      return "read_only";
  }
}

// ---------------------------------------------------------------------------
// Typed wrappers
// ---------------------------------------------------------------------------

export async function createSession(opts?: {
  shell?: string;
  cols?: number;
  rows?: number;
  backend?: BackendID;
  policy?: { mode: PolicyMode; duration?: string };
  launch_command?: string;
  // execute_launch_command asks the server to run launch_command in the new
  // PTY (via server-side paste) so the command runs exactly once without the
  // client typing it after the WebSocket connects.
  execute_launch_command?: boolean;
	agent_type?: AgentType;
	owner?: string;
	display_label?: string;
	target_id?: string;
	working_dir?: string;
	tmux_mouse_mode?: boolean;
	idempotency_key?: string;
}): Promise<SessionInfo> {
  const resp = await sessionsClient.create({
    shell: opts?.shell ?? "",
    cols: opts?.cols ?? 0,
    rows: opts?.rows ?? 0,
    backend: opts?.backend ?? "",
    launchCommand: opts?.launch_command ?? "",
    executeLaunchCommand: opts?.execute_launch_command ?? false,
    agentType: opts?.agent_type ?? "",
		owner: opts?.owner ?? "",
    displayLabel: opts?.display_label ?? "",
	targetId: opts?.target_id ?? "",
	workingDir: opts?.working_dir ?? "",
	tmuxMouseMode: opts?.tmux_mouse_mode ?? false,
    // First-party UI client: tag provenance explicitly so an origin-less
    // create (which the server normalizes to programmatic) can only come from
    // a non-UI caller.
    origin: SessionOrigin.UI,
    ...(opts?.policy
      ? { policy: { mode: opts.policy.mode, duration: opts.policy.duration ?? "" }, hasPolicy: true }
      : {}),
  }, opts?.idempotency_key ? { headers: { "X-Idempotency-Key": opts.idempotency_key } } : undefined);
  return decodeSession(resp.session);
}

/**
 * Snapshot of startup persistent-session recovery, returned alongside the
 * session list. Reattaching surviving tmux sessions runs asynchronously so the
 * API serves immediately; while `in_progress` is true the list may be
 * incomplete and the UI should say so rather than imply the user has no
 * sessions. See session_lifecycle.go (RecoveryProgress) + sessions.proto.
 */
export interface RecoverySnapshot {
  in_progress: boolean;
  total: number;
  recovered: number;
  awaiting_recovery: number;
  adopted: number;
}

export async function listSessionsWithRecovery(): Promise<{
  sessions: SessionInfo[];
  recovery: RecoverySnapshot;
}> {
  const resp = await sessionsClient.list({});
  const r = resp.recovery;
  return {
	    sessions: resp.sessions.map(decodeSession),
    recovery: {
      in_progress: r?.inProgress ?? false,
      total: r?.total ?? 0,
      recovered: r?.recovered ?? 0,
      awaiting_recovery: r?.awaitingRecovery ?? 0,
      adopted: r?.adopted ?? 0,
    },
  };
}

export async function listSessions(): Promise<SessionInfo[]> {
  return (await listSessionsWithRecovery()).sessions;
}

export async function getSession(id: string): Promise<SessionInfo> {
	const resp = await sessionsClient.get({ id });
  return decodeSession(resp.session);
}

export async function deleteSession(id: string): Promise<void> {
	await sessionsClient.delete({ id });
}

export async function archiveSession(id: string): Promise<void> {
  await sessionsClient.archive({ id });
  window.dispatchEvent(new CustomEvent("web-console:archive-changed"));
}

export async function unarchiveSession(id: string): Promise<void> {
  await sessionsClient.unarchive({ id });
  window.dispatchEvent(new CustomEvent("web-console:archive-changed"));
}

export async function listArchivedSessions(): Promise<{ sessions: ArchivedSession[]; total: number }> {
  const response = await sessionsClient.listArchived({});
  return {
    sessions: response.sessions.map((session) => ({
      id: session.id,
      archived_at: session.archivedAt,
      created_at: session.createdAt,
      agent_type: (session.agentType as AgentType) || "none",
      agent_session_id: session.agentSessionId || undefined,
      cwd: session.cwd || undefined,
      pane_name: session.paneName || session.id.slice(0, 8),
      header_color: session.headerColor || undefined,
      group_name: session.groupName || undefined,
      message_count: Number(session.messageCount),
      restore_state: restoreStateName(session.restoreState),
      restore_state_reason: session.restoreStateReason || undefined,
      awaiting_recovery: session.awaitingRecovery,
    })),
    total: response.total,
  };
}

export async function getArchiveRetention(): Promise<ArchiveRetentionSnapshot> {
	const response = await sessionsClient.getArchiveRetention({});
	return {
		policy: {
			message_less_age_days: response.policy?.messageLessAgeDays ?? 0,
			agent_home_age_days: response.policy?.agentHomeAgeDays ?? 0,
			max_bytes: Number(response.policy?.maxBytes ?? 0),
		},
		stats: {
			entry_count: Number(response.stats?.entryCount ?? 0),
			message_count: Number(response.stats?.messageCount ?? 0),
			transcript_bytes: Number(response.stats?.transcriptBytes ?? 0),
			agent_home_bytes: Number(response.stats?.agentHomeBytes ?? 0),
			total_bytes: Number(response.stats?.totalBytes ?? 0),
		},
	};
}

export async function listRecoverableSessions(): Promise<RecoverableSession[]> {
  const resp = await sessionsClient.listRecoverable({});
  return resp.sessions.map(decodeRecoverable);
}

export async function recoverSession(oldId: string, idempotencyKey?: string): Promise<RecoverResult> {
  const resp = await sessionsClient.recover(
    { id: oldId },
    idempotencyKey ? { headers: { "X-Idempotency-Key": idempotencyKey } } : undefined,
  );
  return {
    old_session_id: resp.oldSessionId,
    new_session_id: resp.newSessionId,
    agent_type: resp.agentType || undefined,
    command_sent: resp.commandSent || undefined,
    codex_home_copied: resp.codexHomeCopied,
  };
}

export async function reopenSession(oldId: string, idempotencyKey: string): Promise<RecoverResult> {
  const resp = await sessionsClient.reopen(
    { id: oldId },
    { headers: { "X-Idempotency-Key": idempotencyKey } },
  );
  return {
    old_session_id: resp.oldSessionId,
    new_session_id: resp.newSessionId,
    agent_type: resp.agentType || undefined,
    command_sent: resp.commandSent || undefined,
    codex_home_copied: resp.codexHomeCopied,
  };
}

export async function dismissRecoverableSession(oldId: string): Promise<void> {
  await sessionsClient.dismissRecoverable({ id: oldId });
}

// [REQ:P1-001a] Session Policy API - client
export async function getSessionPolicy(id: string): Promise<PolicyResponse> {
  const resp = await sessionsClient.getPolicy({ id });
  return decodePolicyView(resp.policy);
}

export async function updateSessionPolicy(
  id: string,
  policy: { mode: string; duration?: string },
): Promise<PolicyResponse> {
  const resp = await sessionsClient.updatePolicy({
    id,
    policy: { mode: policy.mode, duration: policy.duration ?? "" },
  });
  return decodePolicyView(resp.policy);
}

// [REQ:P0-004b] api-base WebSocket Integration — session terminal WS URL.
// Kept in the sessions domain because the WS endpoint is part of the
// session lifecycle even though it bypasses Connect-RPC.
export function buildSessionWsUrl(sessionId: string, device?: { id: string; label: string; deviceClass?: string }): string {
  const apiBase = API_BASE_WITH_SUFFIX;
  const wsBase = apiBase.startsWith("https://")
    ? `wss://${apiBase.slice("https://".length)}`
    : apiBase.startsWith("http://")
      ? `ws://${apiBase.slice("http://".length)}`
      : apiBase;
  const url = buildWsUrl(`/sessions/${sessionId}/ws`, { baseUrl: wsBase });
  if (!device) return url;
  const query = new URLSearchParams({ deviceId: device.id, deviceLabel: device.label });
  if (device.deviceClass) query.set("deviceClass", device.deviceClass);
  return `${url}${url.includes("?") ? "&" : "?"}${query}`;
}
