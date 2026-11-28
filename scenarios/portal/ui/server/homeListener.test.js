import { describe, expect, it, vi } from 'vitest'
import { createHomeListener } from './homeListener.js'

describe('Portal home gateway transport', () => {
  it('uses the current lifecycle port for the ordinary LAN gateway', () => {
    const app = { listen: vi.fn() }
    const listener = createHomeListener(app, { env: {} })
    listener.listen(23456, '0.0.0.0', () => {})
    expect(listener.secure).toBe(false)
    expect(app.listen).toHaveBeenCalledWith(23456, '0.0.0.0', expect.any(Function))
  })
  it('serves the same app with validated TLS rather than a separate proxy', () => {
    const app = {}, server = { listen: vi.fn() }, httpsServer = vi.fn(() => server)
    const readFile = vi.fn(path => Buffer.from(path === '/cert' ? 'public-certificate' : 'private-fixture'))
    const listener = createHomeListener(app, { env: { PORTAL_TLS_CERT_FILE: '/cert', PORTAL_TLS_KEY_FILE: '/key' }, readFile, httpsServer })
    listener.listen(27890, '0.0.0.0', () => {})
    expect(listener.secure).toBe(true)
    expect(httpsServer).toHaveBeenCalledWith({ cert: Buffer.from('public-certificate'), key: Buffer.from('private-fixture'), minVersion: 'TLSv1.2' }, app)
    expect(server.listen).toHaveBeenCalledWith(27890, '0.0.0.0', expect.any(Function))
  })
  it('never falls back to plaintext after partial or broken TLS setup', () => {
    for (const env of [{ PORTAL_TLS_CERT_FILE: '/cert' }, { PORTAL_TLS_KEY_FILE: '/key' }]) expect(() => createHomeListener({}, { env })).toThrow('both certificate and key')
    expect(() => createHomeListener({}, { env: { PORTAL_TLS_CERT_FILE: '/cert', PORTAL_TLS_KEY_FILE: '/key' }, readFile: () => { throw new Error('unavailable') } })).toThrow('unavailable')
  })
})
