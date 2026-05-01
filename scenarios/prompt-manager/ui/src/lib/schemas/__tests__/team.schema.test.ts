import { describe, expect, it } from 'vitest'
import {
  TeamDetailsSchema,
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
    expect(result.decisionMode).toBe('yolo')
    expect(result.operatingContract.schemaVersion).toBe(1)
    expect(result.operatingContract.governance.decisionMode).toBe('yolo')
  })
})

describe('TeamDetailsSchema', () => {
  it('normalizes nullable role and member arrays to empty arrays', () => {
    const result = TeamDetailsSchema.parse({
      id: 'scenario-qa',
      displayName: 'Scenario QA',
      enabled: true,
      runtime: { mode: 'multi-process' },
      coordination: buildIndependentCoordination(),
      execution: buildBoundedParallelExecution(2),
      decisionMode: 'yolo',
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
