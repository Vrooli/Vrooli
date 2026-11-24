import { describe, expect, it } from 'vitest'
import * as fs from 'node:fs'
import * as path from 'node:path'
import { fileURLToPath } from 'node:url'

const here = path.dirname(fileURLToPath(import.meta.url))
const schemasDir = path.resolve(here, '../../../schemas')

describe('published schemas', () => {
  it('parses every schema and gives it a unique id', () => {
    const files = fs.readdirSync(schemasDir).filter((name) => name.endsWith('.schema.json'))
    expect(files.length).toBeGreaterThanOrEqual(6)
    const ids = new Set<string>()
    files.forEach((name) => {
      const schema = JSON.parse(fs.readFileSync(path.join(schemasDir, name), 'utf-8')) as { $id?: string; $schema?: string }
      expect(schema.$id, name).toBeTruthy()
      expect(schema.$schema).toContain('json-schema.org')
      expect(ids.has(schema.$id!)).toBe(false)
      ids.add(schema.$id!)
    })
  })
})
