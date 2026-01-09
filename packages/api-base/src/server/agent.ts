import * as http from 'node:http'

interface ResolveOptions {
  agent?: http.Agent
  keepAlive?: boolean
}

let sharedAgent: http.Agent | null = null

const managedAgents = new Set<http.Agent>()

function createSharedAgent(): http.Agent {
  const agent = new http.Agent({
    keepAlive: true,
    keepAliveMsecs: 1000,
    maxSockets: Infinity,
    scheduling: 'lifo',
  })
  managedAgents.add(agent)
  return agent
}

/**
 * Resolve which HTTP agent to use for proxying requests.
 *
 * - If a custom agent is provided, it is returned as-is
 * - If keep-alive is explicitly disabled, undefined is returned (Node default)
 * - Otherwise a shared keep-alive agent is created/reused
 */
export function resolveProxyAgent(options: ResolveOptions): http.Agent | undefined {
  if (options.agent) {
    return options.agent
  }

  if (options.keepAlive === false) {
    return undefined
  }

  if (!sharedAgent) {
    sharedAgent = createSharedAgent()
  }

  return sharedAgent
}

/**
 * Destroy shared agents. Intended for tests to avoid dangling sockets.
 */
export function resetProxyAgentsForTesting(): void {
  for (const agent of managedAgents) {
    agent.destroy()
  }
  managedAgents.clear()
  sharedAgent = null
}
