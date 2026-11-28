import { describe, expect, it, vi } from 'vitest'
import { createApplicationProxy } from './applicationProxy.js'

describe('Portal application gateway owner boundary', () => {
  it('uses bounded local API metadata and the shared proxy engine', async () => {
    const metadata = { id: 'nutrition-planner', port_mappings: { UI_PORT: 20002, API_PORT: 17000 } }
    const fetcher = vi.fn().mockResolvedValue({ ok: true, json: async () => metadata })
    let options
    const controller = { router: {}, destroy() {} }
    const createHost = vi.fn(value => { options = value; return controller })
    expect(createApplicationProxy({ apiPort: 17476, fetcher, createHost })).toBe(controller)
    expect(options.hostScenario).toBe('portal')
    expect(options.patchFetch).toBe(true)
    expect(options.cacheProxyHtml).toBe(false)
    expect(await options.fetchAppMetadata('nutrition-planner')).toEqual(metadata)
    expect(fetcher.mock.calls[0][0]).toBe('http://127.0.0.1:17476/api/v1/applications/nutrition-planner/proxy-target')
    expect(fetcher.mock.calls[0][1].redirect).toBe('error')
  })
  it('rejects recursive or arbitrary targets before any network request', async () => {
    const fetcher = vi.fn()
    let options
    createApplicationProxy({ apiPort: 17476, fetcher, createHost: value => { options = value; return {} } })
    for (const target of ['portal', '../private', 'http://host', 'Bad_Name', '']) {
      await expect(options.fetchAppMetadata(target)).rejects.toThrow('Invalid hosted application')
    }
    expect(fetcher).not.toHaveBeenCalled()
  })
  it('retains discovery failure instead of falling back to a guessed port', async () => {
    let options
    createApplicationProxy({ apiPort: 17476, fetcher: async () => ({ ok: false }), createHost: value => { options = value; return {} } })
    await expect(options.fetchAppMetadata('nutrition-planner')).rejects.toThrow('Application gateway unavailable')
  })
})
