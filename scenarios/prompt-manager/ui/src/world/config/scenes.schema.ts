/**
 * Schema for a scene data file (config/scenes/*.json).
 *
 * A scene is data, never a component: it names the asset set, the prop id used
 * for each place kind, the camera poses, the palette and any lighting period
 * overrides an indoor scene needs.
 */
import { z } from 'zod'
import { PeriodSchema } from './tuning.schema'

const hex = z.string().regex(/^#[0-9a-fA-F]{6}$/)

export const CameraPoseSchema = z.object({
  azimuthDeg: z.number().min(-180).max(180).describe('Orbit angle around the slab centre (degrees)'),
  polarDeg: z.number().min(0).max(90).describe('Angle from straight above (degrees)'),
  distanceFactor: z.number().min(0.2).max(5).describe('Distance as a multiple of the distance at which the layout footprint fills camera.frameFill of the viewport (multiplier)'),
  targetY: z.number().min(-10).max(50).describe('Height of the look-at point above the slab (metres)'),
})

export const PropIdSchema = z.string().regex(/^[a-z0-9_]+$/)

export const SceneSchema = z.object({
  id: z.enum(['park', 'office']),
  title: z.string().min(1),
  environment: z.enum(['outdoor', 'indoor']),
  assetSet: z.string().regex(/^[a-z0-9-]+$/).describe('Directory under public/assets/world holding this scene props'),
  biomeSet: z.enum(['park', 'office']).describe('Biome set that supplies terrain colours and ground-bound props'),
  props: z.object({
    desk: PropIdSchema,
    chair: PropIdSchema,
    table: PropIdSchema,
    seat: PropIdSchema,
    campfire: PropIdSchema,
    lamp: PropIdSchema,
    board: PropIdSchema,
  }),
  palette: z.object({
    /** Terrain beyond the lot, out to the fogged horizon. */
    horizon: hex,
    ground: hex,
    roomFloor: hex,
    roomWall: hex,
    commons: hex,
    path: hex,
  }),
  /** Uniform scale applied to every baked prop so kit units read as metres in this world. */
  propScale: z.number().min(0.1).max(10),
  /** Extra scale for trees on top of propScale. */
  treeScale: z.number().min(0.1).max(10),
  camera: z.object({
    hero: CameraPoseSchema,
    establishing: CameraPoseSchema,
  }),
  lighting: z
    .object({
      periods: z
        .object({
          dawn: PeriodSchema.partial().optional(),
          day: PeriodSchema.partial().optional(),
          dusk: PeriodSchema.partial().optional(),
          night: PeriodSchema.partial().optional(),
        })
        .optional(),
    })
    .optional(),
})

export type Scene = z.infer<typeof SceneSchema>
export type CameraPose = z.infer<typeof CameraPoseSchema>
