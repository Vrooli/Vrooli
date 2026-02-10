import type { SeatPosition } from '@/types/furniture'
import type { FurnitureType } from '@/types/furniture'

export type WorldSeatsConfig = Record<string, SeatPosition[]>

/** Typed version for internal use */
export type WorldSeatsMap = Partial<Record<FurnitureType, SeatPosition[]>>
