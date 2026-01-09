import fs from 'fs'
import path from 'path'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const distDir = path.join(__dirname, 'dist')

fs.rmSync(distDir, { recursive: true, force: true })
fs.mkdirSync(distDir, { recursive: true })

const excluded = new Set([
  'node_modules',
  'dist',
  'package.json',
  'package-lock.json',
  'pnpm-lock.yaml',
  'package.json.backup',
  'package-lock.json.backup',
  'build.mjs'
])

let copied = 0
for (const entry of fs.readdirSync(__dirname)) {
  if (entry.startsWith('.')) {
    continue
  }
  if (excluded.has(entry)) {
    continue
  }
  const source = path.join(__dirname, entry)
  if (!fs.existsSync(source)) {
    continue
  }
  const destination = path.join(distDir, entry)
  fs.cpSync(source, destination, { recursive: true, dereference: true })
  copied += 1
}

console.log(`[accessibility-compliance-hub][ui] Copied ${copied} entries into dist`)
