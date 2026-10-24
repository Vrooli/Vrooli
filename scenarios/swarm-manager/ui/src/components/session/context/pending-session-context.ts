import type { AgentSessionContextType, AgentSessionKind } from "../../../types";
import { CONTEXT_TYPE_CAPS, sessionKindAllowsContextType, totalContextCapForKind } from "./session-context-config";
import { contextKey, type SessionContextOption } from "./session-context-refs";

const STORAGE_KEY = "swarm-manager.pending-session-context.v1";

type StagedContextBySession = Record<string, SessionContextOption[]>;

export interface MergeContextResult {
  items: SessionContextOption[];
  applied: SessionContextOption[];
  rejected: Array<{ option: SessionContextOption; reason: string }>;
}

export function stageContextForSession(sessionId: string, option: SessionContextOption): void {
  if (!sessionId) return;
  const staged = readStagedContext();
  const existing = staged[sessionId] ?? [];
  staged[sessionId] = mergeUnique(existing, [option]);
  writeStagedContext(staged);
}

export function peekStagedContextForSession(sessionId: string): SessionContextOption[] {
  return readStagedContext()[sessionId] ?? [];
}

export function clearStagedContextForSession(sessionId: string, applied: SessionContextOption[]): void {
  if (applied.length === 0) return;
  const staged = readStagedContext();
  const appliedKeys = new Set(applied.map((option) => contextKey(option.type, option.ref)));
  const remaining = (staged[sessionId] ?? []).filter((option) => !appliedKeys.has(contextKey(option.type, option.ref)));
  if (remaining.length > 0) {
    staged[sessionId] = remaining;
  } else {
    delete staged[sessionId];
  }
  writeStagedContext(staged);
}

export function mergeContextOptions(
  current: SessionContextOption[],
  incoming: SessionContextOption[],
  sessionKind: AgentSessionKind,
): MergeContextResult {
  const next = [...current];
  const applied: SessionContextOption[] = [];
  const rejected: MergeContextResult["rejected"] = [];
  const keys = new Set(current.map((option) => contextKey(option.type, option.ref)));

  for (const option of incoming) {
    const key = contextKey(option.type, option.ref);
    if (keys.has(key)) {
      applied.push(option);
      continue;
    }
    const reason = rejectionReason(next, option, sessionKind);
    if (reason) {
      rejected.push({ option, reason });
      continue;
    }
    next.push(option);
    keys.add(key);
    applied.push(option);
  }

  return { items: next, applied, rejected };
}

function rejectionReason(
  current: SessionContextOption[],
  option: SessionContextOption,
  sessionKind: AgentSessionKind,
): string | null {
  if (!sessionKindAllowsContextType(sessionKind, option.type)) {
    return `${labelForType(option.type)} context is not allowed for this session kind.`;
  }
  const totalCap = totalContextCapForKind(sessionKind);
  if (current.length >= totalCap) {
    return `This session kind allows ${totalCap} context items per message.`;
  }
  const typeCap = CONTEXT_TYPE_CAPS[option.type];
  const typeCount = current.filter((item) => item.type === option.type).length;
  if (typeCount >= typeCap) {
    return `${labelForType(option.type)} allows ${typeCap} selections.`;
  }
  return null;
}

function mergeUnique(current: SessionContextOption[], incoming: SessionContextOption[]): SessionContextOption[] {
  const keys = new Set(current.map((option) => contextKey(option.type, option.ref)));
  const next = [...current];
  for (const option of incoming) {
    const key = contextKey(option.type, option.ref);
    if (!keys.has(key)) {
      keys.add(key);
      next.push(option);
    }
  }
  return next;
}

function readStagedContext(): StagedContextBySession {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return {};
    const parsed = JSON.parse(raw) as unknown;
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) return {};
    const staged: StagedContextBySession = {};
    for (const [sessionId, value] of Object.entries(parsed as Record<string, unknown>)) {
      if (!Array.isArray(value)) continue;
      staged[sessionId] = value.filter(isSessionContextOption);
    }
    return staged;
  } catch {
    return {};
  }
}

function writeStagedContext(staged: StagedContextBySession): void {
  try {
    if (Object.keys(staged).length === 0) {
      window.localStorage.removeItem(STORAGE_KEY);
      return;
    }
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(staged));
  } catch {
    // Storage loss should not block the operator from staying on the page.
  }
}

function isSessionContextOption(value: unknown): value is SessionContextOption {
  if (!value || typeof value !== "object") return false;
  const candidate = value as Partial<SessionContextOption>;
  return typeof candidate.type === "string"
    && typeof candidate.ref === "string"
    && typeof candidate.title === "string";
}

function labelForType(type: AgentSessionContextType): string {
  return type.replace(/_/g, " ");
}
