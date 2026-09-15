/**
 * Accessible drag-and-drop ordering primitives for the objective editor.
 *
 * Pointer dragging is an enhancement over the always-present move-up/move-down
 * buttons. The keyboard sensor keeps drag reordering reachable without a
 * pointer, so the same list supports touch, mouse and keyboard operation.
 */

import {
  DndContext,
  KeyboardSensor,
  PointerSensor,
  closestCenter,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DraggableAttributes,
  type DraggableSyntheticListeners,
} from '@dnd-kit/core'
import { SortableContext, sortableKeyboardCoordinates, useSortable, verticalListSortingStrategy } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import type { ReactNode } from 'react'

export interface SortableHandle {
  attributes: DraggableAttributes
  listeners: DraggableSyntheticListeners
  isDragging: boolean
}

export function SortableList({ ids, onReorder, children }: {
  ids: string[]
  onReorder: (from: number, to: number) => void
  children: ReactNode
}) {
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  )

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event
    if (!over || active.id === over.id) return
    const from = ids.indexOf(String(active.id))
    const to = ids.indexOf(String(over.id))
    if (from < 0 || to < 0) return
    onReorder(from, to)
  }

  return <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
    <SortableContext items={ids} strategy={verticalListSortingStrategy}>{children}</SortableContext>
  </DndContext>
}

export function SortableRow({ id, className, render }: {
  id: string
  className?: string
  render: (handle: SortableHandle) => ReactNode
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id })
  return <li
    ref={setNodeRef}
    style={{ transform: CSS.Transform.toString(transform), transition, opacity: isDragging ? 0.6 : 1 }}
    className={className}
  >
    {render({ attributes, listeners, isDragging })}
  </li>
}
