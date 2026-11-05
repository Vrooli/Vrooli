import { fireEvent, render, screen } from '@/test-utils/renderWithProviders'
import { describe, expect, it, vi } from 'vitest'
import { overridesFromWire, overridesToWire } from '../../data/layout'
import { EditorToolbar } from '../editor/Toolbar'

function props(overrides: Partial<Parameters<typeof EditorToolbar>[0]> = {}) {
  return {
    editing: true,
    onEditingChange: vi.fn(),
    canUndo: false,
    canRedo: false,
    onUndo: vi.fn(),
    onRedo: vi.fn(),
    onReset: vi.fn(),
    selectedRoomLabel: null,
    onRemoveSelected: vi.fn(),
    overrideCount: 0,
    saving: false,
    ...overrides,
  }
}

describe('EditorToolbar', () => {
  it('enables undo and redo only when history allows and reports the change count', () => {
    const { rerender } = render(<EditorToolbar {...props()} />)
    expect(screen.getByTestId('world-editor-undo')).toBeDisabled()
    expect(screen.getByTestId('world-editor-redo')).toBeDisabled()
    expect(screen.getByTestId('world-editor-reset')).toBeDisabled()
    expect(screen.getByTestId('world-editor-status')).toHaveTextContent('0 changes')
    const p = props({ canUndo: true, canRedo: true, overrideCount: 2, selectedRoomLabel: 'Alpha' })
    rerender(<EditorToolbar {...p} />)
    expect(screen.getByTestId('world-editor-undo')).toBeEnabled()
    fireEvent.click(screen.getByTestId('world-editor-undo'))
    fireEvent.click(screen.getByTestId('world-editor-redo'))
    fireEvent.click(screen.getByTestId('world-editor-remove'))
    expect(p.onUndo).toHaveBeenCalled()
    expect(p.onRedo).toHaveBeenCalled()
    expect(p.onRemoveSelected).toHaveBeenCalled()
    expect(screen.getByTestId('world-editor-status')).toHaveTextContent('2 changes')
  })

  it('collapses to the toggle when not editing', () => {
    const p = props({ editing: false })
    render(<EditorToolbar {...p} />)
    expect(screen.queryByTestId('world-editor-undo')).not.toBeInTheDocument()
    fireEvent.click(screen.getByTestId('world-editor-toggle'))
    expect(p.onEditingChange).toHaveBeenCalledWith(true)
  })
})

describe('layout wire mapping', () => {
  it('round-trips overrides through the WorldService shape', () => {
    const overrides = [
      { placeId: 'room:a', position: [3, -4] as const, rotation: 0.5 },
      { placeId: 'room:b', removed: true },
    ]
    const wire = overridesToWire('park', overrides)
    expect(wire.scene).toBe('park')
    expect(wire.overrides[0]).toEqual({ placeId: 'room:a', position: { x: 3, z: -4 }, rotation: 0.5, removed: false })
    const back = overridesFromWire(wire)
    expect(back).toEqual([
      { placeId: 'room:a', position: [3, -4], rotation: 0.5, removed: undefined },
      { placeId: 'room:b', position: undefined, rotation: undefined, removed: true },
    ])
  })
})
