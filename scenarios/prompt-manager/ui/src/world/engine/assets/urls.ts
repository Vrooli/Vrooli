import { getProxyInfo } from '@vrooli/api-base'

/**
 * Absolute URL for a file under public/assets/world, honouring the proxy base
 * path the UI is served behind. Never a CDN: every world asset is bundled.
 */
export function worldAssetUrl(relativePath: string): string {
  const proxyInfo = getProxyInfo()
  const proxyPath = proxyInfo ? (proxyInfo.primary.path ?? proxyInfo.basePath) : undefined
  const base = proxyPath ? proxyPath.replace(/\/+$/, '') : ''
  return `${base}/assets/world/${relativePath.replace(/^\/+/, '')}`
}

export const WORLD_ASSETS = {
  skyHdr: 'env/sky_1k.hdr',
  labelFont: 'fonts/NotoSans-Latin.ttf',
} as const
