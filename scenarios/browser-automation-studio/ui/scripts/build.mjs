import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { spawnSync } from 'node:child_process'

const packageRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const args = process.argv.slice(2)
const outputIndex = args.findIndex((arg) => arg.toLowerCase() === '--outdir')
const requestedOutput = outputIndex >= 0 ? args[outputIndex + 1] : ''
if (outputIndex >= 0 && (!requestedOutput || requestedOutput.startsWith('--'))) {
  throw new Error('--outDir requires a directory path')
}
const outputDir = resolve(packageRoot, requestedOutput || 'dist')
const modeIndex = args.findIndex((arg) => arg === '--mode')
const profileBuild = args.some((arg) => arg === '--profile') || (modeIndex >= 0 && args[modeIndex + 1] === 'profile')

function runNode(args) {
  const result = spawnSync(process.execPath, args, { cwd: packageRoot, stdio: 'inherit' })
  if (result.error) throw result.error
  if (result.status !== 0) throw new Error(`${args[0]} exited with status ${result.status}`)
}

function runVite(args) {
  const vite = join(packageRoot, 'node_modules/vite/bin/vite.js')
  const result = spawnSync(process.execPath, [vite, ...args], { cwd: packageRoot, stdio: 'inherit' })
  if (result.error) throw result.error
  if (result.status !== 0) throw new Error(`vite exited with status ${result.status}`)
}

runNode(['scripts/generate-selector-manifest.mjs'])
const viteArgs = ['build', '--outDir', outputDir]
if (profileBuild) viteArgs.push('--mode', 'profile')
runVite(viteArgs)
runNode(['scripts/gzip-dist.mjs', '--outDir', outputDir])
