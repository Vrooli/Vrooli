import { spawnSync } from 'node:child_process'
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { pathToFileURL } from 'node:url'

const tempDir = mkdtempSync(join(tmpdir(), 'prompt-manager-selectors-'))
const pnpmBin = process.platform === 'win32' ? 'pnpm.cmd' : 'pnpm'

try {
  const compile = spawnSync(
    pnpmBin,
    [
      'exec',
      'tsc',
      'src/constants/selectors.ts',
      '--module',
      'nodenext',
      '--moduleResolution',
      'nodenext',
      '--target',
      'es2022',
      '--outDir',
      tempDir,
    ],
    {
      cwd: process.cwd(),
      encoding: 'utf8',
      stdio: 'pipe',
    },
  )
  if (compile.status !== 0) {
    throw new Error(compile.stderr || compile.stdout || 'failed to compile selectors.ts')
  }

  const selectorsModule = await import(pathToFileURL(join(tempDir, 'selectors.js')).href)
  const manifestPath = 'src/constants/selectors.manifest.json'
  const manifestJson = `${JSON.stringify(selectorsModule.selectorsManifest, null, 2)}\n`

  writeFileSync(manifestPath, manifestJson, 'utf8')
  process.stdout.write(`Generated ${manifestPath}\n`)
} finally {
  rmSync(tempDir, { recursive: true, force: true })
}
