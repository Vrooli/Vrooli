import { resolve } from 'node:path'
import { mkdtempSync, mkdirSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { describe, expect, it } from 'vitest'
import { literalAllowlists, scanLiterals, scanLiteralSource } from './literals'

describe('literal scanner', () => {
  it('walks production TS and TSX files while leaving test fixtures out of the gate', () => {
    const root = mkdtempSync(resolve(tmpdir(), 'world-literal-test-'))
    try {
      mkdirSync(resolve(root, '__tests__'))
      writeFileSync(resolve(root, 'introduced.ts'), 'const x = 3.7')
      writeFileSync(resolve(root, 'mesh.tsx'), 'const mesh = <mesh opacity={0.37} />')
      writeFileSync(resolve(root, '__tests__/fixture.ts'), 'const value = 999')
      writeFileSync(resolve(root, 'mesh.test.tsx'), 'const value = 999')
      expect(scanLiterals(root, literalAllowlists.sim).map((entry) => [entry.file, entry.literal])).toEqual([
        ['introduced.ts', '3.7'], ['mesh.tsx', '0.37'],
      ])
    } finally { rmSync(root, { recursive: true, force: true }) }
  })

  it('finds a deliberately introduced sim constant with its original line', () => {
    const source = '/* comment\n with 3.7 */\nconst ignored = "3.7"\nconst x = 3.7\n'
    expect(scanLiteralSource(source, 'deliberate.ts', literalAllowlists.sim)).toEqual([
      { file: 'deliberate.ts', line: 4, column: 11, literal: '3.7', context: 'const x = 3.7', kind: 'number' },
    ])
  })

  it('ignores ordinary text but scans executable template expressions, JSX colours and shader settings', () => {
    const source = 'const text = `text 3.7 ${4.2}`; const jsx = <mesh color="#abcdef" opacity={0.37} />; const material = { vertexShader: `float x = 300.0; vec4 p = vec4(1.0);` }'
    expect(scanLiteralSource(source, 'fixture.tsx', literalAllowlists.scene).map((entry) => entry.literal)).toEqual(['4.2', '#abcdef', '0.37', '300.0'])
  })

  it('recognizes structural indices without suppressing the same number elsewhere', () => {
    expect(scanLiteralSource('const x = items[42]; const size = 42', 'fixture.ts', literalAllowlists.engine).map((entry) => entry.literal)).toEqual(['42'])
  })

  it('inspects shader tails, injected GLSL and functional colour strings', () => {
    expect(scanLiteralSource('const material = { "vertexShader": `float a = 24.0; ${source} float b = 300.0;` }; const colour = "rgba(255,255,255,0.55)"', 'fixture.ts', literalAllowlists.scene).map((entry) => entry.literal)).toEqual(['24.0', '300.0', 'rgba(255,255,255,0.55)'])
    expect(scanLiteralSource('const injection = `float a = 24.0;`', 'injection.glsl.ts', literalAllowlists.engine).map((entry) => entry.literal)).toEqual(['24.0'])
  })

  it('requires a reason and a selector for every allowance', () => {
    expect(() => scanLiteralSource('const x = 3.7', 'fixture.ts', [{ literal: '3.7', reason: '' }])).toThrow()
    expect(() => scanLiteralSource('const x = 3.7', 'fixture.ts', [{ reason: 'anything' }])).toThrow()
    expect(() => scanLiteralSource('const x = 3.7', 'fixture.ts', [{ file: 'fixture.ts', scope: 'x', reason: 'Too broad' }])).toThrow()
    Object.values(literalAllowlists).flat().forEach((entry) => expect(entry.reason.trim().length).toBeGreaterThan(0))
  })

  it('restricts algorithm exceptions to the exact file, declaration and listed values', () => {
    const source = 'function actorSeed() { return 16777619 }; const unrelated = 16777619'
    expect(scanLiteralSource(source, 'actors/pose.ts', literalAllowlists.scene).map((f) => f.literal)).toEqual(['16777619'])
    expect(scanLiteralSource(source, 'other.ts', literalAllowlists.scene)).toHaveLength(2)
    const shader = 'const SIMPLEX_NOISE_3D = `float n = 42.0; float artistic = 0.37;`; const other = `float n = 42.0;`'
    expect(scanLiteralSource(shader, 'materials/slime.glsl.ts', literalAllowlists.engine).map((f) => f.literal)).toEqual(['0.37', '42.0'])
    for (const allowlist of Object.values(literalAllowlists)) {
      expect(scanLiteralSource('const futureLever = 3.7', 'introduced.ts', allowlist)).toHaveLength(1)
    }
  })
})

describe('all world layers expose their behaviour and visual settings', () => {
  it.each(Object.entries(literalAllowlists))('%s has no unconfigured literals', (layer, allowlist) => {
    expect(scanLiterals(resolve(import.meta.dirname, '..', layer), allowlist)).toEqual([])
  })
})
