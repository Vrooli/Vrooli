/**
 * Proves the world layer rule is enforced by ESLint, not only documented.
 * Runs the real project config against the fixtures under sim/__lint__.
 */
import { resolve } from 'node:path'
import { ESLint } from 'eslint'
import { describe, expect, it } from 'vitest'

const UI_ROOT = resolve(import.meta.dirname, '../../..')
const FIXTURES = resolve(UI_ROOT, 'src/world/sim/__lint__')

async function lint(file: string) {
  const eslint = new ESLint({ cwd: UI_ROOT, ignore: false })
  const [result] = await eslint.lintFiles([resolve(FIXTURES, file)])
  return result?.messages.filter((m) => m.severity === 2) ?? []
}

describe('world layer rule', () => {
  it('rejects sim importing scene with no-restricted-paths', async () => {
    const errors = await lint('bad-layer-import.fixture.ts')
    expect(errors.map((e) => e.ruleId)).toEqual(['import/no-restricted-paths'])
    expect(errors[0]?.message).toContain('world/sim must not import world/scene')
  }, 120_000)

  it('rejects sim importing three with no-restricted-imports', async () => {
    const errors = await lint('bad-three-import.fixture.ts')
    expect(errors.map((e) => e.ruleId)).toEqual(['no-restricted-imports'])
  }, 120_000)

  it('accepts sim importing config', async () => {
    const errors = await lint('good-config-import.fixture.ts')
    expect(errors).toEqual([])
  }, 120_000)
})
