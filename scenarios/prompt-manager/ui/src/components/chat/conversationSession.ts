/**
 * Durable conversation session identity.
 *
 * A conversation belongs to one immutable identity: a base agent, or an agent
 * acting as a member of one specific team. The same agent may hold several
 * memberships, so the identity key includes the team. Switching identity selects
 * or starts a different session; it never mutates a live conversation's identity
 * or replays another context's history.
 *
 * The module is storage-injected so it can be exercised without a browser, and
 * the default storage is localStorage. Consumers (the 3-D world and the agent
 * detail view) share this one contract.
 */

export interface ConversationIdentity {
  agentId: string
  teamId?: string
}

export interface PendingTurn {
  requestId: string
  message: string
}

/**
 * A first turn that has been dispatched but whose accepted run is not yet
 * recorded. The server derives the conversation task id from `requestId`, so
 * persisting it before the request lets a reload/retry reuse the same identity
 * instead of starting a second conversation when the first response was lost.
 */
export interface PendingStart {
  requestId: string
  message: string
}

export interface ConversationSession {
  key: string
  agentId: string
  teamId?: string
  runId: string
  firstRequestId: string
  pendingTurn?: PendingTurn
  updatedAt: string
}

export interface SessionStorage {
  getItem(key: string): string | null
  setItem(key: string, value: string): void
  removeItem(key: string): void
}

const STORAGE_PREFIX = 'prompt-manager.conversation.'

/** Stable key for an identity. A missing/blank team is the base-agent context. */
export function conversationIdentityKey(identity: ConversationIdentity): string {
  const team = identity.teamId?.trim()
  return `${identity.agentId.trim()}::${team ? team : 'base'}`
}

/** True when a stored session belongs to exactly this identity. */
export function isSessionCompatible(
  session: Pick<ConversationSession, 'key'>,
  identity: ConversationIdentity,
): boolean {
  return session.key === conversationIdentityKey(identity)
}

function resolveStorage(storage?: SessionStorage): SessionStorage | null {
  if (storage) return storage
  if (typeof window !== 'undefined') return window.localStorage
  return null
}

/**
 * Read the durable session for an identity. A malformed or mismatched record is
 * treated as absent so a corrupt or stale entry cannot leak another context.
 */
export function readConversationSession(
  identity: ConversationIdentity,
  storage?: SessionStorage,
): ConversationSession | null {
  const store = resolveStorage(storage)
  if (!store) return null
  const raw = store.getItem(STORAGE_PREFIX + conversationIdentityKey(identity))
  if (!raw) return null
  try {
    const parsed = JSON.parse(raw) as ConversationSession
    if (typeof parsed.runId !== 'string' || !isSessionCompatible(parsed, identity)) {
      return null
    }
    return parsed
  } catch {
    return null
  }
}

/** Persist a session under its own identity key. */
export function writeConversationSession(
  session: ConversationSession,
  storage?: SessionStorage,
): void {
  const store = resolveStorage(storage)
  if (!store) return
  store.setItem(STORAGE_PREFIX + session.key, JSON.stringify(session))
}

/** Forget the session for an identity (for example after an explicit end). */
export function clearConversationSession(
  identity: ConversationIdentity,
  storage?: SessionStorage,
): void {
  const store = resolveStorage(storage)
  if (!store) return
  store.removeItem(STORAGE_PREFIX + conversationIdentityKey(identity))
}

const PENDING_START_SUFFIX = '.pending-start'

/** Read a dispatched-but-unconfirmed first turn for an identity. */
export function readPendingStart(
  identity: ConversationIdentity,
  storage?: SessionStorage,
): PendingStart | null {
  const store = resolveStorage(storage)
  if (!store) return null
  const raw = store.getItem(STORAGE_PREFIX + conversationIdentityKey(identity) + PENDING_START_SUFFIX)
  if (!raw) return null
  try {
    const parsed = JSON.parse(raw) as PendingStart
    if (typeof parsed.requestId !== 'string' || typeof parsed.message !== 'string') return null
    return parsed
  } catch {
    return null
  }
}

/** Persist the first-turn identity before dispatching it. */
export function writePendingStart(
  identity: ConversationIdentity,
  pending: PendingStart,
  storage?: SessionStorage,
): void {
  const store = resolveStorage(storage)
  if (!store) return
  store.setItem(STORAGE_PREFIX + conversationIdentityKey(identity) + PENDING_START_SUFFIX, JSON.stringify(pending))
}

/** Forget the first-turn identity once the accepted run is recorded. */
export function clearPendingStart(
  identity: ConversationIdentity,
  storage?: SessionStorage,
): void {
  const store = resolveStorage(storage)
  if (!store) return
  store.removeItem(STORAGE_PREFIX + conversationIdentityKey(identity) + PENDING_START_SUFFIX)
}

/** Build a fresh session record for an accepted first turn. */
export function createConversationSession(
  identity: ConversationIdentity,
  runId: string,
  firstRequestId: string,
): ConversationSession {
  return {
    key: conversationIdentityKey(identity),
    agentId: identity.agentId,
    teamId: identity.teamId?.trim() || undefined,
    runId,
    firstRequestId,
    updatedAt: new Date().toISOString(),
  }
}

/**
 * Merge a patch into an existing session. Returns the updated record, or null
 * when no compatible session exists. This is how a pending turn is recorded so a
 * reload can resume the same accepted conversation instead of sending again.
 */
export function updateConversationSession(
  identity: ConversationIdentity,
  patch: Partial<Omit<ConversationSession, 'key'>>,
  storage?: SessionStorage,
): ConversationSession | null {
  const current = readConversationSession(identity, storage)
  if (!current) return null
  const next: ConversationSession = { ...current, ...patch, key: current.key, updatedAt: new Date().toISOString() }
  writeConversationSession(next, storage)
  return next
}
