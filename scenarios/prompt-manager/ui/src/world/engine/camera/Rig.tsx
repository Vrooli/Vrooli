import { CameraControls, useProgress } from '@react-three/drei'
import { useThree } from '@react-three/fiber'
import { useFrame } from '@react-three/fiber'
import { useCallback, useEffect, useImperativeHandle, useMemo, useRef, useState, type Ref } from 'react'
import { Box3, MathUtils, Vector3, type Group, type Mesh } from 'three'
import type CameraControlsImpl from 'camera-controls'
import type { CameraPose, CameraTuning, Scene } from '../../config'
import { updateDiagnostics } from '../diagnostics/store'
import type { WorldBounds } from '../types'
import { bindExploreInput, DEFAULT_NAVIGATION, type NavigationCommand, type NavigationMode, type NavigationPreferences, type NavigationPreset, type NavigationState, type NavigationTool } from './navigation'
import { NAV_MOTION, NAV_VISUALS, WALK, type CameraMemory, type SavedWalkingView } from '../../config/navigation'
import { useWalkingInput } from './hooks/useWalkingInput'
import { NavigationMarkers } from './NavigationMarkers'
import { Walker, type WalkSurface } from './walking'
import { CameraController } from './controller'
import { decideIntro } from './intro'
import { shouldFollow } from './follow'
import { createObstacleSweep } from './obstacles'
import { PoseRoute } from './poseRoute'
import { bindNavigationKeys, focusVisibility, createZoomSurfaceQuery, navigationDelta, WorldCameraControls } from './input'
import { cameraRange, frameDistance, freezeCamera, orbitClamps, visiblePoseForBox, poseToPosition, type FocusedPose } from './pose'

export interface CameraRigHandle {
  /** Stable across handle refreshes, unique to this mounted rig. */
  readonly instance: symbol
  visitor(): { position: [number, number]; yaw: number } | null
  setMode(mode: NavigationMode): void
  command(command: NavigationCommand): void
  preset(preset: NavigationPreset): void
  lockLook(): void
  /** Return to the scene's hero pose. */
  home(animate?: boolean): void
  /** Frame a box (an actor's rest bounds, a room) with the tuning padding. */
  focus(box: Box3, animate?: boolean): void
  /** Keep the look-at target on a moving point until cleared. */
  follow(target: (() => [number, number, number] | null) | null): void
  /** Move to an explicit pose. */
  setPose(pose: CameraPose, animate?: boolean): void
  /** Current look-at target in world space. */
  target(): [number, number, number]
  /** Enable or disable pointer interaction (the editor disables it while dragging). */
  setEnabled(enabled: boolean): void
}

type AbsolutePose = { position: readonly [number, number, number]; target: readonly [number, number, number] }

interface CameraRigProps {
  memory?: CameraMemory
  navigation?: NavigationPreferences
  tool?: NavigationTool
  walkSurface?: WalkSurface
  onVisitorMove?: (visitor: { position: [number, number]; yaw: number } | null) => void
  onNavigationState?: (state: NavigationState) => void
  onFollowDetached?: () => void
  initialPose?: AbsolutePose
  groundCeiling?: WorldCameraControls['groundCeiling']
  epoch: number
  ref?: Ref<CameraRigHandle>
  scene: Scene
  camera: CameraTuning
  bounds: WorldBounds
  /** Play the establishing-to-hero dolly on mount. */
  initialPresentation?: boolean
  intro: boolean
  /** Reduced motion skips every transition. */
  reducedMotion: boolean
  /** Apply the current selection after controls have mounted and the start pose exists. */
  onReady?: () => void
}

/**
 * drei CameraControls configured as a diorama camera: clamped polar, azimuth
 * and distance, a boundary box over the slab, footprint-based framing, an
 * eased intro dolly and
 * imperative home / focus / setPose for the HUD and the editor.
 */
export function CameraRig({ ref, epoch, scene, camera, bounds, intro, reducedMotion, initialPresentation = true, initialPose, memory, onReady, groundCeiling, navigation = DEFAULT_NAVIGATION, tool = 'orbit', walkSurface, onVisitorMove, onNavigationState, onFollowDetached }: CameraRigProps) {
  const instance = useMemo(() => Symbol('camera-rig'), [])
  const controls = useRef<CameraControlsImpl | null>(null)
  const poseRoute = useRef<PoseRoute | null>(null)
  const walker = useRef<Walker | null>(null)
  const walkSurfaceRef = useRef(walkSurface)
  walkSurfaceRef.current = walkSurface
  const walkSweep = useRef<ReturnType<typeof createObstacleSweep> | null>(null)
  const savedPolar = useRef({ min: 0, max: Math.PI })
  const [mode, setModeState] = useState<NavigationMode>('explore')
  const modeRef = useRef<NavigationMode>('explore')
  const savedExplore = useRef<(AbsolutePose & { zoom?: number }) | null>(null)
  const viewReady = useRef(false)
  const avatar = useRef<Group>(null)
  const pivot = useRef<Mesh>(null)
  const locked = useRef(false)
  const message = useRef('')
  const reportAt = useRef(0)
  const stateCallback = useRef(onNavigationState)
  stateCallback.current = onNavigationState
  const canvas = useThree((s) => s.gl.domElement)
  const invalidate = useThree((s) => s.invalidate)
  const renderedScene = useThree((s) => s.scene)
  const ownership = useMemo(() => new CameraController(
    () => { if (controls.current) freezeCamera(controls.current) },
    (cameraOwnership, cameraHistory) => updateDiagnostics({ cameraOwnership, cameraHistory }),
  ), [])
  const aspect = useThree((s) => s.viewport.aspect)
  const { active } = useProgress()
  const introStarted = useRef(false)
  const followRef = useRef<(() => [number, number, number] | null) | null>(null)
  const lastFollowTarget = useRef<[number, number, number] | null>(null)
  const keys = useRef(new Set<string>())
  const committedEpoch = useRef(epoch)
  const previousReducedMotion = useRef(reducedMotion)
  const framingFactor = Math.max(scene.camera.hero.distanceFactor, scene.camera.establishing.distanceFactor)
  const clamps = useMemo(() => orbitClamps(camera, scene.camera.hero.azimuthDeg, aspect, bounds, framingFactor), [camera, scene.camera.hero.azimuthDeg, aspect, bounds, framingFactor])
  const far = cameraRange(camera, aspect, bounds, framingFactor).far
  const animate = !reducedMotion

  useEffect(() => {
    canvas.tabIndex = 0
    canvas.classList.add('focus-visible:outline', 'focus-visible:outline-2', 'focus-visible:outline-sky-500')
    canvas.setAttribute('aria-label', '3D world navigation')
    return () => { canvas.classList.remove('focus-visible:outline', 'focus-visible:outline-2', 'focus-visible:outline-sky-500'); canvas.removeAttribute('tabindex'); canvas.removeAttribute('aria-label') }
  }, [canvas])
  useEffect(() => {
    if (controls.current) (controls.current as WorldCameraControls).groundCeiling = groundCeiling
  }, [groundCeiling])
  useEffect(() => {
    const c = controls.current as WorldCameraControls | null
    if (!c) return
    c.zoomSurfaceTravel = createZoomSurfaceQuery(renderedScene, c.camera)
    const obstacles = createObstacleSweep(renderedScene, cameraObstacleGuard => updateDiagnostics({ cameraObstacleGuard }))
    walkSweep.current = createObstacleSweep(renderedScene, undefined, true)
    c.obstacleSweep = obstacles
    c.obstacleRecovery = obstacles.recover
    c.obstacleCeiling = obstacles.ceiling
    c.obstacleBounds = obstacles.bounds
    c.onZoomGuard = cameraZoomGuard => updateDiagnostics({ cameraZoomGuard })
    return () => { c.zoomSurfaceTravel = undefined; c.onZoomGuard = undefined; c.obstacleSweep = undefined; c.obstacleRecovery = undefined; c.obstacleCeiling = undefined; c.obstacleBounds = undefined }
  }, [renderedScene])
  useEffect(() => {
    const lens = controls.current?.camera
    if (!lens) return
    const fovChanged = 'fov' in lens && lens.fov !== camera.fov
    if (!fovChanged && lens.near === camera.near && lens.far === far) return
    if ('fov' in lens) lens.fov = camera.fov
    lens.near = camera.near
    lens.far = far
    lens.updateProjectionMatrix()
  }, [camera.fov, camera.near, far])

  // Poses are multiples of the distance at which the layout outline fills
  // camera.frameFill of the viewport from that pose, so a world that grows
  // with the team graph keeps the same framing.
  const resolvePose = (pose: CameraPose | FocusedPose | AbsolutePose) => {
    if ('position' in pose) return pose
    const frame = 'frame' in pose ? pose.frame : { points: bounds.outline, center: bounds.footprint.center, height: camera.frameHeight, polarDeg: pose.polarDeg, azimuthDeg: pose.azimuthDeg, targetY: pose.targetY, fovDeg: camera.fov, aspect, minimumProjectionAspect: camera.minimumProjectionAspect, minimumFrameFill: camera.minimumFrameFill }
    const fit = Math.max(frameDistance(frame, 'fill' in pose ? pose.fill : camera.frameFill), Number.EPSILON)
    return poseToPosition(pose, frame.center, fit)
  }
  const applyPose = (pose: CameraPose | FocusedPose, transition: boolean) => {
    const c = controls.current
    if (!c) return
    const { position, target } = resolvePose(pose)
    return c.setLookAt(position[0], position[1], position[2], target[0], target[1], target[2], transition && animate)
  }

  const navigate = useCallback(() => {
    if (!controls.current?.enabled) return
    ownership.navigate('navigation')
    introStarted.current = true
    controls.current.smoothTime = followRef.current ? camera.followSmoothTime : camera.smoothTime
    updateDiagnostics({ introDone: true })
  }, [ownership, camera.followSmoothTime, camera.smoothTime])

  const remember = () => {
    const c = controls.current
    if (!memory || !c || !viewReady.current || poseRoute.current) return
    const body = walker.current
    const explore = body && savedExplore.current ? savedExplore.current : {
      position: c.getPosition(new Vector3(), false).toArray(), target: c.getTarget(new Vector3(), false).toArray(), zoom: c.camera.zoom,
    }
    memory.save({ version: 1, mode: modeRef.current,
      explore: { position: [...explore.position], target: [...explore.target], zoom: explore.zoom ?? 1 },
      body: body ? { position: body.position.toArray(), yaw: body.yaw, pitch: body.pitch, boom: body.boom } : undefined })
  }
  const rememberRef = useRef(remember)
  rememberRef.current = remember
  useEffect(() => {
    const save = () => { rememberRef.current(); memory?.flush() }
    window.addEventListener('pagehide', save)
    document.addEventListener('visibilitychange', save)
    return () => { save(); window.removeEventListener('pagehide', save); document.removeEventListener('visibilitychange', save) }
  }, [memory])

  const publishNavigation = () => {
    remember()
    const c = controls.current as WorldCameraControls | null
    const direction = c?.camera.getWorldDirection(new Vector3())
    const heading = direction ? Math.atan2(direction.x, -direction.z) : 0
    const state = { mode: modeRef.current, heading: Math.round(MathUtils.radToDeg(MathUtils.euclideanModulo(heading, Math.PI * 2))),
      blocked: walker.current?.blocked ?? c?.blocked ?? false, locked: locked.current, message: message.current }
    onVisitorMove?.(walker.current ? { position: [walker.current.position.x, walker.current.position.z], yaw: walker.current.yaw } : null)
    stateCallback.current?.(state)
    updateDiagnostics({ cameraNavigation: { ...state, position: c?.camera.position.toArray() ?? [0, 0, 0], target: c?.getTarget(new Vector3(), false).toArray() ?? [0, 0, 0],
      playerPosition: walker.current?.position.toArray() ?? null, lensZoom: c?.camera.zoom ?? 1,
      inputSequence: c?.inputSequence ?? 0, inputAction: c?.lastInputAction ?? null } })
  }
  const changeMode = (next: NavigationMode, restoredBody?: SavedWalkingView) => {
    const c = controls.current as WorldCameraControls | null
    if (!c || next === modeRef.current || ownership.read().owner === 'editing') return
    keys.current.clear()
    if (next === 'explore') {
      if (document.pointerLockElement === canvas) document.exitPointerLock()
      locked.current = false
      walker.current = null
      c.externalCamera = false
      c.enabled = true
      c.resetClearance()
      c.minDistance = clamps.minDistance
      c.maxDistance = clamps.maxDistance
      c.minPolarAngle = savedPolar.current.min
      c.maxPolarAngle = savedPolar.current.max
      c.camera.near = camera.near
      if ('fov' in c.camera) c.camera.fov = camera.fov
      c.camera.updateProjectionMatrix()
      void c.zoomTo(1, false)
      if (savedExplore.current) {
        const { position, target } = savedExplore.current
        void c.setLookAt(...position, ...target, false)
        void c.zoomTo(savedExplore.current.zoom ?? 1, false)
        c.update(0)
      }
    } else if (!walker.current) {
      if (!walkSurface) { message.current = 'Walking is unavailable until the world is ready.'; publishNavigation(); return }
      const body = new Walker((x, z, radius) => walkSurfaceRef.current?.(x, z, radius) ?? { height: 0, walkable: false },
        (from, to, radius, span) => walkSweep.current?.(from, to, radius, span) ?? 1,
        (position, radius, span) => walkSweep.current?.overlaps(position, radius, span) ?? false)
      const target = c.getTarget(new Vector3(), false)
      if (!(restoredBody ? body.restore(restoredBody) : body.spawn(target.x, target.z))) { message.current = 'No clear walking space nearby. Pan to an open area and try again.'; publishNavigation(); return }
      ownership.navigate('navigation')
      freezeCamera(c)
      followRef.current = null
      ownership.follow(false)
      savedExplore.current = { position: c.getPosition(new Vector3(), false).toArray(), target: target.toArray(), zoom: c.camera.zoom }
      savedPolar.current = { min: c.minPolarAngle, max: c.maxPolarAngle }
      const direction = c.camera.getWorldDirection(new Vector3())
      if (!restoredBody) { body.yaw = Math.atan2(direction.x, -direction.z); body.pitch = 0 }
      walker.current = body
      c.enabled = false
      c.externalCamera = true
      c.camera.near = WALK.near
      if ('fov' in c.camera) c.camera.fov = WALK.fov
      c.camera.zoom = 1
      c.camera.updateProjectionMatrix()
    }
    modeRef.current = next
    if (next === 'explore') ownership.navigate('navigation')
    else ownership.walking(next)
    canvas.focus({ preventScroll: true })
    setModeState(next)
    message.current = ''
    publishNavigation()
    invalidate()
  }
  const manualCommand = (command: NavigationCommand) => {
    const c = controls.current as WorldCameraControls | null
    if (!c || ownership.read().owner === 'editing') return
    if (command === 'stop') {
      keys.current.clear()
      ownership.navigate('navigation')
      followRef.current = null
      lastFollowTarget.current = null
      if (!walker.current) ownership.follow(false)
      onFollowDetached?.()
      if (!walker.current) freezeCamera(c)
      return
    }
    if (walker.current) {
      if (command === 'jump') walker.current.jump()
      else if (command === 'zoom-in' || command === 'zoom-out') walker.current.boom = Math.max(WALK.minBoom, Math.min(WALK.maxBoom, walker.current.boom + (command === 'zoom-in' ? -WALK.boomStep : WALK.boomStep)))
      else if (command.startsWith('orbit')) walker.current.look(command === 'orbit-left' ? -NAV_MOTION.lookButtonPixels : NAV_MOTION.lookButtonPixels, 0, 1, false)
      else walker.current.step(NAV_MOTION.walkButtonSeconds, command === 'up' ? 1 : command === 'down' ? -1 : 0, command === 'right' ? 1 : command === 'left' ? -1 : 0, false)
    } else {
      navigate()
      if (command === 'zoom-in' || command === 'zoom-out') c.dollyPixels(command === 'zoom-in' ? -NAV_MOTION.zoomButtonPixels : NAV_MOTION.zoomButtonPixels, 0, 0)
      else if (command.startsWith('orbit')) void c.rotate(command === 'orbit-left' ? -NAV_MOTION.orbitButtonRadians : NAV_MOTION.orbitButtonRadians, 0, false)
      else c.panPixels(command === 'left' ? -NAV_MOTION.panButtonPixels : command === 'right' ? NAV_MOTION.panButtonPixels : 0, command === 'up' ? -NAV_MOTION.panButtonPixels : command === 'down' ? NAV_MOTION.panButtonPixels : 0)
    }
    invalidate()
  }

  const commandPose = (pose: CameraPose | FocusedPose | AbsolutePose, transition: boolean, owner: 'overview' | 'intro' | 'focus') => {
    const c = controls.current
    if (!c) return
    if (modeRef.current !== 'explore') changeMode('explore')
    const command = ownership.begin(owner)
    if (command === null) return
    void c.zoomTo(1, false)
    void c.setFocalOffset(0, 0, 0, false)
    followRef.current = null
    lastFollowTarget.current = null
    introStarted.current = true
    c.smoothTime = owner === 'intro' ? camera.introSeconds / 3 : camera.smoothTime
    updateDiagnostics({ introDone: owner !== 'intro' })
    const complete = () => {
      if (!ownership.complete(command)) return
      c.smoothTime = followRef.current ? camera.followSmoothTime : camera.smoothTime
      updateDiagnostics({ introDone: true })
    }
    const { position, target } = resolvePose(pose)
    const route = new PoseRoute(c as WorldCameraControls, new Vector3(...position), new Vector3(...target),
      transition && animate ? owner === 'intro' ? camera.introSeconds : Math.max(0.15, camera.smoothTime * 3) : 0,
      complete, () => { ownership.navigate('navigation'); updateDiagnostics({ introDone: true }) },
      cameraPoseRoute => updateDiagnostics({ cameraPoseRoute: { ...cameraPoseRoute, owner } }))
    poseRoute.current = route
    ownership.onCancel(command, () => {
      route.cancel()
      if (poseRoute.current === route) poseRoute.current = null
    })
    if (!transition || !animate) route.finishImmediately()
  }

  useImperativeHandle(ref, () => ({
    instance,
    visitor: () => walker.current ? { position: [walker.current.position.x, walker.current.position.z], yaw: walker.current.yaw } : null,
    setMode: changeMode,
    command: manualCommand,
    preset: preset => {
      if (modeRef.current !== 'explore') changeMode('explore')
      const c = controls.current
      if (!c) return
      const target = c.getTarget(new Vector3(), false)
      const radius = c.distance
      const polar = preset === 'top' ? NAV_MOTION.topPolar : preset === 'front' ? Math.PI / 2 : Math.PI / 4
      const yaw = preset === 'isometric' ? Math.PI / 4 : 0
      c.minPolarAngle = Math.min(clamps.minPolar, polar)
      c.maxPolarAngle = Math.max(clamps.maxPolar, polar)
      commandPose({ position: [target.x + radius * Math.sin(polar) * Math.sin(yaw), target.y + radius * Math.cos(polar), target.z + radius * Math.sin(polar) * Math.cos(yaw)], target: target.toArray() }, true, 'overview')
    },
    lockLook: () => {
      if (!walker.current) return
      canvas.focus({ preventScroll: true })
      try {
        const requestLock = (canvas as unknown as { requestPointerLock?: () => Promise<void> | void }).requestPointerLock
        const request = requestLock?.call(canvas)
        if (!requestLock) { message.current = 'Mouse capture unavailable. Drag the world to look around.'; publishNavigation() }
        request?.catch(() => { message.current = 'Mouse capture was denied. Drag the world to look around.'; publishNavigation() })
      } catch { message.current = 'Mouse capture unavailable. Drag the world to look around.'; publishNavigation() }
    },
    home: (a = true) => commandPose(scene.camera.hero, a, 'overview'),
    setPose: (pose, a = true) => commandPose(pose, a, 'overview'),
    focus: (box, a = true) => {
      const c = controls.current
      if (!c) return
      // Explicit selection supersedes an establishing shot, including initial URL focus.
      commandPose(visiblePoseForBox(box, { polarDeg: MathUtils.radToDeg(c.polarAngle), azimuthDeg: MathUtils.radToDeg(c.azimuthAngle) }, camera, aspect, clamps, (eye, target) => focusVisibility(renderedScene, eye, target)), a, 'focus')
    },
    follow: (target) => {
      if (walker.current) { if (target) changeMode('explore'); else return }
      ownership.follow(target !== null)
      followRef.current = target
      // focus() already commands this initial target; a resting actor needs no updates.
      lastFollowTarget.current = target?.() ?? null
      if (controls.current) controls.current.smoothTime = target ? camera.followSmoothTime : camera.smoothTime
    },
    target: () => {
      const c = controls.current
      if (!c) return [bounds.center[0], 0, bounds.center[1]]
      const t = c.getTarget(new Vector3())
      return [t.x, t.y, t.z]
    },
    setEnabled: (enabled) => {
      if (!enabled && walker.current) changeMode('explore')
      keys.current.clear()
      ownership.edit(!enabled)
      introStarted.current = true
      updateDiagnostics({ introDone: true })
      if (controls.current) controls.current.enabled = enabled
    },
  }))

  // Constraint updates must not reset an operator's pose.
  useEffect(() => {
    const c = controls.current
    if (!c) return
    const box = new Box3(
      new Vector3(bounds.center[0] - bounds.width / 2, -1, bounds.center[1] - bounds.depth / 2),
      new Vector3(bounds.center[0] + bounds.width / 2, camera.boundaryHeight, bounds.center[1] + bounds.depth / 2),
    )
    c.setBoundary(box)
  }, [bounds.center, bounds.width, bounds.depth, camera.boundaryHeight])

  // Starting pose is initialized for this committed rig, independently of bounds updates.
  useEffect(() => {
    const c = controls.current
    if (!c) return
    c.smoothTime = camera.smoothTime
    const remembered = initialPose ? null : memory?.read()
    if (remembered) {
      void c.setLookAt(...remembered.explore.position, ...remembered.explore.target, false)
      void c.zoomTo(remembered.explore.zoom, false)
      c.update(0)
      introStarted.current = true
      updateDiagnostics({ introDone: true })
      if (remembered.mode !== 'explore') changeMode(remembered.mode, remembered.body)
    } else if (initialPose && initialPresentation) {
      void applyPose(scene.camera.establishing, false)
      commandPose(initialPose, !reducedMotion, 'intro')
    } else if (!decideIntro(intro, reducedMotion, initialPresentation).play) {
      void applyPose(scene.camera.hero, false)
      updateDiagnostics({ introDone: true })
      introStarted.current = true
    } else {
      introStarted.current = false
      void applyPose(scene.camera.establishing, false)
      updateDiagnostics({ introDone: false })
    }
    viewReady.current = true
    // Only on mount / scene change.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [scene.id])

  // Intro dolly once assets have loaded.
  useEffect(() => {
    const c = controls.current
    if (!c || introStarted.current || active) return
    commandPose(scene.camera.hero, true, 'intro')
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [active, scene.id])

  useEffect(() => {
    if (controls.current) onReady?.()
  }, [onReady, scene.id])

  useEffect(() => {
    if (committedEpoch.current === epoch) return
    committedEpoch.current = epoch
    // The presenter resets readiness for every commit. Keep the current pose and
    // follow subscription, but retire any framing command from the previous world.
    ownership.navigate('generation')
    if (walker.current) {
      const body = walker.current
      const safe = body.spawn(body.position.x, body.position.z)
      if (!safe) changeMode('explore')
      if (modeRef.current !== 'explore') ownership.walking(modeRef.current)
      message.current = safe ? 'World updated; walking position checked for clearance.' : 'No clear walking space remains. Returned to Explore.'
      publishNavigation()
    }
    introStarted.current = true
    updateDiagnostics({ introDone: true })
    // This transaction runs once per committed world, using the current mode.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [epoch, ownership])

  useEffect(() => {
    const changed = previousReducedMotion.current !== reducedMotion
    previousReducedMotion.current = reducedMotion
    if (!changed || !reducedMotion) return
    // Preference changes apply to already scheduled motion, including input
    // damping, without issuing another pose or replaying a cancelled intro.
    ownership.navigate('reduced-motion')
    if (controls.current) freezeCamera(controls.current)
    keys.current.clear()
    introStarted.current = true
    updateDiagnostics({ introDone: true })
  }, [reducedMotion, ownership])

  // Write route poses before drei updates controls and applies clearance.
  useFrame((_, dt) => { poseRoute.current?.tick(dt) }, -2)

  // Follow mode and keyboard orbit / dolly, applied per frame.
  useFrame((_, dt) => {
    const c = controls.current
    if (!c || !c.enabled || poseRoute.current) return
    const follow = followRef.current
    if (follow) {
      const next = follow()
      if (!next) {
        followRef.current = null
        lastFollowTarget.current = null
        ownership.follow(false)
        ownership.navigate('target-removed')
        c.smoothTime = camera.smoothTime
      } else if (shouldFollow(lastFollowTarget.current, next, camera.followEpsilon)) {
        // Immediate translation preserves the user's orbit/dolly endpoints.
        // setTarget would keep camera position fixed and recompute those endpoints.
        void c.moveTo(next[0], next[1], next[2], false)
        lastFollowTarget.current = next
      }
    }
    const pressed = keys.current
    if (pressed.size === 0) return
    const delta = navigationDelta(dt)
    const orbit = MathUtils.degToRad(camera.keyOrbitDegPerSec) * delta * navigation.sensitivity
    if (pressed.has('ArrowLeft')) void c.rotate(-orbit, 0, false)
    if (pressed.has('ArrowRight')) void c.rotate(orbit, 0, false)
    if (pressed.has('ArrowUp')) void c.rotate(0, -orbit, false)
    if (pressed.has('ArrowDown')) void c.rotate(0, orbit, false)
    const move = camera.keyDollyPerSec * delta * navigation.sensitivity
    if (pressed.has('=') || pressed.has('+')) void c.dolly(move, false)
    if (pressed.has('-')) void c.dolly(-move, false)
    const has = (key: string) => pressed.has(key) || pressed.has(key.toUpperCase())
    if (has('a') || has('d') || has('w') || has('s')) void c.truck((Number(has('d')) - Number(has('a'))) * move, (Number(has('s')) - Number(has('w'))) * move, false)
    invalidate()
  })

  useEffect(() => bindNavigationKeys(keys.current, () => (controls.current?.enabled || walker.current !== null) && document.activeElement === canvas, navigate), [canvas, navigate])
  useEffect(() => {
    const c = controls.current as WorldCameraControls | null
    if (!c || mode !== 'explore') return
    return bindExploreInput(canvas, c, camera, navigation, tool, reducedMotion, navigate)
  }, [canvas, camera, navigation, tool, reducedMotion, navigate, mode])

  useWalkingInput({ mode, canvas, walker, keys, navigation, invalidate,
    onExit: () => changeMode('explore'),
    onLock: value => { locked.current = value; if (value) message.current = ''; publishNavigation() },
    onError: value => { message.current = value; publishNavigation() },
  })

  useFrame(() => {
    const c = controls.current as WorldCameraControls | null
    if (!c) return
    if (pivot.current) {
      pivot.current.position.copy(c.getTarget(new Vector3(), false))
      pivot.current.scale.setScalar(Math.max(NAV_VISUALS.pivotMinScale, c.distance * NAV_VISUALS.pivotDistanceScale))
      pivot.current.visible = modeRef.current === 'explore'
    }
    if (performance.now() - reportAt.current > NAV_MOTION.telemetryMs) { reportAt.current = performance.now(); publishNavigation() }
  }, NAV_MOTION.telemetryPriority)
  useFrame((_, dt) => {
    const body = walker.current
    const c = controls.current as WorldCameraControls | null
    if (!body || !c) return
    if (document.hidden) { keys.current.clear(); return }
    // Instanced furniture uploads after the commit effect. Recheck the body
    // against the actual rendered geometry, including when no key is held.
    if (!body.validPosition()) {
      if (!body.spawn(body.position.x, body.position.z)) {
        changeMode('explore')
        message.current = 'No clear walking space remains. Returned to Explore.'
        publishNavigation()
        return
      }
    }
    body.keyboard(dt, keys.current, MathUtils.degToRad(camera.keyOrbitDegPerSec))
    const view = body.view(modeRef.current === 'third-person')
    c.externalTarget.copy(view.target)
    c.camera.position.copy(view.eye)
    c.camera.lookAt(view.target)
    c.camera.updateMatrixWorld()
    if (avatar.current) { avatar.current.position.copy(body.position); avatar.current.rotation.y = -body.yaw; avatar.current.visible = modeRef.current === 'third-person' && view.eye.distanceTo(body.position.clone().add(new Vector3(0, WALK.eyeHeight, 0))) > WALK.avatarHideDistance }
    invalidate()
  }, NAV_MOTION.walkingPriority)

  useEffect(() => () => ownership.dispose(), [ownership])

  return (
    <>
    <CameraControls
      impl={WorldCameraControls}
      ref={controls}
      makeDefault
      minPolarAngle={clamps.minPolar}
      maxPolarAngle={clamps.maxPolar}
      minAzimuthAngle={clamps.minAzimuth}
      maxAzimuthAngle={clamps.maxAzimuth}
      minDistance={clamps.minDistance}
      maxDistance={clamps.maxDistance}
      smoothTime={followRef.current ? camera.followSmoothTime : camera.smoothTime}
      dollyToCursor={camera.dollyToCursor}
      truckSpeed={camera.truckSpeed}
      dollySpeed={camera.dollySpeed}
    />
    <NavigationMarkers pivot={pivot} avatar={avatar} mode={mode} />
    </>
  )
}
