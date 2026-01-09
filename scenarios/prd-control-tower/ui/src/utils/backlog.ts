import { buildApiUrl } from './apiClient'
import type {
  BacklogConvertResponse,
  BacklogConvertResult,
  BacklogCreateResponse,
  BacklogEntry,
  EntityType,
} from '../types'

export interface PendingIdea {
  id: string
  ideaText: string
  entityType: EntityType
  suggestedName: string
  notes: string
}

const bulletPattern = /^[\s\-*•‣∙·]+/

function stripLeadingNumber(value: string): string {
  return value.replace(/^\d+[.)]\s*/, '').trim()
}

export function sanitizeSlugInput(value: string): string {
  return value
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-+|-+$/g, '')
}

export function slugifyIdea(input: string): string {
  let slug = input
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-+|-+$/g, '')

  if (!slug) {
    slug = `idea-${Date.now()}`
  }

  if (slug.length > 80) {
    slug = slug.slice(0, 80)
  }

  return slug
}

/**
 * Auto-detect entity type from idea text using keyword matching
 */
function autoDetectEntityType(text: string): EntityType | null {
  const lower = text.toLowerCase()

  // Resource keywords
  const resourceKeywords = [
    'resource', 'integration', 'api wrapper', 'client library', 'sdk',
    'connector', 'adapter', 'driver', 'plugin', 'extension'
  ]

  // Scenario keywords
  const scenarioKeywords = [
    'scenario', 'app', 'application', 'tool', 'service', 'platform',
    'dashboard', 'manager', 'builder', 'generator', 'analyzer', 'monitor',
    'automation', 'workflow', 'system', 'studio', 'hub', 'assistant'
  ]

  // Check for explicit prefixes first
  if (resourceKeywords.some(keyword => lower.includes(keyword))) {
    return 'resource'
  }

  if (scenarioKeywords.some(keyword => lower.includes(keyword))) {
    return 'scenario'
  }

  return null
}

export function parseBacklogInput(raw: string, defaultType: EntityType = 'scenario'): PendingIdea[] {
  if (!raw.trim()) {
    return []
  }

  const fallback = defaultType ?? 'scenario'
  const lines = raw.split(/\r?\n/)
  const ideas: PendingIdea[] = []

  lines.forEach((line, index) => {
    let text = line.trim()
    if (!text) {
      return
    }

    text = text.replace(bulletPattern, '').trim()
    text = stripLeadingNumber(text)

    let entityType: EntityType = fallback
    const lower = text.toLowerCase()

    // Explicit type markers
    if (lower.startsWith('[scenario]')) {
      entityType = 'scenario'
      text = text.slice(10).trim()
    } else if (lower.startsWith('[resource]')) {
      entityType = 'resource'
      text = text.slice(10).trim()
    } else {
      // Auto-detect from content
      const detected = autoDetectEntityType(text)
      if (detected) {
        entityType = detected
      }
    }

    if (!text) {
      return
    }

    const suggestedName = slugifyIdea(text)
    ideas.push({
      id: `${index}-${suggestedName}`,
      ideaText: text,
      entityType,
      suggestedName,
      notes: '',
    })
  })

  return ideas
}

export async function createBacklogEntriesRequest(ideas: PendingIdea[]): Promise<BacklogEntry[]> {
  const payload = {
    entries: ideas.map((idea) => ({
      idea_text: idea.ideaText,
      entity_type: idea.entityType,
      suggested_name: idea.suggestedName,
      notes: idea.notes,
    })),
  }

  const response = await fetch(buildApiUrl('/backlog'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })

  if (!response.ok) {
    const message = await response.text()
    throw new Error(message || 'Failed to add backlog items')
  }

  const data: BacklogCreateResponse = await response.json()
  return data.entries
}

export async function convertBacklogEntriesRequest(ids: string[]): Promise<BacklogConvertResult[]> {
  const response = await fetch(buildApiUrl('/backlog/convert'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ entry_ids: ids }),
  })

  if (!response.ok) {
    const message = await response.text()
    throw new Error(message || 'Failed to convert backlog entries')
  }

  const data: BacklogConvertResponse = await response.json()
  return data.results || []
}

export async function updateBacklogEntryRequest(id: string, payload: { notes: string }): Promise<BacklogEntry> {
  const response = await fetch(buildApiUrl(`/backlog/${id}`), {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })

  if (!response.ok) {
    const message = await response.text()
    throw new Error(message || 'Failed to update backlog entry')
  }

  return (await response.json()) as BacklogEntry
}
