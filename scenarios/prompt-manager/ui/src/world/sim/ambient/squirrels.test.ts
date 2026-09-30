import { describe, expect, it } from 'vitest'
import { biomeSets, scenes, tuning } from '../../config'
import { heightAt } from '../terrain/field'
import { ambientPolicy } from '../../config/ambient'
import { makeWorldInput } from '../__tests__/fixtures'
import { generateWorld } from '../world'
import { runCooperatively } from '../cooperative'
import { isWalkable } from '../nav/grid'
import { nextSquirrelBoundary, squirrelPose, squirrelRouteSteps } from './squirrels'

describe('tree-bound squirrel behavior', () => {
  it.each(['park', 'office'] as const)('uses actual %s trees and traversable foraging routes without changing navigation', async scene => {
    const world = generateWorld(makeWorldInput({ seed: 1, scene, teams: 5, agents: 25 }), tuning)
    const original = world.nav.walkable.slice()
    const routes = await runCooperatively(squirrelRouteSteps(world.seed, world.terrain, world.habitats, world.nav, world.decor, biomeSets[scenes[world.scene].biomeSet]))
    expect(routes.length).toBeGreaterThan(0)
    expect(routes.length).toBeLessThanOrEqual(ambientPolicy.squirrels.routeBudget)
    expect(await runCooperatively(squirrelRouteSteps(world.seed, world.terrain, world.habitats, world.nav, world.decor, biomeSets[scenes[world.scene].biomeSet]))).toEqual(routes)
    for (const route of routes) {
      const tree = world.decor.find(spot => spot.id === route.treeId)
      if (!tree) throw new Error('Squirrel has no habitat tree')
      expect(tree.position).toEqual(route.tree)
      // Measured leaf-material bounds in the baked kit, before ground lift.
      // This catches using a generic climb height unrelated to the visible tree.
      const canopyBottom = tree.propId === 'tree_default' ? .75036 : .45040
      const biome = biomeSets[scenes[world.scene].biomeSet]
      expect(Math.hypot(route.trunk[0] - tree.position[0], route.trunk[1] - tree.position[1]), 'gripping paws should meet the measured stem').toBeLessThan(.07 * biome.propScale * biome.treeScale * tree.scale)
      const foliageY = heightAt(world.terrain, ...tree.position) + canopyBottom * biome.propScale * biome.treeScale * tree.scale
      const unshifted = { ...route, phase: 0 }
      expect(squirrelPose(unshifted, world.terrain, 5).state).toBe('forage')
      expect(squirrelPose(unshifted, world.terrain, 11).state).toBe('dart')
      expect(squirrelPose(unshifted, world.terrain, 14).state).toBe('climb')
      expect(squirrelPose(unshifted, world.terrain, 17).animating).toBe(false)
      expect(squirrelPose(unshifted, world.terrain, 27).animating).toBe(false)
      expect(squirrelPose(unshifted, world.terrain, 14.5).position[1]).toBeGreaterThan(squirrelPose(unshifted, world.terrain, 13.5).position[1])
      for (let time = 0; time < 30; time += .125) {
        const pose = squirrelPose(unshifted, world.terrain, time)
        if (pose.state === 'climb' || pose.state === 'perch') expect(pose.position[1] + .4, `${tree.propId} head enters foliage at ${time}s`).toBeLessThan(foliageY)
        if (pose.state === 'dart' || pose.state === 'forage' || pose.state === 'rest') {
          for (const dx of [-.22, 0, .22]) for (const dz of [-.22, 0, .22]) expect(isWalkable(world.nav, [pose.position[0] + dx, pose.position[2] + dz])).toBe(true)
        }
        expect(nextSquirrelBoundary(unshifted, time)).toBeGreaterThan(time)
      }
      for (const boundary of [0, 10, 12, 13, 15, 20, 22, 23, 25, 30]) {
        const a = squirrelPose(unshifted, world.terrain, boundary - .00001), b = squirrelPose(unshifted, world.terrain, boundary + .00001)
        expect(Math.hypot(...a.position.map((v, i) => v - (b.position[i] ?? 0)))).toBeLessThan(.001)
      }
    }
    expect(world.nav.walkable).toEqual(original)
    expect(await runCooperatively(squirrelRouteSteps(world.seed, world.terrain, world.habitats, world.nav, [], biomeSets[scenes[world.scene].biomeSet]))).toEqual([])
    expect(await runCooperatively(squirrelRouteSteps(world.seed, world.terrain, new Uint8Array(world.habitats.length).fill(5), world.nav, world.decor, biomeSets[scenes[world.scene].biomeSet]))).toEqual([])
  })
})
