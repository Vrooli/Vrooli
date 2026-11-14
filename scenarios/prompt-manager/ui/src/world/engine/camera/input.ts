import CameraControls from 'camera-controls'
import { InstancedMesh, Matrix3, Matrix4, Mesh, Raycaster, Spherical, Vector2, Vector3, type Box3, type Intersection, type Object3D, type OrthographicCamera, type PerspectiveCamera } from 'three'
import { NAV_MOTION } from '../../config/navigation'
import type { CameraTuning } from '../../config'
import { nearPlaneRadius } from './pose'

type InputMap = CameraTuning['input']
const A = CameraControls.ACTION
const mouse = {
  none: A.NONE, rotate: A.ROTATE, truck: A.TRUCK, offset: A.OFFSET, dolly: A.DOLLY, zoom: A.ZOOM,
} satisfies Record<InputMap['mouse']['left'], number>
const single = {
  none: A.NONE, rotate: A.TOUCH_ROTATE, truck: A.TOUCH_TRUCK, 'screen-pan': A.TOUCH_SCREEN_PAN,
  offset: A.TOUCH_OFFSET, dolly: A.DOLLY, zoom: A.ZOOM,
} satisfies Record<InputMap['touch']['one'], number>
const multi = {
  none: A.NONE, rotate: A.TOUCH_ROTATE, truck: A.TOUCH_TRUCK, 'screen-pan': A.TOUCH_SCREEN_PAN,
  offset: A.TOUCH_OFFSET, dolly: A.TOUCH_DOLLY, zoom: A.TOUCH_ZOOM,
  'dolly-truck': A.TOUCH_DOLLY_TRUCK, 'dolly-screen-pan': A.TOUCH_DOLLY_SCREEN_PAN,
  'dolly-offset': A.TOUCH_DOLLY_OFFSET, 'dolly-rotate': A.TOUCH_DOLLY_ROTATE,
  'zoom-truck': A.TOUCH_ZOOM_TRUCK, 'zoom-screen-pan': A.TOUCH_ZOOM_SCREEN_PAN,
  'zoom-offset': A.TOUCH_ZOOM_OFFSET, 'zoom-rotate': A.TOUCH_ZOOM_ROTATE,
} satisfies Record<InputMap['touch']['two'], number>

function action(map: Readonly<Record<string, number>>, value: string): number {
  if (!Object.prototype.hasOwnProperty.call(map, value)) throw new Error(`Unmapped camera input action: ${value}`)
  const result = map[value]
  if (result === undefined) throw new Error(`Undefined camera input action: ${value}`)
  return result
}

/** Structural numeric slots include runtime actions omitted by upstream typings. */
interface InputControls {
  mouseButtons: Record<keyof InputMap['mouse'], number>
  touches: Record<keyof InputMap['touch'], number>
}

/** Resolve all seven slots before mutating controls; invalid maps never partially apply. */
export function applyInputMap(controls: InputControls, input: InputMap): void {
  const mouseButtons = {
    left: action(mouse, input.mouse.left), middle: action(mouse, input.mouse.middle),
    right: action(mouse, input.mouse.right), wheel: action(mouse, input.mouse.wheel),
  }
  const touches = {
    one: action(single, input.touch.one), two: action(multi, input.touch.two), three: action(multi, input.touch.three),
  }
  controls.mouseButtons = mouseButtons
  controls.touches = touches
}

const navigationKeys = new Set(['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', '=', '+', '-', 'w', 'a', 's', 'd', 'W', 'A', 'S', 'D', 'Shift'])

const normalizedWheels = new WeakSet<WheelEvent>()

/** Normalize the existing event at the canvas boundary; never redispatch wheel input. */
export function normalizeWheelUnits(event: WheelEvent, element: HTMLElement): void {
  if (normalizedWheels.has(event)) return
  const mode = event.deltaMode
  if (mode === 0) { normalizedWheels.add(event); return }
  const style = getComputedStyle(element)
  const fontSize = Number.parseFloat(style.fontSize) || 16
  const lineHeight = Number.parseFloat(style.lineHeight) || fontSize * 1.2
  const rect = element.getBoundingClientRect()
  const scaleX = mode === 1 ? lineHeight : Math.max(1, rect.width)
  const scaleY = mode === 1 ? lineHeight : Math.max(1, rect.height)
  // WheelEvent accessors live on its prototype. Shadow only the unit-bearing
  // values on this event; identity, trust, modifiers and cancellation stay intact.
  Object.defineProperties(event, {
    deltaX: { value: event.deltaX * scaleX, configurable: true },
    deltaY: { value: event.deltaY * scaleY, configurable: true },
    deltaZ: { value: event.deltaZ * scaleY, configurable: true },
    deltaMode: { value: 0, configurable: true },
  })
  normalizedWheels.add(event)
}

/** UI controls own their keys, including children of dialogs and editable regions. */
export function ownsKeyboardInput(target: EventTarget | null): boolean {
  return target instanceof Element && target.closest('input, textarea, select, button, [contenteditable]:not([contenteditable="false"]), [role="dialog"], [aria-modal="true"], [role="slider"], [role="spinbutton"], [role="combobox"], [role="listbox"], [role="menu"], [role="radiogroup"], [role="tablist"]') !== null
}

/** Never integrate a tab's accumulated suspension time as one navigation step. */
export function navigationDelta(seconds: number): number {
  return Number.isFinite(seconds) ? seconds > NAV_MOTION.maxOrdinaryFrame ? NAV_MOTION.integrationStep : Math.max(seconds, 0) : 0
}

/** Only declared physical surfaces participate; clouds, labels and editor handles do not. */
export function createZoomSurfaceQuery(root: Object3D, camera: PerspectiveCamera | OrthographicCamera) {
  const raycaster = new Raycaster()
  ;(raycaster as Raycaster & { firstHitOnly: boolean }).firstHitOnly = true
  const cursor = new Vector2()
  const matrix = new Matrix4()
  const instance = new Matrix4()
  const normalMatrix = new Matrix3()
  const normal = new Vector3()
  const forward = new Vector3()
  const offset = new Vector3()
  const clampedTarget = new Vector3()
  const eye = new Vector3()
  const surfaces: Object3D[] = []
  const hits: Intersection[] = []
  return (x: number, y: number, radius: number, target?: Vector3, boundary?: Box3): number | null => {
    camera.updateMatrixWorld()
    root.updateWorldMatrix(true, true)
    raycaster.setFromCamera(cursor.set(x, y), camera)
    raycaster.far = camera.far
    surfaces.length = 0
    root.traverseVisible(object => {
      if (!(object instanceof Mesh)) return
      for (let parent: Object3D | null = object; parent; parent = parent.parent) {
        if (parent.userData.cameraSurface === true) { surfaces.push(object); break }
      }
    })
    hits.length = 0
    for (const surface of surfaces) {
      if (surface instanceof InstancedMesh) {
        // Drei disables the batch raycast in favor of interactive proxies.
        // Physical queries use the batch geometry without changing UI picking.
        surface.computeBoundingSphere()
        InstancedMesh.prototype.raycast.call(surface, raycaster, hits)
      } else raycaster.intersectObject(surface, false, hits)
      // A nearer physical hit bounds subsequent terrain/mesh traversal.
      for (const hit of hits) raycaster.far = Math.min(raycaster.far, hit.distance)
    }
    hits.sort((a, b) => a.distance - b.distance)
    const hit = hits[0]
    if (!hit?.face) return null
    matrix.copy(hit.object.matrixWorld)
    if (hit.object instanceof InstancedMesh && hit.instanceId !== undefined) {
      hit.object.getMatrixAt(hit.instanceId, instance)
      matrix.multiply(instance)
    }
    normal.copy(hit.face.normal).applyNormalMatrix(normalMatrix.getNormalMatrix(matrix))
    if (normal.dot(raycaster.ray.direction) > 0) normal.negate()
    const incidence = Math.abs(normal.dot(raycaster.ray.direction))
    // A sphere needs more distance along a grazing ray to clear the hit plane.
    const travel = Math.max(0, hit.distance - radius / Math.max(incidence, 1e-8))
    if (!target || !boundary || travel === 0) return travel
    camera.getWorldDirection(forward)
    const radialScale = raycaster.ray.direction.dot(forward)
    offset.copy(raycaster.ray.direction).addScaledVector(forward, -radialScale)
    // Cursor dolly translates the target sideways while shortening the orbit
    // radius. Boundary clamping can remove part of that translation, bending
    // the eye's path away from the cursor ray. Each axis clamps at most once;
    // check every resulting linear segment, including an unsafe interior turn.
    const stops = [0, travel]
    for (const axis of ['x', 'y', 'z'] as const) {
      if (offset[axis] === 0) continue
      const edge = offset[axis] > 0 ? boundary.max[axis] : boundary.min[axis]
      const crossing = (edge - target[axis]) / offset[axis]
      if (crossing > 0 && crossing < travel) stops.push(crossing)
    }
    stops.sort((a, b) => a - b)
    let previousTravel = 0
    let previousMargin = 0
    for (const distance of stops) {
      clampedTarget.copy(target).addScaledVector(offset, distance).clamp(boundary.min, boundary.max)
      eye.copy(camera.position).addScaledVector(forward, distance * radialScale).add(clampedTarget).sub(target)
      const margin = eye.sub(hit.point).dot(normal) - radius
      if (margin < 0) {
        if (distance === 0) return 0
        return previousTravel + (distance - previousTravel) * previousMargin / (previousMargin - margin)
      }
      previousTravel = distance
      previousMargin = margin
    }
    return travel
  }
}

/** Bound the controller's own integration, not only our keyboard increments. */
export class WorldCameraControls extends CameraControls {
  /** Walking owns the lens and body clearance while active. */
  externalCamera = false
  readonly externalTarget = new Vector3()
  inputSequence = 0
  lastInputAction: 'pan' | 'dolly' | null = null
  blocked = false
  private blockedUntil = 0
  private markBlocked(): void { this.blocked = true; this.blockedUntil = performance.now() + NAV_MOTION.blockedNoticeMs }
  groundCeiling?: (minX: number, minZ: number, maxX: number, maxZ: number) => number
  obstacleSweep?: (from: Vector3, to: Vector3, radius: number) => number
  obstacleRecovery?: (position: Vector3, radius: number) => number
  obstacleCeiling?: (from: Vector3, to: Vector3, radius: number) => number
  obstacleBounds?: (from: Vector3, to: Vector3, radius: number) => Box3 | null
  zoomSurfaceTravel?: ReturnType<typeof createZoomSurfaceQuery>
  onZoomGuard?: (sample: { travel: number | null; limited: boolean; queryMs: number }) => void
  private readonly clearancePosition = new Vector3()
  private readonly clearanceTarget = new Vector3()
  private readonly previousEye = new Vector3()
  private readonly recoveryPosition = new Vector3()
  private readonly clearanceAngles = new Spherical()
  private readonly clearanceEndAngles = new Spherical()
  private hasPreviousEye = false
  constructor(...args: ConstructorParameters<typeof CameraControls>) {
    super(...args)
    const nativeDolly = this._dollyInternal
    // Wrap the shared wheel/touch entry point, keeping one input/motion writer.
    this._dollyInternal = (delta, x, y) => {
      nativeDolly(delta, x, y)
      const camera = this.camera
      if (!this.zoomSurfaceTravel || !('fov' in camera) || this._sphericalEnd.radius >= this._spherical.radius) return
      const radius = nearPlaneRadius(camera.near, camera.fov, camera.aspect, camera.zoom) + 1e-5
      const started = performance.now()
      const cursorX = this.dollyToCursor ? x : 0
      const cursorY = this.dollyToCursor ? y : 0
      const travel = this.zoomSurfaceTravel(cursorX, cursorY, radius, this._target, this._boundary)
      const queryMs = performance.now() - started
      if (travel === null || !Number.isFinite(travel)) {
        this.onZoomGuard?.({ travel: null, limited: false, queryMs })
        return
      }
      const tangent = Math.tan(camera.getEffectiveFOV() * Math.PI / 360)
      const rayScale = Math.hypot(1, cursorX * tangent * camera.aspect, cursorY * tangent)
      const safeRadius = Math.max(this._sphericalEnd.radius, this._spherical.radius - Math.max(0, travel) / rayScale)
      this.onZoomGuard?.({ travel, limited: safeRadius > this._sphericalEnd.radius, queryMs })
      if (this.dollyToCursor) this._changedDolly += safeRadius - this._sphericalEnd.radius
      this._sphericalEnd.radius = safeRadius
    }
  }
  dollyPixels(deltaY: number, x: number, y: number): void {
    this.inputSequence++
    this.lastInputAction = 'dolly'
    this._dollyInternal(deltaY / NAV_MOTION.wheelPixelsPerUnit, x, y)
    if (deltaY !== 0 && Math.abs(this._spherical.radius - this._sphericalEnd.radius) < NAV_MOTION.motionEpsilon) this.markBlocked()
    this._spherical.radius = this._sphericalEnd.radius
    this._radiusVelocity.value = 0
    this.update(0)
  }
  panPixels(x: number, y: number): void {
    this.inputSequence++
    this.lastInputAction = 'pan'
    this._getClientRect(this._elementRect)
    this._truckInternal(x, y, false, false)
    this._target.copy(this._targetEnd)
    this._targetVelocity.set(0, 0, 0)
    this.update(0)
  }
  resetClearance(): void { this.hasPreviousEye = false }
  override getTarget(out: Vector3, receiveEndValue = true): Vector3 {
    return this.externalCamera ? out.copy(this.externalTarget) : super.getTarget(out, receiveEndValue)
  }
  override update(delta: number): boolean {
    if (this.externalCamera) return false
    const seconds = navigationDelta(delta)
    const steps = Math.max(1, Math.ceil(seconds / NAV_MOTION.integrationStep))
    let changed = false
    this.blocked = performance.now() < this.blockedUntil
    for (let i = 0; i < steps; i++) changed = this.integrate(seconds / steps) || changed
    return changed
  }
  private integrate(delta: number): boolean {
    const changed = super.update(delta)
    if (!this.groundCeiling && !this.obstacleSweep && !this.obstacleRecovery) { this.hasPreviousEye = false; return changed }
    const camera = this.camera
    if (!('fov' in camera)) return changed
    const radius = nearPlaneRadius(camera.near, camera.fov, camera.aspect, camera.zoom)
    const { x, y, z } = camera.position
    const floor = this.groundCeiling?.(x - radius, z - radius, x + radius, z + radius) ?? -Infinity
    let dx = 0
    let dy = Math.max(0, floor + radius - y)
    let dz = 0
    const recoveryBase = this.hasPreviousEye ? this.previousEye : camera.position
    const recoveryFloor = this.groundCeiling?.(recoveryBase.x - radius, recoveryBase.z - radius, recoveryBase.x + radius, recoveryBase.z + radius) ?? -Infinity
    this.recoveryPosition.copy(recoveryBase)
    this.recoveryPosition.y = Math.max(recoveryBase.y, recoveryFloor + radius)
    const recoveryLift = this.recoveryPosition.y - recoveryBase.y + (this.obstacleRecovery?.(this.recoveryPosition, radius) ?? 0)
    if (recoveryLift > 0) {
      // A new layout can enclose a stationary eye. Recover locally and keep
      // its horizontal location and view direction, rather than replay home.
      dx = recoveryBase.x - x
      dy = recoveryBase.y + recoveryLift - y
      dz = recoveryBase.z - z
    } else if (this.hasPreviousEye && this.previousEye.distanceToSquared(camera.position) > 1e-16) {
      const previous = this.previousEye
      const previousFloor = this.groundCeiling?.(previous.x - radius, previous.z - radius, previous.x + radius, previous.z + radius) ?? -Infinity
      const sweptFloor = this.groundCeiling?.(Math.min(previous.x, x) - radius, Math.min(previous.z, z) - radius, Math.max(previous.x, x) + radius, Math.max(previous.z, z) + radius) ?? -Infinity
      // A bounding prism encloses the whole near-plane sphere sweep. If the
      // previous eye is still safe, reject translation through its ceiling;
      // the operator can ascend or move around the obstruction. Terrain
      // regeneration that invalidates the previous eye uses local recovery.
      if (previous.y + 1e-8 >= previousFloor + radius && Math.min(previous.y, y) + 1e-8 < sweptFloor + radius) {
        dx = previous.x - x
        dy = previous.y - y
        dz = previous.z - z
      }
      // Sweep the actual proposed translation after terrain correction, so
      // wheel, pan, orbit and mixed input share the same structure guard.
      const candidate = this.clearancePosition.set(x + dx, y + dy, z + dz)
      const fraction = this.obstacleSweep?.(previous, candidate, radius) ?? 1
      if (fraction < 1) {
        candidate.lerpVectors(previous, candidate, fraction)
        dx = candidate.x - x
        dy = candidate.y - y
        dz = candidate.z - z
      }
    }
    if (dx === 0 && dy === 0 && dz === 0) {
      this.previousEye.copy(camera.position)
      this.hasPreviousEye = true
      return changed
    }
    this.markBlocked()
    const position = this.getPosition(this.clearancePosition, false)
    const target = this.getTarget(this.clearanceTarget, false)
    const angles = this.getSpherical(this.clearanceAngles, false)
    const endAngles = this.getSpherical(this.clearanceEndAngles, true)
    const remainingYaw = endAngles.theta - angles.theta
    const remainingPolar = endAngles.phi - angles.phi
    void this.setLookAt(position.x + dx, position.y + dy, position.z + dz, target.x + dx, target.y + dy, target.z + dz, false)
    // Eye and target translate equally, so orientation is unchanged. Publish
    // that translation directly without a zero-delta damping step, which
    // would erase angular velocity on every contact frame.
    camera.position.set(x + dx, y + dy, z + dz)
    camera.updateMatrixWorld()
    // Translation is constrained, but discarding the pending angular endpoint
    // would repeatedly restart damping and trap smooth input at the horizon.
    if (remainingYaw !== 0 || remainingPolar !== 0) void this.rotate(remainingYaw, remainingPolar, true)
    this.previousEye.copy(camera.position)
    this.hasPreviousEye = true
    return true
  }
}

/** Bubble-phase fallback: editors and dialogs can consume Escape first. */
export function bindCameraHome(home: () => void): () => void {
  const key = (event: KeyboardEvent) => {
    if (event.key !== 'Escape' || event.defaultPrevented || event.ctrlKey || event.metaKey || event.altKey || ownsKeyboardInput(event.target)) return
    event.preventDefault()
    home()
  }
  window.addEventListener('keydown', key)
  return () => window.removeEventListener('keydown', key)
}

export function bindNavigationKeys(pressed: Set<string>, enabled: () => boolean, navigate: () => void = () => {}): () => void {
  const clear = () => pressed.clear()
  const down = (event: KeyboardEvent) => {
    if (event.defaultPrevented || event.ctrlKey || event.metaKey || event.altKey || !enabled() || ownsKeyboardInput(event.target)) {
      clear()
      return
    }
    if (!navigationKeys.has(event.key)) return
    if (pressed.size === 0) navigate()
    pressed.add(event.key.length === 1 ? event.key.toLowerCase() : event.key)
    event.preventDefault()
  }
  const up = (event: KeyboardEvent) => { pressed.delete(event.key.length === 1 ? event.key.toLowerCase() : event.key) }
  const focus = (event: FocusEvent) => { if (ownsKeyboardInput(event.target)) clear() }
  window.addEventListener('keydown', down)
  window.addEventListener('keyup', up)
  window.addEventListener('blur', clear)
  document.addEventListener('visibilitychange', clear)
  document.addEventListener('focusin', focus)
  return () => {
    window.removeEventListener('keydown', down)
    window.removeEventListener('keyup', up)
    window.removeEventListener('blur', clear)
    document.removeEventListener('visibilitychange', clear)
    document.removeEventListener('focusin', focus)
    clear()
  }
}

/** Called only by explicit focus commands, never by manual orbit or each frame. */
export function focusVisibility(root: Object3D, eye: Vector3, target: Vector3): boolean {
  const raycaster = new Raycaster(eye, target.clone().sub(eye).normalize(), 0, eye.distanceTo(target))
  ;(raycaster as Raycaster & { firstHitOnly: boolean }).firstHitOnly = true
  const hits: Intersection[] = []
  root.updateWorldMatrix(true, true)
  root.traverseVisible(object => {
    if (!(object instanceof Mesh) || hits.length) return
    let eligible = false
    for (let parent: Object3D | null = object; parent; parent = parent.parent) {
      if (parent.userData.cameraSurface || parent.userData.cameraOccluder) { eligible = true; break }
    }
    if (!eligible) return
    if (object instanceof InstancedMesh) {
      object.computeBoundingSphere()
      InstancedMesh.prototype.raycast.call(object, raycaster, hits)
    } else raycaster.intersectObject(object, false, hits)
  })
  return hits.length === 0
}
