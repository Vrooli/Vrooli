import { chmodSync, mkdtempSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { afterEach, test } from 'node:test'
import assert from 'node:assert/strict'
import { createHash } from 'node:crypto'
import { stageClosure, supportCellFor, validateSupportMatrix } from './stage-browser-closure.mjs'

const supportMatrix = JSON.parse(readFileSync(new URL('./browser-support-matrix.json', import.meta.url), 'utf8'))

const roots = []
function fixture(platform='linux-x64', hostPlatform='ubuntu24.04-x64') {
  const root = mkdtempSync(join(tmpdir(), 'bas-browser-closure-')); roots.push(root)
  const artifact = join(root, 'artifact'), output = join(root, 'bundle', 'resources', 'playwright')
  for (const [dir, file] of [['chromium-1169','chrome-linux/chrome'],['chromium_headless_shell-1169','chrome-linux/headless_shell'],['ffmpeg-1011','ffmpeg-linux']]) {
    const path = join(artifact,dir,file); mkdirSync(path.slice(0,path.lastIndexOf('/')),{recursive:true}); writeFileSync(path,`fixture ${dir}`,{mode:0o755})
  }
  return {
    artifactRoot:artifact, outputDir:output, platform, hostPlatform,
    driver:{name:'@vrooli/playwright-driver',dependencies:{'rebrowser-playwright':'1.52.0'}},
    wrapper:{name:'rebrowser-playwright',version:'1.52.0'},
    core:{name:'rebrowser-playwright-core',version:'1.52.0'},
    browsers:[
      {name:'chromium',revision:'1169',browserVersion:'136.0.7103.25'},
      {name:'chromium-headless-shell',revision:'1169',browserVersion:'136.0.7103.25'},
      {name:'ffmpeg',revision:'1011'},
    ],
  }
}
afterEach(() => { for (const root of roots.splice(0)) rmSync(root,{recursive:true,force:true}) })

test('stages only browsers.json pinned versions and emits checksum relative path', () => {
  const f=fixture(), result=stageClosure(f)
  assert.equal(result.schema_version,'v3')
  assert.deepEqual(result.target_lifecycle_status, {
    runtime_launch: 'unqualified-no-receipt', browser_task: 'unqualified-no-receipt',
    web_console_pane: 'unqualified-no-receipt', update_rollback: 'unqualified-no-receipt', uninstall: 'unqualified-no-receipt',
  })
  assert.deepEqual(result.runtime_identity, {
    schema_version: 'v1', engine: 'chromium', browser_version: '1169',
    driver: 'rebrowser-playwright-core', driver_version: '1.52.0',
    bundle_version: 'br1-chromium-1169-rebrowser-playwright-core-1.52.0',
    sha256: result.runtime_identity.sha256,
  })
  assert.match(result.runtime_identity.sha256,/^[0-9a-f]{64}$/)
  assert.equal(result.runtime_identity.sha256,'209def461e976572bba7f722fc1f5f1c010653d3ef11e9b78677ad2d4c4359f5','digest must match packages/browser-closure framing fixture')
  assert.equal(Object.hasOwn(result,'version'),false)
  assert.equal(Object.hasOwn(result,'chromium_revision'),false)
  assert.equal(result.executable_path,'chromium-1169/chrome-linux/chrome')
  assert.equal(result.playwright_host_platform,'ubuntu24.04-x64')
  assert.equal(result.bundle_layout_status, 'source-fixture-tested')
  assert.equal(result.browser_artifact_status, 'unqualified-no-provisioned-artifact')
  assert.equal(result.clean_host_status, 'unqualified-no-receipt')
  assert.equal(result.online_acquisition_status, 'unsupported-no-governed-browser-source')
  assert.equal(result.offline_transfer_status, 'source-fixture-tested-linux-x64')
  assert.equal(result.offline_transfer_scope, 'Deterministic archive and install fixtures run on linux-x64 only.')
  assert.equal(result.offline_browser_artifact_status, 'unqualified-no-provisioned-artifact')
  assert.equal(result.offline_clean_host_status, 'unqualified-no-receipt')
  const bytes=readFileSync(join(f.outputDir,result.executable_path))
  assert.equal(result.executable_sha256, createHash('sha256').update(bytes).digest('hex'))
  assert.throws(()=>readFileSync(join(f.outputDir,'chromium-1170/chrome-linux/chrome')))
  writeFileSync(join(f.artifactRoot,'chromium_headless_shell-1169','chrome-linux','headless_shell'),'changed payload')
  const changed=stageClosure({...f,outputDir:join(f.outputDir,'second')})
  assert.notEqual(changed.runtime_identity.sha256,result.runtime_identity.sha256,'bundle digest must include non-launcher payload files')
})

test('staging rejects the same package/browser skew as launch verification before replacing output', () => {
  const f=fixture()
  mkdirSync(f.outputDir,{recursive:true})
  writeFileSync(join(f.outputDir,'existing-generation'),'keep')
  f.core.version='1.64.102'
  assert.throws(()=>stageClosure(f),/unsupported driver\/browser package pair.*does not match core/)
  assert.equal(readFileSync(join(f.outputDir,'existing-generation'),'utf8'),'keep')
})

test('support matrix marks only Linux x64 source-qualified and keeps hosted worker unsupported', () => {
  assert.equal(supportMatrix.schema_version, 'v3')
  assert.equal(supportCellFor('linux-x64').bundle_layout, 'source-fixture-tested')
  for (const target of ['linux-arm64', 'darwin-x64', 'darwin-arm64', 'windows-x64']) assert.equal(supportCellFor(target).bundle_layout, 'unqualified-no-node')
  assert.throws(() => supportCellFor('windows-arm64'), /no staged bundle layout contract/)
  assert.throws(() => supportCellFor('unlisted-platform'), /no target cell/)
  assert.equal(validateSupportMatrix(supportMatrix)['private-hosted-worker'].status, 'unsupported')
  for (const status of Object.values(supportMatrix['private-hosted-worker'].lifecycle_contract)) {
    assert.equal(status, 'unsupported-no-browser-task-owner')
  }
  for (const [platform, cell] of Object.entries(supportMatrix.targets)) {
    const expected = platform === 'windows-arm64' ? 'unsupported-no-layout' : platform === 'linux-x64' ? 'unqualified-no-receipt' : 'unqualified-no-node'
    assert.deepEqual(Object.keys(cell.lifecycle_contract).sort(), ['browser_task', 'runtime_launch', 'uninstall', 'update_rollback', 'web_console_pane'])
    for (const status of Object.values(cell.lifecycle_contract)) assert.equal(status, expected)
  }
})

test('rejects forged or incomplete support states before replacing staged output', () => {
  for (const mutate of [
    (matrix) => { matrix.targets['linux-x64'].browser_artifact = 'supported' },
    (matrix) => { matrix.targets['linux-x64'].clean_host = 'supported' },
    (matrix) => { matrix.targets['linux-x64'].lifecycle_contract.browser_task = 'supported' },
    (matrix) => { delete matrix.targets['linux-x64'].lifecycle_contract.uninstall },
    (matrix) => { matrix.targets['windows-arm64'].lifecycle_contract.runtime_launch = 'unqualified-no-receipt' },
    (matrix) => { matrix['private-hosted-worker'].status = 'supported' },
    (matrix) => { matrix['private-hosted-worker'].lifecycle_contract.workspace_isolation = 'supported' },
    (matrix) => { matrix['private-hosted-worker'].lifecycle_contract.completion_cancel_crash_teardown = 'supported' },
    (matrix) => { delete matrix['private-hosted-worker'].lifecycle_contract.artifact_integrity_receipt },
    (matrix) => { matrix['private-hosted-worker'].lifecycle_contract.unreviewed_capability = 'unsupported-no-browser-task-owner' },
    (matrix) => { matrix['private-hosted-worker'].owner_gap = '  ' },
    (matrix) => { matrix['private-hosted-worker'].owner_gap = matrix['private-hosted-worker'].owner_gap.replace('remote-worker teardown owner', '') },
    (matrix) => { delete matrix.targets['windows-arm64'] },
    (matrix) => { matrix.delivery_channels.online_acquisition.status = 'supported' },
    (matrix) => { matrix.delivery_channels.offline_transfer.clean_host = 'supported' },
    (matrix) => { delete matrix.delivery_channels.offline_transfer },
  ]) {
    const f = fixture()
    mkdirSync(f.outputDir, { recursive: true })
    writeFileSync(join(f.outputDir, 'existing-generation'), 'keep')
    const matrix = JSON.parse(JSON.stringify(supportMatrix))
    mutate(matrix)
    assert.throws(() => stageClosure({ ...f, matrix }), /invalid|unsupported with a concrete|owner gap|inventory|lifecycle cell|lifecycle contract|acquisition|transfer|must remain/)
    assert.equal(readFileSync(join(f.outputDir, 'existing-generation'), 'utf8'), 'keep')
  }
})

test('rejects unsupported platform, missing pinned directory and non-executable', () => {
  const f=fixture()
  assert.throws(()=>stageClosure({...f,platform:'aix-x64'}),/unsupported target/)
  assert.throws(()=>stageClosure({...f,platform:'darwin-arm64'}),/does not match target/)
  assert.throws(()=>stageClosure({...f,browsers:f.browsers.map(x=>x.name==='chromium'?{...x,revision:'9999'}:x)}),/revisions differ/)
  assert.throws(()=>stageClosure({...f,packageVersion:'unknown'}),/invalid pinned core package version/)
  const exe=join(f.artifactRoot,'chromium-1169','chrome-linux','chrome'); writeFileSync(exe,'no execute'); chmodSync(exe,0o644)
  assert.throws(()=>stageClosure(f),/not executable/)
})

test('stages exact Windows and macOS executable layouts including revision overrides', () => {
  const cases=[
    {platform:'windows-x64',host:'win64',path:'chromium-1169/chrome-win/chrome.exe',shell:'chromium_headless_shell-1169/chrome-win/headless_shell.exe',ffmpeg:'ffmpeg-1011/ffmpeg-win64.exe'},
    {platform:'darwin-x64',host:'mac12',path:'chromium-1169/chrome-mac/Chromium.app/Contents/MacOS/Chromium',shell:'chromium_headless_shell-1169/chrome-mac/headless_shell',ffmpeg:'ffmpeg_mac12_special-1010/ffmpeg-mac'},
    {platform:'darwin-arm64',host:'mac12-arm64',path:'chromium-1169/chrome-mac/Chromium.app/Contents/MacOS/Chromium',shell:'chromium_headless_shell-1169/chrome-mac/headless_shell',ffmpeg:'ffmpeg_mac12_arm64_special-1010/ffmpeg-mac',override:'mac12-arm64'},
  ]
  for(const tc of cases){
    const root=mkdtempSync(join(tmpdir(),'bas-platform-layout-'));roots.push(root)
    const artifactRoot=join(root,'artifact'), outputDir=join(root,'out')
    const browsers=[{name:'chromium',revision:'1169'},{name:'chromium-headless-shell',revision:'1169'},{name:'ffmpeg',revision:'1011',revisionOverrides:{mac12:'1010','mac12-arm64':'1010'}}]
    for(const rel of [tc.path,tc.shell,tc.ffmpeg]){const path=join(artifactRoot,rel);mkdirSync(path.slice(0,path.lastIndexOf('/')),{recursive:true});writeFileSync(path,`fixture ${tc.platform} ${rel}`,{mode:0o755})}
    const result=stageClosure({artifactRoot,outputDir,packageVersion:'1.52.0',browsers,platform:tc.platform,hostPlatform:tc.host})
    assert.equal(result.platform,tc.platform)
    assert.equal(result.playwright_host_platform,tc.host)
    assert.equal(result.bundle_layout_status, 'unqualified-no-node')
    assert.equal(result.target_lifecycle_status.runtime_launch, 'unqualified-no-node')
    assert.equal(result.executable_path,tc.path)
    assert.ok(readFileSync(join(outputDir,tc.path)).length>0)
    if(tc.override) assert.equal(result.ffmpeg_revision,'1010')
  }
})
