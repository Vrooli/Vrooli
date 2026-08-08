import { describe, expect, it } from 'vitest'
import { formatActions } from './formatEntities'
import type { Action } from '@/types'

function makeAction(overrides: Partial<Action> = {}): Action {
  return {
    kind: 'action',
    schemaVersion: 1,
    id: 'team.swarm.work.list',
    name: 'List Team Work',
    description: 'Review recent team work.',
    status: 'active',
    owner: { type: 'scenario', id: 'prompt-manager' },
    command: { argv: ['swarm-manager', 'backlog', 'list', '--json'] },
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
      workItems: {
        type: 'json',
        description: 'Work item records',
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
    expect(formatActions([makeAction()], 'cli')).toBe('prompt-manager action show team.swarm.work.list')
  })

  it('formats Markdown with contract fields', () => {
    const markdown = formatActions([makeAction()], 'markdown')

    expect(markdown).toContain('## List Team Work')
    expect(markdown).toContain('**Owner:** scenario:prompt-manager')
    expect(markdown).toContain('**Command:** `swarm-manager backlog list --json`')
    expect(markdown).toContain('- `team` (team): Team ID')
    expect(markdown).toContain('- `workItems` (json): Work item records')
    expect(markdown).toContain('**Permissions:** apiRead')
  })

  it('formats deterministic XML with escaped values', () => {
    const xml = formatActions([makeAction({ name: 'List <Work>' })], 'xml')

    expect(xml).toContain('<actions>')
    expect(xml).toContain('<name>List &lt;Work Items&gt;</name>')
    expect(xml).toContain('<field name="team">')
    expect(xml).toContain('<permission>apiRead</permission>')
  })

  it('formats JSON with full Action contracts', () => {
    const parsed = JSON.parse(formatActions([makeAction()], 'json')) as { actions: Action[]; count: number }

    expect(parsed.count).toBe(1)
    expect(parsed.actions[0]?.id).toBe('team.swarm.work.list')
    expect(parsed.actions[0]?.inputs.team?.description).toBe('Team ID')
  })
})
