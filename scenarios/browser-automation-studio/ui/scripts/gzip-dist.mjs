import fs from 'node:fs'
import path from 'node:path'
import zlib from 'node:zlib'

const distDir = path.resolve(process.cwd(), 'dist')

const CONTENT_TYPES = new Map([
  ['.js', true],
  ['.css', true],
  ['.html', true],
  ['.json', true],
  ['.svg', true],
])

function shouldGzip(filePath) {
  if (filePath.endsWith('.gz')) return false
  const ext = path.extname(filePath)
  return CONTENT_TYPES.has(ext)
}

function walk(dir, onFile) {
  const entries = fs.readdirSync(dir, { withFileTypes: true })
  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      walk(fullPath, onFile)
      continue
    }
    if (entry.isFile()) {
      onFile(fullPath)
    }
  }
}

if (!fs.existsSync(distDir)) {
  process.exit(0)
}

let written = 0

walk(distDir, (filePath) => {
  if (!shouldGzip(filePath)) return

  const gzPath = `${filePath}.gz`
  const srcStat = fs.statSync(filePath)
  const gzStat = fs.existsSync(gzPath) ? fs.statSync(gzPath) : null
  if (gzStat && gzStat.mtimeMs >= srcStat.mtimeMs) {
    return
  }

  const src = fs.readFileSync(filePath)
  const gz = zlib.gzipSync(src, { level: 6 })
  fs.writeFileSync(gzPath, gz)
  written += 1
})

if (process.env.VROOLI_VERBOSE_GZIP === '1') {
  // eslint-disable-next-line no-console
  console.log(`[gzip-dist] wrote ${written} file(s)`)
}

