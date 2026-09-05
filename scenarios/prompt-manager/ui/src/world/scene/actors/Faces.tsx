import { useFrame } from '@react-three/fiber'
import { useEffect, useMemo, useRef } from 'react'
import { BoxGeometry, Color, ConeGeometry, InstancedMesh, MeshStandardMaterial, Object3D, SphereGeometry } from 'three'
import type { ActorTuning } from '../../config'
import { useWorldStore } from '../WorldStoreContext'
import { bodyOffset, type BodyPose } from './pose'
import { POSE, POSE_STRIDE, readPose, usePoseBuffer } from './PoseBuffer'

/**
 * Eyes, mouths and ear nubs: three instanced draws that follow the body
 * pose. Blinks squash the eyes; ears and mouth style come from the actor's
 * deterministic variant.
 */
export function Faces({ tuning }: { tuning: ActorTuning }) {
  const look = tuning.look
  const store = useWorldStore()
  const ids = store.getState().actorOrder
  const capacity = Math.max(1, ids.length)
  const eyes = useRef<InstancedMesh | null>(null)
  const mouths = useRef<InstancedMesh | null>(null)
  const ears = useRef<InstancedMesh | null>(null)
  const dummy = useMemo(() => new Object3D(), [])
  const eyeGeometry = useMemo(() => new SphereGeometry(1, look.eyeWidthSegments, look.eyeHeightSegments), [look.eyeWidthSegments, look.eyeHeightSegments])
  const mouthGeometry = useMemo(() => new BoxGeometry(1, 1, 1), [])
  const earGeometry = useMemo(() => new ConeGeometry(1, 2, look.earSegments), [look.earSegments])
  const darkMaterial = useMemo(() => new MeshStandardMaterial({ color: look.eyeColor, roughness: look.eyeRoughness }), [look.eyeColor, look.eyeRoughness])
  const mouthMaterial = useMemo(() => new MeshStandardMaterial({ color: look.mouthColor, roughness: look.mouthRoughness }), [look.mouthColor, look.mouthRoughness])
  const earMaterial = useMemo(() => new MeshStandardMaterial({ roughness: look.earRoughness }), [look.earRoughness])
  useEffect(() => () => eyeGeometry.dispose(), [eyeGeometry])
  useEffect(() => () => mouthGeometry.dispose(), [mouthGeometry])
  useEffect(() => () => earGeometry.dispose(), [earGeometry])
  useEffect(() => () => darkMaterial.dispose(), [darkMaterial])
  useEffect(() => () => mouthMaterial.dispose(), [mouthMaterial])
  useEffect(() => () => earMaterial.dispose(), [earMaterial])
  const color = useMemo(() => new Color(), [])
  const poses = usePoseBuffer()
  const pose = useMemo<BodyPose>(() => ({ x: 0, y: 0, z: 0, facing: 0, scaleXZ: 0, scaleY: 0 }), [])

  useFrame(() => {
    const eyeMesh = eyes.current
    const mouthMesh = mouths.current
    const earMesh = ears.current
    if (!eyeMesh || !mouthMesh || !earMesh) return
    const state = store.getState()
    ids.forEach((id, i) => {
      const actor = state.actors[id]
      if (!actor) return
      readPose(poses, i, pose)
      if ((poses.data[i * POSE_STRIDE + POSE.visible] ?? 0) === 0) {
        dummy.scale.set(0, 0, 0)
        dummy.updateMatrix()
        eyeMesh.setMatrixAt(i * 2, dummy.matrix)
        eyeMesh.setMatrixAt(i * 2 + 1, dummy.matrix)
        earMesh.setMatrixAt(i * 2, dummy.matrix)
        earMesh.setMatrixAt(i * 2 + 1, dummy.matrix)
        mouthMesh.setMatrixAt(i, dummy.matrix)
        return
      }
      const r = pose.scaleXZ
      const blink = actor.anim.blinking ? look.blinkScaleY : 1
      for (let side = 0; side < 2; side += 1) {
        const sign = side === 0 ? -1 : 1
        const [x, y, z] = bodyOffset(pose, sign * look.eyeSpacing * r, look.eyeHeight * pose.scaleY, look.eyeForward * r)
        dummy.position.set(x, y, z)
        dummy.rotation.set(0, pose.facing, 0)
        dummy.scale.set(look.eyeRadius * r, look.eyeRadius * r * blink, look.eyeRadius * r)
        dummy.updateMatrix()
        eyeMesh.setMatrixAt(i * 2 + side, dummy.matrix)
        const earScale = actor.variant.ears === 0 ? 0 : look.earSize * r * (actor.variant.ears === 2 ? look.largeEarScale : 1)
        const [ex, ey, ez] = bodyOffset(pose, sign * look.earSpread * r, look.earHeight * pose.scaleY, 0)
        dummy.position.set(ex, ey, ez)
        dummy.rotation.set(0, pose.facing, sign * -look.earTiltRad)
        dummy.scale.set(earScale, earScale, earScale)
        dummy.updateMatrix()
        earMesh.setMatrixAt(i * 2 + side, dummy.matrix)
        color.set(actor.colors.head)
        earMesh.setColorAt(i * 2 + side, color)
      }
      const [mx, my, mz] = bodyOffset(pose, 0, (look.eyeHeight - look.mouthDrop) * pose.scaleY, look.mouthForward * r)
      dummy.position.set(mx, my, mz)
      dummy.rotation.set(0, pose.facing, 0)
      const mouthWidth = look.mouthWidth * r * (look.mouthVariantScales[actor.variant.mouth] ?? 1)
      const mouthHeight = look.mouthHeight * r * (actor.anim.emote ? look.emoteMouthScale : 1)
      dummy.scale.set(mouthWidth, mouthHeight, mouthHeight)
      dummy.updateMatrix()
      mouthMesh.setMatrixAt(i, dummy.matrix)
    })
    eyeMesh.instanceMatrix.needsUpdate = true
    mouthMesh.instanceMatrix.needsUpdate = true
    earMesh.instanceMatrix.needsUpdate = true
    if (earMesh.instanceColor) earMesh.instanceColor.needsUpdate = true
  })

  return (
    <>
      <instancedMesh ref={eyes} args={[eyeGeometry, darkMaterial, capacity * 2]} frustumCulled={false} />
      <instancedMesh ref={mouths} args={[mouthGeometry, mouthMaterial, capacity]} frustumCulled={false} />
      <instancedMesh ref={ears} args={[earGeometry, earMaterial, capacity * 2]} frustumCulled={false} />
    </>
  )
}
