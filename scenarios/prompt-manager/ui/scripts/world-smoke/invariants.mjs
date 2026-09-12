#!/usr/bin/env node
/** Renderer-free evidence for the same shared scene/seed sweep used by tests. */
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { createServer } from 'vite'
import ts from 'typescript'

const args = process.argv.slice(2)
const at = args.indexOf('--evidence-dir')
if (at < 0 || !args[at + 1]) throw new Error('--evidence-dir is required')
const evidenceDir = resolve(args[at + 1])
const uiRoot = resolve(import.meta.dirname, '../..')
const loader = await createServer({ root: uiRoot, configFile: false, server: { middlewareMode: true, hmr: false, watch: null }, logLevel: 'error', appType: 'custom' })
try {
  const { makeWorld } = await loader.ssrLoadModule('/src/world/sim/__tests__/fixtures.ts')
  const { SWEEP_SEEDS } = await loader.ssrLoadModule('/src/world/sim/__tests__/seeds.ts')
  const { checkWorldInvariants, checkVegetationDry } = await loader.ssrLoadModule('/src/world/sim/invariants.ts')
  const { terrainForBounds } = await loader.ssrLoadModule('/src/world/sim/layout/centre.ts')
  const { tuning, scenes, SCENE_IDS } = await loader.ssrLoadModule('/src/world/config/index.ts')
  const source = ts.createSourceFile('invariants.ts', readFileSync(resolve(uiRoot, 'src/world/sim/invariants.ts'), 'utf8'), ts.ScriptTarget.Latest, true)
  const declaration = source.statements.find((node) => ts.isTypeAliasDeclaration(node) && node.name.text === 'InvariantRule')
  if (!declaration || !ts.isUnionTypeNode(declaration.type)) throw new Error('Invariant rule vocabulary is unavailable')
  const rules = declaration.type.types.map((node) => {
    if (!ts.isLiteralTypeNode(node) || !ts.isStringLiteral(node.literal)) throw new Error('Unexpected invariant rule declaration')
    return node.literal.text
  })
  const rows = SCENE_IDS.flatMap((scene) => SWEEP_SEEDS.map((seed) => {
    const state = makeWorld({ scene, seed, teams: 5, agents: 25 })
    const violations = checkWorldInvariants(state, tuning)
    const vegetation = checkVegetationDry(state, terrainForBounds(scenes[scene], tuning.terrain, state.bounds), tuning.layout)
    if (violations.some((v) => !rules.includes(v.rule))) throw new Error('Unknown invariant violation rule')
    return { scene, seed, actors: Object.keys(state.actors).length, decor: state.decor.length, violations, vegetationViolations: vegetation.length, byRule: Object.fromEntries(rules.map((rule) => [rule, violations.filter((v) => v.rule === rule).length])) }
  }))
  const pass = rows.every((row) => row.violations.length === 0 && row.vegetationViolations === 0)
  const report = { capturedAt: new Date().toISOString(), method: 'renderer-free production invariant checks; 25 actors; shared SWEEP_SEEDS', pass, rules, rows }
  mkdirSync(evidenceDir, { recursive: true })
  writeFileSync(resolve(evidenceDir, 'invariants.json'), JSON.stringify(report, null, 2) + '\n')
  const markdown = ['# Shared scene/seed invariant sweep', '', report.method + '.', '', 'Command (from the UI directory):', '', '```sh', 'node scripts/world-smoke/invariants.mjs --evidence-dir ' + evidenceDir, '```', '', 'Per-rule counts and full violations are in `invariants.json`. Counts report violations, not branch coverage; scene-inapplicable rules can also report zero.', '', '| Scene | Seed | Actors | Decor checked | Vegetation violations | All violations |', '| --- | ---: | ---: | ---: | ---: | ---: |', ...rows.map((r) => `| ${r.scene} | ${r.seed} | ${r.actors} | ${r.decor} | ${r.vegetationViolations} | ${r.violations.length} |`), '', 'Verdict: ' + (pass ? 'PASS' : 'FAIL') + '.', '']
  writeFileSync(resolve(evidenceDir, 'invariants.md'), markdown.join('\n'))
  console.log(`${pass ? 'PASS' : 'FAIL'} ${rows.length} scene/seed cases; ${rules.length} rule counts per case; ${evidenceDir}/invariants.json`)
  if (!pass) process.exitCode = 1
} finally {
  await loader.close()
}
