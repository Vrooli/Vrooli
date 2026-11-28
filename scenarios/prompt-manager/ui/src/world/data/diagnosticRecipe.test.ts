import { describe, expect, it } from 'vitest'
import { tuning, withTuningOverride } from '../config'
import { createWorld, terrainDigest } from '../sim'
import { syntheticRoster } from './synthetic'
import { createDiagnosticRecipe, restoreDiagnosticRecipe, parseRecipeFile, recipeLocation, type DiagnosticRecipeInput } from './diagnosticRecipe'

function input(): DiagnosticRecipeInput {
  return {
    seed: 42, scene: 'park', roster: { kind: 'synthetic', actors: 5, perTeam: 5 },
    tuning: withTuningOverride({ layout: { deskPitch: 2 } }, tuning),
    layout: [{ placeId: 'room:demo-team-0', position: [15, 15] }], clearPoints: [[0, 0]],
    view: { period: 'dusk', weather: 'snow', profile: 'high', zoomTarget: 'center', ambientLife: false, lightingMode: 'preset', clock: { mode: 'fixed', utcMilliseconds: 1772323200000 }, navigationOverlay: true, biomeOverlay: false, habitatOverlay: false, cameraCollisionOverlay: true, camera: { position: [10, 10, 10], target: [0, 0, 0] } },
    renderer: { vendor: null, renderer: null, width: 1400, height: 900, dpr: 1 },
  }
}

describe('allowlisted diagnostic recipes', () => {
  it('restores old recipes while discarding retired scene controls', () => {
    const recipe = createDiagnosticRecipe(input())
    Object.assign(recipe.tuning.layout, { wallHeight: .7 })
    Object.assign(recipe.tuning.layout.floorplan, { splitRatio: { min: .3, max: .7 }, doorWidth: 1 })
    Object.assign(recipe.tuning.weather, { cloudAltitude: 170, cloudPlaneSpan: 3, cloudOpacityScale: .28 })
    const restored = restoreDiagnosticRecipe(JSON.parse(JSON.stringify(recipe)))
    expect(restored.tuning.layout).not.toHaveProperty('wallHeight')
    expect(restored.tuning.layout.floorplan).not.toHaveProperty('splitRatio')
    expect(restored.tuning.layout.floorplan).not.toHaveProperty('doorWidth')
    expect(restored.tuning.weather).not.toHaveProperty('cloudPlaneSpan')
    expect(restored.input.overrides).toEqual(recipe.layout)
    expect(restored.tuning.layout.deskPitch).toBe(2)
  })
  it('retains the civil timezone and rejects invalid recipe zones', () => {
    const recipe = createDiagnosticRecipe(input())
    const clock = { mode: 'fixed' as const, utcMilliseconds: 1835440496000, timeZone: 'America/New_York' }
    expect(restoreDiagnosticRecipe({ ...recipe, view: { ...recipe.view, clock } }).recipe.view.clock).toEqual(clock)
    expect(() => restoreDiagnosticRecipe({ ...recipe, view: { ...recipe.view, clock: { ...clock, timeZone: 'Unknown/Place' } } })).toThrow('Unknown civil timezone')
  })
  it('preserves continuous lighting mode and keeps legacy recipes on named presets', () => {
    const recipe = createDiagnosticRecipe({ ...input(), view: { ...input().view, lightingMode: 'clock' } })
    expect(parseRecipeFile(JSON.stringify(recipe)).view.lightingMode).toBe('clock')
    const { lightingMode: _lightingMode, ...legacyView } = recipe.view
    expect(restoreDiagnosticRecipe({ ...recipe, view: legacyView }).recipe.view.lightingMode).toBe('preset')
  })
  it('roundtrips a pinned UTC clock and leaves older recipes in live mode', () => {
    const recipe = createDiagnosticRecipe(input())
    expect(parseRecipeFile(JSON.stringify(recipe)).view.clock).toEqual(input().view.clock)
    const { clock: _clock, ...legacyView } = recipe.view
    expect(restoreDiagnosticRecipe({ ...recipe, view: legacyView }).recipe.view.clock).toBeNull()
  })
  it('preserves ambient off and enables ambient life for recipes without the field', () => {
    const recipe = createDiagnosticRecipe(input())
    expect(parseRecipeFile(JSON.stringify(recipe)).view.ambientLife).toBe(false)
    const { ambientLife: _ambientLife, ...legacyView } = recipe.view
    expect(restoreDiagnosticRecipe({ ...recipe, view: legacyView }).recipe.view.ambientLife).toBe(true)
  })
  it('defaults older recipes to no biome overlay and rejects overlapping grid choices', () => {
    const recipe = createDiagnosticRecipe(input())
    const { biomeOverlay: _legacyAbsent, ...legacyView } = recipe.view
    expect(restoreDiagnosticRecipe({ ...recipe, view: legacyView }).recipe.view.biomeOverlay).toBe(false)
    expect(() => restoreDiagnosticRecipe({ ...recipe, view: { ...recipe.view, biomeOverlay: true } })).toThrow('Choose only one grid overlay')
  })
  it('bounds imported files and constructs a synthetic URL without unrelated query data', () => {
    const recipe = createDiagnosticRecipe(input())
    expect(parseRecipeFile(JSON.stringify(recipe))).toEqual(recipe)
    expect(() => parseRecipeFile(' '.repeat(1024 * 1024 + 1))).toThrow('at most 1 MiB')
    const url = new URL(recipeLocation(recipe), 'http://localhost')
    expect(url.pathname).toBe('/world')
    expect(url.searchParams.get('actors')).toBe('5')
    expect(url.searchParams.get('recipe')).toBe('session')
    expect(url.searchParams.has('capture')).toBe(false)
  })
  it('restores the same synthetic terrain, edited places, and actor roster from JSON', () => {
    const original = input()
    const restored = restoreDiagnosticRecipe(JSON.parse(JSON.stringify(createDiagnosticRecipe(original))))
    const expected = createWorld({ seed: original.seed, now: 0, scene: original.scene,
      ...syntheticRoster(5, 5, 42), overrides: original.layout, clearPoints: original.clearPoints }, original.tuning)
    const actual = createWorld(restored.input, restored.tuning)
    expect(terrainDigest(actual)).toBe(terrainDigest(expected))
    expect(actual.places).toEqual(expected.places)
    expect(actual.places['room:demo-team-0']?.position).toEqual([15, 15])
    expect(actual.actors).toEqual(expected.actors)
    expect(restored.tuning.layout.deskPitch).toBe(2)
    expect(restored.recipe.view).toEqual(original.view)
  })

  it('projects allowed fields and excludes messages, live roster and unrelated request data', () => {
    const original = input()
    const secret = 'PRIVATE-SENTINEL'
    Object.assign(original, { messages: [secret], agents: [{ name: secret }], request: { token: secret } })
    Object.assign(original.renderer, { request: secret })
    Object.assign(original.tuning.layout, { message: secret })
    const room = original.layout[0]
    if (!room) throw new Error('Missing test room')
    Object.assign(room, { agentMessage: secret })
    const json = JSON.stringify(createDiagnosticRecipe(original))
    expect(json).not.toContain(secret)
    expect(() => restoreDiagnosticRecipe({ ...JSON.parse(json), messages: [secret] })).toThrow()
    room.placeId = 'room:private-team'
    expect(() => createDiagnosticRecipe(original)).toThrow()
  })

  it('rejects incompatible source definitions, versions and invalid bounds', () => {
    const recipe = createDiagnosticRecipe(input())
    expect(() => restoreDiagnosticRecipe({ ...recipe, schemaVersion: 2 })).toThrow()
    expect(() => restoreDiagnosticRecipe({ ...recipe, source: { ...recipe.source, generatorVersion: 'old' } })).toThrow('different world generator')
    expect(() => restoreDiagnosticRecipe({ ...recipe, source: { ...recipe.source, environmentFingerprint: '00000000' } })).toThrow('different world generator')
    expect(() => restoreDiagnosticRecipe({ ...recipe, roster: { ...recipe.roster, actors: 10001 } })).toThrow()
  })
})
