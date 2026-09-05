import { useFrame } from '@react-three/fiber'
import { useEffect, useMemo, useRef } from 'react'
import { BoxGeometry, Color, InstancedMesh, MeshBasicMaterial, MeshStandardMaterial, Object3D, PlaneGeometry, SphereGeometry, TorusGeometry } from 'three'
import type { ActorTuning } from '../../config'
import { equipmentTier } from '../../sim'
import { useWorldStore } from '../WorldStoreContext'
import { bodyOffset, type BodyPose } from './pose'
import { POSE, POSE_STRIDE, readPose, usePoseBuffer } from './PoseBuffer'
import { accessoryColors } from './accessoryColors'

/**
 * Equipment (by skill tier), status marks (failed lamp, working spinner,
 * gathered marker) and emote bursts: three instanced draws driven by sim
 * state. Marks and emotes bypass tone mapping so bloom picks them up.
 */
export function ActorExtras({ tuning }: { tuning: ActorTuning }) {
  const settings = tuning.extras
  const colors = useMemo(() => accessoryColors(settings), [settings])
  const store = useWorldStore()
  const ids = store.getState().actorOrder
  const capacity = Math.max(1, ids.length)
  const gear = useRef<InstancedMesh | null>(null)
  const rings = useRef<InstancedMesh | null>(null)
  const marks = useRef<InstancedMesh | null>(null)
  const emotes = useRef<InstancedMesh | null>(null)
  const dummy = useMemo(() => new Object3D(), [])
  const color = useMemo(() => new Color(), [])
  const gearGeometry = useMemo(() => new BoxGeometry(1, settings.gearHeight, settings.gearDepth), [settings.gearHeight, settings.gearDepth])
  const ringGeometry = useMemo(() => new TorusGeometry(1, settings.ringThickness, settings.ringRadialSegments, settings.ringTubularSegments), [settings.ringThickness, settings.ringRadialSegments, settings.ringTubularSegments])
  const markGeometry = useMemo(() => new SphereGeometry(1, settings.markWidthSegments, settings.markHeightSegments), [settings.markWidthSegments, settings.markHeightSegments])
  const emoteGeometry = useMemo(() => new PlaneGeometry(1, 1), [])
  const gearMaterial = useMemo(() => new MeshStandardMaterial({ roughness: settings.gearRoughness }), [settings.gearRoughness])
  const glowMaterial = useMemo(() => new MeshBasicMaterial({ toneMapped: false }), [])
  const emoteMaterial = useMemo(() => new MeshBasicMaterial({ toneMapped: false, transparent: true, opacity: settings.emoteOpacity, depthWrite: false }), [settings.emoteOpacity])
  useEffect(() => () => gearGeometry.dispose(), [gearGeometry])
  useEffect(() => () => ringGeometry.dispose(), [ringGeometry])
  useEffect(() => () => markGeometry.dispose(), [markGeometry])
  useEffect(() => () => emoteGeometry.dispose(), [emoteGeometry])
  useEffect(() => () => gearMaterial.dispose(), [gearMaterial])
  useEffect(() => () => glowMaterial.dispose(), [glowMaterial])
  useEffect(() => () => emoteMaterial.dispose(), [emoteMaterial])
  const poses = usePoseBuffer()
  const pose = useMemo<BodyPose>(() => ({ x: 0, y: 0, z: 0, facing: 0, scaleXZ: 0, scaleY: 0 }), [])

  useFrame((frame) => {
    const gearMesh = gear.current
    const ringMesh = rings.current
    const markMesh = marks.current
    const emoteMesh = emotes.current
    if (!gearMesh || !ringMesh || !markMesh || !emoteMesh) return
    const state = store.getState()
    const t = tuning
    const look = t.look
    const spin = frame.clock.elapsedTime * settings.spinRate
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
      const size = (settings.tierSizes[tier] ?? 0) * look.equipmentScale * r
      const [gx, gy, gz] = bodyOffset(pose, 0, look.equipmentHeight * pose.scaleY, -look.equipmentBack * r)
      dummy.position.set(gx, gy, gz)
      dummy.rotation.set(0, pose.facing, 0)
      dummy.scale.set(size, size, size)
      dummy.updateMatrix()
      gearMesh.setMatrixAt(i, dummy.matrix)
      color.set(settings.tierColors[tier] ?? settings.tierColors[0] ?? settings.offColor)
      gearMesh.setColorAt(i, color)
      // working ring
      const working = actor.state === 'working'
      dummy.position.set(pose.x, look.markerHeight, pose.z)
      dummy.rotation.set(Math.PI / 2, 0, spin)
      const ringScale = working ? look.markerRadius : 0
      dummy.scale.set(ringScale, ringScale, ringScale)
      dummy.updateMatrix()
      ringMesh.setMatrixAt(i, dummy.matrix)
      ringMesh.setColorAt(i, working ? colors.working : colors.off)
      // failed / gathered marker
      const failed = actor.state === 'failed'
      const gathered = actor.state === 'gathered' || actor.state === 'walkingToTable'
      const markScale = failed || gathered ? look.markerRadius * settings.markScale : 0
      dummy.position.set(pose.x, look.markerHeight, pose.z)
      dummy.rotation.set(0, 0, 0)
      dummy.scale.set(markScale, markScale, markScale)
      dummy.updateMatrix()
      markMesh.setMatrixAt(i, dummy.matrix)
      markMesh.setColorAt(i, failed ? colors.failed : gathered ? colors.gathered : colors.off)
      // emote
      const emote = actor.anim.emote
      if (emote) {
        const progress = 1 - emote.remaining / t.emoteSeconds
        dummy.position.set(pose.x, look.emoteHeight + look.emoteRise * progress, pose.z)
        dummy.quaternion.copy(frame.camera.quaternion)
        const s = look.emoteSize * (1 - progress * settings.emoteShrink)
        dummy.scale.set(s, s, s)
        emoteMesh.setColorAt(i, colors.emotes[emote.kind])
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
