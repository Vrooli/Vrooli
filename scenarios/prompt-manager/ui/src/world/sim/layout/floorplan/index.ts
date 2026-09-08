import type { LayoutInput, LayoutStrategy } from '../strategy'
import type { AgentInput, TeamInput, Place, Seat, Vec2, WorldBounds } from '../../model'
import { sortSteps } from '../../cooperative'
import { applyOverridesSteps, type GeneratedLayout } from '../generate'
import { interiorDeskAt, interiorFor, interiorTablePosition } from '../interior'
import { officeWings } from './plate'
import { doorwayFor } from './doors'
import { architecture } from '../../../config/architecture'

function ringSeats(placeId: string, center: Vec2, radius: number, count: number): Seat[] {
  return Array.from({ length: count }, (_, index) => {
    const angle = (index / count) * Math.PI * 2
    const position: Vec2 = [center[0] + Math.sin(angle) * radius, center[1] + Math.cos(angle) * radius]
    return { id: `seat:${placeId}:${index}`, placeId, position, facing: Math.atan2(center[0] - position[0], center[1] - position[1]), sitting: true }
  })
}

/** Authored office wings with capacity-sized rooms and separate shared facilities. */
export const floorplanStrategy: LayoutStrategy = {
  generate(input): GeneratedLayout {
    const steps = floorplanSteps(input)
    let step = steps.next()
    while (!step.done) step = steps.next()
    return step.value
  },
  generateSteps: floorplanSteps,
}

export function* floorplanSteps({ teams, agents, tuning, options }: LayoutInput): Generator<{ completed: number; total: number }, GeneratedLayout> {
    const agentById = new Map<string, AgentInput>()
    for (const [index, agent] of agents.entries()) {
      if (index % 128 === 0) yield { completed: index, total: agents.length }
      agentById.set(agent.id, agent)
    }
    const ordered = yield* sortSteps(teams, (a, b) => a.id.localeCompare(b.id))
    const roomSpecs: Array<{ team: TeamInput; members: string[]; columns: number; width: number; depth: number }> = []
    for (const [index, team] of ordered.entries()) {
      if (index % 128 === 0) yield { completed: index, total: ordered.length }
      const members: string[] = []
      for (const [memberIndex, id] of team.memberIds.entries()) {
        if (memberIndex % 128 === 0) yield { completed: memberIndex, total: team.memberIds.length }
        if (agentById.has(id)) members.push(id)
      }
      const columns = Math.max(1, Math.ceil(Math.sqrt(members.length)))
      const rows = Math.max(1, Math.ceil(members.length / columns))
      const style = interiorFor(options.seed, team.id, members.length, [tuning.roomWidth, tuning.roomDepth], tuning)
      const across = (style.transposeDesks ? rows : columns) * tuning.deskPitch
      const inward = (style.transposeDesks ? columns : rows) * Math.max(tuning.deskPitch, architecture.office.rowPitch)
      const extras = tuning.deskInset * 2 + tuning.tableSeatRadius * 2
      roomSpecs.push({
        team,
        members,
        columns,
        width: Math.max(tuning.roomWidth, (style.deskWall === 'back' ? across : inward) + extras),
        depth: Math.max(tuning.roomDepth, (style.deskWall === 'back' ? inward : across) + extras),
      })
    }
    const sizes: Vec2[] = []
    const specByTeam = new Map<string, typeof roomSpecs[number]>()
    for (const [index, spec] of roomSpecs.entries()) {
      if (index % 128 === 0) yield { completed: index, total: roomSpecs.length }
      sizes.push([spec.width, spec.depth])
      specByTeam.set(spec.team.id, spec)
    }
    const plan = officeWings(sizes, tuning)
    const plate = plan.plate
    const assignments = ordered.flatMap((team, index) => { const leaf = plan.rooms[index]; return leaf ? [{ team, leaf }] : [] })
    const places: Place[] = []
    const deskSeatByAgent: Record<string, string> = {}

    let completedRooms = 0
    for (const assignment of assignments) {
        yield { completed: completedRooms++, total: assignments.length }
        const spec = specByTeam.get(assignment.team.id)
        if (!spec) continue
        const { leaf } = assignment
        const x = leaf.x
        const z = leaf.z
        const rotation = leaf.side === 'north' ? Math.PI : 0
        const roomId = `room:${spec.team.id}`
        const roomSize: Vec2 = [leaf.width, leaf.depth]
        places.push({ id: roomId, kind: 'room', teamId: spec.team.id, position: [x, z], rotation, size: roomSize, seats: [], label: spec.team.name,
          space: { kind: 'office', variant: 'studio', occupantIds: [...spec.members], entrance: [0, roomSize[1] / 2],
            meeting: [0, roomSize[1] / 2 + tuning.cellSize], gathering: [0, roomSize[1] / 2 - 2.5], shelters: [] } })
        const corridorZ = z + (leaf.side === 'north' ? -1 : 1) * (leaf.depth / 2 + tuning.floorplan.corridorWidth / 2)
        places.push(doorwayFor(assignment, { x: x, z: corridorZ, width: leaf.width, depth: tuning.floorplan.corridorWidth }, tuning))
        const interior = interiorFor(options.seed, spec.team.id, spec.members.length, roomSize, tuning, options.fillerIds?.length)
        const toWorld = (local: Vec2): Vec2 => [x + local[0] * Math.cos(rotation) + local[1] * Math.sin(rotation), z - local[0] * Math.sin(rotation) + local[1] * Math.cos(rotation)]
        for (const [index, agentId] of spec.members.entries()) {
          if (index % 128 === 0) yield { completed: index, total: spec.members.length }
          const desk = interiorDeskAt(interior, spec.members.length, roomSize, tuning, index)
          const deskPosition = toWorld(desk.position)
          const seatPosition = toWorld(desk.seat)
          const deskId = `desk:${agentId}`
          const seatId = `seat:desk:${agentId}`
          const seat: Seat = { id: seatId, placeId: deskId, position: seatPosition, facing: rotation + desk.rotation + Math.PI, sitting: false }
          places.push({ id: deskId, kind: 'desk', teamId: spec.team.id, ownerAgentId: agentId, parentId: roomId, position: deskPosition, rotation: rotation + desk.rotation, size: [tuning.deskPitch / 2, tuning.deskInset], seats: [seat], label: agentById.get(agentId)?.name ?? agentId })
          deskSeatByAgent[agentId] = seatId
        }
        const tableLocal = interiorTablePosition(interior, roomSize, tuning)
        if (tableLocal) {
          const tablePosition = toWorld(tableLocal)
          const tableId = `table:${spec.team.id}`
          places.push({ id: tableId, kind: 'table', teamId: spec.team.id, parentId: roomId, position: tablePosition, rotation, size: [tuning.tableRadius * 2, tuning.tableRadius * 2], seats: ringSeats(tableId, tablePosition, tuning.tableSeatRadius, tuning.tableSeats), label: `${spec.team.name} table` })
        }
    }

    plan.corridors.forEach((corridor, index) => places.push({ id: index === 0 ? 'corridor:primary' : `corridor:secondary:${index - 1}`, kind: 'corridor', position: [corridor.x, corridor.z], rotation: 0, size: [corridor.width, corridor.depth], seats: [], label: index === 0 ? 'Entrance hall' : `Office wing ${index}` }))
    for (const [id, room, label] of [['lounge', plan.lounge, 'Coffee lounge'], ['kitchen', plan.kitchen, 'Kitchenette']] as const) {
      const rotation = Math.PI / 2
      const roomId = `shared:${id}`
      const size: Vec2 = [room.depth, room.width]
      places.push({ id: roomId, kind: 'room', position: [room.x, room.z], rotation, size, seats: [], label,
        space: { kind: 'office', variant: 'studio', occupantIds: [], entrance: [0, size[1] / 2], meeting: [0, size[1] / 2 + tuning.cellSize], gathering: [0, 0], shelters: [] } })
      places.push({ id: `door:shared:${id}`, kind: 'door', parentId: roomId, position: [room.x + room.width / 2, room.z], rotation, size: [tuning.floorplan.doorWidth, tuning.cellSize], seats: [], label: `${label} entrance` })
    }
    const lounge: Vec2 = [plan.lounge.x, plan.lounge.z]
    places.push({ id: 'gathering', kind: 'gathering', parentId: 'shared:lounge', position: lounge, rotation: 0, size: [tuning.floorplan.lobbyRadius * 2, tuning.floorplan.lobbyRadius * 2], seats: [], label: options.gatheringLabel ?? 'Lounge' })
    places.push({ id: 'hearth', kind: 'hearth', parentId: 'shared:lounge', position: lounge, rotation: 0, size: [tuning.commonsSeatRadius, tuning.commonsSeatRadius], seats: ringSeats('hearth', lounge, tuning.commonsSeatRadius, tuning.commonsSeats), label: 'Coffee lounge' })
    places.push({ id: 'board', kind: 'board', parentId: 'shared:kitchen', position: [plan.kitchen.x - 2, plan.kitchen.z - 2.5], rotation: 0, size: [tuning.deskInset, tuning.deskPitch], seats: [], label: 'Run status' })

    yield* applyOverridesSteps(places, options.overrides ?? [])

    const decor: GeneratedLayout['decor'] = []
    for (const [index, room] of places.entries()) {
      if (index % 128 === 0) yield { completed: index, total: places.length }
      if (room.kind !== 'room') continue
      if (!room.teamId || !options.fillerIds?.length) continue
      const members = specByTeam.get(room.teamId)?.team.memberIds.length ?? 0
      const interior = interiorFor(options.seed, room.teamId, members, room.size, tuning, options.fillerIds.length)
      for (const filler of interior.fillers) {
        const propId = options.fillerIds[filler.propIndex % options.fillerIds.length]
        if (!propId) continue
        const cos = Math.cos(room.rotation)
        const sin = Math.sin(room.rotation)
        const position: Vec2 = [room.position[0] + filler.local[0] * cos + filler.local[1] * sin, room.position[1] - filler.local[0] * sin + filler.local[1] * cos]
        decor.push({ id: `filler:${room.teamId}:${filler.index}`, kind: 'decor', scaleRef: 'prop', propId, variant: filler.index, position, rotation: room.rotation + filler.rotation, scale: 1, roomId: room.id })
      }
    }
    const outline: Vec2[] = [[-plate.width / 2, -plate.depth / 2], [plate.width / 2, -plate.depth / 2], [plate.width / 2, plate.depth / 2], [-plate.width / 2, plate.depth / 2]]
    const bounds: WorldBounds = { width: options.terrain.radius * 2, depth: options.terrain.radius * 2, center: [0, 0], footprint: { width: plate.width, depth: plate.depth, center: [0, 0] }, outline }
    return { places, bounds, decor, deskSeatByAgent }
}
