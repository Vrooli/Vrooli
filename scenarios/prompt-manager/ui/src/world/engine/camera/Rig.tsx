import { CameraControls, useProgress } from '@react-three/drei'
import { useThree } from '@react-three/fiber'
import { useFrame } from '@react-three/fiber'
import { useEffect, useImperativeHandle, useMemo, useRef, type Ref } from 'react'
import { Box3, MathUtils, Vector3 } from 'three'
import type CameraControlsImpl from 'camera-controls'
import type { CameraPose, CameraTuning, Scene } from '../../config'
import { updateDiagnostics } from '../diagnostics/store'
import type { WorldBounds } from '../types'
import { decideIntro } from './intro'
import { shouldFollow } from './follow'
import { applyInputMap } from './input'
import { frameDistance, orbitClamps, poseForBox, poseToPosition, type FocusedPose } from './pose'

export interface CameraRigHandle {
  /** Return to the scene's hero pose. */
  home(animate?: boolean): void
  /** Frame a box (an actor's rest bounds, a room) with the tuning padding. */
  focus(box: Box3, animate?: boolean): void
  /** Keep the look-at target on a moving point until cleared. */
  follow(target: (() => [number, number, number]) | null): void
  /** Move to an explicit pose. */
  setPose(pose: CameraPose, animate?: boolean): void
  /** Current look-at target in world space. */
  target(): [number, number, number]
  /** Enable or disable pointer interaction (the editor disables it while dragging). */
  setEnabled(enabled: boolean): void
}

interface CameraRigProps {
  ref?: Ref<CameraRigHandle>
  scene: Scene
  camera: CameraTuning
  bounds: WorldBounds
  /** Play the establishing-to-hero dolly on mount. */
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
export function CameraRig({ ref, scene, camera, bounds, intro, reducedMotion, onReady }: CameraRigProps) {
  const controls = useRef<CameraControlsImpl | null>(null)
  const aspect = useThree((s) => s.viewport.aspect)
  const { active } = useProgress()
  const introStarted = useRef(false)
  const followRef = useRef<(() => [number, number, number]) | null>(null)
  const lastFollowTarget = useRef<[number, number, number] | null>(null)
  const keys = useRef(new Set<string>())
  const clamps = useMemo(() => orbitClamps(camera, scene.camera.hero.azimuthDeg), [camera, scene.camera.hero.azimuthDeg])
  const animate = !reducedMotion

  useEffect(() => {
    if (controls.current) applyInputMap(controls.current, camera.input)
  }, [camera.input])

  // Poses are multiples of the distance at which the layout outline fills
  // camera.frameFill of the viewport from that pose, so a world that grows
  // with the team graph keeps the same framing.
  const applyPose = (pose: CameraPose | FocusedPose, transition: boolean) => {
    const c = controls.current
    if (!c) return
    const frame = 'frame' in pose ? pose.frame : { points: bounds.outline, center: bounds.footprint.center, height: camera.frameHeight, polarDeg: pose.polarDeg, azimuthDeg: pose.azimuthDeg, targetY: pose.targetY, fovDeg: camera.fov, aspect, minimumProjectionAspect: camera.minimumProjectionAspect, minimumFrameFill: camera.minimumFrameFill }
    const fit = Math.max(frameDistance(frame, 'fill' in pose ? pose.fill : camera.frameFill), Number.EPSILON)
    const { position, target } = poseToPosition(pose, frame.center, fit)
    void c.setLookAt(position[0], position[1], position[2], target[0], target[1], target[2], transition && animate)
  }

  useImperativeHandle(ref, () => ({
    home: (a = true) => applyPose(scene.camera.hero, a),
    setPose: (pose, a = true) => applyPose(pose, a),
    focus: (box, a = true) => {
      const c = controls.current
      if (!c) return
      // Explicit selection supersedes an establishing shot, including initial URL focus.
      introStarted.current = true
      updateDiagnostics({ introDone: true })
      applyPose(poseForBox(box, { polarDeg: MathUtils.radToDeg(c.polarAngle), azimuthDeg: MathUtils.radToDeg(c.azimuthAngle) }, camera, aspect, clamps), a)
    },
    follow: (target) => {
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
      if (controls.current) controls.current.enabled = enabled
    },
  }))

  // Boundary and starting pose.
  useEffect(() => {
    const c = controls.current
    if (!c) return
    const box = new Box3(
      new Vector3(bounds.center[0] - bounds.width / 2, -1, bounds.center[1] - bounds.depth / 2),
      new Vector3(bounds.center[0] + bounds.width / 2, camera.boundaryHeight, bounds.center[1] + bounds.depth / 2),
    )
    c.setBoundary(box)
    c.smoothTime = camera.smoothTime
    if (!decideIntro(intro, reducedMotion).play) {
      applyPose(scene.camera.hero, false)
      updateDiagnostics({ introDone: true })
      introStarted.current = true
    } else {
      applyPose(scene.camera.establishing, false)
      updateDiagnostics({ introDone: false })
    }
    // Only on mount / scene change.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [scene.id])

  // Intro dolly once assets have loaded.
  useEffect(() => {
    const c = controls.current
    if (!c || introStarted.current || active) return
    introStarted.current = true
    c.smoothTime = camera.introSeconds / 3
    const onRest = () => {
      c.removeEventListener('rest', onRest)
      c.smoothTime = followRef.current ? camera.followSmoothTime : camera.smoothTime
      updateDiagnostics({ introDone: true })
    }
    c.addEventListener('rest', onRest)
    applyPose(scene.camera.hero, true)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [active, scene.id])

  useEffect(() => {
    if (controls.current) onReady?.()
  }, [onReady, scene.id])

  // Follow mode and keyboard orbit / dolly, applied per frame.
  useFrame((_, dt) => {
    const c = controls.current
    if (!c) return
    const follow = followRef.current
    if (follow) {
      const next = follow()
      if (shouldFollow(lastFollowTarget.current, next, camera.followEpsilon)) {
        // Immediate translation preserves the user's orbit/dolly endpoints.
        // setTarget would keep camera position fixed and recompute those endpoints.
        void c.moveTo(next[0], next[1], next[2], false)
        lastFollowTarget.current = next
      }
    }
    const pressed = keys.current
    if (pressed.size === 0) return
    const orbit = MathUtils.degToRad(camera.keyOrbitDegPerSec) * dt
    if (pressed.has('ArrowLeft')) void c.rotate(-orbit, 0, true)
    if (pressed.has('ArrowRight')) void c.rotate(orbit, 0, true)
    if (pressed.has('ArrowUp')) void c.rotate(0, -orbit, true)
    if (pressed.has('ArrowDown')) void c.rotate(0, orbit, true)
    if (pressed.has('=') || pressed.has('+')) void c.dolly(camera.keyDollyPerSec * dt, true)
    if (pressed.has('-')) void c.dolly(-camera.keyDollyPerSec * dt, true)
  })

  useEffect(() => {
    const tracked = new Set(['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', '=', '+', '-'])
    const isTyping = (target: EventTarget | null) => target instanceof HTMLElement && (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.tagName === 'SELECT' || target.isContentEditable)
    const down = (event: KeyboardEvent) => {
      if (!tracked.has(event.key) || isTyping(event.target)) return
      keys.current.add(event.key)
      event.preventDefault()
    }
    const up = (event: KeyboardEvent) => keys.current.delete(event.key)
    const pressed = keys.current
    window.addEventListener('keydown', down)
    window.addEventListener('keyup', up)
    return () => {
      window.removeEventListener('keydown', down)
      window.removeEventListener('keyup', up)
      pressed.clear()
    }
  }, [])

  return (
    <CameraControls
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
  )
}
