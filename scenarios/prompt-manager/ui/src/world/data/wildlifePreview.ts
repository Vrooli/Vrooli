import type { WildlifePreviewId } from '../config/ambient'
import { ambientPolicy } from '../config/ambient'
import { biomeSets, scenes } from '../config'
import { birdPose, birdRouteSteps } from '../sim/ambient/birds'
import { butterflyPose, butterflySiteSteps } from '../sim/ambient/butterflies'
import { fireflyPose, fireflySiteSteps } from '../sim/ambient/fireflies'
import { fishCandidate, fishPondSteps } from '../sim/ambient/fish'
import { squirrelPose, squirrelRouteSteps } from '../sim/ambient/squirrels'
import { rabbitPose, rabbitRouteSteps } from '../sim/ambient/rabbits'
import { hashString } from '../sim/rng'
import { runCooperatively } from '../sim/cooperative'
import type { WorldState } from '../sim/model'

export interface WildlifePreview {
  utcMilliseconds: number
  position: readonly [number, number, number]
  radius: number
  period: 'day' | 'night'
  description: string
}

/** Preview seeks the normal deterministic schedule and uses committed habitat.
 * It never invents an animal, relocates a site or changes simulation occupancy.
 */
export async function prepareWildlifePreview(world: WorldState, kind: WildlifePreviewId, utcMilliseconds: number, signal?: AbortSignal, reducedMotion = false, vegetationDensityScale = 1): Promise<WildlifePreview | null> {
  const seconds = utcMilliseconds / 1000
  const options = { signal }
  if (kind === 'fireflies') {
    const [site] = await runCooperatively(fireflySiteSteps(world.seed, world.terrain, world.habitats, world.habitatRegions), options)
    return site ? { utcMilliseconds, position: fireflyPose(site, reducedMotion ? 0 : seconds).position, radius: .35, period: 'night', description: 'wetland firefly' } : null
  }
  if (kind === 'butterflies') {
    const [site] = await runCooperatively(butterflySiteSteps(world.seed, world.terrain, world.habitats, world.decor), options)
    return site ? { utcMilliseconds, position: butterflyPose(site, world.terrain, seconds).position, radius: .4, period: 'day', description: 'meadow butterfly' } : null
  }
  if (kind === 'birds') {
    const [route] = await runCooperatively(birdRouteSteps(world.seed, world.terrain, world.habitats), options)
    return route ? { utcMilliseconds, position: birdPose(route, seconds).position, radius: .85, period: 'day', description: 'habitat bird route' } : null
  }
  if (kind === 'rabbit-idle' || kind === 'rabbit-travel') {
    const [route] = await runCooperatively(rabbitRouteSteps(world.seed, world.terrain, world.habitats, world.nav), options)
    if (!route) return null
    const { idleSeconds: idle, travelSeconds: travel } = ambientPolicy.rabbits
    const period = (idle + travel) * 2
    const time = Math.floor(seconds / period) * period - route.phase * period + (kind === 'rabbit-idle' ? idle / 2 : idle + travel * .375)
    const pose = rabbitPose(route, world.terrain, time)
    return { utcMilliseconds: time * 1000, position: [pose.position[0], pose.position[1] + pose.hop + .4, pose.position[2]], radius: .7, period: 'day', description: kind === 'rabbit-idle' ? 'idle rabbit' : 'rabbit mid-hop' }
  }
  if (kind === 'squirrel-forage' || kind === 'squirrel-climb') {
    const routes = await runCooperatively(squirrelRouteSteps(world.seed, world.terrain, world.habitats, world.nav, world.decor, biomeSets[scenes[world.scene].biomeSet]), options)
    const route = routes.find(candidate => hashString(`density:${candidate.treeId}`) / 0xffffffff <= vegetationDensityScale)
    if (!route) return null
    const period = ambientPolicy.squirrels.periodSeconds
    const time = Math.floor(seconds / period) * period - route.phase * period + (kind === 'squirrel-forage' ? 5 : 14)
    const pose = squirrelPose(route, world.terrain, time)
    return { utcMilliseconds: time * 1000, position: [pose.position[0], pose.position[1] + .35, pose.position[2]], radius: .85, period: 'day', description: kind === 'squirrel-forage' ? 'foraging squirrel' : 'squirrel on its habitat tree' }
  }
  const ponds = await runCooperatively(fishPondSteps(world.waterGeometry, world.terrain, world.habitats), options)
  const event = fishCandidate(world.seed, Math.floor(seconds / ambientPolicy.fish.meanEligibleSeconds), ponds)
  if (!event) return null
  const offset = kind === 'fish-jump' ? ambientPolicy.fish.jumpSeconds / 2 : ambientPolicy.fish.jumpSeconds + (kind === 'fish-splash' ? .15 : ambientPolicy.fish.effectSeconds * .5)
  return { utcMilliseconds: (event.start + offset) * 1000,
    position: [event.site.position[0], event.site.position[1] + (kind === 'fish-jump' ? .45 : 0), event.site.position[2]],
    radius: .65, period: 'day', description: `${kind.replace('fish-', 'fish ')} in pond ${event.pond}` }
}
