import { createScenarioProxyHost } from '@vrooli/api-base/server'

// Catalogue and port discovery stay with Portal's API. api-base owns HTML,
// assets, API rewriting and WebSocket forwarding; no private proxy lives here.
export function createApplicationProxy({ apiPort, fetcher = fetch, createHost = createScenarioProxyHost }) {
  return createHost({
    hostScenario: 'portal',
    cacheTtlMs: 15000,
    timeoutMs: 10000,
    cacheProxyHtml: false,
    patchFetch: true,
    proxiedAppHeader: 'X-Portal-App',
    childBaseTagAttribute: 'data-portal',
    async fetchAppMetadata(appId) {
      if (!/^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$/.test(appId) || appId === 'portal') {
        throw new Error('Invalid hosted application')
      }
      const response = await fetcher(`http://127.0.0.1:${apiPort}/api/v1/applications/${appId}/proxy-target`, {
        signal: AbortSignal.timeout(6000),
        redirect: 'error',
      })
      if (!response.ok) throw new Error('Application gateway unavailable')
      return response.json()
    },
  })
}
