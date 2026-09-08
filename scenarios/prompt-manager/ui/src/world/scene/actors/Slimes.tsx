import { writeInstanceMatrix } from './pose'
import { useFrame } from '@react-three/fiber'
import { useEffect, useMemo, useRef, useState } from 'react'
import { Color, InstancedBufferAttribute, InstancedMesh, Matrix4, Object3D, SphereGeometry } from 'three'
import type { ActorTuning, QualityProfile } from '../../config'
import { createSlimeMaterial, updateSlimeMaterial } from '../../engine/materials/slime'
import { useWorldStore } from '../WorldStoreContext'
import { actorSeed, type BodyPose } from './pose'
import { POSE, POSE_STRIDE, readPose, usePoseBuffer } from './PoseBuffer'

const COLOR_STRIDE = 3

/**
 * Every slime body in one instanced draw. Per frame: read the sim state,
 * write instance matrices and the squash attribute; never setState.
 */
export function Slimes({ tuning, profile, onSelect, onHover }: { tuning: ActorTuning; profile: QualityProfile; onSelect?: (id: string | null) => void; onHover?: (id: string | null) => void }) {
  const store = useWorldStore()
  const ids = store.getState().actorOrder
  const capacity = Math.max(1, ids.length)
  const meshRef = useRef<InstancedMesh | null>(null)
  const dummy = useMemo(() => new Object3D(), [])
  const matrix = useMemo(() => new Matrix4(), [])
  const color = useMemo(() => new Color(), [])
  const poses = usePoseBuffer()
  const pose = useMemo<BodyPose>(() => ({ x: 0, y: 0, z: 0, facing: 0, scaleXZ: 0, scaleY: 0 }), [])

  const { geometry, bodyColors } = useMemo(() => {
    const geo = new SphereGeometry(1, tuning.mesh.widthSegments, tuning.mesh.heightSegments)
    const state = store.getState()
    const bodyColors = new Array<string | undefined>(capacity)
    const colors = new Float32Array(capacity * COLOR_STRIDE)
    const seeds = new Float32Array(capacity)
    const shifts = new Float32Array(capacity)
    const squash = new Float32Array(capacity).fill(1)
    ids.forEach((id, i) => {
      const actor = state.actors[id]
      if (!actor) return
      bodyColors[i] = actor.colors.body
      color.set(actor.colors.body)
      colors[i * COLOR_STRIDE] = color.r
      colors[i * COLOR_STRIDE + 1] = color.g
      colors[i * COLOR_STRIDE + 2] = color.b
      const seed = actorSeed(id)
      seeds[i] = seed
    })
    geo.setAttribute('aColor', new InstancedBufferAttribute(colors, COLOR_STRIDE))
    geo.setAttribute('aSeed', new InstancedBufferAttribute(seeds, 1))
    geo.setAttribute('aTimeShift', new InstancedBufferAttribute(shifts, 1))
    geo.setAttribute('aSquash', new InstancedBufferAttribute(squash, 1))
    return { geometry: geo, bodyColors }
  }, [store, ids, capacity, color, tuning.mesh.widthSegments, tuning.mesh.heightSegments])

  useEffect(() => {
    const seeds = geometry.getAttribute('aSeed')
    const shifts = geometry.getAttribute('aTimeShift') as InstancedBufferAttribute
    for (let i = 0; i < shifts.count; i++) shifts.setX(i, seeds.getX(i) * tuning.mesh.timeShiftSeconds)
    shifts.needsUpdate = true
  }, [geometry, tuning.mesh.timeShiftSeconds])

  const [material] = useState(() => createSlimeMaterial(tuning, profile.wobble))
  useEffect(() => updateSlimeMaterial(material, { material: tuning.material, wobbleIntensity: tuning.wobbleIntensity }, profile.wobble),
    [material, tuning.material, tuning.wobbleIntensity, profile.wobble])
  useEffect(() => () => geometry.dispose(), [geometry])
  useEffect(() => () => material.dispose(), [material])

  useFrame((frame) => {
    const mesh = meshRef.current
    if (!mesh) return
    const state = store.getState()
    const squash = geometry.getAttribute('aSquash') as InstancedBufferAttribute
    const colors = geometry.getAttribute('aColor') as InstancedBufferAttribute
    let colorChanged = false
    let squashChanged = false
    material.slime.uTime.value = frame.clock.elapsedTime
    for (const [i, id] of ids.entries()) {
      const actor = state.actors[id]
      if (!actor) continue
      if (bodyColors[i] !== actor.colors.body) {
        color.set(actor.colors.body)
        colors.setXYZ(i, color.r, color.g, color.b)
        bodyColors[i] = actor.colors.body
        colorChanged = true
      }
      readPose(poses, i, pose)
      dummy.position.set(pose.x, pose.y, pose.z)
      dummy.rotation.set(0, pose.facing, 0)
      dummy.scale.set(pose.scaleXZ, pose.scaleY, pose.scaleXZ)
      dummy.updateMatrix()
      matrix.copy(dummy.matrix)
      writeInstanceMatrix(mesh, i, matrix)
      const nextSquash = poses.data[i * POSE_STRIDE + POSE.squash] ?? 1
      if (squash.getX(i) !== nextSquash) {
        squash.setX(i, nextSquash)
        squash.addUpdateRange(i, 1)
        squashChanged = true
      }
    }
    if (colorChanged) colors.needsUpdate = true

    if (squashChanged) squash.needsUpdate = true
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
