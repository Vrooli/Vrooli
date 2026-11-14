#!/usr/bin/env node
/**
 * Regenerates the world tuning lever table in docs/reference/configuration.md.
 * Loads the TypeScript schema through Vite's SSR module loader so the script
 * needs no separate TS runtime.
 */
import { createServer } from 'vite'
import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'

const uiRoot = resolve(import.meta.dirname, '..', '..')
const docPath = resolve(uiRoot, '..', 'docs', 'reference', 'configuration.md')

const server = await createServer({
  root: uiRoot,
  configFile: resolve(uiRoot, 'vite.config.ts'),
  server: { middlewareMode: true, hmr: false, watch: null },
  logLevel: 'error',
  appType: 'custom',
})
try {
  const mod = await server.ssrLoadModule('/src/world/config/tuningDocs.ts')
  const before = readFileSync(docPath, 'utf8')
  const after = mod.spliceTuningDocs(before)
  if (after !== before) {
    writeFileSync(docPath, after)
    console.log(`updated ${docPath}`)
  } else {
    console.log('configuration.md already current')
  }
} finally {
  await server.close()
}
