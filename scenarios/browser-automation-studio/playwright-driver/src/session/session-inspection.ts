import type { Config } from '../config';
import type { SessionPhase, SessionState } from '../types';

export interface SessionInfo {
  id: string;
  phase: SessionPhase;
  instructionCount: number;
  createdAt: string;
  lastUsedAt: string;
  isRecording: boolean;
  url: string;
}

export interface SessionSummary {
  total: number;
  active: number;
  idle: number;
  active_recordings: number;
  idle_timeout_ms: number;
  capacity: number;
}

export interface SessionListEntry {
  id: string;
  phase: SessionPhase;
  created_at: string;
  last_used_at: string;
  idle_time_ms: number;
  is_idle: boolean;
  is_recording: boolean;
  instruction_count: number;
  owner_execution_id: string;
  workflow_id?: string;
  current_url?: string;
  page_count: number;
}

export function inspectSession(session: SessionState): SessionInfo {
  let url = 'about:blank';
  try { if (!session.page.isClosed()) url = session.page.url(); } catch { /* closed page */ }
  return { id: session.id, phase: session.phase, instructionCount: session.instructionCount, createdAt: session.createdAt.toISOString(), lastUsedAt: session.lastUsedAt.toISOString(), isRecording: session.pipelineManager?.isRecording() ?? false, url };
}

export function summarizeSessions(sessions: Iterable<SessionState>, config: Config, now = Date.now()): SessionSummary {
  let total = 0; let active = 0; let activeRecordings = 0;
  for (const session of sessions) {
    total++;
    if (now - session.lastUsedAt.getTime() < config.session.idleTimeoutMs) active++;
    if (session.pipelineManager?.isRecording()) activeRecordings++;
  }
  return { total, active, idle: total - active, active_recordings: activeRecordings, idle_timeout_ms: config.session.idleTimeoutMs, capacity: config.session.maxConcurrent };
}

export function listSessions(sessions: Iterable<SessionState>, config: Config, now = Date.now()): SessionListEntry[] {
  const list: SessionListEntry[] = [];
  for (const session of sessions) {
    const idleTimeMs = now - session.lastUsedAt.getTime();
    let currentUrl: string | undefined;
    try { currentUrl = session.page.url(); } catch { /* closed page */ }
    list.push({ id: session.id, phase: session.phase, created_at: session.createdAt.toISOString(), last_used_at: session.lastUsedAt.toISOString(), idle_time_ms: idleTimeMs, is_idle: idleTimeMs >= config.session.idleTimeoutMs, is_recording: session.pipelineManager?.isRecording() ?? false, instruction_count: session.instructionCount, owner_execution_id: session.ownerExecutionId, workflow_id: session.spec.workflow_id, current_url: currentUrl, page_count: session.pages.length });
  }
  return list.sort((a, b) => new Date(b.last_used_at).getTime() - new Date(a.last_used_at).getTime());
}

export function countActiveSessions(sessions: Iterable<SessionState>, idleTimeoutMs: number, now = Date.now()): number {
  let count = 0;
  for (const session of sessions) if (now - session.lastUsedAt.getTime() < idleTimeoutMs) count++;
  return count;
}
