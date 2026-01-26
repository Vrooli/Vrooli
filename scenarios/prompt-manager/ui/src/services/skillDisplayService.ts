/**
 * Service for displaying multiple skills into various output formats.
 */

import type { Skill } from '@/types'
import type { DisplayFormat, DisplayResponse } from '@/types/world'

/**
 * Escape XML special characters.
 */
function escapeXml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&apos;')
}

/**
 * Estimate token count (rough approximation).
 * Uses ~4 characters per token as average.
 */
function estimateTokens(text: string): number {
  return Math.ceil(text.length / 4)
}

/**
 * Display skills into XML format.
 */
function displayToXml(skills: Skill[]): string {
  const lines: string[] = [
    `<skills count="${skills.length}">`,
  ]

  skills.forEach((skill) => {
    lines.push(`  <skill id="${escapeXml(skill.id)}" name="${escapeXml(skill.name)}"><![CDATA[`)
    lines.push(skill.content || '')
    lines.push(`]]></skill>`)
  })

  lines.push(`</skills>`)

  return lines.join('\n')
}

/**
 * Display skills into Markdown format.
 */
function displayToMarkdown(skills: Skill[]): string {
  const lines: string[] = [
    `# Combined Skills (${skills.length})`,
    '',
    `*Generated: ${new Date().toISOString()}*`,
    '',
    '---',
    '',
  ]

  skills.forEach((skill, index) => {
    const modes = skill.modes.join(' / ')
    const tags = skill.tags.map((t) => `\`${t}\``).join(' ')

    lines.push(`## ${index + 1}. ${skill.name}`)
    lines.push('')

    if (skill.description) {
      lines.push(`> ${skill.description}`)
      lines.push('')
    }

    if (modes) {
      lines.push(`**Modes:** ${modes}`)
    }

    if (tags) {
      lines.push(`**Tags:** ${tags}`)
    }

    if (modes || tags) {
      lines.push('')
    }

    lines.push('### Content')
    lines.push('')
    lines.push('```')
    lines.push(skill.content || '')
    lines.push('```')
    lines.push('')
    lines.push('---')
    lines.push('')
  })

  return lines.join('\n')
}

/**
 * Display skills into JSON format.
 */
function displayToJson(skills: Skill[]): string {
  const data = {
    combined: true,
    count: skills.length,
    generated: new Date().toISOString(),
    skills: skills.map((p) => ({
      id: p.id,
      name: p.name,
      description: p.description,
      modes: p.modes,
      tags: p.tags,
      content: p.content,
    })),
  }

  return JSON.stringify(data, null, 2)
}

/**
 * Display multiple skills into the specified format.
 */
export function displaySkills(
  skills: Skill[],
  format: DisplayFormat = 'xml'
): DisplayResponse {
  if (skills.length === 0) {
    return {
      combined: '',
      skillCount: 0,
      totalTokens: 0,
      format,
    }
  }

  let combined: string

  switch (format) {
    case 'xml':
      combined = displayToXml(skills)
      break
    case 'markdown':
      combined = displayToMarkdown(skills)
      break
    case 'json':
      combined = displayToJson(skills)
      break
    default:
      combined = displayToXml(skills)
  }

  return {
    combined,
    skillCount: skills.length,
    totalTokens: estimateTokens(combined),
    format,
  }
}

/**
 * Generate a preview of displayed skills (first N characters).
 */
export function generatePreview(
  skills: Skill[],
  format: DisplayFormat = 'xml',
  maxLength: number = 500
): string {
  const result = displaySkills(skills, format)

  if (result.combined.length <= maxLength) {
    return result.combined
  }

  return result.combined.substring(0, maxLength) + '\n...[truncated]'
}

/**
 * Validate skills for display.
 */
export function validateForDisplay(skills: Skill[]): {
  valid: boolean
  errors: string[]
  warnings: string[]
} {
  const errors: string[] = []
  const warnings: string[] = []

  if (skills.length === 0) {
    errors.push('No skills selected')
  }

  if (skills.length === 1) {
    warnings.push('Only one skill selected - display is optional')
  }

  // Check for missing content
  const missingContent = skills.filter((p) => !p.content.trim())
  if (missingContent.length > 0) {
    warnings.push(
      `${missingContent.length} skill(s) have no content: ${missingContent.map((p) => p.name).join(', ')}`
    )
  }

  // Check for draft skills
  const drafts = skills.filter((p) => p.draft)
  if (drafts.length > 0) {
    warnings.push(
      `${drafts.length} skill(s) are drafts: ${drafts.map((p) => p.name).join(', ')}`
    )
  }

  // Estimate displayed size
  const totalContent = skills.reduce(
    (sum, p) => sum + p.content.length,
    0
  )
  if (totalContent > 50000) {
    warnings.push(
      `Displayed content is large (~${Math.round(totalContent / 1000)}KB). Consider selecting fewer skills.`
    )
  }

  return {
    valid: errors.length === 0,
    errors,
    warnings,
  }
}

/**
 * Copy displayed content to clipboard.
 */
export async function copyToClipboard(
  skills: Skill[],
  format: DisplayFormat = 'xml'
): Promise<{ success: boolean; error?: string }> {
  try {
    const result = displaySkills(skills, format)
    await navigator.clipboard.writeText(result.combined)
    return { success: true }
  } catch (error) {
    return {
      success: false,
      error: error instanceof Error ? error.message : 'Failed to copy',
    }
  }
}
