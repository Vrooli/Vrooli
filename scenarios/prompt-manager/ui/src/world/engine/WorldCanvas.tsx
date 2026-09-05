import { Canvas, useThree } from '@react-three/fiber'
import { useEffect, type ReactNode } from 'react'
import { PCFSoftShadowMap } from 'three'
import type { CameraTuning, QualityProfile } from '../config'
import { snapshotWorld } from './capture'

interface WorldCanvasProps {
  profile: QualityProfile
  camera: CameraTuning
  children: ReactNode
  /** Passed to the underlying canvas element for BAS and the smoke tool. */
  testId?: string
  onCreated?: () => void
  capture?: boolean
}

/**
 * The single R3F Canvas. The renderer tone-maps with AgX (set by the post
 * chain) so the sky background and the scene agree; the profile owns dpr and
 * shadow settings; the camera tuning owns the lens.
 */
export function WorldCanvas({ profile, camera, children, testId = 'world-canvas', onCreated, capture = false }: WorldCanvasProps) {
  return (
    <Canvas
      shadows={profile.shadows ? { type: PCFSoftShadowMap } : false}
      dpr={[1, profile.dpr]}
      frameloop="demand"
      gl={{ antialias: false, powerPreference: 'high-performance', stencil: false, preserveDrawingBuffer: capture }}
      camera={{ fov: camera.fov, near: camera.near, far: camera.far, position: camera.initialPosition }}
      onCreated={onCreated}
      data-testid={testId}
      style={{ position: 'absolute', inset: 0 }}
    >
      {capture && <CaptureBridge />}
      {children}
    </Canvas>
  )
}

function CaptureBridge() {
  const get = useThree((state) => state.get)
  useEffect(() => {
    const bridge = { snapshot: (seconds: number, frames: number) => snapshotWorld(get(), seconds, frames) }
    window.__worldCapture = bridge
    return () => { if (window.__worldCapture === bridge) delete window.__worldCapture }
  }, [get])
  return null
}
