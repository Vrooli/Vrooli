import { fetchScenarioIssuesStatus } from '../services/issues'
import type { ScenarioIssuesSummary } from '../types'

const CACHE_TTL_MS = 2 * 60 * 1000

type ScenarioIssuesStatus = 'idle' | 'loading' | 'ready' | 'error'

export interface ScenarioIssuesEntry {
  status: ScenarioIssuesStatus
  summary: ScenarioIssuesSummary | null
  error: string | null
  fetchedAt: number | null
  stale: boolean
}

const defaultEntry: ScenarioIssuesEntry = {
  status: 'idle',
  summary: null,
  error: null,
  fetchedAt: null,
  stale: false,
}

function buildKey(entityType?: string, entityName?: string): string | null {
  if (!entityType || !entityName) return null
  return `${entityType}:${entityName}`.toLowerCase()
}

class ScenarioIssuesStore {
  private entries = new Map<string, ScenarioIssuesEntry>()
  private pending = new Map<string, Promise<ScenarioIssuesEntry>>()
  private listeners = new Set<() => void>()

  subscribe(listener: () => void): () => void {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  private emit() {
    this.listeners.forEach((listener) => listener())
  }

  getEntry(entityType?: string, entityName?: string): ScenarioIssuesEntry {
    const key = buildKey(entityType, entityName)
    if (!key) return defaultEntry
    return this.entries.get(key) ?? defaultEntry
  }

  async fetch(entityType: string, entityName: string, options?: { force?: boolean }): Promise<ScenarioIssuesEntry> {
    const key = buildKey(entityType, entityName)
    if (!key) return defaultEntry

    const force = options?.force ?? false
    const existing = this.entries.get(key)
    const now = Date.now()

    if (
      !force &&
      existing &&
      existing.status === 'ready' &&
      existing.fetchedAt &&
      !existing.stale &&
      now - existing.fetchedAt < CACHE_TTL_MS
    ) {
      return existing
    }

    if (!force && this.pending.has(key)) {
      return this.pending.get(key) as Promise<ScenarioIssuesEntry>
    }

    const request = (async () => {
      this.entries.set(key, {
        ...(existing ?? defaultEntry),
        status: 'loading',
        error: null,
      })
      this.emit()

      try {
        const summary = await fetchScenarioIssuesStatus(entityType, entityName, { useCache: !force })
        const entry: ScenarioIssuesEntry = {
          status: 'ready',
          summary,
          error: null,
          fetchedAt: Date.now(),
          stale: summary?.stale ?? false,
        }
        this.entries.set(key, entry)
        this.emit()
        return entry
      } catch (error) {
        const entry: ScenarioIssuesEntry = {
          status: 'error',
          summary: existing?.summary ?? null,
          error: error instanceof Error ? error.message : 'Failed to load issue status',
          fetchedAt: Date.now(),
          stale: true,
        }
        this.entries.set(key, entry)
        this.emit()
        throw error
      } finally {
        this.pending.delete(key)
      }
    })()

    this.pending.set(key, request)
    return request
  }

  flagIssueReported(entityType: string, entityName: string) {
    const key = buildKey(entityType, entityName)
    if (!key) return
    const existing = this.entries.get(key)
    if (!existing) return
    this.entries.set(key, { ...existing, stale: true })
    this.emit()
  }
}

export const scenarioIssuesStore = new ScenarioIssuesStore()
