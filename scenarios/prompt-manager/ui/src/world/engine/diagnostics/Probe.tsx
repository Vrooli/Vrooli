import { addAfterEffect, addEffect, useFrame, useThree } from '@react-three/fiber'
import { useProgress } from '@react-three/drei'
import { useCallback, useEffect, useRef } from 'react'
import { Box3, InstancedMesh, Raycaster, Vector3, WebGLRenderTarget } from 'three'
import type { PeriodId, QualityProfile, QualityProfileId, QualityTuning, SceneId } from '../../config'
import type { WorldBounds } from '../types'
import { frameStats, readDiagnostics, recordFrame, updateDiagnostics } from './store'
import { GpuTimer } from './gpuTimer'
import { addPassDraws, beginPassDrawFrame, disposePassTimer, passDrawsFor, passTimerFor } from './passTimer'

interface ProbeProps {
  settings: QualityTuning['diagnostics']
  frameHeight: number
  scene: SceneId
  profileId: QualityProfileId
  profile: QualityProfile
  auto: boolean
  period: PeriodId
  /** Read the current camera look-at target from the rig. */
  getTarget: () => [number, number, number]
  bounds: WorldBounds
  measureEnabled: boolean
}

/**
 * Reads renderer.info and frame timing every frame and publishes a snapshot
 * every few frames. Lives inside the Canvas; never sets React state.
 */
export function DiagnosticsProbe({ settings, frameHeight, scene, profileId, profile, auto, period, getTarget, bounds, measureEnabled }: ProbeProps) {
  const gl = useThree((s) => s.gl)
  const threeScene = useThree((s) => s.scene)
  const camera = useThree((s) => s.camera)
  const { active, progress } = useProgress()
  const frames = useRef(0)
  const framesPerSecond = useRef(0)
  const frameWindow = useRef({ startedAt: performance.now(), frames: 0 })
  const raycaster = useRef(new Raycaster())
  const direction = useRef(new Vector3())
  const corner = useRef(new Vector3())
  const gpuTimer = useRef<GpuTimer | null>(null)

  useEffect(() => {
    updateDiagnostics({ minimumReadyFps: settings.minimumReadyFps })
  }, [settings.minimumReadyFps])

  // Outline fill as the camera actually sees it: project every framed point
  // and take the largest normalised extent. Independent of the rig's
  // closed-form framing, so the smoke tool cross-checks one against the other.
  const measureFill = useCallback(() => {
    let fill = 0
    for (const [x, z] of bounds.outline) {
      for (const y of [0, frameHeight]) {
        corner.current.set(x, y, z).project(camera)
        fill = Math.max(fill, Math.abs(corner.current.x), Math.abs(corner.current.y))
      }
    }
    return fill
  }, [bounds.outline, camera, frameHeight])

  const measure = useCallback((attributeGroups = true) => {
    camera.getWorldDirection(direction.current)
    raycaster.current.set(camera.position, direction.current)
    const hits = raycaster.current.intersectObjects(threeScene.children, true)
    const nearest = hits.find((hit) => hit.object.visible && hit.distance > 0)
    const box = new Box3()
    const sceneGraph = threeScene.children.map((child) => {
      box.makeEmpty()
      let instances = 0
      child.traverse((object) => {
        if (object instanceof InstancedMesh) instances += object.count
      })
      box.setFromObject(child)
      return {
        name: child.name || child.type,
        type: child.type,
        visible: child.visible,
        children: child.children.length,
        minY: box.isEmpty() ? NaN : Number(box.min.y.toFixed(2)),
        maxY: box.isEmpty() ? NaN : Number(box.max.y.toFixed(2)),
        instances,
      }
    })
    const groupCosts: Array<{ name: string; calls: number; triangles: number }> = []
    if (attributeGroups) {
      const originalTarget = gl.getRenderTarget()
      const originalAutoReset = gl.info.autoReset
      const originalBackground = threeScene.background
      const visibility = threeScene.children.map((child) => child.visible)
      const scratch = new WebGLRenderTarget(1, 1, { depthBuffer: false, stencilBuffer: false })
      try {
        gl.info.autoReset = false
        for (const child of threeScene.children) child.visible = false
        gl.setRenderTarget(scratch)
        // Charge the environment cube once, not once per isolated child
        // (including lights and empty groups that draw no geometry).
        gl.info.reset()
        gl.render(threeScene, camera)
        if (gl.info.render.calls > 0 || gl.info.render.triangles > 0) {
          groupCosts.push({ name: 'background', calls: gl.info.render.calls, triangles: gl.info.render.triangles })
        }
        threeScene.background = null
        threeScene.children.forEach((child, index) => {
          if (!visibility[index]) return
          child.visible = true
          gl.info.reset()
          gl.render(threeScene, camera)
          groupCosts.push({ name: child.name || child.type, calls: gl.info.render.calls, triangles: gl.info.render.triangles })
          child.visible = false
        })
        const passDraws = passDrawsFor(gl)
        if (passDraws.shadow.calls > 0 || passDraws.shadow.triangles > 0) groupCosts.push({ name: 'shadow-pass', ...passDraws.shadow })
        if (passDraws.post.calls > 0 || passDraws.post.triangles > 0) groupCosts.push({ name: 'post-pass', ...passDraws.post })
        // EffectComposer owns the frame after the scene render and may create
        // its internal passes after diagnostic effects mount. Renderer totals
        // are authoritative, so account for that pipeline remainder instead
        // of reporting known post-process work as "unattributed".
        const frame = readDiagnostics()
        const attributedCalls = groupCosts.reduce((sum, group) => sum + group.calls, 0)
        const attributedTriangles = groupCosts.reduce((sum, group) => sum + group.triangles, 0)
        const pipelineCalls = Math.max(0, frame.drawCalls - attributedCalls)
        const pipelineTriangles = Math.max(0, frame.triangles - attributedTriangles)
        if (pipelineCalls > 0 || pipelineTriangles > 0) {
          groupCosts.push({ name: 'compositor-pipeline', calls: pipelineCalls, triangles: pipelineTriangles })
        }
      } finally {
        threeScene.background = originalBackground
        threeScene.children.forEach((child, index) => { child.visible = visibility[index] ?? true })
        gl.setRenderTarget(originalTarget)
        gl.info.autoReset = originalAutoReset
        gl.info.reset()
        scratch.dispose()
      }
    }
    updateDiagnostics({
      nearestHit: nearest ? nearest.distance : -1,
      footprintFill: measureFill(),
      footprint: { width: bounds.footprint.width, depth: bounds.footprint.depth, center: [bounds.footprint.center[0], bounds.footprint.center[1]] },
      sceneGraph,
      ...(attributeGroups ? { groupCosts } : {}),
      ...(attributeGroups ? {
        drawCallsUnattributed: Math.max(0, readDiagnostics().drawCalls - groupCosts.reduce((sum, group) => sum + group.calls, 0)),
        trianglesUnattributed: Math.max(0, readDiagnostics().triangles - groupCosts.reduce((sum, group) => sum + group.triangles, 0)),
      } : {}),
    })
  }, [bounds, camera, gl, measureFill, threeScene])

  useEffect(() => {
    updateDiagnostics({ measure })
    return () => updateDiagnostics({ measure: () => undefined })
  }, [measure])

  // The post chain renders several passes per frame; renderer.info must
  // accumulate across all of them and be reset by the probe, not per render.
  useEffect(() => {
    gl.info.autoReset = false
    return () => {
      gl.info.autoReset = true
    }
  }, [gl])

  useEffect(() => {
    if (!measureEnabled) return
    const timer = passTimerFor(gl, gl.getContext() as WebGL2RenderingContext, settings)
    const shadowMap = gl.shadowMap
    const originalShadowRender = shadowMap.render.bind(shadowMap)
    shadowMap.render = function (...args: Parameters<typeof originalShadowRender>) {
      const beforeCalls = gl.info.render.calls
      const beforeTriangles = gl.info.render.triangles
      timer.begin('shadow')
      try {
        return originalShadowRender(...args)
      } finally {
        timer.end('shadow')
        addPassDraws(gl, 'shadow', gl.info.render.calls - beforeCalls, gl.info.render.triangles - beforeTriangles)
      }
    }
    const removeBefore = addEffect(() => {
      beginPassDrawFrame(gl)
      timer.beginFrame()
    })
    const removeAfter = addAfterEffect(() => timer.endFrame())
    return () => {
      removeBefore()
      removeAfter()
      shadowMap.render = originalShadowRender
      disposePassTimer(gl)
    }
  }, [gl, measureEnabled, settings])

  useEffect(() => {
    const timer = new GpuTimer(gl.getContext() as WebGL2RenderingContext, settings)
    gpuTimer.current = timer
    const removeBefore = addEffect(() => {
      timer.drain()
      timer.begin()
    })
    const removeAfter = addAfterEffect(() => {
      timer.end()
      timer.drain()
    })
    return () => {
      removeBefore()
      removeAfter()
      timer.dispose()
      gpuTimer.current = null
    }
  }, [gl, settings])

  useEffect(() => {
    const debugInfo = gl.getContext().getExtension('WEBGL_debug_renderer_info')
    const gpu = debugInfo ? String(gl.getContext().getParameter(debugInfo.UNMASKED_RENDERER_WEBGL)) : ''
    updateDiagnostics({ gpu })
  }, [gl])

  useEffect(() => {
    updateDiagnostics({ assetsLoaded: !active && progress >= 100 })
  }, [active, progress])

  useEffect(() => {
    updateDiagnostics({ scene, profile: profileId, auto, period, ao: profile.ao, bloom: profile.bloom, msaa: profile.msaa })
  }, [scene, profileId, auto, period, profile.ao, profile.bloom, profile.msaa])

  useFrame((state, delta) => {
    recordFrame(delta, settings.frameWindow)
    frames.current += 1
    frameWindow.current.frames += 1
    const now = performance.now()
    const elapsed = now - frameWindow.current.startedAt
    if (elapsed >= settings.fpsWindowMs) {
      framesPerSecond.current = Math.round((frameWindow.current.frames * 1000) / elapsed)
      frameWindow.current = { startedAt: now, frames: 0 }
    }
    if (frames.current % settings.publishEveryFrames !== 0) {
      gl.info.reset()
      return
    }
    const { p50, p95 } = frameStats()
    const gpu = gpuTimer.current?.stats()
    const pass = measureEnabled ? passTimerFor(gl, gl.getContext() as WebGL2RenderingContext).stats() : null
    updateDiagnostics({
      framesRendered: framesPerSecond.current,
      drawCalls: gl.info.render.calls,
      triangles: gl.info.render.triangles,
      programs: gl.info.programs?.length ?? 0,
      geometries: gl.info.memory.geometries,
      textures: gl.info.memory.textures,
      frameMsP50: p50,
      frameMsP95: p95,
      gpuMsP50: gpu?.p50 ?? 0,
      gpuMsP95: gpu?.p95 ?? 0,
      gpuSamples: gpu?.samples ?? 0,
      gpuTimerReason: gpu?.reason ?? 'timer not initialized',
      ...(pass && !pass.reason ? { passMs: { shadow: pass.shadow, main: pass.main, post: pass.post, total: pass.total } } : {}),
      dpr: state.viewport.dpr,
      toneMapping: toneMappingName(gl.toneMapping),
      cameraPosition: [camera.position.x, camera.position.y, camera.position.z],
      cameraTarget: getTarget(),
    })
    // Passive diagnostics must not add direct renders inside the GPU query.
    // Explicit callers use the exposed measure() function to request attribution.
    if (measureEnabled) measure(false)
    gl.info.reset()
  })

  return null
}

const TONE_MAPPING_NAMES: Record<number, string> = {
  0: 'none',
  1: 'linear',
  2: 'reinhard',
  3: 'cineon',
  4: 'aces',
  6: 'agx',
  7: 'neutral',
  5: 'custom',
}

function toneMappingName(value: number): string {
  return TONE_MAPPING_NAMES[value] ?? String(value)
}
