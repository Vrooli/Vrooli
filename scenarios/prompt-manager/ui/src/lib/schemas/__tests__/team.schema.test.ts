import { describe, expect, it } from 'vitest'
import {
  TeamDetailsSchema,
  UpdateTeamRequestSchema,
  buildBoundedParallelExecution,
  buildDefaultCreateTeamRequest,
  buildIndependentCoordination,
  buildLeaderLedCoordination,
  buildPeerCoordination,
  buildSerializedExecution,
} from '../team.schema'

describe('team coordination preset builders', () => {
  it('builds the independent preset without coordination overhead', () => {
    const result = buildIndependentCoordination()

    expect(result.pattern).toBe('independent')
    expect(result.reportingMode).toBe('none')
    expect(result.messagingMode).toBe('disabled')
    expect(result.capabilities.showOrgContext).toBe(false)
    expect(result.capabilities.injectInbox).toBe(false)
    expect(result.capabilities.allowPeerTriggers).toBe(false)
    expect(result.capabilities.requireHandoff).toBe(true)
  })

  it('builds the peer preset with async inbox coordination enabled', () => {
    const result = buildPeerCoordination()

    expect(result.pattern).toBe('peer')
    expect(result.reportingMode).toBe('org-chart')
    expect(result.messagingMode).toBe('async-inbox')
    expect(result.capabilities.showOrgContext).toBe(true)
    expect(result.capabilities.injectInbox).toBe(true)
    expect(result.capabilities.allowPeerTriggers).toBe(true)
  })

  it('builds the leader-led preset for single-process runtime', () => {
    const result = buildLeaderLedCoordination('director', 'single-process')

    expect(result.pattern).toBe('leader-led')
    expect(result.leadAgentId).toBe('director')
    expect(result.reportingMode).toBe('leader')
    expect(result.messagingMode).toBe('in-session')
    expect(result.capabilities.injectInbox).toBe(false)
    expect(result.capabilities.allowPeerTriggers).toBe(false)
  })

  it('builds the leader-led preset for multi-process runtime', () => {
    const result = buildLeaderLedCoordination('director', 'multi-process')

    expect(result.pattern).toBe('leader-led')
    expect(result.leadAgentId).toBe('director')
    expect(result.messagingMode).toBe('async-inbox')
    expect(result.capabilities.injectInbox).toBe(true)
  })
})

describe('team execution builders', () => {
  it('builds bounded parallel execution with the requested concurrency', () => {
    expect(buildBoundedParallelExecution(4)).toEqual({
      queuePolicy: 'bounded-parallel',
      maxConcurrentRuns: 4,
    })
  })

  it('builds serialized execution with a single active run', () => {
    expect(buildSerializedExecution()).toEqual({
      queuePolicy: 'serialized',
      maxConcurrentRuns: 1,
    })
  })
})

describe('buildDefaultCreateTeamRequest', () => {
  it('defaults new teams to the independent multi-process preset', () => {
    const result = buildDefaultCreateTeamRequest('Scenario QA')

    expect(result.displayName).toBe('Scenario QA')
    expect(result.runtime.mode).toBe('multi-process')
    expect(result.coordination.pattern).toBe('independent')
    expect(result.coordination.messagingMode).toBe('disabled')
    expect(result.execution.queuePolicy).toBe('bounded-parallel')
    expect(result.execution.maxConcurrentRuns).toBe(2)
    expect(result.operatingContract.schemaVersion).toBe(1)
  })
})

describe('TeamDetailsSchema', () => {
  it('preserves legacy teams without inventing classification or objective grants', () => {
    const result = TeamDetailsSchema.parse({
      ...buildDefaultCreateTeamRequest('Legacy'), id: 'legacy', enabled: false,
      memberCount: 0, roles: [], members: [],
      objectivesServed: [{ id: 'objective:quality', customEvidence: 'receipt:7' }],
      createdAt: '2026-04-09T00:00:00Z', updatedAt: '2026-04-09T00:00:00Z',
    })
    expect(result.purpose).toBeUndefined()
    expect(result.lifetime).toBeUndefined()
    expect(result.effortRefs).toBeUndefined()
    expect(result.objectivesServed?.[0]).toEqual({ id: 'objective:quality', customEvidence: 'receipt:7' })
    expect(result.enabled).toBe(false)
  })

  it('accepts independent classification and explicit clearing while rejecting invalid metadata', () => {
    expect(UpdateTeamRequestSchema.parse({ purpose: 'delivery', lifetime: 'standing' })).toEqual({ purpose: 'delivery', lifetime: 'standing' })
    expect(UpdateTeamRequestSchema.parse({ purpose: '', lifetime: '', effortRefs: [] })).toEqual({ purpose: '', lifetime: '', effortRefs: [] })
    expect(UpdateTeamRequestSchema.safeParse({ purpose: 'temporary' }).success).toBe(false)
    expect(UpdateTeamRequestSchema.safeParse({ lifetime: 'delivery' }).success).toBe(false)
    expect(UpdateTeamRequestSchema.safeParse({ effortRefs: ['   '] }).success).toBe(false)
  })

  it('normalizes nullable role and member arrays to empty arrays', () => {
    const result = TeamDetailsSchema.parse({
      id: 'scenario-qa',
      displayName: 'Scenario QA',
      enabled: true,
      runtime: { mode: 'multi-process' },
      coordination: buildIndependentCoordination(),
      execution: buildBoundedParallelExecution(2),
      operatingContract: buildDefaultCreateTeamRequest('Scenario QA').operatingContract,
      memberCount: 0,
      roles: null,
      members: null,
      createdAt: '2026-04-09T00:00:00Z',
      updatedAt: '2026-04-09T00:00:00Z',
    })

    expect(result.roles).toEqual([])
    expect(result.members).toEqual([])
  })
})
