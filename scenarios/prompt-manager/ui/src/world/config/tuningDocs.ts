import { tuningSettingImpact, type SettingImpact } from './settingImpact'
import { numericOverrides, choiceSettings, integerSettings } from './settings'
/**
 * Renders the tuning schema as the markdown lever table that lives in
 * docs/reference/configuration.md between the world-tuning markers.
 *
 * Both `pnpm world:tuning-docs` (writes the doc) and `tuning.test.ts` (asserts
 * the doc is current) call `renderTuningDocs`, so the doc cannot drift.
 */
import { z } from 'zod'
import { WorldTuningSchema } from './tuning.schema'
import { tuning } from './tuning'
import { SceneSchema } from './scenes.schema'
import { scenes } from './scenes'
import { BiomeSetSchema, VegetationEntrySchema } from './biomes.schema'
import { biomeSets } from './biomes'

export const TUNING_DOC_BEGIN = '<!-- world-tuning:begin -->'
export const TUNING_DOC_END = '<!-- world-tuning:end -->'

interface JsonSchemaNode {
  type?: string | string[]
  properties?: Record<string, JsonSchemaNode>
  items?: JsonSchemaNode
  description?: string
  minimum?: number
  maximum?: number
  enum?: unknown[]
  const?: unknown
  readOnly?: boolean
  minItems?: number
  maxItems?: number
}

export interface LeverRow {
  impact?: SettingImpact
  readOnly?: boolean
  choices?: string[]
  path: string
  type: string
  bounds: string
  value: string
  description: string
}

function valueAt(root: unknown, path: string[]): unknown {
  let cur: unknown = root
  for (const key of path) {
    if (typeof cur !== 'object' || cur === null) return undefined
    cur = (cur as Record<string, unknown>)[key]
  }
  return cur
}

function describeType(node: JsonSchemaNode): string {
  if (node.enum) return node.enum.map((v) => JSON.stringify(v)).join(' | ')
  if (node.const !== undefined) return `const ${JSON.stringify(node.const)}`
  if (Array.isArray(node.type)) return node.type.join(' | ')
  if (node.type === 'array') return `array<${node.items ? describeType(node.items) : 'unknown'}>`
  return node.type ?? 'unknown'
}

function describeBounds(node: JsonSchemaNode): string {
  const parts: string[] = []
  if (node.minimum !== undefined) parts.push(`min ${node.minimum}`)
  if (node.maximum !== undefined) parts.push(`max ${node.maximum}`)
  if (node.minItems !== undefined && node.minItems === node.maxItems) parts.push(`length ${node.minItems}`)
  return parts.join(', ') || '—'
}

export function collectLevers(root: unknown = z.toJSONSchema(WorldTuningSchema), values: unknown = tuning): LeverRow[] {
  const rows: LeverRow[] = []
  const walk = (node: JsonSchemaNode, path: string[]) => {
    if (node.properties) {
      for (const [key, child] of Object.entries(node.properties)) walk(child, [...path, key])
      return
    }
    const value = valueAt(values, path)
    rows.push({
      path: path.join('.'),
      impact: tuningSettingImpact(path.join('.')),
      ...(node.readOnly ? { readOnly: true } : {}),
      ...(node.enum?.every(value => typeof value === 'string') ? { choices: node.enum } : {}),
      type: describeType(node),
      bounds: describeBounds(node),
      value: value === undefined ? '—' : JSON.stringify(value),
      description: node.description ?? '',
    })
  }
  walk(root as JsonSchemaNode, [])
  return rows
}

/** Composition lives in scene/biome data, not world.tuning.json. Keep those new
 * levers in the same generated reference without inventing absent overrides. */
export function collectCompositionLevers(): LeverRow[] {
  const sceneSchema = z.toJSONSchema(SceneSchema.pick({ centre: true, emissive: true, biomeSet: true, assetSet: true }))
  const { assetSet, propScale, treeScale } = BiomeSetSchema.shape
  const landscapeSchema = z.toJSONSchema(z.object({ assetSet, propScale, treeScale }))
  return [
    ...Object.entries(scenes).flatMap(([id, scene]) => collectLevers(sceneSchema, scene).map((row) => ({ ...row, path: `scenes.${id}.${row.path}` }))),
    ...Object.entries(biomeSets).flatMap(([id, set]) => collectLevers(landscapeSchema, set).map((row) => ({ ...row, path: `biomes.${id}.${row.path}` }))),
    ...collectLevers(z.toJSONSchema(VegetationEntrySchema), {}).map((row) => ({ ...row, path: `vegetationEntry.${row.path}` })),
  ]
}

/** Every leaf lever must carry a description; returns the offenders. */
export function undocumentedLevers(rows: LeverRow[] = collectLevers()): string[] {
  return rows.filter((r) => !r.description.trim()).map((r) => r.path)
}

function escapeCell(text: string): string {
  return text.replace(/\|/g, '\\|').replace(/\n/g, ' ')
}

export function renderTuningDocs(rows: LeverRow[] = [...collectLevers(), ...collectCompositionLevers()]): string {
  const groups = new Map<string, LeverRow[]>()
  for (const row of rows) {
    const group = row.path.split('.')[0] ?? row.path
    const list = groups.get(group) ?? []
    list.push(row)
    groups.set(group, list)
  }
  const lines: string[] = [
    TUNING_DOC_BEGIN,
    '## World tuning levers',
    '',
    'Source: `ui/src/world/config/world.tuning.json`, validated by `tuning.schema.ts`.',
    'Composition rows also come from `scenes/*.json` and `biomes.json`; vegetation-entry',
    'rows define each per-prop record. An em dash means no override or no shared default.',
    'Structural constants are separately reviewed by the literal gate. Edit the JSON, keep values inside their bounds, and run',
    '`pnpm world:tuning-docs` to refresh this table (a test fails when it is stale).',
    'In development the HUD settings panel has a Levers tab that edits these live.',
    '',
  ]
  for (const [group, list] of groups) {
    lines.push(`### \`${group}\``, '', '| Lever | Type | Bounds | Default | Effect |', '|---|---|---|---|---|')
    for (const row of list) {
      lines.push(
        `| \`${row.path}\` | ${escapeCell(row.type)} | ${escapeCell(row.bounds)} | \`${escapeCell(row.value)}\` | ${escapeCell(row.description)} |`,
      )
    }
    lines.push('')
  }
  lines.push('### Integer URL settings', '', '| Setting | Bounds | Default | Impact | Persistence | Effect |', '|---|---|---|---|---|---|')
  for (const setting of Object.values(integerSettings)) lines.push(`| \`${setting.id}\` | ${setting.minimum}–${setting.maximum} | ${setting.defaultValue} | ${setting.impact} | ${setting.persistence} | ${escapeCell(setting.description)} |`)
  lines.push('', 'Values must contain decimal digits only. Invalid values use the declared default and display an error.', '')
  lines.push('### Choice URL settings', '', '| Setting | Choices | Default | Impact | Persistence | Effect |', '|---|---|---|---|---|---|')
  for (const setting of Object.values(choiceSettings)) lines.push(`| \`${setting.id}\` | ${setting.choices.map(choice => choice.id).join(', ')} | ${setting.defaultValue} | ${setting.impact} | ${setting.persistence} | ${escapeCell(setting.description)} |`)
  lines.push('', 'Choice IDs are case-sensitive. Invalid values use the declared default and display an error; absent values may use saved operator preferences.', '')
  lines.push('### Numeric diagnostic overrides', '', '| Setting | Bounds | Default | Unit | Impact | Effect |', '|---|---|---|---|---|---|')
  for (const setting of Object.values(numericOverrides)) lines.push(`| \`${setting.id}\` | ${setting.schema.minValue}–${setting.schema.maxValue}${(setting.schema.format === 'safeint') ? ', integer' : ''} | ${setting.defaultSource} | ${setting.unit} | ${setting.impact} | ${escapeCell(setting.description)} |`)
  lines.push('', 'These development overrides accept plain decimal notation. Invalid values retain the active profile or automatic default and display an error; values are not clamped or rounded.', '')
  lines.push('### Layout setting impacts', '', '| Setting | Impact |', '|---|---|')
  for (const row of collectLevers().filter(row => row.path.startsWith('layout.'))) lines.push(`| \`${row.path}\` | ${row.impact ?? 'undeclared'} |`)
  lines.push('', 'Renderer-owned surface, wall-height and lamp-placement edits retain generated terrain, navigation and room placement. Structural layout edits still regenerate.', '')
  lines.push('### Actor setting impacts', '', '| Setting | Impact |', '|---|---|')
  for (const row of collectLevers().filter(row => row.path.startsWith('actor.'))) lines.push(`| \`${row.path}\` | ${row.impact ?? 'undeclared'} |`)
  lines.push('', 'Live edits update simulation or instance data; material edits update surface properties or procedural texture content; geometry edits rebuild the affected shape. Body radius changes regenerate navigation clearance.', '')
  lines.push('### Simulation setting impacts', '', '| Setting | Impact |', '|---|---|')
  for (const row of collectLevers().filter(row => row.path.startsWith('sim.'))) lines.push(`| \`${row.path}\` | ${row.impact ?? 'undeclared'} |`)
  lines.push('', 'Simulation edits retain generated world buffers. Cache capacity and event-history limits apply at commit; motion and scheduling rules use the new values on subsequent simulation steps.', '')
  lines.push('### Camera setting impacts', '', '| Setting | Impact |', '|---|---|')
  for (const row of collectLevers().filter(row => row.path.startsWith('camera.'))) lines.push(`| \`${row.path}\` | ${row.readOnly ? 'initialization only' : row.impact ?? 'undeclared'} |`)
  lines.push('', 'Camera edits retain generation buffers. Lens, input and navigation values update the current rig; framing and transition settings govern the next corresponding camera operation. Initial position is a bootstrap value, not an editable live control.', '')
  lines.push('### Label setting impacts', '', '| Setting | Impact |', '|---|---|')
  for (const row of collectLevers().filter(row => row.path.startsWith('labels.'))) lines.push(`| \`${row.path}\` | ${row.impact ?? 'undeclared'} |`)
  lines.push('', 'Label surface edits update text materials; size edits synchronize glyph geometry and budget edits resize the bounded text pool. Selection, placement and refresh settings update the existing label pipeline. No label setting regenerates terrain or navigation.', '')
  lines.push(TUNING_DOC_END)
  return lines.join('\n')
}

/** Replace (or append) the generated block inside a markdown document. */
export function spliceTuningDocs(markdown: string, rendered: string = renderTuningDocs()): string {
  const begin = markdown.indexOf(TUNING_DOC_BEGIN)
  const end = markdown.indexOf(TUNING_DOC_END)
  if (begin === -1 || end === -1 || end < begin) {
    return `${markdown.trimEnd()}\n\n${rendered}\n`
  }
  return `${markdown.slice(0, begin)}${rendered}${markdown.slice(end + TUNING_DOC_END.length)}`
}

/** The generated block currently present in a markdown document, or null. */
export function extractTuningDocs(markdown: string): string | null {
  const begin = markdown.indexOf(TUNING_DOC_BEGIN)
  const end = markdown.indexOf(TUNING_DOC_END)
  if (begin === -1 || end === -1 || end < begin) return null
  return markdown.slice(begin, end + TUNING_DOC_END.length)
}
