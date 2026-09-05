import { useThree, type ThreeEvent } from '@react-three/fiber'
import { useMemo, useRef } from 'react'
import { Plane, Raycaster, Vector2, Vector3 } from 'three'
import type { EditorTuning } from '../../config'
import type { Place, Vec2 } from '../../sim'
import { snapPosition } from '../../sim'
import { useWorldStore } from '../WorldStoreContext'

interface RoomHandlesProps {
  editor: EditorTuning
  selectedRoomId: string | null
  onSelectRoom: (roomId: string | null) => void
  /** Called continuously while dragging (preview) and once on release (commit). */
  onMove: (roomId: string, position: Vec2, commit: boolean) => void
  /** Disable the camera controls while a drag is active. */
  onDragging: (dragging: boolean) => void
}

/**
 * Edit-mode drag handles over every room: a translucent plate that captures
 * the pointer, projects it onto the ground plane and snaps to the editor
 * grid. Live preview goes through the store; the commit lands in history.
 */
export function RoomHandles({ editor, selectedRoomId, onSelectRoom, onMove, onDragging }: RoomHandlesProps) {
  const store = useWorldStore()
  const state = store.getState()
  const camera = useThree((s) => s.camera)
  const size = useThree((s) => s.size)
  const ground = useMemo(() => new Plane(new Vector3(0, 1, 0), 0), [])
  const raycaster = useMemo(() => new Raycaster(), [])
  const hit = useMemo(() => new Vector3(), [])
  const ndc = useMemo(() => new Vector2(), [])
  const drag = useRef<{ roomId: string; offset: Vec2 } | null>(null)
  const rooms = useMemo(() => state.placeOrder.map((id) => state.places[id]).filter((p): p is Place => p?.kind === 'room'), [state.placeOrder, state.places])

  const groundPoint = (event: ThreeEvent<PointerEvent>): Vec2 | null => {
    const rect = (event.nativeEvent.target as HTMLElement | null)?.getBoundingClientRect()
    const width = rect?.width ?? size.width
    const height = rect?.height ?? size.height
    const left = rect?.left ?? 0
    const top = rect?.top ?? 0
    ndc.set(((event.nativeEvent.clientX - left) / width) * 2 - 1, -((event.nativeEvent.clientY - top) / height) * 2 + 1)
    raycaster.setFromCamera(ndc, camera)
    return raycaster.ray.intersectPlane(ground, hit) ? [hit.x, hit.z] : null
  }

  return (
    <group name="room-handles">
      {rooms.map((room) => (
        <mesh
          key={room.id}
          position={[room.position[0], editor.handleLift, room.position[1]]}
          rotation={[-Math.PI / 2, 0, room.rotation]}
          onPointerDown={(event) => {
            event.stopPropagation()
            const point = groundPoint(event)
            if (!point) return
            onSelectRoom(room.id)
            drag.current = { roomId: room.id, offset: [room.position[0] - point[0], room.position[1] - point[1]] }
            onDragging(true)
            ;(event.nativeEvent.target as HTMLElement | null)?.setPointerCapture(event.nativeEvent.pointerId)
          }}
          onPointerMove={(event) => {
            const active = drag.current
            if (!active || active.roomId !== room.id) return
            const point = groundPoint(event)
            if (!point) return
            onMove(room.id, snapPosition([point[0] + active.offset[0], point[1] + active.offset[1]], editor.snap), false)
          }}
          onPointerUp={(event) => {
            const active = drag.current
            if (!active || active.roomId !== room.id) return
            drag.current = null
            onDragging(false)
            const point = groundPoint(event)
            const live = store.getState().places[room.id]
            const final = point ? snapPosition([point[0] + active.offset[0], point[1] + active.offset[1]], editor.snap) : live?.position ?? room.position
            onMove(room.id, final, true)
          }}
        >
          <planeGeometry args={[room.size[0], room.size[1]]} />
          <meshBasicMaterial color={selectedRoomId === room.id ? editor.selectedColor : editor.handleColor} transparent opacity={selectedRoomId === room.id ? editor.selectedOpacity : editor.handleOpacity} depthWrite={false} />
        </mesh>
      ))}
    </group>
  )
}
