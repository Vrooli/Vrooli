// DOC: docs/reference/cross-platform-effort/machine-linking-ux-2026-08-26.html
//
// The machine axis is app state, not a page filter: one control in the header
// sets the subject of every panel below it. Everything a surface needs to say
// about a machine — is it answering, how old is the reading, what may we do to
// it — is derived here once, so the picker, the identity strip and any future
// machine-aware panel cannot drift into three different vocabularies for the
// same node facts.
import type { Machine } from '../../types';

/**
 * How a machine is currently answering.
 *
 *  - `local`       this computer; always answers, never has a grant
 *  - `live`        a remote machine that is dispatchable with a fresh heartbeat
 *  - `stale`       registered and dispatchable, but the heartbeat has aged out
 *  - `unreachable` cannot be dispatched to at all
 */
export type PresenceTone = 'local' | 'live' | 'stale' | 'unreachable';

/** What a caller is permitted to do on a machine, shortest truthful form. */
export type GrantLevel = 'local' | 'none' | 'read' | 'operate' | 'full';

export interface MachinePresence {
  tone: PresenceTone;
  isLocal: boolean;
  /** "linux · x86_64" or "darwin · amd64 · 8s ago". */
  meta: string;
  /** "darwin/amd64" — the platform alone, for prose that already names the machine. */
  platform: string;
  /** Age of the last heartbeat, or undefined when the machine never reported one. */
  age?: string;
  /**
   * Why this machine cannot answer. Present only when `tone` is `unreachable`,
   * so a caller can render the reason instead of inventing one.
   */
  unreachableReason?: string;
  grant: GrantLevel;
  /** Chip text for `grant`: "read only", "operate", … Empty for the local machine. */
  grantLabel: string;
  /** The full operator-facing grant sentence from the API, when it sent one. */
  grantDetail?: string;
}

/**
 * Compact age for a presence line. Distinct from formatDurationSeconds because
 * a machine that has been silent for a week must read as "7d", not "177h 41m":
 * the reader is judging whether the number is alarming, not measuring it.
 */
export const formatPresenceAge = (seconds: number): string => {
  const rounded = Math.max(0, Math.round(seconds));
  if (rounded < 60) return `${rounded}s`;
  if (rounded < 3600) return `${Math.floor(rounded / 60)}m`;
  if (rounded < 86400) return `${Math.floor(rounded / 3600)}h`;
  return `${Math.floor(rounded / 86400)}d`;
};

/**
 * Reduce concrete Bridge scopes to the level a person acts on. Scope syntax is
 * kept out of product controls; `grantDetail` carries the API's own sentence
 * for anyone who wants the long form.
 */
export const grantLevelFor = (machine: Machine): GrantLevel => {
  if (!machine.id) return 'local';
  const effects = (machine.scopes ?? [])
    .map(scope => scope.toLowerCase().trim().split(':')[1] ?? '')
    .filter(Boolean);
  if (effects.includes('destructive') || effects.includes('*')) return 'full';
  if (effects.includes('write')) return 'operate';
  if (effects.includes('read')) return 'read';
  // A node with no scopes at all can still be listed; say so rather than
  // implying read access the caller does not hold.
  return 'none';
};

const GRANT_LABELS: Record<GrantLevel, string> = {
  local: '',
  none: 'no actions',
  read: 'read only',
  operate: 'operate',
  full: 'full control'
};

/**
 * The single derivation of a machine's presence. Reachability is decided here
 * so it can be shown *in* the picker rather than as an error after a choice:
 * choosing a machine that cannot answer should be visibly a choice.
 */
export const describeMachine = (machine: Machine): MachinePresence => {
  const isLocal = !machine.id;
  const platformParts = [machine.os, machine.arch].filter(Boolean) as string[];
  const platform = platformParts.length > 0 ? platformParts.join('/') : 'unknown platform';
  const age = machine.heartbeat_age_seconds === undefined
    ? undefined
    : formatPresenceAge(machine.heartbeat_age_seconds);

  const grant = grantLevelFor(machine);
  const base = {
    isLocal,
    platform,
    age,
    grant,
    grantLabel: GRANT_LABELS[grant],
    grantDetail: machine.grant
  };

  if (isLocal) {
    return { ...base, tone: 'local', meta: platformParts.join(' · ') || 'this computer' };
  }

  const metaParts = [...platformParts];
  if (age) metaParts.push(`${age} ago`);
  const meta = metaParts.join(' · ');

  if (!machine.dispatchable) {
    // Prefer the specific readiness fact that failed. "not responding" is the
    // honest summary only when the heartbeat is what aged out; a node can be
    // undispatchable while still beating.
    const failed = (machine.readiness ?? []).filter(fact => !fact.passed).map(fact => fact.identity);
    const unreachableReason = failed.includes('heartbeat_fresh')
      ? 'not responding'
      : failed.includes('channel_held')
      ? 'no open channel'
      : failed.includes('protocol_compatible')
      ? 'incompatible agent version'
      : failed.includes('registry_record')
      ? 'not registered'
      : 'not dispatchable';
    return {
      ...base,
      tone: 'unreachable',
      unreachableReason,
      meta: age ? `${unreachableReason} · ${age} ago` : unreachableReason
    };
  }

  return { ...base, tone: machine.heartbeat_fresh ? 'live' : 'stale', meta };
};

/**
 * The tone a surface should render for a subject: the machine's own presence,
 * narrowed by whether the polling loop is currently failing. A machine that
 * cannot be dispatched to outranks staleness — there is no "last reading" to
 * show for a channel that was never open.
 */
export const machinePresenceTone = (presence: MachinePresence, isStale: boolean): PresenceTone => {
  if (presence.tone === 'unreachable') return 'unreachable';
  if (isStale && !presence.isLocal) return 'stale';
  return presence.tone;
};

/**
 * Order for the picker: this computer first, then machines that can answer,
 * then the ones that cannot — each group alphabetical. A machine that stopped
 * answering keeps its place in the list rather than disappearing, because a
 * disappearing row reads as "deleted" and sends people to look for it.
 */
export const sortMachinesForPicker = (machines: Machine[]): Machine[] => {
  const rank = (machine: Machine): number => {
    if (!machine.id) return 0;
    return machine.dispatchable ? 1 : 2;
  };
  return [...machines].sort((a, b) => {
    const byRank = rank(a) - rank(b);
    return byRank !== 0 ? byRank : a.name.localeCompare(b.name);
  });
};
