import { useFrame, useThree } from '@react-three/fiber'
import { useEffect, useMemo, useRef, useState } from 'react'
import { BufferAttribute, BufferGeometry, LineBasicMaterial } from 'three'
import { createObstacleSweep } from '../engine/camera/obstacles'
import { nearPlaneRadius } from '../engine/camera/pose'

/** Committed structural camera boxes, using the guard's exact expansion logic. */
export function CameraCollisionOverlay({ epoch, onStatus }: { epoch: number; onStatus: (status: string) => void }) {
  const scene = useThree(state => state.scene)
  const camera = useThree(state => state.camera)
  const obstacles = useMemo(() => createObstacleSweep(scene), [scene])
  const sample = useRef({ epoch: -1, radius: -1, frames: 0 })
  const [positions, setPositions] = useState(() => new Float32Array())
  const geometry = useMemo(() => {
    const result = new BufferGeometry()
    result.setAttribute('position', new BufferAttribute(positions, 3))
    return result
  }, [positions])
  const material = useMemo(() => new LineBasicMaterial({ color: '#ff9800', depthTest: false, depthWrite: false, toneMapped: false }), [])
  useEffect(() => () => geometry.dispose(), [geometry])
  useEffect(() => () => material.dispose(), [material])
  useFrame(() => {
    if (!('fov' in camera)) return
    const radius = nearPlaneRadius(camera.near, camera.fov, camera.aspect, camera.zoom)
    if (sample.current.epoch !== epoch || Math.abs(sample.current.radius - radius) > 1e-8) sample.current = { epoch, radius, frames: 0 }
    // Allow committed instance transforms to reach the scene graph first.
    if (sample.current.frames++ !== 1) return
    const result = obstacles.outlines(radius)
    setPositions(result.positions)
    onStatus(`${result.count} structural boxes · ${radius.toFixed(3)} m near-plane clearance`)
  })
  return <lineSegments name="camera-collision-overlay" geometry={geometry} material={material} dispose={null} renderOrder={1001} raycast={() => {}} />
}
