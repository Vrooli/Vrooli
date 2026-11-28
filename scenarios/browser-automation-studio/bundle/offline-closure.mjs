#!/usr/bin/env node
// Offline closure producer/installer. Metadata is derived from the installed
// driver owner; all file/archive verification is in the shared Go contract.
import { createRequire } from 'node:module'
import { dirname, resolve, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { spawnSync } from 'node:child_process'
import { readFileSync } from 'node:fs'

function fail(message) { console.error(`offline-closure: ${message}`); process.exit(1) }
const [command, ...args] = process.argv.slice(2)
if (!['package','install','rollback','remove'].includes(command)) fail('usage: node offline-closure.mjs package|install|rollback|remove [--archive <path> --manifest <path>] [--closure <path>|--destination <path>]')
const val = (key) => { const i=args.indexOf(key); return i>=0?args[i+1]:undefined }
const archiveRaw=val('--archive'), manifestRaw=val('--manifest')
const archive=archiveRaw?resolve(process.cwd(),archiveRaw):undefined
const manifest=manifestRaw?resolve(process.cwd(),manifestRaw):undefined
if(['package','install'].includes(command)&&(!archive||!manifest)) fail('--archive and --manifest are required')
const scriptDir=dirname(fileURLToPath(import.meta.url))
const driverDir=resolve(scriptDir,'..','playwright-driver')
const req=createRequire(join(driverDir,'package.json'))
const wrapperDir=dirname(req.resolve('rebrowser-playwright/package.json'))
const corePackagePath=createRequire(join(wrapperDir,'package.json')).resolve('playwright-core/package.json')
const corePackage=JSON.parse(readFileSync(corePackagePath,'utf8'))
if(corePackage.name!=='rebrowser-playwright-core') fail(`unexpected core package: ${corePackage.name}`)
const browsers=JSON.parse(readFileSync(join(dirname(corePackagePath),'browsers.json'),'utf8')).browsers
const hostPlatformPath=join(dirname(corePackagePath),'lib','server','utils','hostPlatform.js')
const currentHostPlatform=createRequire(hostPlatformPath)(hostPlatformPath).hostPlatform
const chromium=browsers.find(row=>row.name==='chromium')
const headless=browsers.find(row=>row.name==='chromium-headless-shell')
const ffmpeg=browsers.find(row=>row.name==='ffmpeg')
if(!chromium?.revision||!headless?.revision||!ffmpeg?.revision) fail('pinned Chromium closure revisions missing from browsers.json')
const repoRoot=resolve(scriptDir,'../../..')
const platform=val('--target-platform')||`${process.platform==='win32'?'windows':process.platform}-${process.arch}`
const hostPlatform=val('--playwright-host-platform')||currentHostPlatform
const chromiumRevision=chromium.revisionOverrides?.[hostPlatform]||chromium.revision
const headlessRevision=headless.revisionOverrides?.[hostPlatform]||headless.revision
const ffmpegRevision=ffmpeg.revisionOverrides?.[hostPlatform]||ffmpeg.revision
const goArgs=['run','./scenarios/browser-automation-studio/bundle/offline_closure.go',command,'--platform',platform,'--core-version',corePackage.version,'--chromium-revision',String(chromiumRevision),'--headless-shell-revision',String(headlessRevision),'--ffmpeg-revision',String(ffmpegRevision),'--playwright-host-platform',hostPlatform]
if(archive&&manifest) goArgs.push('--archive',archive,'--manifest',manifest)
const localPathRaw=val(command==='package'?'--closure':'--destination')
if(!localPathRaw) fail(command==='package'?'--closure is required':'--destination is required')
const localPath=resolve(process.cwd(),localPathRaw)
goArgs.push(command==='package'?'--closure':'--destination',localPath)
const result=spawnSync('go',goArgs,{cwd:repoRoot,stdio:'inherit'})
if(result.error) fail(result.error.message)
if(result.status!==0) process.exit(result.status||1)
