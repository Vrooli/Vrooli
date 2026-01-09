import { describe, it, expect, afterEach } from 'vitest'
import * as http from 'node:http'
import { resolveProxyAgent, resetProxyAgentsForTesting } from '../../server/agent.js'

describe('resolveProxyAgent', () => {
  afterEach(() => {
    resetProxyAgentsForTesting()
  })

  it('returns a shared keep-alive agent by default', () => {
    const agentA = resolveProxyAgent({})
    const agentB = resolveProxyAgent({})

    expect(agentA).toBeInstanceOf(http.Agent)
    expect(agentA).toBe(agentB)
  })

  it('returns undefined when keep-alive is disabled', () => {
    const agent = resolveProxyAgent({ keepAlive: false })
    expect(agent).toBeUndefined()
  })

  it('returns the provided custom agent', () => {
    const customAgent = new http.Agent({ keepAlive: false })
    const agent = resolveProxyAgent({ agent: customAgent })
    expect(agent).toBe(customAgent)
    customAgent.destroy()
  })
})
