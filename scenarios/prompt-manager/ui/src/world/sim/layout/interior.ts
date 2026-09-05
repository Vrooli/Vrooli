import type { LayoutTuning } from '../../config'
import type { Vec2 } from '../model'
import { Rng, hashString } from '../rng'

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
  const tables = ['front', 'rear-left', 'rear-right'] as const
  const fillerCount = rng.int(tuning.interior.fillerMax + 1)
  const firstLamp = rng.int(4)
  const secondDraw = rng.int(3)
  const secondLamp = secondDraw >= firstLamp ? secondDraw + 1 : secondDraw
  return {
    transposeDesks,
    deskWall: walls[rng.int(walls.length)] ?? 'back',
    table: memberCount >= tuning.interior.tableMinMembers ? tables[rng.int(tables.length)] ?? 'front' : 'none',
    lampCorners: [firstLamp, secondLamp],
    fillers: Array.from({ length: fillerCount }, (_, index) => ({
      index,
      propIndex: rng.int(Math.max(1, fillerKinds)),
      local: [rng.range(-roomSize[0] / 2 + tuning.deskInset, roomSize[0] / 2 - tuning.deskInset), rng.range(-roomSize[1] / 2 + tuning.deskInset, roomSize[1] / 2 - tuning.deskInset)],
      rotation: rng.range(-Math.PI, Math.PI),
    })),
    columns: Math.max(1, Math.ceil(Math.sqrt(memberCount))),
  }
}

/** Convert the pure interior choice into room-local desk and seat transforms. */
export function interiorDesks(choice: InteriorChoice, memberCount: number, roomSize: Vec2, tuning: LayoutTuning): InteriorDesk[] {
  if (memberCount <= 0) return []
  const [width, depth] = roomSize
  const columns = choice.transposeDesks
    ? Math.max(1, Math.ceil(memberCount / choice.columns))
    : choice.columns
  const seatClearance = Math.max(tuning.deskSeatOffset, tuning.deskInset * 0.5 + tuning.cellSize)
  return Array.from({ length: memberCount }, (_, index) => {
    const row = Math.floor(index / columns)
    const column = index % columns
    const rowMembers = Math.min(columns, memberCount - row * columns)
    const across = (column - (rowMembers - 1) * 0.5) * tuning.deskPitch
    if (choice.deskWall === 'left' || choice.deskWall === 'right') {
      const alongWall = Math.max(-depth * 0.5 + tuning.deskInset, Math.min(depth * 0.5 - tuning.deskInset, across))
      const inward = -width * 0.5 + tuning.deskInset + row * tuning.deskPitch
      if (choice.deskWall === 'left') return { position: [inward, -alongWall], seat: [inward + seatClearance, -alongWall], rotation: Math.PI * 0.5 }
      return { position: [-inward, alongWall], seat: [-inward - seatClearance, alongWall], rotation: -Math.PI * 0.5 }
    }
    const inward = -depth * 0.5 + tuning.deskInset + row * tuning.deskPitch
    const backPosition: Vec2 = [Math.max(-width * 0.5 + tuning.deskInset, Math.min(width * 0.5 - tuning.deskInset, across)), inward]
    return { position: backPosition, seat: [backPosition[0], backPosition[1] + seatClearance], rotation: 0 }
  })
}

export function interiorTablePosition(choice: InteriorChoice, roomSize: Vec2, tuning: LayoutTuning): Vec2 | undefined {
  if (choice.table === 'none') return undefined
  const inset = tuning.tableSeatRadius + tuning.deskInset
  if (choice.table === 'rear-left') return [-roomSize[0] * 0.5 + inset, -roomSize[1] * 0.5 + inset]
  if (choice.table === 'rear-right') return [roomSize[0] * 0.5 - inset, -roomSize[1] * 0.5 + inset]
  return [0, roomSize[1] * 0.5 - inset]
}
