import { useFrame } from '@react-three/fiber'
import { useMemo, useRef } from 'react'
import { BoxGeometry, Color, ConeGeometry, InstancedMesh, MeshStandardMaterial, Object3D, SphereGeometry } from 'three'
import { useWorldStore } from '../WorldStoreContext'
import { bodyOffset, bodyPose } from './pose'

const EYE_COLOR = '#1b1b2a'
const MOUTH_COLOR = '#2a1b2a'
const EYE_SEGMENTS = 12

/**
 * Eyes, mouths and ear nubs: three instanced draws that follow the body
 * pose. Blinks squash the eyes; ears and mouth style come from the actor's
 * deterministic variant.
 */
export function Faces() {
  const store = useWorldStore()
  const ids = store.getState().actorOrder
  const capacity = Math.max(1, ids.length)
  const eyes = useRef<InstancedMesh | null>(null)
  const mouths = useRef<InstancedMesh | null>(null)
  const ears = useRef<InstancedMesh | null>(null)
  const dummy = useMemo(() => new Object3D(), [])
  const eyeGeometry = useMemo(() => new SphereGeometry(1, EYE_SEGMENTS, EYE_SEGMENTS), [])
  const mouthGeometry = useMemo(() => new BoxGeometry(1, 1, 1), [])
  const earGeometry = useMemo(() => new ConeGeometry(1, 2, 8), [])
  const darkMaterial = useMemo(() => new MeshStandardMaterial({ color: EYE_COLOR, roughness: 0.4 }), [])
  const mouthMaterial = useMemo(() => new MeshStandardMaterial({ color: MOUTH_COLOR, roughness: 0.6 }), [])
  const earMaterial = useMemo(() => new MeshStandardMaterial({ roughness: 0.7 }), [])
  const color = useMemo(() => new Color(), [])

  useFrame(() => {
    const eyeMesh = eyes.current
    const mouthMesh = mouths.current
    const earMesh = ears.current
    if (!eyeMesh || !mouthMesh || !earMesh) return
    const state = store.getState()
    const t = store.tuning().actor
    const look = t.look
    ids.forEach((id, i) => {
      const actor = state.actors[id]
      if (!actor) return
      const pose = bodyPose(actor, t)
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
        const earScale = actor.variant.ears === 0 ? 0 : look.earSize * r * (actor.variant.ears === 2 ? 1.5 : 1)
        const [ex, ey, ez] = bodyOffset(pose, sign * look.earSpread * r, look.earHeight * pose.scaleY, 0)
        dummy.position.set(ex, ey, ez)
        dummy.rotation.set(0, pose.facing, sign * -0.5)
        dummy.scale.set(earScale, earScale, earScale)
        dummy.updateMatrix()
        earMesh.setMatrixAt(i * 2 + side, dummy.matrix)
        color.set(actor.colors.head)
        earMesh.setColorAt(i * 2 + side, color)
      }
      const [mx, my, mz] = bodyOffset(pose, 0, (look.eyeHeight - look.mouthDrop) * pose.scaleY, look.mouthForward * r)
      dummy.position.set(mx, my, mz)
      dummy.rotation.set(0, pose.facing, 0)
      const mouthWidth = look.mouthWidth * r * (actor.variant.mouth === 2 ? 1.6 : actor.variant.mouth === 1 ? 1.2 : 1)
      const mouthHeight = look.mouthHeight * r * (actor.anim.emote ? 2 : 1)
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
      <instancedMesh ref={ears} args={[earGeometry, earMaterial, capacity * 2]} frustumCulled={false} castShadow />
    </>
  )
}
