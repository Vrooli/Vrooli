import { useFrame } from '@react-three/fiber'
import { useEffect, useMemo, useRef } from 'react'
import { CanvasTexture, InstancedMesh, MeshBasicMaterial, Object3D, PlaneGeometry } from 'three'
import { useWorldStore } from '../WorldStoreContext'
import { bodyPose } from './pose'

const TEXTURE_SIZE = 64
const LIFT = 0.035
const OPACITY = 0.38
/** Shadow disc radius as a multiple of the body radius. */
const SPREAD = 1.15
/** How much the disc shrinks at the top of a hop. */
const HOP_SHRINK = 0.35

/** A soft radial falloff drawn once; the alpha map every shadow disc shares. */
function makeFalloff(): CanvasTexture {
  const canvas = document.createElement('canvas')
  canvas.width = TEXTURE_SIZE
  canvas.height = TEXTURE_SIZE
  const context = canvas.getContext('2d')
  if (context) {
    const gradient = context.createRadialGradient(TEXTURE_SIZE / 2, TEXTURE_SIZE / 2, 0, TEXTURE_SIZE / 2, TEXTURE_SIZE / 2, TEXTURE_SIZE / 2)
    gradient.addColorStop(0, 'rgba(255,255,255,1)')
    gradient.addColorStop(0.55, 'rgba(255,255,255,0.55)')
    gradient.addColorStop(1, 'rgba(255,255,255,0)')
    context.fillStyle = gradient
    context.fillRect(0, 0, TEXTURE_SIZE, TEXTURE_SIZE)
  }
  return new CanvasTexture(canvas)
}

/**
 * One instanced draw of soft contact discs under every actor. Cheaper and
 * steadier than a re-rendered contact-shadow pass: no extra scene render,
 * follows hops (the disc shrinks as the body rises) and works in every profile.
 */
export function ActorShadows() {
  const store = useWorldStore()
  const ids = store.getState().actorOrder
  const capacity = Math.max(1, ids.length)
  const meshRef = useRef<InstancedMesh | null>(null)
  const dummy = useMemo(() => new Object3D(), [])
  const geometry = useMemo(() => new PlaneGeometry(2, 2).rotateX(-Math.PI / 2), [])
  const material = useMemo(() => {
    const alphaMap = makeFalloff()
    return new MeshBasicMaterial({ color: '#000000', alphaMap, transparent: true, opacity: OPACITY, depthWrite: false, toneMapped: false })
  }, [])
  useEffect(() => () => {
    geometry.dispose()
    material.alphaMap?.dispose()
    material.dispose()
  }, [geometry, material])

  useFrame(() => {
    const mesh = meshRef.current
    if (!mesh) return
    const state = store.getState()
    const t = store.tuning().actor
    ids.forEach((id, i) => {
      const actor = state.actors[id]
      if (!actor) return
      const pose = bodyPose(actor, t)
      const hop = t.hopHeight > 0 ? Math.max(0, pose.y - pose.scaleY) / t.hopHeight : 0
      const radius = pose.scaleXZ * SPREAD * (1 - hop * HOP_SHRINK)
      dummy.position.set(pose.x, LIFT, pose.z)
      dummy.scale.set(radius, 1, radius)
      dummy.updateMatrix()
      mesh.setMatrixAt(i, dummy.matrix)
    })
    mesh.instanceMatrix.needsUpdate = true
  })

  return <instancedMesh ref={meshRef} args={[geometry, material, capacity]} frustumCulled={false} renderOrder={-1} />
}
