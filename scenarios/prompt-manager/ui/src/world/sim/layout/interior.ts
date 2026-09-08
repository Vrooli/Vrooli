import type { LayoutTuning } from '../../config'
import type { Vec2 } from '../model'
import { Rng, hashString } from '../rng'
import { architecture } from '../../config/architecture'

export interface InteriorChoice {
  transposeDesks: boolean
  deskWall: 'back' | 'left' | 'right'
  table: 'front' | 'rear-left' | 'rear-right' | 'none'
  /** Two distinct indices into the clockwise room-corner list. */
  lampCorners: readonly [number, number]
  fillers: Array<{ index: number; propIndex: number; local: Vec2; rotation: number }>
  columns: number
}

export interface InteriorDesk {
  position: Vec2
  seat: Vec2
  rotation: number
}

export function interiorFor(seed: number, teamId: string, memberCount: number, roomSize: Vec2, tuning: LayoutTuning, fillerKinds = 1): InteriorChoice {
  const rng = new Rng(hashString(`interior:${seed}:${teamId}`))
  const transposeDesks = rng.int(2) === 1
  const walls = ['back', 'left', 'right'] as const
  const deskWall = walls[rng.int(walls.length)] ?? 'back'
  const table = deskWall === 'left' ? 'rear-right' : deskWall === 'right' ? 'rear-left' : 'front'
  const firstLamp = rng.int(4)
  const secondDraw = rng.int(3)
  const secondLamp = secondDraw >= firstLamp ? secondDraw + 1 : secondDraw
  const choice: InteriorChoice = {
    transposeDesks, deskWall,
    table: memberCount >= tuning.interior.tableMinMembers ? table : 'none',
    lampCorners: [firstLamp, secondLamp], fillers: [],
    columns: Math.max(1, Math.ceil(Math.sqrt(memberCount))),
  }
  // Authored furnishing sockets: corners and side-wall bays. Doorways and the
  // middle of circulation routes are never random decoration candidates.
  const x = roomSize[0] / 2 - .7, z = roomSize[1] / 2 - .7
  const anchors: Array<{ local: Vec2; rotation: number }> = [
    { local: [-x, z], rotation: Math.PI / 2 }, { local: [x, z], rotation: -Math.PI / 2 },
    { local: [-x, -z], rotation: Math.PI / 2 }, { local: [x, -z], rotation: -Math.PI / 2 },
    { local: [-x, 0], rotation: Math.PI / 2 }, { local: [x, 0], rotation: -Math.PI / 2 },
  ]
  const desks = interiorDesks(choice, memberCount, roomSize, tuning)
  const meeting = interiorTablePosition(choice, roomSize, tuning)
  if (meeting && fillerKinds > 1 && tuning.interior.fillerMax > 0) choice.fillers.push({ index: 0, propIndex: 1, local: meeting, rotation: 0 })
  const start = rng.int(anchors.length)
  for (let offset = 0; offset < anchors.length && choice.fillers.length < tuning.interior.fillerMax; offset++) {
    const anchor = anchors[(start + offset) % anchors.length]
    if (!anchor) continue
    const near = (point: Vec2, clearance: number) => Math.hypot(point[0] - anchor.local[0], point[1] - anchor.local[1]) < clearance
    if (desks.some(desk => near(desk.position, 1.3) || near(desk.seat, 1.3))) continue
    if (meeting && near(meeting, tuning.tableSeatRadius + .7)) continue
    const propIndex = choice.fillers.length % 2 === 0 && fillerKinds > 3 ? 3 : 0
    choice.fillers.push({ index: choice.fillers.length, propIndex, ...anchor })
  }
  return choice
}

/** Convert the pure interior choice into room-local desk and seat transforms. */
export function interiorDesks(choice: InteriorChoice, memberCount: number, roomSize: Vec2, tuning: LayoutTuning): InteriorDesk[] {
  if (memberCount <= 0) return []
  return Array.from({ length: memberCount }, (_, index) => interiorDeskAt(choice, memberCount, roomSize, tuning, index))
}

/** One transform lets cooperative callers schedule large rooms incrementally. */
export function interiorDeskAt(choice: InteriorChoice, memberCount: number, roomSize: Vec2, tuning: LayoutTuning, index: number): InteriorDesk {
  const [width, depth] = roomSize
  const columns = choice.transposeDesks
    ? Math.max(1, Math.ceil(memberCount / choice.columns))
    : choice.columns
  const seatClearance = Math.max(tuning.deskSeatOffset, tuning.deskInset * 0.5 + tuning.cellSize)
  const rowPitch = Math.max(tuning.deskPitch, architecture.office.rowPitch)
    const row = Math.floor(index / columns)
    const column = index % columns
    const rowMembers = Math.min(columns, memberCount - row * columns)
    const across = (column - (rowMembers - 1) * 0.5) * tuning.deskPitch
    if (choice.deskWall === 'left' || choice.deskWall === 'right') {
      const alongWall = Math.max(-depth * 0.5 + tuning.deskInset, Math.min(depth * 0.5 - tuning.deskInset, across))
      const inward = -width * 0.5 + tuning.deskInset + row * rowPitch
      if (choice.deskWall === 'left') return { position: [inward, -alongWall], seat: [inward + seatClearance, -alongWall], rotation: Math.PI * 0.5 }
      return { position: [-inward, alongWall], seat: [-inward - seatClearance, alongWall], rotation: -Math.PI * 0.5 }
    }
    const inward = -depth * 0.5 + tuning.deskInset + row * rowPitch
    const backPosition: Vec2 = [Math.max(-width * 0.5 + tuning.deskInset, Math.min(width * 0.5 - tuning.deskInset, across)), inward]
    return { position: backPosition, seat: [backPosition[0], backPosition[1] + seatClearance], rotation: 0 }
}

export function interiorTablePosition(choice: InteriorChoice, roomSize: Vec2, tuning: LayoutTuning): Vec2 | undefined {
  if (choice.table === 'none') return undefined
  const inset = tuning.tableSeatRadius + tuning.deskInset
  if (choice.table === 'rear-left') return [-roomSize[0] * 0.5 + inset, -roomSize[1] * 0.5 + inset]
  if (choice.table === 'rear-right') return [roomSize[0] * 0.5 - inset, -roomSize[1] * 0.5 + inset]
  return [0, roomSize[1] * 0.5 - inset]
}
