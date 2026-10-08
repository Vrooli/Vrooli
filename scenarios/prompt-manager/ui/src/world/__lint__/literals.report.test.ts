import { mkdirSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { it } from 'vitest'
import { literalAllowlists, scanLiterals } from './literals'

it('writes the complete report alongside the separately enforced visual gate', () => {
  const root = resolve(import.meta.dirname, '..')
  const layers = Object.entries(literalAllowlists).map(([layer, allowlist]) => ({ layer, offenders: scanLiterals(resolve(root, layer), allowlist) }))
  const directory = resolve(root, '../../evidence')
  mkdirSync(directory, { recursive: true })
  writeFileSync(resolve(directory, 'literal-scan.json'), JSON.stringify({ mode: 'enforced', layers }, null, 2) + '\n')
  console.log(JSON.stringify(layers.map(({ layer, offenders }) => ({ layer, count: offenders.length }))))
})
