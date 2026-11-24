#!/usr/bin/env node
// Transfer an explicit Playwright artifact tree into BAS's declared bundle
// assets. This script consumes bytes; it neither downloads nor attests source provenance.
import { createRequire } from 'node:module'
import { createHash } from 'node:crypto'
import { cpSync, lstatSync, mkdirSync, readFileSync, readdirSync, realpathSync, rmSync, writeFileSync } from 'node:fs'
import { dirname, isAbsolute, join, relative, resolve, sep } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'
import { inspectBrowserPair } from './browser-pair-contract.mjs'
import { newRuntimeIdentity } from './runtime-identity.mjs'

function fail(message) { throw new Error(`stage-browser-closure: ${message}`) }

function safeTreeCopy(source, destination, root) {
  const stat = lstatSync(source)
  if (stat.isSymbolicLink()) fail(`symlink in artifact tree: ${source}`)
  if (stat.isDirectory()) {
    mkdirSync(destination, { recursive: true, mode: stat.mode & 0o777 })
    for (const child of readdirSync(source)) safeTreeCopy(join(source, child), join(destination, child), root)
  } else if (stat.isFile()) {
    const real = realpathSync(source)
    const rel = relative(root, real)
    if (rel === '..' || rel.startsWith(`..${sep}`) || isAbsolute(rel)) fail(`artifact file escapes input root: ${source}`)
    mkdirSync(dirname(destination), { recursive: true })
    cpSync(source, destination, { preserveTimestamps: true, verbatimSymlinks: true })
  } else fail(`unsupported artifact entry: ${source}`)
}

function sha256(path) { return createHash('sha256').update(readFileSync(path)).digest('hex') }

const layouts = {
  'linux-x64': { chromium: 'chrome-linux/chrome', shell: 'chrome-linux/headless_shell', ffmpeg: 'ffmpeg-linux' },
  'linux-arm64': { chromium: 'chrome-linux/chrome', shell: 'chrome-linux/headless_shell', ffmpeg: 'ffmpeg-linux' },
  'darwin-x64': { chromium: 'chrome-mac/Chromium.app/Contents/MacOS/Chromium', shell: 'chrome-mac/headless_shell', ffmpeg: 'ffmpeg-mac' },
  'darwin-arm64': { chromium: 'chrome-mac/Chromium.app/Contents/MacOS/Chromium', shell: 'chrome-mac/headless_shell', ffmpeg: 'ffmpeg-mac' },
  'windows-x64': { chromium: 'chrome-win/chrome.exe', shell: 'chrome-win/headless_shell.exe', ffmpeg: 'ffmpeg-win64.exe' },
}

const supportMatrix = JSON.parse(readFileSync(new URL('./browser-support-matrix.json', import.meta.url), 'utf8'))

const stageableLayoutPlatforms = ['linux-x64', 'linux-arm64', 'darwin-x64', 'darwin-arm64', 'windows-x64']
const unsupportedLayoutPlatforms = ['windows-arm64']
const privateWorkerLifecycleCells = [
  'workspace_isolation',
  'browser_closure_mount',
  'run_grant_session_binding',
  'network_effect_policy',
  'resource_timeout_limits',
  'artifact_integrity_receipt',
  'completion_cancel_crash_teardown',
]
const targetLifecycleCells = ['runtime_launch', 'browser_task', 'web_console_pane', 'update_rollback', 'uninstall']

function validateTargetLifecycle(platform, cell) {
  const lifecycle = cell?.lifecycle_contract
  if (!lifecycle || typeof lifecycle !== 'object' || Array.isArray(lifecycle) ||
      JSON.stringify(Object.keys(lifecycle).sort()) !== JSON.stringify([...targetLifecycleCells].sort())) {
    fail(`browser support matrix ${platform} lifecycle inventory is incomplete or unexpected`)
  }
  const expected = platform === 'windows-arm64'
    ? 'unsupported-no-layout'
    : platform === 'linux-x64' ? 'unqualified-no-receipt' : 'unqualified-no-node'
  for (const capability of targetLifecycleCells) {
    if (lifecycle[capability] !== expected) fail(`browser support matrix ${platform} ${capability} must remain ${expected}`)
  }
  return lifecycle
}

export function validateSupportMatrix(matrix) {
  if (!matrix || matrix.schema_version !== 'v3') fail('unsupported browser support matrix schema')
  if (!matrix.targets || typeof matrix.targets !== 'object' || Array.isArray(matrix.targets)) fail('browser support matrix targets are invalid')
  const expectedPlatforms = [...stageableLayoutPlatforms, ...unsupportedLayoutPlatforms].sort()
  const actualPlatforms = Object.keys(matrix.targets).sort()
  if (JSON.stringify(actualPlatforms) !== JSON.stringify(expectedPlatforms)) fail('browser support matrix target inventory is incomplete or unexpected')
  for (const platform of stageableLayoutPlatforms) {
    const cell = matrix.targets[platform]
    const expectedLayout = platform === 'linux-x64' ? 'source-fixture-tested' : 'unqualified-no-node'
    if (cell?.bundle_layout !== expectedLayout || cell?.browser_artifact !== 'unqualified-no-provisioned-artifact' || cell?.clean_host !== 'unqualified-no-receipt') {
      fail(`browser support matrix ${platform} has an invalid layout/artifact/clean-host status combination`)
    }
    validateTargetLifecycle(platform, cell)
  }
  for (const platform of unsupportedLayoutPlatforms) {
    const cell = matrix.targets[platform]
    if (cell?.bundle_layout !== 'unsupported-no-layout' || cell?.browser_artifact !== 'unsupported' || cell?.clean_host !== 'unsupported-no-layout') {
      fail(`browser support matrix ${platform} has an invalid unsupported status combination`)
    }
    validateTargetLifecycle(platform, cell)
  }
  const channels = matrix.delivery_channels
  if (!channels || typeof channels !== 'object' || Array.isArray(channels) ||
      JSON.stringify(Object.keys(channels).sort()) !== JSON.stringify(['offline_transfer', 'online_acquisition'])) {
    fail('browser support matrix delivery channel inventory is incomplete or unexpected')
  }
  const online = channels.online_acquisition
  if (online?.status !== 'unsupported-no-governed-browser-source' || online.owner !== 'scenario-dependency-analyzer' ||
      online.evidence !== 'No admitted browser resource or composite Playwright closure export exists.') {
    fail('online browser acquisition must remain explicitly unsupported by the current SDA owner')
  }
  const offline = channels.offline_transfer
  if (offline?.status !== 'source-fixture-tested-linux-x64' ||
      offline.scope !== 'Deterministic archive and install fixtures run on linux-x64 only.' ||
      offline.browser_artifact !== 'unqualified-no-provisioned-artifact' || offline.clean_host !== 'unqualified-no-receipt') {
    fail('offline browser transfer must preserve its fixture-only and unqualified status')
  }
  const remote = matrix['private-hosted-worker']
  if (!remote || remote.status !== 'unsupported' || typeof remote.owner_gap !== 'string' || remote.owner_gap.trim() === '') {
    fail('private hosted worker must remain unsupported with a concrete existing-owner gap')
  }
  for (const requiredGap of [
    'BAS BackendRemote rejects before local Session.Manager open',
    'Agent Manager SandboxLauncher isolates agent processes',
    'browser closure mount',
    'browser run/grant/session binding',
    'task/artifact receipt contract',
    'remote-worker teardown owner',
  ]) {
    if (!remote.owner_gap.includes(requiredGap)) fail(`private hosted worker owner gap omits ${requiredGap}`)
  }
  const lifecycle = remote.lifecycle_contract
  if (!lifecycle || typeof lifecycle !== 'object' || Array.isArray(lifecycle)) fail('private hosted worker lifecycle contract is missing')
  const actualLifecycleCells = Object.keys(lifecycle).sort()
  if (JSON.stringify(actualLifecycleCells) !== JSON.stringify([...privateWorkerLifecycleCells].sort())) {
    fail('private hosted worker lifecycle contract is incomplete or unexpected')
  }
  for (const cell of privateWorkerLifecycleCells) {
    if (lifecycle[cell] !== 'unsupported-no-browser-task-owner') {
      fail(`private hosted worker lifecycle cell ${cell} must remain explicitly unsupported without a browser-task owner`)
    }
  }
  return matrix
}

export function supportCellFor(platform, matrix = supportMatrix) {
  validateSupportMatrix(matrix)
  const cell = matrix.targets?.[platform]
  if (!cell) fail(`support matrix has no target cell for ${platform}`)
  if (cell.bundle_layout !== 'source-fixture-tested' && cell.bundle_layout !== 'unqualified-no-node') fail(`target ${platform} has no staged bundle layout contract`)
  return cell
}

export function layoutFor(platform) {
  const layout = layouts[platform]
  if (!layout) fail(`unsupported target ${platform}`)
  return layout
}

function revisionFolder(descriptor, hostPlatform) {
  const override = descriptor.revisionOverrides?.[hostPlatform]
  const revision = String(override || descriptor.revision || '')
  if (!/^\d+$/.test(revision)) fail(`pinned browsers.json revision missing for ${descriptor.name}`)
  const name = override ? `${descriptor.name}_${hostPlatform}_special` : descriptor.name
  return `${name.replaceAll('-', '_')}-${revision}`
}

function compatibleHost(platform, hostPlatform) {
  if (!hostPlatform) return false
  if (platform.startsWith('linux-')) return /-(x64|arm64)$/.test(hostPlatform)
  if (platform === 'windows-x64') return hostPlatform === 'win64'
  if (platform === 'darwin-arm64') return /^mac\d+-arm64$/.test(hostPlatform)
  if (platform === 'darwin-x64') return /^mac\d+$/.test(hostPlatform)
  return false
}

export function stageClosure({ artifactRoot, outputDir, driver, wrapper, core, browsers, platform = `${process.platform === 'win32' ? 'windows' : process.platform}-${process.arch}`, hostPlatform, matrix = supportMatrix }) {
  if (!artifactRoot || !outputDir) fail('both --artifact-root and --output-dir are required')
  let pair
  try {
    pair = inspectBrowserPair({
      driver, wrapper, core,
      browsers: browsers?.map((browser) => ({
        ...browser,
        revision: browser.revisionOverrides?.[hostPlatform] || browser.revision,
      })),
    })
  } catch (error) {
    fail(`unsupported driver/browser package pair: ${error instanceof Error ? error.message : String(error)}`)
  }
  const layout = layoutFor(platform)
  const support = supportCellFor(platform, matrix)
  if (!compatibleHost(platform, hostPlatform)) fail(`Playwright host platform ${hostPlatform || '(missing)'} does not match target ${platform}`)
  const input = realpathSync(resolve(artifactRoot))
  const out = resolve(outputDir)
  const cache = resolve(process.env.HOME || '', '.cache/ms-playwright')
  if (input === cache || input.startsWith(cache + sep)) fail('developer Playwright cache is not an admitted artifact source')
  const descriptor = (name) => {
    const descriptor = browsers.find((item) => item.name === name)
    if (!descriptor) fail(`pinned browsers.json revision missing for ${name}`)
    return descriptor
  }
  const chromium = descriptor('chromium')
  const shell = descriptor('chromium-headless-shell')
  const ffmpeg = descriptor('ffmpeg')
  const chromiumRev = pair.chromiumRevision
  const shellRev = pair.headlessShellRevision
  const ffmpegRev = String(ffmpeg.revisionOverrides?.[hostPlatform] || ffmpeg.revision)
  if (!/^\d+$/.test(ffmpegRev)) fail('invalid pinned ffmpeg browsers.json revision')
  const chromiumFolder = revisionFolder(chromium, hostPlatform)
  const shellFolder = revisionFolder(shell, hostPlatform)
  const ffmpegFolder = revisionFolder(ffmpeg, hostPlatform)
  const entries = [
    [chromiumFolder, join(chromiumFolder, layout.chromium)],
    [shellFolder, join(shellFolder, layout.shell)],
    [ffmpegFolder, join(ffmpegFolder, layout.ffmpeg)],
  ]
  for (const [folder, file] of entries) {
    const executableSource = join(input, file)
    const executableStat = lstatSync(executableSource)
    if (!executableStat.isFile() || (platform !== 'windows-x64' && (executableStat.mode & 0o111) === 0)) fail(`pinned artifact executable absent or not executable: ${executableSource}`)
  }
  const staging = `${out}.stage-${process.pid}`
  rmSync(staging, { recursive: true, force: true })
  mkdirSync(staging, { recursive: true })
  try {
    for (const [folder] of entries) {
      const src = join(input, folder)
      if (!lstatSync(src).isDirectory()) fail(`pinned artifact directory missing: ${src}`)
      safeTreeCopy(src, join(staging, folder), input)
    }
    const executablePath = entries[0][1]
    const executable = join(staging, executablePath)
    const runtimeIdentity = newRuntimeIdentity({ directory: staging, engine: 'chromium', browserVersion: chromiumRev, driver: 'rebrowser-playwright-core', driverVersion: pair.coreVersion })
    const closure = {
      schema_version: 'v3', platform, playwright_host_platform: hostPlatform, runtime_identity: runtimeIdentity,
      headless_shell_revision: shellRev, ffmpeg_revision: ffmpegRev,
      executable_path: executablePath, executable_sha256: sha256(executable),
      support_matrix_schema: supportMatrix.schema_version,
      bundle_layout_status: support.bundle_layout,
      browser_artifact_status: support.browser_artifact,
      clean_host_status: support.clean_host,
      online_acquisition_status: matrix.delivery_channels.online_acquisition.status,
      offline_transfer_status: matrix.delivery_channels.offline_transfer.status,
      offline_transfer_scope: matrix.delivery_channels.offline_transfer.scope,
      offline_browser_artifact_status: matrix.delivery_channels.offline_transfer.browser_artifact,
      offline_clean_host_status: matrix.delivery_channels.offline_transfer.clean_host,
      target_lifecycle_status: support.lifecycle_contract,
    }
    writeFileSync(join(staging, 'browser-closure.json'), `${JSON.stringify(closure, null, 2)}\n`, { mode: 0o644 })
    rmSync(out, { recursive: true, force: true })
    cpSync(staging, out, { recursive: true, preserveTimestamps: true })
  } finally { rmSync(staging, { recursive: true, force: true }) }
  return JSON.parse(readFileSync(join(out, 'browser-closure.json'), 'utf8'))
}

function cli() {
  const args = process.argv.slice(2)
  const val = (key) => { const i = args.indexOf(key); return i >= 0 ? args[i + 1] : undefined }
  const artifactRoot = val('--artifact-root')
  const outputDir = val('--output-dir')
  const targetPlatform = val('--target-platform') || `${process.platform === 'win32' ? 'windows' : process.platform}-${process.arch}`
  if (!artifactRoot || !outputDir) fail('usage: node stage-browser-closure.mjs --artifact-root <SDA-admitted-tree> --output-dir <bundle-resources/playwright>')
  const driverDir = resolve(dirname(fileURLToPath(import.meta.url)), '..', 'playwright-driver')
  const req = createRequire(join(driverDir, 'package.json'))
  const driver = JSON.parse(readFileSync(join(driverDir, 'package.json'), 'utf8'))
  const wrapperPackagePath = req.resolve('rebrowser-playwright/package.json')
  const wrapperDir = dirname(wrapperPackagePath)
  const wrapper = JSON.parse(readFileSync(wrapperPackagePath, 'utf8'))
  const corePackagePath = createRequire(join(wrapperDir, 'package.json')).resolve('playwright-core/package.json')
  const core = JSON.parse(readFileSync(corePackagePath, 'utf8'))
  const browsers = JSON.parse(readFileSync(join(dirname(corePackagePath), 'browsers.json'), 'utf8')).browsers
  const hostPlatformModule = join(dirname(corePackagePath), 'lib', 'server', 'utils', 'hostPlatform.js')
  const current = createRequire(hostPlatformModule)(hostPlatformModule).hostPlatform
  const hostPlatform = val('--playwright-host-platform') || current
  const platform = targetPlatform
  const result = stageClosure({ artifactRoot, outputDir, driver, wrapper, core, browsers, platform, hostPlatform })
  process.stdout.write(`${JSON.stringify(result, null, 2)}\n`)
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  try { cli() } catch (error) { console.error(error.message); process.exitCode = 1 }
}
