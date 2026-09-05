import type { LayoutTuning } from '../config'
import { heightAt, type Place, type TerrainField, type Vec2 } from '../sim'
import { interiorFor } from '../sim/layout/interior'
import type { Placement } from './Props'

/** Shared placements keep emissive props and pooled point lights on the same ground. */
export function lampPlacements(places: Place[], seed: number, terrain: TerrainField, tuning: LayoutTuning, fillerCount: number): Placement[] {
  const out: Placement[] = []
  for (const room of places) {
    if (room.kind !== 'room') continue
    const [w, d] = room.size
    const inset = Math.min(w, d) * tuning.lampInsetRatio
    const rotate = (localX: number, localZ: number): Vec2 => [room.position[0] + localX * Math.cos(room.rotation) + localZ * Math.sin(room.rotation), room.position[1] - localX * Math.sin(room.rotation) + localZ * Math.cos(room.rotation)]
    const members = places.filter((place) => place.parentId === room.id && place.kind === 'desk').length
    const choice = interiorFor(seed, room.teamId ?? room.id, members, room.size, tuning, fillerCount)
    const corners: Vec2[] = [[-w / 2 + inset, -d / 2 + inset], [w / 2 - inset, -d / 2 + inset], [w / 2 - inset, d / 2 - inset], [-w / 2 + inset, d / 2 - inset]]
    choice.lampCorners.forEach((cornerIndex, index) => {
      const local = corners[cornerIndex]
      if (!local) return
      const position = rotate(local[0], local[1])
      out.push({ key: `${room.id}:lamp:${index}`, position, y: heightAt(terrain, position[0], position[1]), rotation: room.rotation, scale: 1 })
    })
  }
  for (const corridor of places) {
    if (corridor.kind !== 'corridor') continue
    const horizontal = corridor.size[0] >= corridor.size[1]
    const length = horizontal ? corridor.size[0] : corridor.size[1]
    const count = Math.max(1, Math.floor(length / tuning.corridorLampSpacing))
    for (let index = 0; index < count; index += 1) {
      const offset = ((index + 0.5) / count - 0.5) * length
      const position: Vec2 = horizontal ? [corridor.position[0] + offset, corridor.position[1]] : [corridor.position[0], corridor.position[1] + offset]
      out.push({ key: `${corridor.id}:lamp:${index}`, position, y: heightAt(terrain, position[0], position[1]), rotation: corridor.rotation, scale: tuning.corridorLampScale })
    }
  }
  return out
}
