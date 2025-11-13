import type { EntityType } from '../types'

export interface PendingIdea {
  id: string
  ideaText: string
  entityType: EntityType
  suggestedName: string
}

const bulletPattern = /^[\s\-\*•‣∙·]+/

function stripLeadingNumber(value: string): string {
  return value.replace(/^\d+[\.)]\s*/, '').trim()
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
    if (lower.startsWith('[scenario]')) {
      entityType = 'scenario'
      text = text.slice(10).trim()
    } else if (lower.startsWith('[resource]')) {
      entityType = 'resource'
      text = text.slice(10).trim()
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
    })
  })

  return ideas
}
