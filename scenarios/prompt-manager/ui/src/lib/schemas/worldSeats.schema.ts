import { z } from 'zod'

export const SeatPositionSchema = z.object({
  position: z.tuple([z.number(), z.number(), z.number()]),
  rotation: z.number(),
})

export const WorldSeatsConfigSchema = z.record(z.string(), z.array(SeatPositionSchema))

export type WorldSeatsConfig = z.infer<typeof WorldSeatsConfigSchema>
