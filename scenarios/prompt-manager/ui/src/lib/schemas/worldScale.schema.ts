import { z } from 'zod'

export const WorldScaleConfigSchema = z.object({
  agent: z.number(),
  furniture: z.number(),
  decoration: z.number(),
  overlay: z.number(),
})

export type WorldScaleConfig = z.infer<typeof WorldScaleConfigSchema>
