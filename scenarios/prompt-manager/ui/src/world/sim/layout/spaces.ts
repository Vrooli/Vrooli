import { architecture as A } from '../../config/architecture'
import type { Place, SpaceLayout, Vec2, WorldState } from '../model'
import { hashString } from '../rng'

export interface StructureBox {
  id: string
  position: readonly [number, number, number]
  size: readonly [number, number, number]
  rotation: number
  surface: 'wall' | 'floor' | 'lintel' | 'window' | 'ceiling' | 'furniture'
  color?: string
}

export function spacePoint(place: Place, local: Vec2): Vec2 {
  const c = Math.cos(place.rotation), s = Math.sin(place.rotation)
  return [place.position[0] + local[0] * c + local[1] * s, place.position[1] - local[0] * s + local[1] * c]
}

export function insideSpace(place: Place, point: Vec2): boolean {
  const dx = point[0] - place.position[0], dz = point[1] - place.position[1]
  const c = Math.cos(place.rotation), s = Math.sin(place.rotation)
  return Math.abs(dx * c - dz * s) <= place.size[0] / 2 && Math.abs(dx * s + dz * c) <= place.size[1] / 2
}

/** Assigned members remain manageable while visitors appear in their current room. */
export function spaceActors(state: WorldState, place: Place) {
  const assigned = new Set(place.space?.occupantIds ?? [])
  return state.actorOrder.flatMap(id => {
    const actor = state.actors[id]
    return actor && (assigned.has(id) || insideSpace(place, actor.position)) ? [actor] : []
  })
}

/** Whole shelters are the unit of growth; existing member slots retain their IDs. */
export function campsiteSize(members: number): Vec2 {
  const shelters = Math.max(1, Math.ceil(members / A.shelterCapacity))
  const columns = Math.max(1, Math.ceil(Math.sqrt(shelters)))
  const rows = Math.ceil(shelters / columns)
  return [columns * A.shelterPitch + A.siteMargin * 2, rows * A.shelterPitch + A.gatheringDepth + A.siteMargin]
}

export function campsiteFor(teamId: string, members: string[], size: Vec2): SpaceLayout {
  const count = Math.max(1, Math.ceil(members.length / A.shelterCapacity))
  const columns = Math.max(1, Math.ceil(Math.sqrt(count)))
  const variants = ['tent', 'cabin', 'rv'] as const
  return {
    kind: 'campsite', variant: variants[hashString(teamId) % variants.length] ?? 'tent', occupantIds: [...members],
    entrance: [0, size[1] / 2], meeting: [0, size[1] / 2 - 1.4], gathering: [0, size[1] / 2 - 4.2],
    shelters: Array.from({ length: count }, (_, i) => ({ id: `shelter:${teamId}:${i}`,
      position: [(i % columns - (columns - 1) / 2) * A.shelterPitch, -size[1] / 2 + A.siteMargin + A.shelterDepth / 2 + Math.floor(i / columns) * A.shelterPitch],
      size: [A.shelterWidth, A.shelterDepth] })),
  }
}

export function campsiteSeat(space: SpaceLayout, member: number): Vec2 {
  const shelter = space.shelters[Math.floor(member / A.shelterCapacity)]
  if (!shelter) throw new Error('Campsite has insufficient shelter capacity')
  const slot = member % A.shelterCapacity
  return [shelter.position[0] + (slot % 2 === 0 ? -1.25 : 1.25), shelter.position[1] + (slot < 2 ? -.95 : .65)]
}

/** Rendering and agent navigation consume these exact walls and doorway gaps. */
export function spaceStructures(place: Place): StructureBox[] {
  if (!place.space) return []
  const shells = place.space.kind === 'office' ? [{ id: place.id, position: [0, 0] as Vec2, size: place.size }] : place.space.shelters
  const boxes: StructureBox[] = []
  for (const shell of shells) {
    const [w, d] = shell.size, [x, z] = shell.position
    const height = place.space.variant === 'tent' ? .65 : A.wallHeight
    const add = (id: string, local: Vec2, y: number, size: StructureBox['size'], surface: StructureBox['surface']) => {
      const p = spacePoint(place, local)
      boxes.push({ id: `${shell.id}:${id}`, position: [p[0], y, p[1]], rotation: place.rotation, size, surface })
    }
    add('floor', [x, z], 0, [w, A.floorThickness, d], 'floor')
    const windowWall = (id: string, center: Vec2, length: number, side: boolean) => {
      const { windowSill: sill, windowHeight: opening, windowFraction } = A.office
      const span = length * windowFraction, pier = (length - span) / 2
      const size = (along: number, h: number): StructureBox['size'] => side ? [A.wallThickness, h, along] : [along, h, A.wallThickness]
      add(`${id}:sill`, center, sill / 2, size(length, sill), 'wall')
      add(`${id}:head`, center, (height + sill + opening) / 2, size(length, height - sill - opening), 'wall')
      for (const sign of [-1, 1]) {
        const offset = sign * (span + pier) / 2
        add(`${id}:pier:${sign}`, [center[0] + (side ? 0 : offset), center[1] + (side ? offset : 0)], sill + opening / 2, size(pier, opening), 'wall')
      }
      add(`${id}:glass`, center, sill + opening / 2, size(span, opening), 'window')
    }
    if (place.space.kind === 'office') {
      windowWall('back', [x, z - d / 2], w + A.wallThickness, false)
      add('ceiling', [x, z], height + A.office.ceilingThickness / 2, [w, A.office.ceilingThickness, d], 'ceiling')
      add('open-door', [x - A.doorWidth / 2 - .13, z + d / 2 + A.doorWidth / 4], A.doorHeight / 2, [.09, A.doorHeight, A.doorWidth / 2], 'wall')
    } else add('back', [x, z - d / 2], height / 2, [w + A.wallThickness, height, A.wallThickness], 'wall')
    for (const sign of [-1, 1]) {
      if (place.space.kind === 'office') windowWall(`side:${sign}`, [x + sign * w / 2, z], d, true)
      else add(`side:${sign}`, [x + sign * w / 2, z], height / 2, [A.wallThickness, height, d], 'wall')
      const segment = (w - A.doorWidth) / 2
      add(`front:${sign}`, [x + sign * (A.doorWidth + segment) / 2, z + d / 2], height / 2, [segment, height, A.wallThickness], 'wall')
    }
    if (height > A.doorHeight) {
      const lintel = height - A.doorHeight
      add('lintel', [x, z + d / 2], A.doorHeight + lintel / 2, [A.doorWidth, lintel, A.wallThickness], 'lintel')
    }
    if (place.id === 'shared:kitchen') {
      add('counter', [0, -d / 2 + .55], .43, [4.8, .86, 1], 'furniture')
      add('island', [0, .8], .43, [2.8, .86, .85], 'furniture')
      add('fridge', [w / 2 - .8, -d / 2 + .65], 1.05, [1.2, 2.1, 1.15], 'furniture')
    }
  }
  return boxes.map(box => box.surface === 'furniture' ? { ...box, color: box.id.endsWith('fridge') ? '#d8ded8' : A.palette.roof } : box)
}
