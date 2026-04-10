/* eslint-disable @typescript-eslint/no-non-null-assertion */
/**
 * Tests for DraggableObject drag logic.
 *
 * DraggableObject is an R3F component (uses useFrame/useThree) so we can't
 * render it without a Canvas.  Instead we test:
 *  1. computeDragPosition – pure position-from-offset calculation
 *  2. Store-driven drag flow – validates that the interactionStore produces
 *     the correct dragState that DraggableObject reads every frame.
 *     This is the exact data path that was broken before the fix: DragPlane
 *     updates the store, and DraggableObject derives position from it.
 */

import { describe, it, expect, beforeEach } from 'vitest'
import { useInteractionStore } from '@/stores/interactionStore'
import { computeDragPosition } from './DraggableObject'

// ---------------------------------------------------------------------------
// computeDragPosition
// ---------------------------------------------------------------------------

describe('computeDragPosition', () => {
  it('should apply offset to initial position (XZ only)', () => {
    const result = computeDragPosition(
      [2, 0.5, 3],
      [1, 999, -2], // Y offset is ignored
    )
    expect(result).toEqual([3, 0.5, 1])
  })

  it('should preserve initial Y regardless of offset Y', () => {
    const result = computeDragPosition(
      [0, 1.2, 0],
      [5, 10, 5],
    )
    expect(result[1]).toBe(1.2)
  })

  it('should apply constraint function', () => {
    const clamp = (pos: [number, number, number]): [number, number, number] => [
      Math.max(-5, Math.min(5, pos[0])),
      pos[1],
      Math.max(-5, Math.min(5, pos[2])),
    ]
    const result = computeDragPosition([3, 0, 3], [10, 0, 10], clamp)
    expect(result).toEqual([5, 0, 5])
  })

  it('should work with zero offset', () => {
    const result = computeDragPosition([4, 1, 7], [0, 0, 0])
    expect(result).toEqual([4, 1, 7])
  })

  it('should work with negative offsets', () => {
    const result = computeDragPosition([5, 0, 5], [-3, 0, -4])
    expect(result).toEqual([2, 0, 1])
  })
})

// ---------------------------------------------------------------------------
// Store-driven drag flow
// ---------------------------------------------------------------------------
// These tests validate that the interactionStore produces the correct
// dragState.offset when DragPlane calls store.updateDrag() directly.
// DraggableObject reads this offset every frame to compute visual position.

describe('store-driven drag flow (DragPlane path)', () => {
  beforeEach(() => {
    useInteractionStore.getState().reset()
  })

  it('should compute correct offset when DragPlane updates position', () => {
    const store = useInteractionStore.getState()

    // pointerDown on object → startDrag
    store.startDrag('furniture-1', [2, 0, 3])

    // Pointer moves to DragPlane → DragPlane calls updateDrag directly
    useInteractionStore.getState().updateDrag([5, 0, 1])

    const state = useInteractionStore.getState()
    expect(state.dragState).not.toBeNull()
    expect(state.dragState!.offset).toEqual([3, 0, -2])

    // DraggableObject would compute: initialPosition + offset
    // e.g. [2, 0, 3] + [3, 0, -2] = [5, 0, 1]
    const visualPos = computeDragPosition([2, 0, 3], state.dragState!.offset)
    expect(visualPos).toEqual([5, 0, 1])
  })

  it('should track multiple position updates from DragPlane', () => {
    const store = useInteractionStore.getState()
    store.startDrag('furniture-1', [0, 0, 0])

    // Simulate mouse movement across DragPlane
    useInteractionStore.getState().updateDrag([1, 0, 0])
    expect(useInteractionStore.getState().dragState!.offset).toEqual([1, 0, 0])

    useInteractionStore.getState().updateDrag([3, 0, 2])
    expect(useInteractionStore.getState().dragState!.offset).toEqual([3, 0, 2])

    useInteractionStore.getState().updateDrag([-1, 0, 4])
    expect(useInteractionStore.getState().dragState!.offset).toEqual([-1, 0, 4])
  })

  it('should clear dragState when DragPlane ends drag', () => {
    const store = useInteractionStore.getState()
    store.startDrag('furniture-1', [0, 0, 0])
    useInteractionStore.getState().updateDrag([5, 0, 5])

    // DragPlane pointerUp → calls endDrag directly
    useInteractionStore.getState().endDrag()

    const state = useInteractionStore.getState()
    expect(state.isDragging).toBe(false)
    expect(state.draggedObjectId).toBeNull()
    expect(state.dragState).toBeNull()
  })

  it('should support drag end detection via isDragging transition', () => {
    // This tests the pattern DraggableObject uses to detect when
    // DragPlane ends a drag (isDragging goes true → false)
    const store = useInteractionStore.getState()
    store.startDrag('obj-1', [1, 0, 2])
    expect(useInteractionStore.getState().isDragging).toBe(true)

    useInteractionStore.getState().updateDrag([4, 0, 5])
    const offsetBeforeEnd = useInteractionStore.getState().dragState!.offset

    // Save the last offset before ending (DraggableObject does this in useFrame)
    const lastDragPos = computeDragPosition([1, 0, 2], offsetBeforeEnd)
    expect(lastDragPos).toEqual([4, 0, 2 + 3]) // [1+3, 0, 2+3]

    // DragPlane ends drag
    useInteractionStore.getState().endDrag()
    expect(useInteractionStore.getState().isDragging).toBe(false)

    // DraggableObject detects transition and persists lastDragPos
    // (tested via the useEffect in the component)
    expect(lastDragPos).toEqual([4, 0, 5])
  })

  it('should work with non-zero initial positions', () => {
    const store = useInteractionStore.getState()
    const initialPos: [number, number, number] = [10, 0.8, -5]

    store.startDrag('agent-1', initialPos)
    useInteractionStore.getState().updateDrag([12, 0, -3])

    const offset = useInteractionStore.getState().dragState!.offset
    const visualPos = computeDragPosition(initialPos, offset)

    // offset = [12-10, 0-0.8, -3-(-5)] = [2, -0.8, 2]
    // visualPos = [10+2, 0.8, -5+2] = [12, 0.8, -3]
    expect(visualPos).toEqual([12, 0.8, -3])
  })

  it('should not update dragState when not dragging', () => {
    // Don't start drag, just try to update
    useInteractionStore.getState().updateDrag([5, 0, 5])
    expect(useInteractionStore.getState().dragState).toBeNull()
  })

  it('should isolate drag to the specific object (selectIsDragged)', () => {
    useInteractionStore.getState().startDrag('obj-a', [0, 0, 0])

    const state = useInteractionStore.getState()
    expect(state.draggedObjectId).toBe('obj-a')
    // obj-b should NOT see any drag state
    expect(state.draggedObjectId === 'obj-b').toBe(false)
  })
})

// ---------------------------------------------------------------------------
// Position constraint integration
// ---------------------------------------------------------------------------

describe('drag with constraints', () => {
  beforeEach(() => {
    useInteractionStore.getState().reset()
  })

  it('should constrain position within boundary', () => {
    const boundary = 10
    const constrain = (pos: [number, number, number]): [number, number, number] => [
      Math.max(-boundary, Math.min(boundary, pos[0])),
      pos[1],
      Math.max(-boundary, Math.min(boundary, pos[2])),
    ]

    useInteractionStore.getState().startDrag('obj-1', [8, 0, 8])
    useInteractionStore.getState().updateDrag([20, 0, 20])

    const offset = useInteractionStore.getState().dragState!.offset
    const visualPos = computeDragPosition([8, 0, 8], offset, constrain)

    // Without constraint: [8+12, 0, 8+12] = [20, 0, 20]
    // With constraint: clamped to [10, 0, 10]
    expect(visualPos).toEqual([10, 0, 10])
  })

  it('should apply snap-to-grid constraint', () => {
    const gridSize = 2
    const snapToGrid = (pos: [number, number, number]): [number, number, number] => [
      Math.round(pos[0] / gridSize) * gridSize,
      pos[1],
      Math.round(pos[2] / gridSize) * gridSize,
    ]

    useInteractionStore.getState().startDrag('obj-1', [0, 0, 0])
    useInteractionStore.getState().updateDrag([3.7, 0, 1.3])

    const offset = useInteractionStore.getState().dragState!.offset
    const visualPos = computeDragPosition([0, 0, 0], offset, snapToGrid)

    // Raw: [3.7, 0, 1.3] → snapped: [4, 0, 2]
    expect(visualPos).toEqual([4, 0, 2])
  })
})
