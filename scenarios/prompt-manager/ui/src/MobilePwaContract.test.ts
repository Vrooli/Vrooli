import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import indexHTML from '../index.html?raw'
import manifestJSON from '../public/public/site.webmanifest?raw'

const globalsCSS = readFileSync('src/styles/globals.css', 'utf8')

describe('Prompt Manager mobile PWA contract', () => {
  it('keeps installed navigation in standalone mode on every client route', () => {
    expect(indexHTML).toContain('rel="manifest" href="/public/site.webmanifest"')
    expect(indexHTML).toContain('name="apple-mobile-web-app-capable" content="yes"')
    expect(indexHTML).toContain('name="mobile-web-app-capable" content="yes"')
    expect(indexHTML).toContain('name="viewport" content="width=device-width, initial-scale=1.0, viewport-fit=cover"')

    expect(JSON.parse(manifestJSON)).toMatchObject({
      display: 'standalone',
      start_url: '/',
      scope: '/',
    })
    expect(JSON.parse(manifestJSON).icons).toEqual(expect.arrayContaining([
      expect.objectContaining({ sizes: '192x192' }),
      expect.objectContaining({ sizes: '512x512', purpose: 'maskable' }),
    ]))
    expect(globalsCSS).toContain('padding: env(safe-area-inset-top)')
  })
})
