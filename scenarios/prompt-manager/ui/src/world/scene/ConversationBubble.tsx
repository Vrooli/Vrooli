import { Html } from '@react-three/drei'
import { useFrame } from '@react-three/fiber'
import { useRef, useState, type ReactNode } from 'react'
import { Group, Vector3 } from 'three'
import { conversationReady } from '../sim/conversation'
import { useWorldStore } from './WorldStoreContext'
import { POSE, POSE_STRIDE, usePoseBuffer } from './actors/PoseBuffer'

/** Scene owns arrival and projection; the route supplies the HUD contents. */
export function ConversationBubble({ id, walking, children }: { id: string; walking: boolean; children: ReactNode }) {
  const store = useWorldStore()
  const poses = usePoseBuffer()
  const group = useRef<Group>(null)
  const [ready, setReady] = useState(false)
  const readyRef = useRef(false)
  useFrame(() => {
    const state = store.getState(), index = state.actorOrder.indexOf(id), offset = index * POSE_STRIDE
    if (group.current && index >= 0) group.current.position.set(poses.data[offset + POSE.x] ?? 0, (poses.data[offset + POSE.y] ?? 0) + .8, poses.data[offset + POSE.z] ?? 0)
    const next = conversationReady(state, id, walking)
    if (next !== readyRef.current) { readyRef.current = next; setReady(next) }
  })
  return <group ref={group} name="member-conversation" userData={{ walkObstacle: false }}>
    {ready && <Html center zIndexRange={[45, 40]} calculatePosition={(object, camera, size) => {
      const position = new Vector3().setFromMatrixPosition(object.matrixWorld).project(camera)
      return [Math.max(size.width > 900 ? 420 : 170, Math.min(size.width - 170, (position.x + 1) * size.width / 2)), Math.max(270, Math.min(size.height - 250, (1 - position.y) * size.height / 2))]
    }}>{children}</Html>}
  </group>
}
