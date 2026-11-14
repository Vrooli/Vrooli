import { useThree, type ThreeEvent } from '@react-three/fiber'
import { useEffect, useMemo, useRef, useState } from 'react'
import { Plane, Raycaster, Vector2, Vector3 } from 'three'
import type { EditorTuning } from '../../config'
import type { Place, Vec2 } from '../../sim'
import { heightAt, snapPosition } from '../../sim'
import { useWorldStore } from '../WorldStoreContext'

interface RoomHandlesProps {
  editor: EditorTuning
  selectedRoomId: string | null
  onSelectRoom: (roomId: string | null) => void
  /** Called once on release; the footprint preview is local to this component. */
  onMove: (roomId: string, position: Vec2) => void
  /** Disable the camera controls while a drag is active. */
  onDragging: (dragging: boolean) => void
}

/**
 * Edit-mode drag handles over every room: a translucent plate that captures
 * the pointer, projects it onto the ground plane and snaps to the editor
 * grid. The footprint previews locally; only release commits a layout change.
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
  const drag = useRef<{ roomId: string; offset: Vec2; position: Vec2; pointerId: number; target: HTMLElement | null } | null>(null)
  const [preview, setPreview] = useState<{ roomId: string; position: Vec2 } | null>(null)
  const draggingCallback = useRef(onDragging)
  draggingCallback.current = onDragging
  useEffect(() => {
    const cancel = () => {
      if (!drag.current) return
      const active = drag.current
      drag.current = null
      if (active.target?.hasPointerCapture(active.pointerId)) active.target.releasePointerCapture(active.pointerId)
      setPreview(null)
      draggingCallback.current(false)
    }
    const escape = (event: KeyboardEvent) => {
      if (event.key !== 'Escape' || !drag.current) return
      event.preventDefault()
      event.stopImmediatePropagation()
      cancel()
    }
    window.addEventListener('pointercancel', cancel)
    window.addEventListener('lostpointercapture', cancel, true)
    window.addEventListener('blur', cancel)
    window.addEventListener('keydown', escape, true)
    return () => {
      window.removeEventListener('pointercancel', cancel)
      window.removeEventListener('lostpointercapture', cancel, true)
      window.removeEventListener('blur', cancel)
      window.removeEventListener('keydown', escape, true)
      const active = drag.current
      drag.current = null
      if (active?.target?.hasPointerCapture(active.pointerId)) active.target.releasePointerCapture(active.pointerId)
      if (active) draggingCallback.current(false)
    }
  }, [])
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
          position={preview?.roomId === room.id ? [preview.position[0], heightAt(state.terrain, ...preview.position) + editor.handleLift, preview.position[1]] : [room.position[0], heightAt(state.terrain, ...room.position) + editor.handleLift, room.position[1]]}
          rotation={[-Math.PI / 2, 0, room.rotation]}
          onPointerDown={(event) => {
            event.stopPropagation()
            ground.constant = -heightAt(state.terrain, ...room.position)
            const point = groundPoint(event)
            if (!point) return
            onSelectRoom(room.id)
            drag.current = { roomId: room.id, offset: [room.position[0] - point[0], room.position[1] - point[1]], position: room.position, pointerId: event.nativeEvent.pointerId, target: event.target as unknown as HTMLElement }
            onDragging(true)
            ;(event.target as unknown as HTMLElement).setPointerCapture(event.nativeEvent.pointerId)
          }}
          onPointerMove={(event) => {
            const active = drag.current
            if (!active || active.roomId !== room.id) return
            const point = groundPoint(event)
            if (!point) return
            const position = snapPosition([point[0] + active.offset[0], point[1] + active.offset[1]], editor.snap)
            active.position = position
            setPreview({ roomId: room.id, position })
          }}
          onPointerUp={(event) => {
            const active = drag.current
            if (!active || active.roomId !== room.id) return
            drag.current = null
            setPreview(null)
            onDragging(false)
            if (active.target?.hasPointerCapture(active.pointerId)) active.target.releasePointerCapture(active.pointerId)
            const point = groundPoint(event)
            const final = point ? snapPosition([point[0] + active.offset[0], point[1] + active.offset[1]], editor.snap) : active.position
            onMove(room.id, final)
          }}
        >
          <planeGeometry args={[room.size[0], room.size[1]]} />
          <meshBasicMaterial color={selectedRoomId === room.id ? editor.selectedColor : editor.handleColor} transparent opacity={selectedRoomId === room.id ? editor.selectedOpacity : editor.handleOpacity} depthWrite={false} />
        </mesh>
      ))}
    </group>
  )
}
