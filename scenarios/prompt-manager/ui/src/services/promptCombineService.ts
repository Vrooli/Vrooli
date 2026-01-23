/**
 * Service for combining multiple prompts into various output formats.
 */

import type { Prompt } from '@/types'
import type { CombineFormat, CombineResponse } from '@/types/world'

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
 * Combine prompts into XML format.
 */
function combineToXml(prompts: Prompt[]): string {
  const lines: string[] = [
    `<?xml version="1.0" encoding="UTF-8"?>`,
    `<combined-prompts count="${prompts.length}">`,
  ]

  prompts.forEach((prompt) => {
    const modes = prompt.modes.join('/')
    const tags = prompt.tags.join(', ')

    lines.push(`  <prompt id="${escapeXml(prompt.id)}" name="${escapeXml(prompt.name)}"${modes ? ` modes="${escapeXml(modes)}"` : ''}>`)

    if (prompt.description) {
      lines.push(`    <description>${escapeXml(prompt.description)}</description>`)
    }

    if (tags) {
      lines.push(`    <tags>${escapeXml(tags)}</tags>`)
    }

    lines.push(`    <content><![CDATA[`)
    lines.push(prompt.content || '')
    lines.push(`]]></content>`)
    lines.push(`  </prompt>`)
  })

  lines.push(`</combined-prompts>`)

  return lines.join('\n')
}

/**
 * Combine prompts into Markdown format.
 */
function combineToMarkdown(prompts: Prompt[]): string {
  const lines: string[] = [
    `# Combined Prompts (${prompts.length})`,
    '',
    `*Generated: ${new Date().toISOString()}*`,
    '',
    '---',
    '',
  ]

  prompts.forEach((prompt, index) => {
    const modes = prompt.modes.join(' / ')
    const tags = prompt.tags.map((t) => `\`${t}\``).join(' ')

    lines.push(`## ${index + 1}. ${prompt.name}`)
    lines.push('')

    if (prompt.description) {
      lines.push(`> ${prompt.description}`)
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
    lines.push(prompt.content || '')
    lines.push('```')
    lines.push('')
    lines.push('---')
    lines.push('')
  })

  return lines.join('\n')
}

/**
 * Combine prompts into JSON format.
 */
function combineToJson(prompts: Prompt[]): string {
  const data = {
    combined: true,
    count: prompts.length,
    generated: new Date().toISOString(),
    prompts: prompts.map((p) => ({
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
 * Combine multiple prompts into the specified format.
 */
export function combinePrompts(
  prompts: Prompt[],
  format: CombineFormat = 'xml'
): CombineResponse {
  if (prompts.length === 0) {
    return {
      combined: '',
      promptCount: 0,
      totalTokens: 0,
      format,
    }
  }

  let combined: string

  switch (format) {
    case 'xml':
      combined = combineToXml(prompts)
      break
    case 'markdown':
      combined = combineToMarkdown(prompts)
      break
    case 'json':
      combined = combineToJson(prompts)
      break
    default:
      combined = combineToXml(prompts)
  }

  return {
    combined,
    promptCount: prompts.length,
    totalTokens: estimateTokens(combined),
    format,
  }
}

/**
 * Generate a preview of combined prompts (first N characters).
 */
export function generatePreview(
  prompts: Prompt[],
  format: CombineFormat = 'xml',
  maxLength: number = 500
): string {
  const result = combinePrompts(prompts, format)

  if (result.combined.length <= maxLength) {
    return result.combined
  }

  return result.combined.substring(0, maxLength) + '\n...[truncated]'
}

/**
 * Validate prompts for combination.
 */
export function validateForCombine(prompts: Prompt[]): {
  valid: boolean
  errors: string[]
  warnings: string[]
} {
  const errors: string[] = []
  const warnings: string[] = []

  if (prompts.length === 0) {
    errors.push('No prompts selected')
  }

  if (prompts.length === 1) {
    warnings.push('Only one prompt selected - combining is optional')
  }

  // Check for missing content
  const missingContent = prompts.filter((p) => !p.content.trim())
  if (missingContent.length > 0) {
    warnings.push(
      `${missingContent.length} prompt(s) have no content: ${missingContent.map((p) => p.name).join(', ')}`
    )
  }

  // Check for draft prompts
  const drafts = prompts.filter((p) => p.draft)
  if (drafts.length > 0) {
    warnings.push(
      `${drafts.length} prompt(s) are drafts: ${drafts.map((p) => p.name).join(', ')}`
    )
  }

  // Estimate combined size
  const totalContent = prompts.reduce(
    (sum, p) => sum + p.content.length,
    0
  )
  if (totalContent > 50000) {
    warnings.push(
      `Combined content is large (~${Math.round(totalContent / 1000)}KB). Consider selecting fewer prompts.`
    )
  }

  return {
    valid: errors.length === 0,
    errors,
    warnings,
  }
}

/**
 * Copy combined content to clipboard.
 */
export async function copyToClipboard(
  prompts: Prompt[],
  format: CombineFormat = 'xml'
): Promise<{ success: boolean; error?: string }> {
  try {
    const result = combinePrompts(prompts, format)
    await navigator.clipboard.writeText(result.combined)
    return { success: true }
  } catch (error) {
    return {
      success: false,
      error: error instanceof Error ? error.message : 'Failed to copy',
    }
  }
}
