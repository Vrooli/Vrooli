/**
 * Client-side formatting for non-skill entities.
 *
 * Skills use the backend POST /skills/read endpoint for combined rendering.
 * Agents, teams, topics, and actions are formatted client-side since they lack
 * equivalent backend display endpoints.
 */

import type { CombineFormat } from '@/stores/combineStore'
import type { AIAgentSearchResult, AITeamSearchResult, TopicMatchResult } from '@/lib/schemas'
import type { Action } from '@/types'

// --- Agents ---

export function formatAgents(results: AIAgentSearchResult[], format: CombineFormat): string {
  switch (format) {
    case 'xml':
      return [
        '<agents>',
        ...results.map((r) => [
          `  <agent id="${escapeXml(r.id)}">`,
          `    <displayName>${escapeXml(r.displayName)}</displayName>`,
          `    <description>${escapeXml(r.description)}</description>`,
          `    <status>${escapeXml(r.status)}</status>`,
          r.tags.length > 0 ? `    <tags>${r.tags.map((t) => escapeXml(t)).join(', ')}</tags>` : null,
          '  </agent>',
        ].filter(Boolean).join('\n')),
        '</agents>',
      ].join('\n')

    case 'markdown':
      return results.map((r) => [
        `## ${r.displayName}`,
        '',
        r.description ? r.description : '_No description_',
        '',
        `**Status:** ${r.status}`,
        r.tags.length > 0 ? `**Tags:** ${r.tags.join(', ')}` : null,
      ].filter(Boolean).join('\n')).join('\n\n---\n\n')

    case 'json':
      return JSON.stringify({
        agents: results.map((r) => ({
          id: r.id,
          displayName: r.displayName,
          description: r.description,
          status: r.status,
          tags: r.tags,
        })),
        count: results.length,
      }, null, 2)

    case 'cli':
      return results.map((r) => `prompt-manager agent show ${r.id}`).join('\n')
  }
}

// --- Teams ---

export function formatTeams(results: AITeamSearchResult[], format: CombineFormat): string {
  switch (format) {
    case 'xml':
      return [
        '<teams>',
        ...results.map((r) => [
          `  <team id="${escapeXml(r.id)}">`,
          `    <displayName>${escapeXml(r.displayName)}</displayName>`,
          `    <mission>${escapeXml(r.mission)}</mission>`,
          `    <enabled>${r.enabled}</enabled>`,
          `    <memberCount>${r.memberCount}</memberCount>`,
          '  </team>',
        ].join('\n')),
        '</teams>',
      ].join('\n')

    case 'markdown':
      return results.map((r) => [
        `## ${r.displayName}`,
        '',
        r.mission ? r.mission : '_No mission_',
        '',
        `**Status:** ${r.enabled ? 'enabled' : 'disabled'}`,
        `**Members:** ${r.memberCount}`,
      ].join('\n')).join('\n\n---\n\n')

    case 'json':
      return JSON.stringify({
        teams: results.map((r) => ({
          id: r.id,
          displayName: r.displayName,
          mission: r.mission,
          enabled: r.enabled,
          memberCount: r.memberCount,
        })),
        count: results.length,
      }, null, 2)

    case 'cli':
      return results.map((r) => `prompt-manager team show ${r.id}`).join('\n')
  }
}

// --- Topics ---

export function formatTopics(results: TopicMatchResult[], format: CombineFormat): string {
  switch (format) {
    case 'xml':
      return [
        '<topics>',
        ...results.map((r) => [
          `  <topic id="${escapeXml(r.topic.id)}">`,
          `    <name>${escapeXml(r.topic.name)}</name>`,
          r.topic.description ? `    <description>${escapeXml(r.topic.description)}</description>` : null,
          r.topic.skills.length > 0 ? `    <skills>${r.topic.skills.map((s) => escapeXml(s)).join(', ')}</skills>` : null,
          '  </topic>',
        ].filter(Boolean).join('\n')),
        '</topics>',
      ].join('\n')

    case 'markdown':
      return results.map((r) => [
        `## ${r.topic.name}`,
        '',
        r.topic.description ? r.topic.description : '_No description_',
        '',
        r.topic.skills.length > 0 ? `**Skills:** ${r.topic.skills.join(', ')}` : null,
      ].filter(Boolean).join('\n')).join('\n\n---\n\n')

    case 'json':
      return JSON.stringify({
        topics: results.map((r) => ({
          id: r.topic.id,
          name: r.topic.name,
          description: r.topic.description,
          skills: r.topic.skills,
        })),
        count: results.length,
      }, null, 2)

    case 'cli':
      return results.map((r) => `prompt-manager topic show ${r.topic.id}`).join('\n')
  }
}

// --- Actions ---

export function formatActions(actions: Action[], format: CombineFormat): string {
  switch (format) {
    case 'xml':
      return [
        '<actions>',
        ...actions.map((action) => [
          `  <action id="${escapeXml(action.id)}">`,
          `    <name>${escapeXml(action.name)}</name>`,
          action.description ? `    <description>${escapeXml(action.description)}</description>` : null,
          `    <status>${escapeXml(action.status)}</status>`,
          `    <owner>${escapeXml(formatActionOwner(action))}</owner>`,
          `    <command>${escapeXml(action.command.argv.join(' '))}</command>`,
          action.tags.length > 0 ? `    <tags>${action.tags.map((tag) => escapeXml(tag)).join(', ')}</tags>` : null,
          formatActionFieldsXml('inputs', action.inputs),
          formatActionFieldsXml('outputs', action.outputs),
          formatActionPermissionsXml(action),
          formatActionExamplesXml(action),
          '  </action>',
        ].filter(Boolean).join('\n')),
        '</actions>',
      ].join('\n')

    case 'markdown':
      return actions.map((action) => [
        `## ${action.name}`,
        '',
        action.description ? action.description : '_No description_',
        '',
        `**ID:** ${action.id}`,
        `**Status:** ${action.status}`,
        `**Owner:** ${formatActionOwner(action)}`,
        `**Command:** \`${action.command.argv.join(' ')}\``,
        action.tags.length > 0 ? `**Tags:** ${action.tags.join(', ')}` : null,
        formatActionFieldsMarkdown('Inputs', action.inputs),
        formatActionFieldsMarkdown('Outputs', action.outputs),
        formatActionPermissionsMarkdown(action),
        formatActionExamplesMarkdown(action),
      ].filter(Boolean).join('\n')).join('\n\n---\n\n')

    case 'json':
      return JSON.stringify({
        actions,
        count: actions.length,
      }, null, 2)

    case 'cli':
      return actions.map((action) => `prompt-manager action show ${action.id}`).join('\n')
  }
}

// --- Helpers ---

function escapeXml(str: string): string {
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&apos;')
}

function formatActionOwner(action: Action): string {
  return `${action.owner.type}:${action.owner.id}`
}

function formatActionFieldsXml(label: 'inputs' | 'outputs', fields: Record<string, { type: string; description?: string }>): string | null {
  const entries = Object.entries(fields).sort(([left], [right]) => left.localeCompare(right))
  if (entries.length === 0) return null
  return [
    `    <${label}>`,
    ...entries.map(([name, field]) => [
      `      <field name="${escapeXml(name)}">`,
      `        <type>${escapeXml(field.type)}</type>`,
      field.description ? `        <description>${escapeXml(field.description)}</description>` : null,
      '      </field>',
    ].filter(Boolean).join('\n')),
    `    </${label}>`,
  ].join('\n')
}

function formatActionFieldsMarkdown(label: 'Inputs' | 'Outputs', fields: Record<string, { type: string; description?: string }>): string | null {
  const entries = Object.entries(fields).sort(([left], [right]) => left.localeCompare(right))
  if (entries.length === 0) return null
  return [
    '',
    `**${label}:**`,
    ...entries.map(([name, field]) => `- \`${name}\` (${field.type})${field.description ? `: ${field.description}` : ''}`),
  ].join('\n')
}

function enabledActionPermissions(action: Action): string[] {
  return Object.entries(action.permissions)
    .filter(([, enabled]) => enabled === true)
    .map(([name]) => name)
    .sort()
}

function formatActionPermissionsXml(action: Action): string | null {
  const permissions = enabledActionPermissions(action)
  if (permissions.length === 0) return null
  return [
    '    <permissions>',
    ...permissions.map((permission) => `      <permission>${escapeXml(permission)}</permission>`),
    '    </permissions>',
  ].join('\n')
}

function formatActionPermissionsMarkdown(action: Action): string | null {
  const permissions = enabledActionPermissions(action)
  if (permissions.length === 0) return null
  return ['', `**Permissions:** ${permissions.join(', ')}`].join('\n')
}

function formatActionExamplesXml(action: Action): string | null {
  if (action.examples.length === 0) return null
  return [
    '    <examples>',
    ...action.examples.map((example) => [
      '      <example>',
      example.description ? `        <description>${escapeXml(example.description)}</description>` : null,
      `        <input>${escapeXml(JSON.stringify(example.input))}</input>`,
      '      </example>',
    ].filter(Boolean).join('\n')),
    '    </examples>',
  ].join('\n')
}

function formatActionExamplesMarkdown(action: Action): string | null {
  if (action.examples.length === 0) return null
  return [
    '',
    '**Examples:**',
    ...action.examples.map((example) => `- ${example.description || 'Example'}: \`${JSON.stringify(example.input)}\``),
  ].join('\n')
}
