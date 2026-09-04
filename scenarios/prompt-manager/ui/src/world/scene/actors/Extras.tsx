import { useFrame } from '@react-three/fiber'
import { useMemo, useRef } from 'react'
import { BoxGeometry, Color, InstancedMesh, MeshBasicMaterial, MeshStandardMaterial, Object3D, PlaneGeometry, SphereGeometry, TorusGeometry } from 'three'
import type { EmoteKind } from '../../sim'
import { equipmentTier } from '../../sim'
import { useWorldStore } from '../WorldStoreContext'
import { bodyOffset, type BodyPose } from './pose'
import { POSE, POSE_STRIDE, readPose, usePoseBuffer } from './PoseBuffer'

/** Equipment tiers: none, paper, folder, briefcase, backpack, as relative sizes. */
const TIER_SIZE = [0, 0.45, 0.65, 0.85, 1.1]
const TIER_COLOR = ['#000000', '#f5f0e6', '#e0b35a', '#6b4a2b', '#3b6ea5']
const MARK_FAILED = new Color('#ff3b3b').multiplyScalar(3)
const MARK_GATHERED = new Color('#ffb020').multiplyScalar(2.2)
const MARK_WORKING = new Color('#59d0ff').multiplyScalar(2.6)
const MARK_OFF = new Color('#000000')
const EMOTE_COLOR: Record<EmoteKind, Color> = {
  start: new Color('#59d0ff').multiplyScalar(2),
  done: new Color('#5ce27a').multiplyScalar(2),
  fail: new Color('#ff3b3b').multiplyScalar(2.5),
  message: new Color('#ffffff').multiplyScalar(1.8),
  gather: new Color('#ffb020').multiplyScalar(2),
}
const SPIN_RATE = 2.4

/**
 * Equipment (by skill tier), status marks (failed lamp, working spinner,
 * gathered marker) and emote bursts: three instanced draws driven by sim
 * state. Marks and emotes bypass tone mapping so bloom picks them up.
 */
export function ActorExtras() {
  const store = useWorldStore()
  const ids = store.getState().actorOrder
  const capacity = Math.max(1, ids.length)
  const gear = useRef<InstancedMesh | null>(null)
  const rings = useRef<InstancedMesh | null>(null)
  const marks = useRef<InstancedMesh | null>(null)
  const emotes = useRef<InstancedMesh | null>(null)
  const dummy = useMemo(() => new Object3D(), [])
  const color = useMemo(() => new Color(), [])
  const gearGeometry = useMemo(() => new BoxGeometry(1, 1.2, 0.5), [])
  const ringGeometry = useMemo(() => new TorusGeometry(1, 0.18, 8, 24), [])
  const markGeometry = useMemo(() => new SphereGeometry(1, 10, 10), [])
  const emoteGeometry = useMemo(() => new PlaneGeometry(1, 1), [])
  const gearMaterial = useMemo(() => new MeshStandardMaterial({ roughness: 0.8 }), [])
  const glowMaterial = useMemo(() => new MeshBasicMaterial({ toneMapped: false }), [])
  const emoteMaterial = useMemo(() => new MeshBasicMaterial({ toneMapped: false, transparent: true, opacity: 0.9, depthWrite: false }), [])
  const poses = usePoseBuffer()
  const pose = useMemo<BodyPose>(() => ({ x: 0, y: 0, z: 0, facing: 0, scaleXZ: 0, scaleY: 0 }), [])

  useFrame((frame) => {
    const gearMesh = gear.current
    const ringMesh = rings.current
    const markMesh = marks.current
    const emoteMesh = emotes.current
    if (!gearMesh || !ringMesh || !markMesh || !emoteMesh) return
    const state = store.getState()
    const t = store.tuning().actor
    const look = t.look
    const spin = frame.clock.elapsedTime * SPIN_RATE
    ids.forEach((id, i) => {
      const actor = state.actors[id]
      if (!actor) return
      readPose(poses, i, pose)
      if ((poses.data[i * POSE_STRIDE + POSE.visible] ?? 0) === 0) {
        dummy.scale.set(0, 0, 0)
        dummy.updateMatrix()
        gearMesh.setMatrixAt(i, dummy.matrix)
        ringMesh.setMatrixAt(i, dummy.matrix)
        markMesh.setMatrixAt(i, dummy.matrix)
        emoteMesh.setMatrixAt(i, dummy.matrix)
        return
      }
      const r = pose.scaleXZ
      // equipment
      const tier = equipmentTier(actor.skillCount, t.equipmentTiers)
      const size = (TIER_SIZE[tier] ?? 0) * look.equipmentScale * r
      const [gx, gy, gz] = bodyOffset(pose, 0, look.equipmentHeight * pose.scaleY, -look.equipmentBack * r)
      dummy.position.set(gx, gy, gz)
      dummy.rotation.set(0, pose.facing, 0)
      dummy.scale.set(size, size, size)
      dummy.updateMatrix()
      gearMesh.setMatrixAt(i, dummy.matrix)
      color.set(TIER_COLOR[tier] ?? TIER_COLOR[0] ?? '#000000')
      gearMesh.setColorAt(i, color)
      // working ring
      const working = actor.state === 'working'
      dummy.position.set(pose.x, look.markerHeight, pose.z)
      dummy.rotation.set(Math.PI / 2, 0, spin)
      const ringScale = working ? look.markerRadius : 0
      dummy.scale.set(ringScale, ringScale, ringScale)
      dummy.updateMatrix()
      ringMesh.setMatrixAt(i, dummy.matrix)
      ringMesh.setColorAt(i, working ? MARK_WORKING : MARK_OFF)
      // failed / gathered marker
      const failed = actor.state === 'failed'
      const gathered = actor.state === 'gathered' || actor.state === 'walkingToTable'
      const markScale = failed || gathered ? look.markerRadius * 0.6 : 0
      dummy.position.set(pose.x, look.markerHeight, pose.z)
      dummy.rotation.set(0, 0, 0)
      dummy.scale.set(markScale, markScale, markScale)
      dummy.updateMatrix()
      markMesh.setMatrixAt(i, dummy.matrix)
      markMesh.setColorAt(i, failed ? MARK_FAILED : gathered ? MARK_GATHERED : MARK_OFF)
      // emote
      const emote = actor.anim.emote
      if (emote) {
        const progress = 1 - emote.remaining / t.emoteSeconds
        dummy.position.set(pose.x, look.emoteHeight + look.emoteRise * progress, pose.z)
        dummy.quaternion.copy(frame.camera.quaternion)
        const s = look.emoteSize * (1 - progress * 0.5)
        dummy.scale.set(s, s, s)
        emoteMesh.setColorAt(i, EMOTE_COLOR[emote.kind])
      } else {
        dummy.position.set(pose.x, 0, pose.z)
        dummy.scale.set(0, 0, 0)
      }
      dummy.updateMatrix()
      emoteMesh.setMatrixAt(i, dummy.matrix)
    })
    for (const mesh of [gearMesh, ringMesh, markMesh, emoteMesh]) {
      mesh.instanceMatrix.needsUpdate = true
      if (mesh.instanceColor) mesh.instanceColor.needsUpdate = true
    }
  })

  return (
    <>
      <instancedMesh ref={gear} args={[gearGeometry, gearMaterial, capacity]} frustumCulled={false} />
      <instancedMesh ref={rings} args={[ringGeometry, glowMaterial, capacity]} frustumCulled={false} />
      <instancedMesh ref={marks} args={[markGeometry, glowMaterial, capacity]} frustumCulled={false} />
      <instancedMesh ref={emotes} args={[emoteGeometry, emoteMaterial, capacity]} frustumCulled={false} />
    </>
  )
}
