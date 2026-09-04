import { useFrame } from '@react-three/fiber'
import { useEffect, useMemo, useRef } from 'react'
import { Color, InstancedBufferAttribute, InstancedMesh, Matrix4, Object3D, SphereGeometry } from 'three'
import type { QualityProfile } from '../../config'
import { createSlimeMaterial, setSlimeWobble } from '../../engine/materials/slime'
import { useWorldStore } from '../WorldStoreContext'
import { actorSeed, type BodyPose } from './pose'
import { POSE, POSE_STRIDE, readPose, usePoseBuffer } from './PoseBuffer'

const BODY_WIDTH_SEGMENTS = 16
const BODY_HEIGHT_SEGMENTS = 10
const COLOR_STRIDE = 3

/**
 * Every slime body in one instanced draw. Per frame: read the sim state,
 * write instance matrices and the squash attribute; never setState.
 */
export function Slimes({ profile, onSelect, onHover }: { profile: QualityProfile; onSelect?: (id: string | null) => void; onHover?: (id: string | null) => void }) {
  const store = useWorldStore()
  const tuning = store.tuning().actor
  const ids = store.getState().actorOrder
  const capacity = Math.max(1, ids.length)
  const meshRef = useRef<InstancedMesh | null>(null)
  const dummy = useMemo(() => new Object3D(), [])
  const matrix = useMemo(() => new Matrix4(), [])
  const color = useMemo(() => new Color(), [])
  const poses = usePoseBuffer()
  const pose = useMemo<BodyPose>(() => ({ x: 0, y: 0, z: 0, facing: 0, scaleXZ: 0, scaleY: 0 }), [])

  const geometry = useMemo(() => {
    const geo = new SphereGeometry(1, BODY_WIDTH_SEGMENTS, BODY_HEIGHT_SEGMENTS)
    const state = store.getState()
    const colors = new Float32Array(capacity * COLOR_STRIDE)
    const seeds = new Float32Array(capacity)
    const shifts = new Float32Array(capacity)
    const squash = new Float32Array(capacity).fill(1)
    ids.forEach((id, i) => {
      const actor = state.actors[id]
      if (!actor) return
      color.set(actor.colors.body)
      colors[i * COLOR_STRIDE] = color.r
      colors[i * COLOR_STRIDE + 1] = color.g
      colors[i * COLOR_STRIDE + 2] = color.b
      const seed = actorSeed(id)
      seeds[i] = seed
      shifts[i] = seed * BODY_WIDTH_SEGMENTS
    })
    geo.setAttribute('aColor', new InstancedBufferAttribute(colors, COLOR_STRIDE))
    geo.setAttribute('aSeed', new InstancedBufferAttribute(seeds, 1))
    geo.setAttribute('aTimeShift', new InstancedBufferAttribute(shifts, 1))
    geo.setAttribute('aSquash', new InstancedBufferAttribute(squash, 1))
    return geo
  }, [store, ids, capacity, color])

  const material = useMemo(() => createSlimeMaterial(tuning, profile.wobble), [tuning, profile.wobble])
  useEffect(() => setSlimeWobble(material, tuning, profile.wobble), [material, tuning, profile.wobble])
  useEffect(() => () => {
    geometry.dispose()
    material.dispose()
  }, [geometry, material])

  useFrame((frame) => {
    const mesh = meshRef.current
    if (!mesh) return
    const state = store.getState()
    const squash = geometry.getAttribute('aSquash') as InstancedBufferAttribute
    material.slime.uTime.value = frame.clock.elapsedTime
    ids.forEach((id, i) => {
      const actor = state.actors[id]
      if (!actor) return
      readPose(poses, i, pose)
      dummy.position.set(pose.x, pose.y, pose.z)
      dummy.rotation.set(0, pose.facing, 0)
      dummy.scale.set(pose.scaleXZ, pose.scaleY, pose.scaleXZ)
      dummy.updateMatrix()
      matrix.copy(dummy.matrix)
      mesh.setMatrixAt(i, matrix)
      squash.setX(i, poses.data[i * POSE_STRIDE + POSE.squash] ?? 1)
    })
    mesh.instanceMatrix.needsUpdate = true
    squash.needsUpdate = true
  })

  return (
    <instancedMesh
      ref={meshRef}
      args={[geometry, material, capacity]}
      frustumCulled={false}
      receiveShadow
      onClick={(event) => {
        event.stopPropagation()
        onSelect?.(event.instanceId === undefined ? null : (ids[event.instanceId] ?? null))
      }}
      onPointerOver={(event) => onHover?.(event.instanceId === undefined ? null : (ids[event.instanceId] ?? null))}
      onPointerOut={() => onHover?.(null)}
    />
  )
}
