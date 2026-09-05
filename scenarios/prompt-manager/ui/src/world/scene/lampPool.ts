import { Camera, PointLight } from 'three'
import { CameraMotionGate } from '../engine/camera/motion'
import type { LightingPeriod, QualityProfile } from '../config'

export function lampPoolSize(profile: QualityProfile, period: LightingPeriod, color?: string): number {
  return color && period.lampEmissive > 0 ? profile.lampLights : 0
}

export interface LampPlacement {
  position: readonly [number, number]
  y?: number
  scale: number
}
export interface LampSettings {
  color: string
  intensity: number
  distance: number
  height: number
}

/** Stable light objects: camera movement updates transforms, never shader light counts. */
export class LampLightPool {
  readonly lights: PointLight[]
  private readonly indices: Int32Array
  private readonly distances: Float64Array
  private readonly motion = new CameraMotionGate()
  private lastPlacements: readonly LampPlacement[] | null = null
  private lastSettings: LampSettings | null = null
  runs = 0
  skips = 0
  constructor(count: number) {
    this.lights = Array.from({ length: count }, () => new PointLight())
    this.indices = new Int32Array(count)
    this.distances = new Float64Array(count)
    for (const light of this.lights) light.intensity = 0
  }
  update(camera: Camera, placements: readonly LampPlacement[], settings: LampSettings, metres: number, radians: number): boolean {
    const moved = this.motion.changed(camera, metres, radians)
    if (!moved && this.lastPlacements === placements && this.lastSettings === settings) {
      this.skips += 1
      return false
    }
    this.lastPlacements = placements
    this.lastSettings = settings
    this.indices.fill(-1)
    this.distances.fill(Infinity)
    // The profile caps K at 32. A bounded insertion list avoids per-item allocation.
    for (let i = 0; i < placements.length; i += 1) {
      const placement = placements[i]
      if (!placement) continue
      const dx = placement.position[0] - this.motion.position.x
      const dy = (placement.y ?? 0) + settings.height * placement.scale - this.motion.position.y
      const dz = placement.position[1] - this.motion.position.z
      const distance = dx * dx + dy * dy + dz * dz
      for (let slot = 0; slot < this.lights.length; slot += 1) {
        if (distance >= (this.distances[slot] ?? Infinity)) continue
        for (let shift = this.lights.length - 1; shift > slot; shift -= 1) {
          this.indices[shift] = this.indices[shift - 1] ?? -1
          this.distances[shift] = this.distances[shift - 1] ?? Infinity
        }
        this.indices[slot] = i
        this.distances[slot] = distance
        break
      }
    }
    for (let slot = 0; slot < this.lights.length; slot += 1) {
      const light = this.lights[slot]
      if (!light) continue
      const placement = placements[this.indices[slot] ?? -1]
      light.intensity = placement ? settings.intensity : 0
      light.color.set(settings.color)
      light.distance = settings.distance
      if (placement) light.position.set(placement.position[0], (placement.y ?? 0) + settings.height * placement.scale, placement.position[1])
    }
    this.runs += 1
    return true
  }
}
