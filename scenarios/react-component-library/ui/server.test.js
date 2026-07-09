import { describe, expect, it } from 'vitest'

import { shouldProxyToApi } from './server.js'

describe('ui server route proxying', () => {
  it('forwards preview harness and runtime routes to the API before SPA fallback', () => {
    expect(shouldProxyToApi('/preview/cmp-7/harness.html')).toBe(true)
    expect(shouldProxyToApi('/preview/runtime/react@18.3.1/index.js')).toBe(true)
    expect(shouldProxyToApi('/preview/runtime/npm/lucide-react@0.424.0/index.js')).toBe(true)
  })

  it('keeps Connect-RPC forwarding without proxying normal SPA routes', () => {
    expect(shouldProxyToApi('/vrooli.react_component_library.v1.components.ComponentsService/ListComponents')).toBe(true)
    expect(shouldProxyToApi('/components')).toBe(false)
    expect(shouldProxyToApi('/previewing')).toBe(false)
    expect(shouldProxyToApi('/assets/index.js')).toBe(false)
  })
})
