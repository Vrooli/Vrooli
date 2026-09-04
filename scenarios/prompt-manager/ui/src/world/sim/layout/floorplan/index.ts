import type { LayoutStrategy } from '../strategy'
import type { Place, Seat, Vec2, WorldBounds } from '../../model'
import { Rng, hashString } from '../../rng'
import { applyOverrides, type GeneratedLayout } from '../generate'
import { interiorDesks, interiorFor, interiorTablePosition } from '../interior'
import { floorplate } from './plate'
import { cutCorridors } from './corridors'
import { bspLeaves } from './bsp'
import { assignRooms } from './assign'
import { doorwayFor } from './doors'

function ringSeats(placeId: string, center: Vec2, radius: number, count: number): Seat[] {
  return Array.from({ length: count }, (_, index) => {
    const angle = (index / count) * Math.PI * 2
    const position: Vec2 = [center[0] + Math.sin(angle) * radius, center[1] + Math.cos(angle) * radius]
    return { id: `seat:${placeId}:${index}`, placeId, position, facing: Math.atan2(center[0] - position[0], center[1] - position[1]), sitting: true }
  })
}

/** Deterministic corridor-first BSP office floorplan. */
export const floorplanStrategy: LayoutStrategy = {
  generate({ teams, agents, tuning, options }): GeneratedLayout {
    const agentById = new Map(agents.map((agent) => [agent.id, agent]))
    const ordered = [...teams].sort((a, b) => a.id.localeCompare(b.id))
    const roomSpecs = ordered.map((team) => {
      const members = team.memberIds.filter((id) => agentById.has(id))
      const columns = Math.max(1, Math.ceil(Math.sqrt(members.length)))
      const rows = Math.max(1, Math.ceil(members.length / columns))
      const gridSpan = Math.max(columns, rows) * tuning.deskPitch + tuning.deskInset * 2
      return {
        team,
        members,
        columns,
        width: Math.max(tuning.roomWidth, gridSpan + tuning.tableSeatRadius * 2),
        depth: Math.max(tuning.roomDepth, gridSpan + tuning.tableSeatRadius * 2),
      }
    })
    const rng = new Rng(hashString(`floorplan:${options.seed}`))
    const plate = floorplate(roomSpecs.map((spec) => [spec.width, spec.depth]), roomSpecs.reduce((sum, spec) => sum + spec.members.length, 0), tuning, rng)
    const corridorPlan = cutCorridors(plate, tuning, rng)
    const leaves = bspLeaves(corridorPlan.blocks, roomSpecs.length, tuning, rng)
    const assignments = assignRooms(ordered, leaves)
    const specByTeam = new Map(roomSpecs.map((spec) => [spec.team.id, spec]))
    const places: Place[] = []
    const deskSeatByAgent: Record<string, string> = {}

    for (const assignment of assignments) {
        const spec = specByTeam.get(assignment.team.id)
        if (!spec) continue
        const { leaf } = assignment
        const x = leaf.x
        const z = leaf.z
        const rotation = leaf.side === 'north' ? Math.PI : 0
        const roomId = `room:${spec.team.id}`
        const roomSize: Vec2 = [leaf.width, leaf.depth]
        places.push({ id: roomId, kind: 'room', teamId: spec.team.id, position: [x, z], rotation, size: roomSize, seats: [], label: spec.team.name })
        places.push(doorwayFor(assignment, corridorPlan.primary, tuning))
        const interior = interiorFor(options.seed, spec.team.id, spec.members.length, roomSize, tuning, options.fillerIds?.length)
        const desks = interiorDesks(interior, spec.members.length, roomSize, tuning)
        const toWorld = (local: Vec2): Vec2 => [x + local[0] * Math.cos(rotation) + local[1] * Math.sin(rotation), z - local[0] * Math.sin(rotation) + local[1] * Math.cos(rotation)]
        spec.members.forEach((agentId, index) => {
          const desk = desks[index]
          if (!desk) return
          const deskPosition = toWorld(desk.position)
          const seatPosition = toWorld(desk.seat)
          const deskId = `desk:${agentId}`
          const seatId = `seat:desk:${agentId}`
          const seat: Seat = { id: seatId, placeId: deskId, position: seatPosition, facing: rotation + desk.rotation + Math.PI, sitting: false }
          places.push({ id: deskId, kind: 'desk', teamId: spec.team.id, ownerAgentId: agentId, parentId: roomId, position: deskPosition, rotation: rotation + desk.rotation, size: [tuning.deskPitch / 2, tuning.deskInset], seats: [seat], label: agentById.get(agentId)?.name ?? agentId })
          deskSeatByAgent[agentId] = seatId
        })
        const tableLocal = interiorTablePosition(interior, roomSize, tuning)
        if (tableLocal) {
          const tablePosition = toWorld(tableLocal)
          const tableId = `table:${spec.team.id}`
          places.push({ id: tableId, kind: 'table', teamId: spec.team.id, parentId: roomId, position: tablePosition, rotation, size: [tuning.tableRadius * 2, tuning.tableRadius * 2], seats: ringSeats(tableId, tablePosition, tuning.tableSeatRadius, tuning.tableSeats), label: `${spec.team.name} table` })
        }
    }

    places.push({ id: 'corridor:primary', kind: 'corridor', position: [corridorPlan.primary.x, corridorPlan.primary.z], rotation: 0, size: [corridorPlan.primary.width, corridorPlan.primary.depth], seats: [], label: 'Main corridor' })
    corridorPlan.secondary.forEach((corridor, index) => places.push({ id: `corridor:secondary:${index}`, kind: 'corridor', position: [corridor.x, corridor.z], rotation: 0, size: [corridor.width, corridor.depth], seats: [], label: `Cross corridor ${index + 1}` }))
    const junctionX = corridorPlan.secondary[0]?.x ?? 0
    const junction: Vec2 = [junctionX, corridorPlan.primary.z]
    places.push({ id: 'gathering', kind: 'gathering', position: junction, rotation: 0, size: [tuning.floorplan.lobbyRadius * 2, tuning.floorplan.lobbyRadius * 2], seats: [], label: options.gatheringLabel ?? 'Lobby' })
    places.push({ id: 'hearth', kind: 'hearth', parentId: 'gathering', position: junction, rotation: 0, size: [tuning.commonsSeatRadius, tuning.commonsSeatRadius], seats: ringSeats('hearth', junction, tuning.commonsSeatRadius, tuning.commonsSeats), label: 'Coffee lounge' })
    places.push({ id: 'board', kind: 'board', position: [junctionX + tuning.boardOffset, corridorPlan.primary.z], rotation: -Math.PI / 2, size: [tuning.deskInset, tuning.deskPitch], seats: [], label: 'Run status' })

    applyOverrides(places, options.overrides ?? [])

    const decor: GeneratedLayout['decor'] = []
    for (const room of places.filter((place) => place.kind === 'room')) {
      if (!room.teamId || !options.fillerIds?.length) continue
      const members = ordered.find((team) => team.id === room.teamId)?.memberIds.length ?? 0
      const interior = interiorFor(options.seed, room.teamId, members, room.size, tuning, options.fillerIds.length)
      for (const filler of interior.fillers) {
        const propId = options.fillerIds[filler.propIndex % options.fillerIds.length]
        if (!propId) continue
        const cos = Math.cos(room.rotation)
        const sin = Math.sin(room.rotation)
        const position: Vec2 = [room.position[0] + filler.local[0] * cos + filler.local[1] * sin, room.position[1] - filler.local[0] * sin + filler.local[1] * cos]
        decor.push({ id: `filler:${room.teamId}:${filler.index}`, kind: 'decor', propId, variant: filler.index, position, rotation: room.rotation + filler.rotation, scale: 1, roomId: room.id })
      }
    }
    const outline: Vec2[] = [[-plate.width / 2, -plate.depth / 2], [plate.width / 2, -plate.depth / 2], [plate.width / 2, plate.depth / 2], [-plate.width / 2, plate.depth / 2]]
    const bounds: WorldBounds = { width: options.terrain.radius * 2, depth: options.terrain.radius * 2, center: [0, 0], footprint: { width: plate.width, depth: plate.depth, center: [0, 0] }, outline }
    return { places, bounds, decor, deskSeatByAgent }
  },
}
