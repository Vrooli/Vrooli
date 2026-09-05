import { useFrame } from '@react-three/fiber'
import { useEffect, useMemo, useRef } from 'react'
import { CanvasTexture, InstancedMesh, MeshBasicMaterial, Object3D, PlaneGeometry } from 'three'
import { useWorldStore } from '../WorldStoreContext'
import { heightAt } from '../../sim'
import type { BodyPose } from './pose'
import { readPose, usePoseBuffer } from './PoseBuffer'
import type { ActorTuning } from '../../config'

/** A soft radial falloff drawn once; the alpha map every shadow disc shares. */
function makeFalloff(settings: ActorTuning['shadow']): CanvasTexture {
  const canvas = document.createElement('canvas')
  canvas.width = settings.textureSize
  canvas.height = settings.textureSize
  const context = canvas.getContext('2d')
  if (context) {
    const half = settings.textureSize / 2
    const gradient = context.createRadialGradient(half, half, 0, half, half, half)
    for (const stop of settings.gradient) gradient.addColorStop(stop.position, stop.color)
    context.fillStyle = gradient
    context.fillRect(0, 0, settings.textureSize, settings.textureSize)
  }
  return new CanvasTexture(canvas)
}

/**
 * One instanced draw of soft contact discs under every actor. Cheaper and
 * steadier than a re-rendered contact-shadow pass: no extra scene render,
 * follows hops (the disc shrinks as the body rises) and works in every profile.
 */
export function ActorShadows({ tuning }: { tuning: ActorTuning }) {
  const settings = tuning.shadow
  const store = useWorldStore()
  const ids = store.getState().actorOrder
  const capacity = Math.max(1, ids.length)
  const meshRef = useRef<InstancedMesh | null>(null)
  const dummy = useMemo(() => new Object3D(), [])
  const poses = usePoseBuffer()
  const pose = useMemo<BodyPose>(() => ({ x: 0, y: 0, z: 0, facing: 0, scaleXZ: 0, scaleY: 0 }), [])
  const geometry = useMemo(() => new PlaneGeometry(2, 2).rotateX(-Math.PI / 2), [])
  const material = useMemo(() => {
    const alphaMap = makeFalloff(settings)
    return new MeshBasicMaterial({ color: settings.color, alphaMap, transparent: true, opacity: settings.opacity, depthWrite: false, toneMapped: false })
  }, [settings])
  useEffect(() => () => geometry.dispose(), [geometry])
  useEffect(() => () => {
    material.alphaMap?.dispose()
    material.dispose()
  }, [material])

  useFrame(() => {
    const mesh = meshRef.current
    if (!mesh) return
    const state = store.getState()
    ids.forEach((id, i) => {
      const actor = state.actors[id]
      if (!actor) return
      readPose(poses, i, pose)
      const hop = actor.anim.hopPhase > 0 ? Math.sin(actor.anim.hopPhase * Math.PI) : 0
      const radius = pose.scaleXZ * settings.spread * (1 - hop * settings.hopShrink)
      dummy.position.set(pose.x, heightAt(state.terrain, pose.x, pose.z) + settings.lift, pose.z)
      dummy.scale.set(radius, 1, radius)
      dummy.updateMatrix()
      mesh.setMatrixAt(i, dummy.matrix)
    })
    mesh.instanceMatrix.needsUpdate = true
  })

  return <instancedMesh ref={meshRef} args={[geometry, material, capacity]} frustumCulled={false} renderOrder={-1} />
}
