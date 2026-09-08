import { Vector3 } from 'three'
import { navigationDelta } from './input'

export interface WalkGround { height: number; walkable: boolean }
export type WalkSurface = (x: number, z: number, radius: number) => WalkGround
type WalkOccupied = (position: Vector3, radius: number, verticalSpan?: number) => boolean
export type WalkSweep = ((from: Vector3, to: Vector3, radius: number, verticalSpan?: number) => number) & { overlaps?: WalkOccupied }
import { WALK } from '../../config/navigation'
export { WALK } from '../../config/navigation'

/** A grounded operator body, independent of autonomous agent simulation. */
export class Walker {
  readonly position = new Vector3()
  yaw = 0
  pitch = 0
  boom = WALK.boom as number
  grounded = true
  verticalVelocity = 0
  blocked = false
  private readonly from = new Vector3()
  private readonly to = new Vector3()
  constructor(readonly surface: WalkSurface, readonly sweep: WalkSweep, readonly occupied: WalkOccupied = sweep.overlaps ?? (() => false)) {}

  safe(x: number, z: number): WalkGround | null {
    const ground = this.surface(x, z, WALK.radius)
    if (!ground.walkable || !Number.isFinite(ground.height)) return null
    this.from.set(x, ground.height + WALK.height / 2, z)
    if (this.occupied(this.from, WALK.radius, WALK.height - WALK.radius * 2)) return null
    return ground
  }

  validPosition(): boolean {
    const ground = this.surface(this.position.x, this.position.z, WALK.radius)
    this.from.copy(this.position).y += WALK.height / 2
    return ground.walkable && Number.isFinite(ground.height) && this.position.y >= ground.height &&
      !this.occupied(this.from, WALK.radius, WALK.height - WALK.radius * 2)
  }

  jump(): boolean {
    if (!this.grounded) return false
    this.grounded = false
    this.verticalVelocity = WALK.jumpSpeed
    return true
  }

  private verticalStep(dt: number): void {
    if (this.grounded) return
    const floor = this.surface(this.position.x, this.position.z, WALK.radius).height
    const dy = this.verticalVelocity * dt - WALK.gravity * dt * dt / 2
    this.verticalVelocity -= WALK.gravity * dt
    this.from.copy(this.position).y += WALK.height / 2
    this.to.copy(this.from).y += dy
    const fraction = this.sweep(this.from, this.to, WALK.radius, WALK.height - WALK.radius * 2)
    this.position.y += dy * fraction
    if (fraction < 1) {
      this.grounded = dy < 0
      this.verticalVelocity = 0
    }
    if (this.position.y <= floor) {
      this.position.y = floor
      this.verticalVelocity = 0
      this.grounded = true
    }
  }

  spawn(x: number, z: number): boolean {
    // A bounded search finds a body-sized site near the current inspection target.
    for (let ring = 0; ring <= WALK.spawnRings; ring++) {
      const count = Math.max(1, ring * 8)
      for (let i = 0; i < count; i++) {
        const angle = i / count * Math.PI * 2
        const nx = x + Math.cos(angle) * ring * WALK.spawnRingMetres, nz = z + Math.sin(angle) * ring * WALK.spawnRingMetres
        const ground = this.safe(nx, nz)
        if (ground) { this.position.set(nx, ground.height, nz); this.grounded = true; this.verticalVelocity = 0; return true }
      }
    }
    return false
  }

  private move(dx: number, dz: number): boolean {
    const { x, y, z } = this.position
    const ground = this.surface(x + dx, z + dz, WALK.radius)
    const distance = Math.hypot(dx, dz)
    if (!ground.walkable || !Number.isFinite(ground.height)) return false
    const rise = Math.abs(ground.height - y)
    // Permit small kerbs, but reject cliffs and sustained excessive slopes.
    if (this.grounded && (rise > WALK.stepHeight || (rise > WALK.kerbTolerance && rise / Math.max(distance, WALK.slopeSampleMetres) > WALK.maxGrade))) return false
    if (!this.grounded && ground.height > y) return false
    const span = WALK.height - WALK.radius * 2
    // Sweep horizontally at foot height first. If blocked, try a bounded
    // up/across/down stair move, checking headroom along the entire route.
    this.from.set(x, y + WALK.height / 2, z)
    this.to.set(x + dx, y + WALK.height / 2, z + dz)
    if (this.sweep(this.from, this.to, WALK.radius, span) < 1 || ground.height > y) {
      if (!this.grounded) return false
      this.to.copy(this.from).y += WALK.stepHeight
      if (this.sweep(this.from, this.to, WALK.radius, span) < 1) return false
      this.from.copy(this.to)
      this.to.x += dx; this.to.z += dz
      if (this.sweep(this.from, this.to, WALK.radius, span) < 1 || this.occupied(this.to, WALK.radius, span)) return false
    }
    const top = this.to.y - WALK.height / 2
    if (this.grounded) {
      this.from.copy(this.to)
      this.to.y = ground.height + WALK.height / 2
      const fraction = this.sweep(this.from, this.to, WALK.radius, span)
      this.position.set(x + dx, top + (ground.height - top) * fraction, z + dz)
    } else this.position.set(x + dx, y, z + dz)
    return true
  }

  step(seconds: number, forward: number, right: number, run: boolean): void {
    this.blocked = false
    const length = Math.hypot(forward, right)
    const dt = navigationDelta(seconds)
    const distance = (length ? dt : 0) * (run ? WALK.runSpeed : WALK.speed)
    const dx = (Math.sin(this.yaw) * forward + Math.cos(this.yaw) * right) / (length || 1) * distance
    const dz = (-Math.cos(this.yaw) * forward + Math.sin(this.yaw) * right) / (length || 1) * distance
    const steps = Math.max(1, Math.ceil(distance / WALK.stepMetres), Math.ceil(dt / WALK.physicsStep))
    for (let i = 0; i < steps; i++) {
      this.verticalStep(dt / steps)
      if (!length || this.move(dx / steps, dz / steps)) continue
      this.blocked = true
      // Slide along a wall without accelerating diagonal movement.
      if (dx !== 0) this.move(dx / steps, 0)
      if (dz !== 0) this.move(0, dz / steps)
    }
  }

  look(dx: number, dy: number, sensitivity: number, invert: boolean): void {
    this.yaw += dx * WALK.lookRadiansPerPixel * sensitivity
    this.pitch = Math.max(-WALK.maxPitch, Math.min(WALK.maxPitch, this.pitch - dy * WALK.lookRadiansPerPixel * sensitivity * (invert ? -1 : 1)))
  }

  view(thirdPerson: boolean): { eye: Vector3; target: Vector3 } {
    const head = this.position.clone().add(new Vector3(0, WALK.eyeHeight, 0))
    const direction = new Vector3(Math.sin(this.yaw) * Math.cos(this.pitch), Math.sin(this.pitch), -Math.cos(this.yaw) * Math.cos(this.pitch))
    if (!thirdPerson) return { eye: head, target: head.clone().add(direction) }
    const desired = head.clone().addScaledVector(direction, -this.boom)
    let fraction = this.sweep(head, desired, WALK.boomRadius)
    const samples = Math.ceil(this.boom / WALK.boomSampleMetres)
    for (let i = 1; i <= samples; i++) {
      const t = i / samples
      const point = head.clone().lerp(desired, t)
      if (point.y < this.surface(point.x, point.z, WALK.boomRadius).height + WALK.boomRadius) { fraction = Math.min(fraction, (i - 1) / samples); break }
    }
    return { eye: head.clone().lerp(desired, Math.max(0, fraction)), target: head.clone().add(direction) }
  }
}
