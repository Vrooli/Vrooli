import { readdirSync, readFileSync } from 'node:fs'
import { join, relative } from 'node:path'
import ts from 'typescript'

export interface LiteralFinding {
  file: string
  line: number
  column: number
  literal: string
  context: string
  kind: 'number' | 'colour' | 'shader-number'
}

export interface LiteralAllowance {
  literal?: string
  literals?: readonly string[]
  scope?: string
  file?: string
  kind?: LiteralFinding['kind'] | 'array-index'
  reason: string
}

const structural: LiteralAllowance[] = [
  ...Array.from({ length: 10 }, (_, value) => ({ literal: String(value), reason: 'Small integers: ordinal ranks, dimensions, indices and vector components; retain the existing sim policy.' })),
  { literal: '0.5', reason: 'One half: midpoint, radius/diameter and symmetric coordinate conversion.' },
  { literal: '1e-9', reason: 'Existing numerical comparison epsilon, not an artistic parameter.' },
  { kind: 'array-index', reason: 'A nonnegative integer indexing a fixed array is structural, not a tuning lever.' },
]

/** Exact paths and values only. No wildcard path or blanket visual suppression. */
export const literalAllowlists: Record<'sim' | 'scene' | 'engine' | 'hud', LiteralAllowance[]> = {
  sim: [
    ...structural,
    { file: 'rng.ts', reason: 'Published PRNG/hash algorithm constants; preserve the original sim exemption.' },
    { file: 'hash.ts', reason: 'Deterministic digest algorithm constants; preserve the original sim exemption.' },
    { file: 'layout/seatMath.ts', reason: 'Geometric seat solver constants; preserve the original sim exemption.' },
    { file: 'terrain/colour.ts', literal: '0x10', reason: 'Hexadecimal radix for colour decoding; the old regex did not inspect hex numbers.' },
    { file: 'terrain/colour.ts', literal: '0xff', reason: 'Eight-bit colour channel normalization; not an adjustable colour.' },
    { file: 'terrain/noise.ts', literal: '0xffffffff', reason: 'Normalize the unsigned32 hash to the unit interval.' },
    { file: 'terrain/noise.ts', literal: '0x9e3779b9', reason: 'Published integer hash mixing increment, part of deterministic noise.' },
    { kind: 'colour', reason: 'The existing sim gate excluded string values. Colour migration belongs to the visual conversion, not the scanner refactor.' },
  ],
  scene: [
    ...structural,
    { file: 'Vegetation.tsx', scope: 'Vegetation', literal: '0xffffffff', reason: 'Unsigned32 maximum normalizes the seeded hash to a unit interval; vegetation density is separately configured.' },
    { file: 'actors/pose.ts', scope: 'actorSeed', literals: ['2166136261', '16777619', '10000'], reason: 'FNV-1a offset/prime and fixed ten-thousand-bin seed mapping; changing these changes actor identity, not an artistic parameter.' },
    { file: 'vegetationCull.ts', scope: 'MATRIX_ELEMENTS', literal: '16', reason: 'A homogeneous4x4 matrix has16 elements; changing its stride corrupts instance matrices.' },
  ],
  engine: [
    ...structural,
    { file: 'diagnostics/Overlay.tsx', scope: 'DiagnosticsOverlay', literal: '100', reason: 'Convert viewport share to a displayed percentage.' },
    { file: 'diagnostics/Probe.tsx', scope: 'DiagnosticsProbe', literal: '100', reason: 'The asset loader reports completion as100 percent, not a configurable threshold.' },
    { file: 'diagnostics/Probe.tsx', scope: 'DiagnosticsProbe', literal: '1000', reason: 'Convert elapsed milliseconds to seconds in the FPS measurement.' },
    { file: 'diagnostics/gpuTimer.ts', scope: 'drain', literal: '1000000', reason: 'WebGL timer results are nanoseconds; diagnostic timings are milliseconds.' },
    { file: 'diagnostics/gpuTimer.ts', scope: 'stats', literal: '0.95', reason: 'The p95 metric is defined as the95th percentile; changing it would mislabel budget evidence.' },
    { file: 'diagnostics/passTimer.ts', scope: 'drain', literal: '1000000', reason: 'WebGL timestamp differences are nanoseconds; pass timings are milliseconds.' },
    { file: 'diagnostics/passTimer.ts', scope: 'stats', literal: '0.95', reason: 'Pass timings report the95th percentile, not a tunable quantile.' },
    { file: 'diagnostics/store.ts', scope: 'recordFrame', literal: '1000', reason: 'R3F frame deltas are seconds; diagnostic frame timings are milliseconds.' },
    { file: 'diagnostics/store.ts', scope: 'frameStats', literal: '0.95', reason: 'The named p95 field must retain its95th-percentile definition.' },
    { file: 'frameDriver.tsx', scope: 'FrameDriver', literal: '1000', reason: 'Convert configured settle seconds to the performance clock millisecond unit.' },
    { file: 'lighting/clock.ts', scope: 'useLightingPeriod', literal: '1000', reason: 'Convert configured polling seconds to setInterval milliseconds.' },
    { file: 'lighting/shadowRefresh.ts', scope: 'useShadowRefresh', literal: '100', reason: 'The asset loader completion indicator is100 percent.' },
    { file: 'materials/slime.glsl.ts', scope: 'SIMPLEX_NOISE_3D', literals: ['289', '34', '10', '1.79284291400159', '0.85373472095314', '0.142857142857', '49', '0.6', '42'], reason: 'Licensed Ashima/Gustavson simplex noise algorithm constants, restricted to that declaration. Wobble strength, scale and speed are configured separately.' },
  ],
  hud: [
    ...structural,
    { file: 'SummaryStrip.tsx', scope: 'SummaryStrip', literal: '100', reason: 'Convert weather pressure from unit ratio to displayed percent.' },
    { file: 'TwoDMode.tsx', scope: 'TwoDMode', literal: '100', reason: 'Convert weather pressure from unit ratio to displayed percent.' },
    { file: 'format.ts', scope: 'formatDuration', literal: '60', reason: 'Sexagesimal seconds/minutes/hours conversion and remainder.' },
    { file: 'format.ts', scope: 'formatCountdown', literal: '60', reason: 'Countdown minute/second quotient and remainder.' },
    { file: 'format.ts', scope: 'formatClock', literal: '1000', reason: 'Convert Unix seconds to JavaScript Date milliseconds.' },
  ],
}

function sourceFiles(root: string): string[] {
  return readdirSync(root, { withFileTypes: true }).flatMap((entry) => {
    const path = join(root, entry.name)
    if (entry.isDirectory()) return ['__tests__', '__lint__', 'node_modules'].includes(entry.name) ? [] : sourceFiles(path)
    return /\.tsx?$/.test(entry.name) && !/\.(?:test|spec|fixture|generated)\.tsx?$/.test(entry.name) ? [path] : []
  }).sort()
}

/** Parse source instead of stripping it: comments/strings cannot shift locations,
 * and executable template interpolations cannot hide a behaviour constant. */
export function scanLiteralSource(source: string, file: string, allowlist: readonly LiteralAllowance[]): LiteralFinding[] {
  for (const entry of allowlist) {
    if (!entry.reason.trim() || (!entry.file && !entry.literal && !entry.kind)) throw new Error('Each literal allowance needs a selector and a written reason')
    if (entry.scope && (!entry.file || (!entry.literal && !entry.literals?.length))) throw new Error('Scoped allowances require an exact file and explicit literal values')
  }
  const normalFile = file.replace(/\\/g, '/')
  const tree = ts.createSourceFile(file, source, ts.ScriptTarget.Latest, true, file.endsWith('.tsx') ? ts.ScriptKind.TSX : ts.ScriptKind.TS)
  const lines = source.split(/\r?\n/)
  const findings: LiteralFinding[] = []
  const add = (literal: string, position: number, kind: LiteralFinding['kind'], node: ts.Node, arrayIndex = false) => {
    // GLSL spells structural integers with a float suffix; it does not make
    // artistic shader values such as point-size300 or opacity0.78 structural.
    const canonical = kind === 'shader-number' ? String(Number(literal)) : literal.replace(/_/g, '')
    const scopes = new Set<string>()
    for (let owner: ts.Node | undefined = node; owner; owner = owner.parent as ts.Node | undefined) {
      if ((ts.isVariableDeclaration(owner) || ts.isFunctionDeclaration(owner) || ts.isMethodDeclaration(owner)) && owner.name) scopes.add(owner.name.getText(tree))
    }
    if (allowlist.some((entry) => (!entry.file || entry.file === normalFile)
      && (!entry.literal || entry.literal === canonical)
      && (!entry.literals || entry.literals.includes(canonical))
      && (!entry.scope || scopes.has(entry.scope))
      && (!entry.kind || entry.kind === kind || (entry.kind === 'array-index' && arrayIndex)))) return
    const location = tree.getLineAndCharacterOfPosition(position)
    findings.push({ file: normalFile, line: location.line + 1, column: location.character + 1, literal, context: lines[location.line] ?? '', kind })
  }
  const shader = (node: ts.Node) => {
    if (normalFile.endsWith('.glsl.ts')) return true
    let owner = node.parent
    if (ts.isTemplateSpan(owner)) owner = owner.parent
    if (ts.isTemplateExpression(owner)) owner = owner.parent
    return ts.isPropertyAssignment(owner) && /^(vertex|fragment)Shader$/.test(ts.isStringLiteral(owner.name) ? owner.name.text : owner.name.getText(tree))
  }
  const visit = (node: ts.Node) => {
    if (ts.isNumericLiteral(node)) {
      const parent = node.parent
      const index = ts.isElementAccessExpression(parent) && parent.argumentExpression === node && Number.isInteger(Number(node.text)) && Number(node.text) >= 0
      add(node.getText(tree), node.getStart(tree), 'number', node, index)
    } else if (ts.isStringLiteral(node) && (/^#(?:[\da-f]{3}|[\da-f]{4}|[\da-f]{6}|[\da-f]{8})$/i.test(node.text) || /^(?:rgb|hsl)a?\(/i.test(node.text))) {
      add(node.text, node.getStart(tree) + 1, 'colour', node)
    } else if ((ts.isNoSubstitutionTemplateLiteral(node) || ts.isTemplateHead(node) || ts.isTemplateMiddle(node) || ts.isTemplateTail(node)) && shader(node)) {
      const text = node.getText(tree).replace(/\/\*[\s\S]*?\*\/|\/\/[^\n]*/g, (comment) => comment.replace(/[^\n]/g, ' '))
      for (const match of text.matchAll(/(?<![\w.])\d+(?:\.\d+)?(?:e[+-]?\d+)?(?![\w.])/gi)) {
        add(match[0], node.getStart(tree) + match.index, 'shader-number', node)
      }
    }
    ts.forEachChild(node, visit)
  }
  visit(tree)
  return findings
}

export function scanLiterals(root: string, allowlist: readonly LiteralAllowance[]): LiteralFinding[] {
  return sourceFiles(root).flatMap((file) => scanLiteralSource(readFileSync(file, 'utf8'), relative(root, file), allowlist))
}
