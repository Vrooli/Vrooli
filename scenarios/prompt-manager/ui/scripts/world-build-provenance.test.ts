import { mkdtempSync, mkdirSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { afterEach, describe, expect, it } from 'vitest'
import { collectWorldBuildProvenance } from './world-build-provenance'

const roots: string[] = []
afterEach(() => { for (const root of roots.splice(0)) rmSync(root, { recursive: true, force: true }) })

describe('world build provenance', () => {
  it('tracks runtime, dependency and asset byte changes independently and excludes test/docs churn', () => {
    const root = mkdtempSync(join(tmpdir(), 'world-provenance-'))
    roots.push(root)
    const write = (path: string, content: string) => {
      const target = join(root, path)
      mkdirSync(join(target, '..'), { recursive: true })
      writeFileSync(target, content)
    }
    for (const path of ['src/world.ts', 'vite.config.ts', 'scripts/world-build-provenance.ts', 'package.json', 'pnpm-lock.yaml', 'public/assets/world/park/tree.glb']) write(path, path)
    const before = collectWorldBuildProvenance(root)
    expect(before.sourceSha256).toMatch(/^[a-f0-9]{64}$/)
    expect(before.assets[0]?.path).toBe('park/tree.glb')
    write('src/world.test.ts', 'test changed')
    write('src/README.md', 'documentation changed')
    expect(collectWorldBuildProvenance(root)).toEqual(before)
    write('src/world.ts', 'runtime changed')
    const sourceChanged = collectWorldBuildProvenance(root)
    expect(sourceChanged.sourceSha256).not.toBe(before.sourceSha256)
    expect(sourceChanged.assetSha256).toBe(before.assetSha256)
    write('public/assets/world/park/tree.glb', 'asset changed')
    const assetChanged = collectWorldBuildProvenance(root)
    expect(assetChanged.assetSha256).not.toBe(before.assetSha256)
    expect(assetChanged.sourceSha256).toBe(sourceChanged.sourceSha256)
    write('pnpm-lock.yaml', 'dependencies changed')
    expect(collectWorldBuildProvenance(root).dependencySha256).not.toBe(before.dependencySha256)
  })
})
