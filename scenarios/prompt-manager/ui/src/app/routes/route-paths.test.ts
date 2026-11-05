import { describe, expect, it } from 'vitest'
import {
  actionDetailPath,
  agentDetailPath,
  appendQuery,
  detailPath,
  graphPath,
  runDetailPath,
  skillDetailPath,
  teamDetailPath,
  topicDetailPath,
  topicWizardPath,
  worldPath,
} from './route-paths'

describe('prompt-manager route paths', () => {
  it('builds home routes', () => {
    expect(worldPath()).toBe('/world')
    expect(graphPath()).toBe('/graph')
    expect(topicWizardPath()).toBe('/topics/new')
  })

  it('builds detail routes with encoded ids', () => {
    expect(skillDetailPath('skill one')).toBe('/skills/skill%20one')
    expect(agentDetailPath('agent/one')).toBe('/agents/agent%2Fone')
    expect(teamDetailPath('team?one')).toBe('/teams/team%3Fone')
    expect(runDetailPath('run#one')).toBe('/runs/run%23one')
    expect(topicDetailPath('topic&one')).toBe('/topics/topic%26one')
    expect(actionDetailPath('action=one')).toBe('/actions/action%3Done')
  })

  it('omits empty query values', () => {
    expect(appendQuery('/skills/a', {
      tab: 'files',
      empty: '',
      missing: null,
      absent: undefined,
      line: 7,
      show: false,
    })).toBe('/skills/a?tab=files&line=7&show=false')
  })

  it('builds generic detail routes', () => {
    expect(detailPath({ entityType: 'skill', id: 's1' })).toBe('/skills/s1')
    expect(detailPath({ entityType: 'agent', id: 'a1', query: { tab: 'prompt' } })).toBe('/agents/a1?tab=prompt')
    expect(detailPath({ entityType: 'team', id: 't1' })).toBe('/teams/t1')
    expect(detailPath({ entityType: 'run', id: 'r1' })).toBe('/runs/r1')
    expect(detailPath({ entityType: 'topic', id: 'tp1' })).toBe('/topics/tp1')
    expect(detailPath({ entityType: 'action', id: 'ac1' })).toBe('/actions/ac1')
  })
})
