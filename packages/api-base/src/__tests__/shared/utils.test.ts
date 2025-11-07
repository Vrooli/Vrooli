/**
 * Tests for shared/utils.ts
 */

import { describe, it, expect } from 'vitest'
import {
  hasAssetExtension,
  startsWithAssetPrefix,
  isAssetRequest,
  toTrimmedString,
  stringLooksLikeUrlOrPath,
  isLocalHostname,
  isLikelyProxyPath,
} from '../../shared/utils.js'

describe('hasAssetExtension', () => {
  it('detects JavaScript extensions', () => {
    expect(hasAssetExtension('/assets/main.js')).toBe(true)
    expect(hasAssetExtension('/assets/main.mjs')).toBe(true)
    expect(hasAssetExtension('/src/component.tsx')).toBe(true)
    expect(hasAssetExtension('/src/component.jsx')).toBe(true)
  })

  it('detects CSS extensions', () => {
    expect(hasAssetExtension('/styles/main.css')).toBe(true)
    expect(hasAssetExtension('/styles/main.map')).toBe(true)
  })

  it('detects image extensions', () => {
    expect(hasAssetExtension('/images/logo.png')).toBe(true)
    expect(hasAssetExtension('/images/photo.jpg')).toBe(true)
    expect(hasAssetExtension('/images/photo.jpeg')).toBe(true)
    expect(hasAssetExtension('/images/icon.svg')).toBe(true)
    expect(hasAssetExtension('/images/banner.gif')).toBe(true)
    expect(hasAssetExtension('/favicon.ico')).toBe(true)
    expect(hasAssetExtension('/assets/hero.webp')).toBe(true)
  })

  it('detects font extensions', () => {
    expect(hasAssetExtension('/fonts/roboto.woff')).toBe(true)
    expect(hasAssetExtension('/fonts/roboto.woff2')).toBe(true)
    expect(hasAssetExtension('/fonts/roboto.ttf')).toBe(true)
    expect(hasAssetExtension('/fonts/roboto.eot')).toBe(true)
  })

  it('detects other asset types', () => {
    expect(hasAssetExtension('/data/config.json')).toBe(true)
    expect(hasAssetExtension('/wasm/module.wasm')).toBe(true)
    expect(hasAssetExtension('/media/audio.mp3')).toBe(true)
    expect(hasAssetExtension('/media/video.mp4')).toBe(true)
    expect(hasAssetExtension('/docs/manual.pdf')).toBe(true)
    expect(hasAssetExtension('/downloads/archive.zip')).toBe(true)
  })

  it('handles query strings correctly', () => {
    expect(hasAssetExtension('/assets/main.js?v=123')).toBe(true)
    expect(hasAssetExtension('/assets/main.js?v=123&lang=en')).toBe(true)
    expect(hasAssetExtension('/assets/logo.png?size=large')).toBe(true)
  })

  it('handles hash fragments correctly', () => {
    expect(hasAssetExtension('/assets/main.js#section')).toBe(true)
    expect(hasAssetExtension('/assets/main.js?v=1#section')).toBe(true)
  })

  it('returns false for paths without extensions', () => {
    expect(hasAssetExtension('/api/users')).toBe(false)
    expect(hasAssetExtension('/apps/preview')).toBe(false)
    expect(hasAssetExtension('/dashboard')).toBe(false)
    expect(hasAssetExtension('/')).toBe(false)
  })

  it('returns false for paths with non-asset extensions', () => {
    expect(hasAssetExtension('/file.txt')).toBe(false)
    expect(hasAssetExtension('/document.doc')).toBe(false)
    expect(hasAssetExtension('/spreadsheet.xls')).toBe(false)
  })

  it('returns false for directories ending with dots', () => {
    expect(hasAssetExtension('/some.folder/')).toBe(false)
    expect(hasAssetExtension('/path.with.dots/')).toBe(false)
  })

  it('handles root-level files', () => {
    expect(hasAssetExtension('/favicon.ico')).toBe(true)
    expect(hasAssetExtension('/robots.txt')).toBe(false)
  })

  it('is case-insensitive for extensions', () => {
    expect(hasAssetExtension('/assets/MAIN.JS')).toBe(true)
    expect(hasAssetExtension('/assets/Logo.PNG')).toBe(true)
    expect(hasAssetExtension('/styles/App.CSS')).toBe(true)
  })

  it('handles empty or invalid inputs', () => {
    expect(hasAssetExtension('')).toBe(false)
    expect(hasAssetExtension('.')).toBe(false)
    expect(hasAssetExtension('..')).toBe(false)
  })

  it('accepts custom extension set', () => {
    const customExtensions = new Set(['.custom', '.special'])
    expect(hasAssetExtension('/file.custom', customExtensions)).toBe(true)
    expect(hasAssetExtension('/file.js', customExtensions)).toBe(false)
  })
})

describe('startsWithAssetPrefix', () => {
  it('detects Vite prefixes', () => {
    expect(startsWithAssetPrefix('/@vite/client')).toBe(true)
    expect(startsWithAssetPrefix('/@react-refresh')).toBe(true)
    expect(startsWithAssetPrefix('/@fs/home/user/project')).toBe(true)
  })

  it('detects build tool prefixes', () => {
    expect(startsWithAssetPrefix('/node_modules/.vite/deps.js')).toBe(true)
    expect(startsWithAssetPrefix('/__webpack_hmr')).toBe(true)
    expect(startsWithAssetPrefix('/_next/static/chunks/main.js')).toBe(true)
  })

  it('detects common asset directory prefixes', () => {
    expect(startsWithAssetPrefix('/assets/main.js')).toBe(true)
    expect(startsWithAssetPrefix('/static/logo.png')).toBe(true)
    expect(startsWithAssetPrefix('/public/images/hero.jpg')).toBe(true)
  })

  it('detects source directory prefixes', () => {
    expect(startsWithAssetPrefix('/src/components/App.tsx')).toBe(true)
  })

  it('returns false for non-asset paths', () => {
    expect(startsWithAssetPrefix('/api/users')).toBe(false)
    expect(startsWithAssetPrefix('/apps/preview')).toBe(false)
    expect(startsWithAssetPrefix('/dashboard')).toBe(false)
    expect(startsWithAssetPrefix('/')).toBe(false)
  })

  it('is case-sensitive for prefixes', () => {
    // Asset prefixes are case-sensitive
    expect(startsWithAssetPrefix('/Assets/main.js')).toBe(false)
    expect(startsWithAssetPrefix('/STATIC/logo.png')).toBe(false)
  })

  it('handles empty or invalid inputs', () => {
    expect(startsWithAssetPrefix('')).toBe(false)
  })

  it('accepts custom prefix array', () => {
    const customPrefixes = ['/custom/', '/special/']
    expect(startsWithAssetPrefix('/custom/file.js', customPrefixes)).toBe(true)
    expect(startsWithAssetPrefix('/assets/file.js', customPrefixes)).toBe(false)
  })
})

describe('isAssetRequest', () => {
  it('detects assets by extension', () => {
    expect(isAssetRequest('/assets/main.js')).toBe(true)
    expect(isAssetRequest('/styles/app.css')).toBe(true)
    expect(isAssetRequest('/logo.png')).toBe(true)
  })

  it('detects assets by prefix', () => {
    expect(isAssetRequest('/@vite/client')).toBe(true)
    expect(isAssetRequest('/assets/file-without-extension')).toBe(true)
    expect(isAssetRequest('/_next/chunk')).toBe(true)
  })

  it('detects assets by both extension and prefix', () => {
    expect(isAssetRequest('/assets/main.js')).toBe(true)
    expect(isAssetRequest('/@vite/client.js')).toBe(true)
  })

  it('returns false for non-asset requests', () => {
    expect(isAssetRequest('/api/users')).toBe(false)
    expect(isAssetRequest('/apps/preview')).toBe(false)
    expect(isAssetRequest('/dashboard')).toBe(false)
    expect(isAssetRequest('/apps/scenario/view')).toBe(false)
  })

  it('handles complex real-world paths', () => {
    // Real Vite dev server paths
    expect(isAssetRequest('/@vite/client')).toBe(true)
    expect(isAssetRequest('/@react-refresh')).toBe(true)
    expect(isAssetRequest('/node_modules/.vite/deps/react.js?v=123')).toBe(true)

    // Real production build paths
    expect(isAssetRequest('/assets/index-abc123.js')).toBe(true)
    expect(isAssetRequest('/assets/vendor-xyz789.css')).toBe(true)

    // NOT assets
    expect(isAssetRequest('/apps/scenario-auditor/preview')).toBe(false)
    expect(isAssetRequest('/api/v1/health')).toBe(false)
  })

  it('prevents SPA fallback from serving HTML for assets', () => {
    // These paths should NOT get index.html
    expect(isAssetRequest('/main.js')).toBe(true)
    expect(isAssetRequest('/app.css')).toBe(true)
    expect(isAssetRequest('/favicon.ico')).toBe(true)

    // These paths SHOULD get index.html (SPA fallback)
    expect(isAssetRequest('/apps/preview')).toBe(false)
    expect(isAssetRequest('/dashboard')).toBe(false)
    expect(isAssetRequest('/')).toBe(false)
  })
})

describe('toTrimmedString', () => {
  it('converts valid strings', () => {
    expect(toTrimmedString('hello')).toBe('hello')
    expect(toTrimmedString('  hello  ')).toBe('hello')
  })

  it('returns undefined for empty strings', () => {
    expect(toTrimmedString('')).toBeUndefined()
    expect(toTrimmedString('   ')).toBeUndefined()
  })

  it('returns undefined for non-strings', () => {
    expect(toTrimmedString(123)).toBeUndefined()
    expect(toTrimmedString(null)).toBeUndefined()
    expect(toTrimmedString(undefined)).toBeUndefined()
  })
})

describe('stringLooksLikeUrlOrPath', () => {
  it('detects URLs', () => {
    expect(stringLooksLikeUrlOrPath('http://example.com')).toBe(true)
    expect(stringLooksLikeUrlOrPath('https://example.com')).toBe(true)
  })

  it('detects paths', () => {
    expect(stringLooksLikeUrlOrPath('/api/users')).toBe(true)
    expect(stringLooksLikeUrlOrPath('/assets/main.js')).toBe(true)
  })

  it('returns false for simple strings', () => {
    expect(stringLooksLikeUrlOrPath('hello')).toBe(false)
    expect(stringLooksLikeUrlOrPath('test')).toBe(false)
  })
})

describe('isLocalHostname', () => {
  it('detects localhost', () => {
    expect(isLocalHostname('localhost')).toBe(true)
    expect(isLocalHostname('LOCALHOST')).toBe(true)
  })

  it('detects 127.0.0.1', () => {
    expect(isLocalHostname('127.0.0.1')).toBe(true)
  })

  it('detects IPv6 localhost', () => {
    expect(isLocalHostname('::1')).toBe(true)
    expect(isLocalHostname('[::1]')).toBe(true)
  })

  it('detects 0.0.0.0', () => {
    expect(isLocalHostname('0.0.0.0')).toBe(true)
  })

  it('returns false for external hosts', () => {
    expect(isLocalHostname('example.com')).toBe(false)
    expect(isLocalHostname('192.168.1.1')).toBe(false)
  })

  it('handles URLs', () => {
    expect(isLocalHostname('http://localhost:3000')).toBe(true)
    expect(isLocalHostname('http://127.0.0.1:8080')).toBe(true)
    expect(isLocalHostname('http://example.com')).toBe(false)
  })
})

describe('isLikelyProxyPath', () => {
  it('detects app-monitor proxy paths', () => {
    expect(isLikelyProxyPath('/apps/scenario-auditor/proxy/')).toBe(true)
    expect(isLikelyProxyPath('/apps/test-app/proxy/assets/main.js')).toBe(true)
  })

  it('detects generic proxy paths', () => {
    expect(isLikelyProxyPath('/proxy')).toBe(true)
    expect(isLikelyProxyPath('/custom/proxy/path')).toBe(true)
  })

  it('returns false for non-proxy paths', () => {
    expect(isLikelyProxyPath('/apps/preview')).toBe(false)
    expect(isLikelyProxyPath('/api/users')).toBe(false)
    expect(isLikelyProxyPath('/')).toBe(false)
  })
})
