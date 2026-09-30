import { Quaternion, Vector3 } from 'three'
import { navigationDelta, type WorldCameraControls } from './input'
import { nearPlaneRadius } from './pose'

interface Waypoint { position: Vector3; target: Vector3 }
export interface PoseRouteStatus { status: 'moving' | 'complete' | 'blocked'; waypoints: number; replans: number; planningMs: number }

/** Plan clear straight eye segments; target interpolation must not bend that path. */
export function planPoseRoute(controls: WorldCameraControls, requestedPosition: Vector3, target: Vector3): Waypoint[] | null {
  const camera = controls.camera
  if (!('fov' in camera)) return null
  const radius = nearPlaneRadius(camera.near, camera.fov, camera.aspect, camera.zoom)
  const from = controls.getPosition(new Vector3(), false)
  const currentTarget = controls.getTarget(new Vector3(), false)
  const end = requestedPosition.clone()
  const floor = (a: Vector3, b: Vector3) => (controls.groundCeiling?.(Math.min(a.x, b.x) - radius, Math.min(a.z, b.z) - radius, Math.max(a.x, b.x) + radius, Math.max(a.z, b.z) + radius) ?? -Infinity) + radius
  end.y = Math.max(end.y, floor(end, end))
  end.y += controls.obstacleRecovery?.(end, radius) ?? 0
  const clear = (a: Vector3, b: Vector3) =>
    Math.min(a.y, b.y) + 1e-8 >= floor(a, b) && (controls.obstacleSweep?.(a, b, radius) ?? 1) >= 1
  const destination = { position: end, target: target.clone() }
  if (clear(from, end)) return [destination]
  const height = Math.max(from.y, end.y, floor(from, end), controls.obstacleCeiling?.(from, end, radius) ?? -Infinity) + 0.1
  const aboveStart = from.clone().setY(height)
  const aboveEnd = end.clone().setY(height)
  const candidates: Array<[Vector3, Vector3]> = [[aboveStart, aboveEnd]]
  const obstacleBounds = controls.obstacleBounds?.(from, end, radius)
  if (obstacleBounds) {
    for (const axis of ['x', 'z'] as const) {
      for (const edge of [obstacleBounds.min[axis] - 0.1, obstacleBounds.max[axis] + 0.1]) {
        const a = from.clone()
        const b = end.clone()
        a[axis] = edge
        b[axis] = edge
        candidates.push([a, b])
      }
    }
  }
  let result: Waypoint[] | null = null
  let shortest = Infinity
  for (const [a, b] of candidates) {
    const length = from.distanceTo(a) + a.distanceTo(b) + b.distanceTo(end)
    if (length >= shortest || !clear(from, a) || !clear(a, b) || !clear(b, end)) continue
    shortest = length
    result = [
      { position: a, target: currentTarget.clone().add(a.clone().sub(from)) },
      { position: b, target: target.clone().add(b.clone().sub(end)) },
      destination,
    ]
  }
  return result
}

/** A single cancelable automatic pose writer, advanced before clearance updates. */
export class PoseRoute {
  private route: Waypoint[] = []
  private index = 0
  private elapsed = 0
  private replans = 0
  private planningMs = 0
  private active = true
  private awaiting = false
  private readonly fromPosition = new Vector3()
  private readonly fromTarget = new Vector3()
  private readonly position = new Vector3()
  private readonly target = new Vector3()
  private readonly startDirection = new Vector3()
  private readonly endDirection = new Vector3()
  private readonly turn = new Quaternion()
  private readonly rotation = new Quaternion()

  constructor(
    private readonly controls: WorldCameraControls,
    private readonly destination: Vector3,
    private readonly destinationTarget: Vector3,
    private readonly seconds: number,
    private readonly complete: () => void,
    private readonly blocked: () => void,
    private readonly report: (status: PoseRouteStatus) => void = () => {},
  ) {
    // Resolve any newly committed overlap before using the eye as a route
    // origin. This runs the same clearance owner as ordinary rendered frames.
    this.controls.update(0)
    this.plan()
  }

  cancel(): void { this.active = false }

  private plan() {
    const started = performance.now()
    const route = planPoseRoute(this.controls, this.destination, this.destinationTarget)
    this.planningMs = performance.now() - started
    if (!route) { this.fail(); return false }
    this.route = route
    this.index = 0
    this.elapsed = 0
    this.awaiting = false
    this.controls.getPosition(this.fromPosition, false)
    this.controls.getTarget(this.fromTarget, false)
    this.report({ status: 'moving', waypoints: route.length, replans: this.replans, planningMs: this.planningMs })
    return true
  }

  private fail() {
    this.active = false
    this.report({ status: 'blocked', waypoints: this.route.length, replans: this.replans, planningMs: this.planningMs })
    this.blocked()
  }

  tick(delta: number): void {
    if (!this.active) return
    // Completion follows the actual guarded frame, never just a rest event.
    if (this.awaiting) {
      const actual = this.controls.getPosition(new Vector3(), false)
      const actualTarget = this.controls.getTarget(new Vector3(), false)
      if (actual.distanceTo(this.position) > 1e-4 || actualTarget.distanceTo(this.target) > 1e-4) {
        if (++this.replans > 3) { this.fail(); return }
        if (!this.plan()) return
      } else if (this.elapsed >= this.seconds / this.route.length) {
        this.index++
        if (this.index === this.route.length) {
          this.active = false
          this.report({ status: 'complete', waypoints: this.route.length, replans: this.replans, planningMs: this.planningMs })
          this.complete()
          return
        }
        this.fromPosition.copy(actual)
        this.fromTarget.copy(actualTarget)
        this.elapsed = 0
      }
    }
    const next = this.route[this.index]
    if (!next) return
    const segmentSeconds = this.seconds / this.route.length
    this.elapsed = Math.min(segmentSeconds, this.elapsed + navigationDelta(delta))
    const fraction = segmentSeconds > 0 ? this.elapsed / segmentSeconds : 1
    const t = fraction * fraction * (3 - 2 * fraction)
    this.position.lerpVectors(this.fromPosition, next.position, t)
    this.startDirection.subVectors(this.fromTarget, this.fromPosition)
    this.endDirection.subVectors(next.target, next.position)
    const distance = this.startDirection.length() * (1 - t) + this.endDirection.length() * t
    this.startDirection.normalize()
    this.endDirection.normalize()
    this.turn.setFromUnitVectors(this.startDirection, this.endDirection)
    this.rotation.identity().slerp(this.turn, t)
    this.target.copy(this.startDirection).applyQuaternion(this.rotation).multiplyScalar(distance).add(this.position)
    void this.controls.setLookAt(...this.position.toArray(), ...this.target.toArray(), false)
    this.awaiting = true
  }

  /** Reduced motion crosses only validated segments without displaying intermediate frames. */
  finishImmediately(): void {
    for (let step = 0; this.active && step < 16; step++) {
      this.tick(0)
      this.controls.update(0)
    }
    if (this.active) this.fail()
  }
}
