import { describe, expect, it } from 'vitest'
import {
  ActionSchema,
  ActionValidationResponseSchema,
  ActionMutationResponseSchema,
} from './action.schema'

function actionPayload(overrides: Record<string, unknown> = {}) {
  return {
    kind: 'action',
    schemaVersion: 1,
    id: 'team.decisions.list',
    name: 'List Team Decisions',
    description: null,
    status: 'draft',
    owner: { type: 'scenario', id: 'prompt-manager' },
    command: { argv: ['prompt-manager', 'team', 'decision-list', '{{team}}', '--json'] },
    inputs: {
      team: {
        type: 'team',
        required: true,
        description: 'Prompt-manager team ID.',
        enum: null,
      },
    },
    outputs: null,
    permissions: null,
    examples: null,
    tags: null,
    validation: { mode: 'contract', argv: null },
    revision: 1,
    createdAt: '2026-04-30T00:00:00Z',
    updatedAt: '2026-04-30T00:00:00Z',
    ...overrides,
  }
}

describe('Action schemas', () => {
  it('normalizes nullable Go collections and optional descriptions', () => {
    const parsed = ActionSchema.parse(actionPayload())

    expect(parsed.description).toBe('')
    expect(parsed.outputs).toEqual({})
    expect(parsed.permissions.destructive).toBe(false)
    expect(parsed.examples).toEqual([])
    expect(parsed.tags).toEqual([])
    expect(parsed.validation?.argv).toEqual([])
    expect(parsed.inputs.team?.enum).toEqual([])
  })

  it('rejects malformed API responses', () => {
    expect(() => ActionSchema.parse(actionPayload({ command: { argv: [1] } }))).toThrow()
    expect(() => ActionSchema.parse(actionPayload({ status: 'running' }))).toThrow()
    expect(() => ActionSchema.parse(actionPayload({ owner: { type: 'external', id: 'x' } }))).toThrow()
  })

  it('parses validation and mutation envelopes', () => {
    const action = actionPayload()
    const validation = ActionValidationResponseSchema.parse({
      actionId: 'team.decisions.list',
      valid: true,
      runnable: false,
      status: 'draft',
      command: {
        certainty: 'command',
        owner: { type: 'scenario', id: 'prompt-manager' },
        target: 'prompt-manager',
        commandPath: null,
        effect: 'read',
        permissions: null,
        runSurfaces: null,
        message: null,
      },
      checks: null,
      action,
    })

    expect(validation.checks).toEqual([])
    expect(validation.command?.permissions).toEqual([])
    expect(validation.command?.message).toBe('')

    const mutation = ActionMutationResponseSchema.parse({
      action,
      validation,
    })

    expect(mutation.action.id).toBe('team.decisions.list')
    expect(mutation.validation.valid).toBe(true)
  })
})

