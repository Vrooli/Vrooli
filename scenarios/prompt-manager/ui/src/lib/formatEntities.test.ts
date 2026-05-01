import { describe, expect, it } from 'vitest'
import { formatActions } from './formatEntities'
import type { Action } from '@/types'

function makeAction(overrides: Partial<Action> = {}): Action {
  return {
    kind: 'action',
    schemaVersion: 1,
    id: 'team.decisions.list',
    name: 'List Team Decisions',
    description: 'Review recent team decisions.',
    status: 'active',
    owner: { type: 'scenario', id: 'prompt-manager' },
    command: { argv: ['prompt-manager', 'team', 'decisions', 'list'] },
    inputs: {
      team: {
        type: 'team',
        description: 'Team ID',
        required: true,
        enum: [],
        pattern: '',
        allowMultiline: false,
      },
    },
    outputs: {
      decisions: {
        type: 'json',
        description: 'Decision records',
      },
    },
    permissions: {
      filesystemRead: false,
      filesystemWrite: false,
      localhostNetwork: false,
      externalNetwork: false,
      apiRead: true,
      apiWrite: false,
      processStart: false,
      processStop: false,
      hostConfigure: false,
      secretRead: false,
      secretWrite: false,
      destructive: false,
    },
    examples: [{ description: 'Meta team', input: { team: 'meta-optimization' } }],
    tags: ['team'],
    revision: 1,
    createdAt: '2026-04-30T00:00:00Z',
    updatedAt: '2026-04-30T00:00:00Z',
    ...overrides,
  }
}

describe('formatActions', () => {
  it('formats CLI commands using action show', () => {
    expect(formatActions([makeAction()], 'cli')).toBe('prompt-manager action show team.decisions.list')
  })

  it('formats Markdown with contract fields', () => {
    const markdown = formatActions([makeAction()], 'markdown')

    expect(markdown).toContain('## List Team Decisions')
    expect(markdown).toContain('**Owner:** scenario:prompt-manager')
    expect(markdown).toContain('**Command:** `prompt-manager team decisions list`')
    expect(markdown).toContain('- `team` (team): Team ID')
    expect(markdown).toContain('- `decisions` (json): Decision records')
    expect(markdown).toContain('**Permissions:** apiRead')
  })

  it('formats deterministic XML with escaped values', () => {
    const xml = formatActions([makeAction({ name: 'List <Decisions>' })], 'xml')

    expect(xml).toContain('<actions>')
    expect(xml).toContain('<name>List &lt;Decisions&gt;</name>')
    expect(xml).toContain('<field name="team">')
    expect(xml).toContain('<permission>apiRead</permission>')
  })

  it('formats JSON with full Action contracts', () => {
    const parsed = JSON.parse(formatActions([makeAction()], 'json')) as { actions: Action[]; count: number }

    expect(parsed.count).toBe(1)
    expect(parsed.actions[0]?.id).toBe('team.decisions.list')
    expect(parsed.actions[0]?.inputs.team?.description).toBe('Team ID')
  })
})
