import type { Transport } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'

import { resolveApiBase } from './resolve.js'

export interface ScenarioConnectTransportOptions {
  baseUrl?: string
  fetch?: typeof fetch
}

// createScenarioConnectTransport configures the standard Connect-Web transport
// used by scenario UIs for proto-typed API calls.
export function createScenarioConnectTransport(opts: ScenarioConnectTransportOptions = {}): Transport {
  return createConnectTransport({
    baseUrl: opts.baseUrl ?? resolveApiBase(),
    fetch: opts.fetch,
  })
}
