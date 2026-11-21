import { createHash } from 'node:crypto'
import { readdirSync, readFileSync } from 'node:fs'
import { join, relative } from 'node:path'

const sha256 = (value: string | Uint8Array) => createHash('sha256').update(value).digest('hex')

function files(root: string, directory: string): string[] {
  return readdirSync(directory, { withFileTypes: true }).flatMap(entry => {
    const path = join(directory, entry.name)
    if (entry.isDirectory()) return files(root, path)
    if (!entry.isFile()) throw new Error(`Provenance requires regular files: ${relative(root, path)}`)
    return [relative(root, path).replace(/\\/g, '/')]
  }).sort()
}

/** Hash file names and bytes, excluding tests and documentation from runtime source. */
export function collectWorldBuildProvenance(root: string) {
  const sourcePaths = files(root, join(root, 'src')).filter(path =>
    !/(?:^|\/)(?:__tests__|test|test-utils)\//.test(path) &&
    !/\.(?:test|spec)\.[^.]+$/.test(path) && !/\.(?:md|mdx)$/.test(path))
  sourcePaths.push('vite.config.ts', 'scripts/world-build-provenance.ts')
  const records = (paths: string[]) => paths.sort().map(path => {
    const bytes = readFileSync(join(root, path))
    return { path, bytes: bytes.byteLength, sha256: sha256(bytes) }
  })
  const sources = records(sourcePaths)
  const dependencies = records(['package.json', 'pnpm-lock.yaml'])
  const assets = records(files(root, join(root, 'public/assets/world')))
    .map(asset => ({ ...asset, path: asset.path.slice('public/assets/world/'.length) }))
  return {
    algorithm: 'sha256' as const,
    sourceScope: 'ui/src runtime files, vite.config.ts, scripts/world-build-provenance.ts' as const,
    sourceSha256: sha256(JSON.stringify(sources)),
    dependencyScope: 'ui/package.json and ui/pnpm-lock.yaml' as const,
    dependencySha256: sha256(JSON.stringify(dependencies)),
    assetSha256: sha256(JSON.stringify(assets)), assets,
  }
}
