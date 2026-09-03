import { useFrame, useThree } from '@react-three/fiber'
import { useProgress } from '@react-three/drei'
import { useEffect, useRef } from 'react'
import { Box3, InstancedMesh, Raycaster, Vector3 } from 'three'
import type { PeriodId, QualityProfile, QualityProfileId, SceneId } from '../../config'
import type { WorldBounds } from '../types'
import { frameStats, recordFrame, updateDiagnostics } from './store'

interface ProbeProps {
  scene: SceneId
  profileId: QualityProfileId
  profile: QualityProfile
  auto: boolean
  period: PeriodId
  /** Read the current camera look-at target from the rig. */
  getTarget: () => [number, number, number]
  bounds: WorldBounds
}

const PUBLISH_EVERY_FRAMES = 6
/** Must match the camera rig's framed height so the measured fill and the requested fill agree. */
const FRAME_HEIGHT = 2

/**
 * Reads renderer.info and frame timing every frame and publishes a snapshot
 * every few frames. Lives inside the Canvas; never sets React state.
 */
export function DiagnosticsProbe({ scene, profileId, profile, auto, period, getTarget, bounds }: ProbeProps) {
  const gl = useThree((s) => s.gl)
  const threeScene = useThree((s) => s.scene)
  const camera = useThree((s) => s.camera)
  const { active, progress } = useProgress()
  const frames = useRef(0)
  const raycaster = useRef(new Raycaster())
  const direction = useRef(new Vector3())
  const corner = useRef(new Vector3())

  // Outline fill as the camera actually sees it: project every framed point
  // and take the largest normalised extent. Independent of the rig's
  // closed-form framing, so the smoke tool cross-checks one against the other.
  const measureFill = () => {
    let fill = 0
    for (const [x, z] of bounds.outline) {
      for (const y of [0, FRAME_HEIGHT]) {
        corner.current.set(x, y, z).project(camera)
        fill = Math.max(fill, Math.abs(corner.current.x), Math.abs(corner.current.y))
      }
    }
    return fill
  }

  // The post chain renders several passes per frame; renderer.info must
  // accumulate across all of them and be reset by the probe, not per render.
  useEffect(() => {
    gl.info.autoReset = false
    return () => {
      gl.info.autoReset = true
    }
  }, [gl])

  useEffect(() => {
    const debugInfo = gl.getContext().getExtension('WEBGL_debug_renderer_info')
    const gpu = debugInfo ? String(gl.getContext().getParameter(debugInfo.UNMASKED_RENDERER_WEBGL)) : ''
    updateDiagnostics({ gpu })
  }, [gl])

  useEffect(() => {
    updateDiagnostics({ assetsLoaded: !active && progress >= 100 })
  }, [active, progress])

  useEffect(() => {
    updateDiagnostics({ scene, profile: profileId, auto, period, ao: profile.ao, bloom: profile.bloom })
  }, [scene, profileId, auto, period, profile.ao, profile.bloom])

  useFrame((state, delta) => {
    recordFrame(delta)
    frames.current += 1
    if (frames.current % PUBLISH_EVERY_FRAMES !== 0) {
      gl.info.reset()
      return
    }
    const { p50, p95 } = frameStats()
    camera.getWorldDirection(direction.current)
    raycaster.current.set(camera.position, direction.current)
    const hits = raycaster.current.intersectObjects(threeScene.children, true)
    const nearest = hits.find((hit) => hit.object.visible && hit.distance > 0)
    const box = new Box3()
    const sceneGraph = threeScene.children.map((child) => {
      box.makeEmpty()
      let instances = 0
      child.traverse((o) => {
        if (o instanceof InstancedMesh) instances += o.count
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
    updateDiagnostics({
      framesRendered: frames.current,
      drawCalls: gl.info.render.calls,
      triangles: gl.info.render.triangles,
      programs: gl.info.programs?.length ?? 0,
      geometries: gl.info.memory.geometries,
      textures: gl.info.memory.textures,
      frameMsP50: p50,
      frameMsP95: p95,
      dpr: state.viewport.dpr,
      toneMapping: toneMappingName(gl.toneMapping),
      cameraPosition: [camera.position.x, camera.position.y, camera.position.z],
      cameraTarget: getTarget(),
      nearestHit: nearest ? nearest.distance : -1,
      footprintFill: measureFill(),
      footprint: { width: bounds.footprint.width, depth: bounds.footprint.depth, center: [bounds.footprint.center[0], bounds.footprint.center[1]] },
      sceneGraph,
    })
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
